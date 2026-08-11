# Changelog

Every PR that changes behavior adds an entry to the `Unreleased` section in the same PR.
Entry format: `* (module) [#PR](pr-link) description`. Sections used (only when non-empty):
`State Machine Breaking`, `API Breaking`, `Features`, `Improvements`, `Bug Fixes`.
On release, `Unreleased` becomes the new version section and the GitHub release notes are
derived from it. History older than v8.1.0 lives in the
[GitHub releases](https://github.com/bze-alphateam/bze/releases) only.

## Unreleased

### Features

* (x/tokenfactory) [#75](https://github.com/bze-alphateam/bze/pull/75) On-chain denom branding: `MsgSetDenomBranding` lets a denom admin attach a branding package (font plus light/dark colour palettes) to a factory denom, with `DenomBranding`/`AllDenomBranding` queries and genesis import/export.
* (x/rewards) [#92](https://github.com/bze-alphateam/bze/pull/92) `MsgDeleteStakingReward`: permissionless removal of a finished staking reward once all stakes have exited, mirroring ExitStaking's final-exit cleanup; a removal-hook veto fails the message explicitly.

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
