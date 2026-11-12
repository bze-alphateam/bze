# Burner Module – User Guide

The burner module lets any user permanently remove tokens from circulation and run luck-based raffles that also destroy unclaimed prizes over time. This guide focuses on how to interact with it from a wallet/CLI point of view.

## What You Can Do
- Contribute coins to the burn pool — native and token factory coins are destroyed on the next scheduled burn, LP tokens are permanently locked, and IBC tokens are locked; anyone can later move locked IBC tokens into their BZE pool, a capped amount at a time.
- Start a raffle for a specific denom, providing the initial prize pot and odds.
- Join an existing raffle by buying tickets and see whether you win a share of the pot.
- Move locked IBC coins from the permanent lock account into liquidity pools paired with native BZE.
- Query current raffles, past winners, and the history of burned coins.

## Burning Coins
Coins sent to the module are held until the periodic burn hook runs (by default once per week). At each burn, the module classifies every coin it holds and handles them according to their type:

- **Native and token factory coins** are burned directly (removed from supply).
- **LP tokens** cannot be burned. They are sent to a **permanent lock account** (`burner_black_hole`) where they remain forever, effectively frozen. The supply is unchanged but the tokens can never be moved or redeemed.
- **IBC tokens** are sent to the permanent lock account, whether or not they have a pool with BZE. The burn never adds them to a pool itself: `MsgMoveIbcLockedCoins` (below) is the only way they reach one, in capped steps.

Example CLI transaction:
```bash
bzed tx burner fund-burner 1000000ubze --from mykey
```
You can send multiple denominations in a single `fund-burner` call (e.g. `1000000ubze,500ulp_token`). The module will classify and route each coin automatically.

After the next scheduled burn, the burned amount will appear in `burnedCoins` queries (see Queries below).

The x/guardian guardian (or governance) can pause the periodic burn together with the other protocol automation (`MsgPauseAutomation`); only governance resumes it. While paused, the coins stay in the burner. A burning week that ends during the pause is not skipped: the first block after the resume burns everything the burner holds by then. Raffles are not paused.

## Raffles
Raffles are per-denom and run for a set duration. Tickets settle a couple of blocks after purchase, so results come quickly.

### Start a Raffle
You fund the initial pot and define the rules:
- `pot`: starting prize pool in the raffle denom (must be in your balance).
- `duration`: number of days the raffle stays open (1–180). Internally this is `duration * 24` hourly epochs.
- `chances`: winning threshold between 1 and 1,000,000 (higher = better odds per ticket).
- `ratio`: decimal between 0.01 and 1.00; the winner receives `ratio * current_pot`.
- `ticket_price`: cost per ticket in the same denom.
- `denom`: the coin being raffled.

Example:
```bash
bzed tx burner start-raffle \
  --pot 10000000ubze \
  --duration 7 \
  --chances 500000 \
  --ratio 0.25 \
  --ticket-price 250000ubze \
  --denom ubze \
  --from mykey
```

### Join a Raffle
- Choose the denom of the active raffle.
- `tickets`: 1–50 tickets per transaction; each costs `ticket_price`.
- Results are evaluated two blocks after submission; each ticket is checked in consecutive blocks.
- If a ticket wins, you receive `ratio * pot` at that moment; otherwise your ticket price is added to the pot.

Example:
```bash
bzed tx burner join-raffle \
  --denom ubze \
  --tickets 3 \
  --from mykey
```

### What Happens After the Raffle Ends
- Duration is tracked in hourly epochs; after `duration` days elapse, the raffle is cleaned up.
- Remaining pot is burned; a `RaffleFinishedEvent` is emitted.
- Up to the latest 100 winners per denom are kept (older ones roll over).

## Queries (CLI / REST / gRPC)
- `bzed query burner raffles` – list active raffles.
- `bzed query burner raffle-winners --denom <denom>` – see recent winners for a denom.
- `bzed query burner all-burned-coins` – paginated list of burned amounts by block height.
- `bzed query burner params` – view current module parameters, including burn frequency.

