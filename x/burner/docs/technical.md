# Burner Module – Technical Notes

This file captures implementation details for integrators and operators. For user-facing actions see `README.md`.

## Module Accounts
- `burner`: holds coins sent via `MsgFundBurner`; contents are periodically burned.
- `burner_raffle`: holds raffle pots and ticket payments.
- `burner_black_hole`: sink account used to lock LP tokens that cannot be burned.

## Burn Pipeline
- Triggered by the weekly epoch hook when `epochNumber % periodic_burning_weeks == 0`.
- `BurnAnyCoins` processes each coin:
  - Native and token factory denoms are burned directly.
  - LP tokens are moved to `burner_black_hole` (supply unchanged, effectively frozen).
  - IBC tokens are swapped to native via the trade module when possible, then burned; otherwise they remain untouched.
- Each successful burn emits `CoinsBurnedEvent` and appends/aggregates an entry in `burnedCoins` keyed by block height.

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

### v8.1.0
- Periodic burning moved from synchronous epoch hook to queued EndBlock processing: `EnqueuePeriodicBurn()` queues the work, `ProcessPeriodicBurnQueue()` processes up to 100 denoms per block
- Raffle cleanup moved from synchronous epoch hook to queued EndBlock processing: `EnqueueRaffleCleanup()` queues epochs, `ProcessRaffleCleanupQueue()` processes up to 50 raffles per block
- IBC token strategy in `BurnAnyCoins()` changed from `ModuleSwapForNativeDenom` to `ModuleAddLiquidityWithNativeDenom`
- `MsgFundBurner` now classifies coins before sending: lockable (LP) to black hole, burnable/exchangeable to burner module
- Added `MsgMoveIbcLockedCoins` (governance-only) to recover IBC coins from the black-hole module via liquidity addition
- `GetRaffleCurrentEpoch()` now returns `(uint64, error)` using `SafeGetEpochCountByIdentifier` for proper error handling
- Rate limit: max 200 raffle participants per block height; minimum pot of 100,000 units enforced
