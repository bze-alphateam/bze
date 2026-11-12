# Burner Module – Technical Notes

This file captures implementation details for integrators and operators. For user-facing actions see `README.md`.

## Module Accounts
- `burner`: holds coins sent via `MsgFundBurner`; contents are periodically burned.
- `burner_raffle`: holds raffle pots and ticket payments.
- `burner_black_hole`: sink account that locks the coins that cannot be burned: LP tokens and IBC tokens.

## Burn Pipeline
- Triggered by the weekly epoch hook when `epochNumber % periodic_burning_weeks == 0`.
- `BurnAnyCoins` processes each coin:
  - Native and token factory denoms are burned directly.
  - LP tokens are moved to `burner_black_hole` (supply unchanged, effectively frozen).
  - IBC tokens (exchangeable or not) are moved to `burner_black_hole`. `BurnAnyCoins` makes no trade module call besides `IsNativeDenom`; it never adds liquidity. The raffle cleanup uses the same path for leftover pots.
- Each successful burn emits `CoinsBurnedEvent` and appends/aggregates an entry in `burnedCoins` keyed by block height.
- Guardian pause: `ProcessPeriodicBurnQueue()` returns before reading any balance while the x/guardian protocol automation pause is on (read through the one-method `GuardianKeeper` interface, `IsAutomationPaused`). The epoch hook still enqueues (it only writes the `Pending` flag), so the queue stays pending through the pause and the first EndBlock after the resume burns the whole balance. The burn is balance based, so several burning weeks during the pause add up to one burn. `BurnAnyCoins` is not gated: the raffle cleanup keeps burning finished raffles' pots.

## Moving Locked IBC Coins
- `MsgMoveIbcLockedCoins` is permissionless. It adds black-hole coins of one IBC denom to the denom's pool with native BZE through `SafeModuleAddLiquidityWithNativeDenom`, and sends the refunded native coins to `burner`.
- Before any coin moves, it checks, in order:
  - **Cooldown:** the store keeps, per denom, the height of the last successful move (prefix `iml/` + denom → `IbcMoveLock`). A move is refused (`ErrIbcMoveCooldown`) while `height - last_move_height < ibc_move_cooldown_blocks`. This check runs before any trade module call. The lock is written only after a successful move. A denom that was never moved has no lock.
  - **Cap:** the pool id comes from `CreatePoolId(native_denom, denom)`, and the IBC reserve is `reserve_base` or `reserve_quote`, whichever side holds the denom. The move takes `min(locked balance, floor(ibc_move_max_reserve_ratio × reserve))`. With no pool or a cap of 0 it is refused (`ErrIbcMoveNoCapacity`).
- The cap uses the pool's live reserve at the time of the move, not the price average. A swap earlier in the same block that grows the reserve is bounded by the module price-drift guard of `SafeModuleAddLiquidityWithNativeDenom`.
- `CanSwapForNativeDenom` and the zap then run on the capped coin. The rest of the balance stays locked.
- The burner reads tradebin only through its `TradeKeeper` interface: `GetParams` (native denom), `CreatePoolId`, `GetLiquidityPool`.
- The locks are exported and imported in genesis (`ibc_move_locks`; denoms must be unique and non-empty).

## Raffle Mechanics
- **Epoch timing:** Raffle durations are counted in hourly epochs (`duration * 24`). Cleanup happens in an epoch hook keyed to the `"hour"` identifier.
- **Starting:** `MsgStartRaffle` verifies denom supply, captures the initial pot from the creator into `burner_raffle`, and sets an expiration height (`current_hour_epoch + duration*24`).
- **Joining:** `MsgJoinRaffle` schedules each ticket for resolution at `block_height + 2 + ticket_index` and collects the ticket price into `burner_raffle`. Maximum 50 tickets per message.
- **Resolution:** EndBlock runs `WithdrawLuckyRaffleParticipants` for the current height:
  - A deterministic pseudo-random check (`< chances` against a 0–999,999 range) combines the block header hash, app hash, and participant address.
  - Winners receive `ratio * current_pot` (truncated), pot is reduced, and a `RaffleWinnerEvent` is emitted.
  - Losers add their ticket price to the pot and trigger `RaffleLostEvent`.
  - Winner records are stored modulo 100 indexes per denom (only the latest 100 kept).
- **Expiration:** At the configured end epoch, remaining pot is burned via `BurnAnyCoins`, the raffle and winner records are deleted, and `RaffleFinishedEvent` is emitted.

## Queries and Storage
- Raffles and participants are stored under prefixed keys for pagination; burned coin history is keyed by block height.
- gRPC/REST routes are exposed for `Params`, `Raffles`, `RaffleWinners`, and `AllBurnedCoins` (`/bze/burner/...`).

## Parameters and Authority
- `periodic_burning_weeks` is the only parameter; authority-gated updates use `MsgUpdateParams`.
- The module expects the epoch module to provide `"week"` and `"hour"` identifiers to drive hooks; misconfiguration will disable scheduled burns/cleanup.

## Version History
### v8.3.0
- `BurnAnyCoins()` no longer adds exchangeable IBC coins as liquidity: they are locked in `burner_black_hole` like every other non-burnable coin (`addLiquidityWithNative` removed)
- `MsgMoveIbcLockedCoins` gets a per-call cap (`ibc_move_max_reserve_ratio`, default 0.03) and a per-denom cooldown (`ibc_move_cooldown_blocks`, default 1000), with the per-denom lock in the store under `iml/` and in genesis as `ibc_move_locks`
- Consensus version 5: `Migrate4to5` writes the defaults of the two new params into the stored params
- The periodic burn queue is not processed while the x/guardian protocol automation pause is on; new `GuardianKeeper` dependency (depinject)
- `ProcessPeriodicBurnQueue()` reads at most 101 balances of the burner account per block (bounded `GetBalancesPage` instead of `GetAllBalances`); it burns the first 100 and keeps the queue pending only when the 101st exists

### v8.1.0
- Periodic burning moved from synchronous epoch hook to queued EndBlock processing: `EnqueuePeriodicBurn()` queues the work, `ProcessPeriodicBurnQueue()` processes up to 100 denoms per block
- Raffle cleanup moved from synchronous epoch hook to queued EndBlock processing: `EnqueueRaffleCleanup()` queues epochs, `ProcessRaffleCleanupQueue()` processes up to 50 raffles per block
- IBC token strategy in `BurnAnyCoins()` changed from `ModuleSwapForNativeDenom` to `ModuleAddLiquidityWithNativeDenom`
- `MsgFundBurner` now classifies coins before sending: lockable (LP) to black hole, burnable/exchangeable to burner module
- Added `MsgMoveIbcLockedCoins` (permissionless) to recover IBC coins from the black-hole module via liquidity addition
- `GetRaffleCurrentEpoch()` now returns `(uint64, error)` using `SafeGetEpochCountByIdentifier` for proper error handling
- Rate limit: max 200 raffle participants per block height; minimum pot of 100,000 units enforced