REST routes (if using REST): `/bze/burner/raffles`, `/bze/burner/raffle_winners?denom=<denom>`, `/bze/burner/all_burned_coins`, `/bze/burner/params`.

## Moving Locked IBC Coins to Liquidity
IBC tokens should never be burned — instead they can be put to use. Every IBC token the burner receives ends up in the permanent lock account, and anyone can call `MsgMoveIbcLockedCoins` to move them into the liquidity pool paired with native BZE. Only IBC denominations are eligible (LP tokens are rejected). On success the resulting LP tokens stay in the lock account, and any refunded native coins are sent back to the burner module for the next burn cycle.

Each move is limited, so a large locked balance can't be dumped into a pool at once:
- **Cap:** one move adds at most `ibc_move_max_reserve_ratio` (default 3%) of the pool's current reserve of that IBC denom. The rest stays locked for later moves. Without a BZE pool for the denom, or when the cap rounds down to 0, the move is refused.
- **Cooldown:** after a successful move of a denom, the next move of that denom is refused until `ibc_move_cooldown_blocks` (default 1000) blocks have passed. Other denoms are not affected, and a failed move does not start the cooldown.

Example:
```bash
bzed tx burner move-ibc-locked-coins --denom <ibc-denom> --from mykey
```

## Permanent Lock Account
The module registers a dedicated account named `burner_black_hole`. This account has no private key and no signing capability — coins sent to it can never be transferred out. It serves as the destination for:
- LP tokens collected through `fund-burner` or the periodic burn.
- IBC tokens the burner receives (the periodic burn and the raffle cleanup lock them).
- LP tokens produced when IBC coins are converted to liquidity.

## Governance and Permissions
- Parameter changes go through `MsgUpdateParams` and are restricted to the module authority (typically the gov module). Users normally do not send this message directly.
- All other messages (`fund-burner`, `start-raffle`, `join-raffle`, `move-ibc-locked-coins`) are open to any account.

## Version History

### v8.3.0
- The periodic burn and the raffle cleanup lock IBC tokens in the black hole instead of adding them to their pool; `MsgMoveIbcLockedCoins` is the only way they reach a pool
- `MsgMoveIbcLockedCoins` moves at most `ibc_move_max_reserve_ratio` of the pool's reserve of the denom per call (default 3%) and refuses a denom moved less than `ibc_move_cooldown_blocks` blocks ago (default 1000); the rest stays locked for later calls
- The periodic burn waits while the x/guardian protocol automation pause is on; a burning week that ended during the pause is burned right after governance resumes it

### v8.1.0
- Queued periodic burning: burns now processed at EndBlock in bounded batches (up to 100 denoms/block) instead of synchronously in epoch hooks
- Queued raffle cleanup: expired raffles processed at EndBlock in bounded batches (up to 50/block) instead of synchronously in epoch hooks
- IBC token burn strategy changed from `ModuleSwapForNativeDenom` to `ModuleAddLiquidityWithNativeDenom`
- The burner calls `SafeModuleAddLiquidityWithNativeDenom`: when the coin's pool price is away from its moving average the coin is locked in the black hole (periodic burn) or `MsgMoveIbcLockedCoins` fails, and it can be sent again later
- `MsgFundBurner` now classifies coins before sending: lockable (LP) to black hole, burnable/exchangeable to burner module
- Added `MsgMoveIbcLockedCoins` (permissionless) to move locked IBC coins from the black-hole account into liquidity pools paired with native BZE
- Raffle participation rate-limited to 200 participants per block height
- Minimum pot enforcement: raffles require at least 100,000 smallest units
- Stricter raffle expiration check: rejects joins when 2 or fewer epochs remain
- `GetRaffleCurrentEpoch()` now returns `(uint64, error)` using `SafeGetEpochCountByIdentifier` for proper error handling
- `periodic_burning_weeks` parameter validation now requires value > 0 (disallows zero)
