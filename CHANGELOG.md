# Changelog

Every PR that changes behavior adds an entry to the `Unreleased` section in the same PR.
Entry format: `* (module) [#PR](pr-link) description`. Sections used (only when non-empty):
`State Machine Breaking`, `API Breaking`, `Features`, `Improvements`, `Bug Fixes`.
On release, `Unreleased` becomes the new version section and the GitHub release notes are
derived from it. History older than v8.0.0 lives in the
[GitHub releases](https://github.com/bze-alphateam/bze/releases) only.

## Unreleased

### Features

* (x/tokenfactory) [#75](https://github.com/bze-alphateam/bze/pull/75) On-chain denom branding: `MsgSetDenomBranding` lets a denom admin attach a branding package (font plus light/dark colour palettes) to a factory denom, with `DenomBranding`/`AllDenomBranding` queries and genesis import/export.
* (x/rewards) [#92](https://github.com/bze-alphateam/bze/pull/92) `MsgDeleteStakingReward`: permissionless removal of a finished staking reward once all stakes have exited, mirroring ExitStaking's final-exit cleanup; a removal-hook veto fails the message explicitly.
* (x/rewards) [#96](https://github.com/bze-alphateam/bze/pull/96) Denom Rewards: add `MsgCreateDenomReward` (creates the unique per-denom staking pool with a param-snapshotted lock and min-stake, capturing the creation fee) and `MsgJoinDenomReward` (stake into a pool — first join or top-up — settling every accrued prize before the staked amount changes).
* (x/rewards) [#97](https://github.com/bze-alphateam/bze/pull/97) Denom Rewards: add `MsgClaimDenomRewards` (pays every pending prize denom in one tx; sub-unit dust keeps accruing) and `MsgExitDenomReward` (all-or-nothing exit that settles pending prizes first, then releases the stake through the existing pending-unlock pipeline — immediately at lock 0, otherwise after lock×24 hour-epochs).
* (x/rewards) [#98](https://github.com/bze-alphateam/bze/pull/98) Denom Rewards: add the money-in messages — `MsgCreateDenomRewardSchedule` (daily-payout campaign with the full budget escrowed up front, plus the flat schedule fee), `MsgUpdateDenomRewardSchedule` (extend a schedule by escrowing the extra budget, no fee) and `MsgDistributeDenomRewards` (instant airdrop applied to the accumulator in the same tx; rejected when the pool has no stakers). Introducing a prize denom new to a pool pays the prize-creation fee once per denom lifetime — identically via schedule or airdrop — and is bounded by the `max_prize_denoms_per_dr` cap.

### Improvements

* (x/rewards) [#74](https://github.com/bze-alphateam/bze/pull/74) Add `StakingRewardHooks` interface (with multi-hooks combinator and keeper `SetHooks` plumbing); participation hooks are emitted on staking join, increase and exit. No-op until a listener module registers.
* (x/rewards) [#91](https://github.com/bze-alphateam/bze/pull/91) Add `BeforeStakingRewardRemoval` hook as a veto point for staking reward deletion; on final exit a veto suppresses the deletion instead of failing the exit.

### Bug Fixes

* (x/burner) [#73](https://github.com/bze-alphateam/bze/pull/73) Add the missing `amino.name` annotation to `MsgMoveIbcLockedCoins` so amino-JSON (e.g. Ledger) signing works.

## [v8.1.1](https://github.com/bze-alphateam/bze/releases/tag/v8.1.1) - 2026-07-05

Coordinated upgrade at height 23855000 (`v811` upgrade handler).

### State Machine Breaking

* (x/tradebin) [#71](https://github.com/bze-alphateam/bze/pull/71) Liquidity pool denoms are now derived from a hash of the two assets prefixed with `lp`, fixing pool creation for asset pairs whose concatenated denom exceeded the bank module's 128-character limit.

### Improvements

* (deps) [#71](https://github.com/bze-alphateam/bze/pull/71) Bump ibc-go and CometBFT patch versions.
* (x/tradebin) [#71](https://github.com/bze-alphateam/bze/pull/71) Generate proto code for tradebin events.

### Bug Fixes

* (build) Package platform-specific binaries correctly in the compressed release archives.

## [v8.1.0](https://github.com/bze-alphateam/bze/releases/tag/v8.1.0) - 2026-04-22

Coordinated upgrade at height 22551810 (`v810` upgrade handler). Module migrations: tradebin v3→v4, rewards v3→v4, txfeecollector v1→v2; the `crisis` module store is removed. The entries below are a digest — the [release notes](https://github.com/bze-alphateam/bze/releases/tag/v8.1.0) carry the exhaustive per-module list, including every new parameter with its default and purpose.

### State Machine Breaking

* (app) [#61](https://github.com/bze-alphateam/bze/pull/61) Removed the `crisis` module entirely (keeper, blockers, invariant registration, store).
* (x/tradebin) [#61](https://github.com/bze-alphateam/bze/pull/61) Queue-based order processing with bounded EndBlock execution; order keys migrated to 32-char/18-decimal precision; queue message keys restructured to composite `{market}/{id}`; `MsgFillOrders` capped at 50 orders with unique prices.
* (x/tradebin) [#67](https://github.com/bze-alphateam/bze/pull/67) v2 parameters: queue-depth gas surcharges (`order_book_extra_gas_window`, `order_book_queue_extra_gas`, `order_book_queue_message_scan_extra_gas`), `fill_orders_extra_gas`, `order_book_per_block_messages`, `min_native_liquidity_for_module_swap`; fee parameters migrated from string to `sdk.Coin`.
* (x/rewards) [#61](https://github.com/bze-alphateam/bze/pull/61) Reward operations (participant unlocks, staking reward distribution, trading reward expiration) moved from synchronous epoch hooks to bounded EndBlock queues (100 items/block); creation fees routed to txfeecollector instead of the community pool; expired pending trading rewards send uncaptured funds to the burner; new `extra_gas_for_exit_stake` parameter.
* (x/txfeecollector) [#61](https://github.com/bze-alphateam/bze/pull/61) Three-way fee split between stakers, burner and community pool via dedicated module accounts; fee conversion moved from epoch hooks to EndBlock; ante handler enforces a dynamic minimum gas price derived from the new `validator_min_gas_fee` parameter and trade module spot prices; new `max_balance_iterations` parameter.
* (x/burner) [#61](https://github.com/bze-alphateam/bze/pull/61) Periodic burning and raffle cleanup processed in bounded EndBlock batches instead of epoch hooks; non-burnable coins are locked; raffles rate-limited to 200 participants per height, with a minimum pot and stricter expiration checks.
* (x/tokenfactory) [#61](https://github.com/bze-alphateam/bze/pull/61) Subdenoms may no longer contain `/`; denom creation fee routed through trade-keeper fee capture instead of the community pool.

### Features

* (x/burner) [#61](https://github.com/bze-alphateam/bze/pull/61) Permissionless `MsgMoveIbcLockedCoins` to move locked IBC coins from the black-hole account into liquidity pools paired with native BZE; `MsgFundBurner` now classifies coins (lockable LP shares to the black hole, burnable/exchangeable to the burner).

### Improvements

* (x/epochs) [#61](https://github.com/bze-alphateam/bze/pull/61) Added `SafeGetEpochCountByIdentifier`, returning an error for missing or catching-up epochs; adopted by burner and rewards.
* (x/tradebin) [`c01958a`](https://github.com/bze-alphateam/bze/commit/c01958a) Added `CaptureAndTryToSwapUserFeesOrSendItAsIs` to handle small trading fees that cannot be swapped.
* (deps) [#61](https://github.com/bze-alphateam/bze/pull/61) Go 1.23→1.25, cosmos-sdk v0.50.15, CometBFT v0.38.21.
* (repo) [#66](https://github.com/bze-alphateam/bze/pull/66) Added MIT license.
* (ci) [#61](https://github.com/bze-alphateam/bze/pull/61) Added test workflow (keeper and types tests on PRs and main); build binaries now carry an arch suffix.

### Bug Fixes

* (x/rewards) [#61](https://github.com/bze-alphateam/bze/pull/61) Leaderboard sorting compared the same value for both items; staking/trading reward counters incremented on every operation instead of only on creation; claiming zero staking rewards now errors; zero lock duration sends funds immediately instead of queueing; multiple active trading rewards for the same market are prevented.
* (x/tradebin) [#61](https://github.com/bze-alphateam/bze/pull/61) Consistent reserve computation for reversed assets in pool creation; duplicate order-cancel prevention via pending-cancel tracking; `MulRaw` applied before `QuoRaw` for precision; history order index format fixed for correct sorting.
* (x/cointrunk) [#61](https://github.com/bze-alphateam/bze/pull/61) Publisher respect tax validation enforces strict bounds (0 < tax < 1); `SetArticle` exposes the auto-incremented article id; added missing article event id.

## [v8.0.2](https://github.com/bze-alphateam/bze/releases/tag/v8.0.2) - 2026-01-26

### Improvements

* (deps) [#63](https://github.com/bze-alphateam/bze/pull/63) Upgrade CometBFT and tidy dependencies.

## [v8.0.1](https://github.com/bze-alphateam/bze/releases/tag/v8.0.1) - 2026-01-26

Tag only — no GitHub release; superseded the same day by v8.0.2.

### Improvements

* (deps) [`667fe07`](https://github.com/bze-alphateam/bze/commit/667fe07) Bump CometBFT.

## [v8.0.0](https://github.com/bze-alphateam/bze/releases/tag/v8.0.0) - 2025-11-19

Coordinated upgrade at height 20237800. Major release: Cosmos SDK upgraded to v0.50 and ibc-go to v8.

### State Machine Breaking

* (deps) Upgraded Cosmos SDK to v0.50 and ibc-go to v8.
* (x/scavenge) Removed the unused scavenge module.
* (x/rewards) Migrated 75,000 BZE (75000000000ubze) from the rewards module account to the community pool — staking reward creation fees that a bug had kept from being forwarded.

### Features

* (x/tradebin) Constant-product AMM liquidity pools; the tradebin module is granted the minter role for LP share denoms.
* (x/txfeecollector) New module: transaction fees can be paid in any denom that has a liquidity pool with BZE; captured fees are exchanged to BZE and sent to delegators.
* (x/burner) Burner can now burn factory tokens; IBC coins with a sufficiently liquid BZE pool are swapped to BZE and the result burned; LP shares are locked in the black-hole address.
