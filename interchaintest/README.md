# bze end-to-end tests

End-to-end suites for the bze chain, built on
[interchaintest](https://github.com/cosmos/interchaintest) v10. Each suite
starts a real network of `bzed` validators in Docker, sends real transactions
through the `bzed` CLI and asserts balances, state and events.

This directory is its own Go module (`github.com/bze-alphateam/bze/interchaintest`).
It does not import the chain module: it only drives the `bzed` binary inside
the image, so the same harness works across bze versions, including old
releases for the upgrade suite. One exception: on v8.1.1 and older, `--gas auto`
combined with `--gas-prices` fails (the fee collector rejected the zero
simulation fee until v8.2.0), so the default `helpers.ExecTx` does not work
there; send txs to those versions with `helpers.ExecTxWith` and `WithFees`.

## Prerequisites

- Docker (the daemon must be reachable by the user running the tests)
- Go 1.26

## Build the images

Images are never built implicitly by the test targets. Build them once, and
again whenever the code under test changes:

```bash
make e2e-image                      # working tree -> bze:local
make e2e-image-ref REF=v8.2.0       # a git ref    -> bze:v8.2.0
```

`scripts/e2e/build-image.sh` does the work. A ref is exported with
`git archive` and built with the current `Dockerfile`, so tags that predate it
build too. `bzed version` in the image reports the git version of what was
built, and the image carries it in the `org.opencontainers.image.version` label.

## Run

```bash
make e2e                            # every suite but the upgrade suite
make e2e-smoke                      # the harness smoke test
make e2e-denomrewards               # denom rewards lifecycle (~5 min)
make e2e-amm                        # AMM swaps, liquidity and the fee path (~3 min)
make e2e-tokenfactory               # staking rewards + tokenfactory admin and branding (~4 min)
make e2e-halt                       # halted denoms: gov halt/unhalt, refused trading, exits
make e2e-ibc                        # blocked inbound IBC: bze <-> simd through Hermes
make e2e-fees                       # fee destinations, minimum gas prices, ante refusals, fee grants, txfeecollector params
make e2e-orderbook                  # order-book engine: fills, remainders, dust, cancels, refusals, queue limits
make e2e-tradingrewards             # trading rewards: create, gov activation, hooks, leaderboard, payout, cleanup (~20 min)
make e2e-upgrade                    # chain upgrade from the previous release (see below)
cd interchaintest && go test -v -run TestSmoke ./...
```

`make e2e` runs every suite but the upgrade suite, which needs the previous
release's image and a plan name the candidate has a handler for (see below).

`go test -short` skips every e2e test and runs only the helpers' unit tests
(`make e2e-unit`, no Docker needed). `make e2e-lint` lints the module.

Each suite starts its own network and removes its containers, volumes and
Docker network when it ends. Suites do not share state.

### Environment variables

| Variable | Default | Effect |
|---|---|---|
| `E2E_BZE_IMAGE_TAG` | `local` | Tag of the `bze` image to run |
| `E2E_TIMEOUT` (make only) | `60m` | `go test -timeout` of the make targets |
| `E2E_UPGRADE_FROM` | `v8.2.0-rc5` | Upgrade suite: image tag the network starts on (the previous release) |
| `E2E_UPGRADE_TO` | `local` | Upgrade suite: image tag the validators switch to at the plan height |
| `E2E_UPGRADE_NAME` | `v8.3.0` | Upgrade suite: plan name, must equal the candidate's `UpgradeName` |
| `KEEP_CONTAINERS` | unset | Any value keeps the containers after the test, for debugging (remove them by hand) |
| `ICTEST_SKIP_FAILURE_CLEANUP` | unset | Any value keeps Docker volumes and temp dirs of failed tests |
| `SHOW_CONTAINER_LOGS` | unset | `always` prints container logs after each test, otherwise only on failure |
| `CONTAINER_LOG_TAIL` | `50` | Number of container log lines printed |

## Before every release tag: the upgrade suite

Every `bze` tag (release candidate or final) is preceded by a local run of the
upgrade suite from the previous release, and its result goes with the tag
request. `TestChainUpgrade` starts two validators on the previous release,
seeds state in the custom modules (only what that release supports), passes a
software-upgrade proposal, lets the chain halt at the plan height, restarts
the validators on the candidate image and checks:

- the chain halts at exactly the plan height, resumes on the candidate, and
  `bzed version` changed;
- the upgrade block changes no supply other than what its own bank events
  mint (inflation) and burn, and no seeded account, module balance, pool, LP
  supply, market, resting order, denom admin, metadata, branding or halted
  denom;
- staking and denom rewards keep their stakes, accumulators, schedules and
  pending amounts, and the next `day` tick pays exactly one day (the epoch
  hooks are still wired);
- a pending unlock is returned at its `hour` epoch;
- one tx per custom module works on the candidate;
- the checks of the handler itself, in `upgradeChecks` (`upgrade_test.go`),
  keyed by upgrade name. A release that adds a handler adds its entry.

Steps, for a candidate `vX.Y.Z` whose previous release is `vP`:

1. Build the previous release: `make e2e-image-ref REF=vP` (gives `bze:vP`).
2. Build the candidate: `make e2e-image` (the working tree, as `bze:local`).
3. Run the suite (about 6 minutes):

   ```bash
   E2E_UPGRADE_FROM=vP E2E_UPGRADE_NAME=vX.Y.Z make e2e-upgrade
   ```

4. Attach the outcome (pass/fail, runtime, the `upgrade block` and version
   log lines) to the tag request.

`E2E_UPGRADE_NAME` must be the `UpgradeName` of the candidate's handler in
`app/upgrades/`. For example, the v8.2.0 handler is checked from v8.1.1 with
`E2E_UPGRADE_FROM=v8.1.1 E2E_UPGRADE_NAME=v8.2.0` after
`make e2e-image-ref REF=v8.1.1`.

All txs use a fixed gas limit, because `--gas auto` does not work on v8.1.1
and older.

## The e2e network

| Setting | Value |
|---|---|
| Chain id | `bze-e2e-1` |
| Validators | 1 (suites can ask for more) |
| Block time | about 1 s (`timeout_commit = 1s`) |
| Gas price | `0.01ubze` |
| Epochs | `day` 30 s, `hour` 5 s, `week` 210 s |
| Gov voting period | 20 s (expedited 10 s) |
| Gov min deposit | 1 BZE (expedited 2 BZE) |
| Denom rewards lock | 1 day, i.e. 24 `hour` epochs, about 2 minutes |

The genesis overrides live in `chain/genesis.go` (`DefaultGenesis`); a suite
adds its own with `chain.WithGenesis(...)`.

## Writing a suite

- Start the network with `chain.Start(t, ctx, opts...)` (options such as
  `chain.WithGenesis` or `chain.WithMinGasPrices`), or with
  `chain.StartIbc` for bze plus an ibc-go `simd` counterparty connected by a
  Hermes relayer (pinned in `chain/ibc.go`; send and follow packets with
  `helpers.SendIbc`, `WaitForRecv`, `WaitForAck`, `WaitForTimeout`).
- Fund users with `interchaintest.GetAndFundTestUsers`.
- Send messages with `helpers.ExecTx` / `helpers.ExecTxWith` (a non-zero
  `Code` is a rejection, not an error; `helpers.WithFlags` appends raw CLI
  flags that win over the defaults, e.g. `--gas 0`).
- Read state with `helpers.Query[T]`, `helpers.Balance`, events with
  `helpers.TxEvents` / `helpers.BlockEvents`.
- Wait for time with `helpers.WaitForEpoch`, pass gov messages with
  `helpers.SubmitAndPassProposal`.
- Tests live in package `interchaintest_test` and are black-box; lint them with
  `make e2e-lint`.
- Add a `make e2e-<suite>` target to the root Makefile.

## Runtime

Building an image from scratch takes about 3–4 minutes (Go build caches make
rebuilds faster). The smoke test takes about a minute, most of it network
start-up.

## Known gaps

- Trading rewards: the 30-day expiry of a pending reward, which burns its
  escrow, takes 720 `hour` epochs (about an hour) and is covered by unit tests
  only. The trading reward candidates (every trader's running total) have no
  query, so the suite sees them only through the leaderboard.
