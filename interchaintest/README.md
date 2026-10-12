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
make e2e                            # every suite
make e2e-smoke                      # the harness smoke test
make e2e-denomrewards               # denom rewards lifecycle (~5 min)
make e2e-amm                        # AMM swaps, liquidity and the fee path (~3 min)
make e2e-tokenfactory               # staking rewards + tokenfactory admin and branding (~4 min)
make e2e-halt                       # halted denoms: gov halt/unhalt, refused trading, exits
make e2e-ibc                        # blocked inbound IBC: bze <-> simd through Hermes
cd interchaintest && go test -v -run TestSmoke ./...
```

`go test -short` skips every e2e test and runs only the helpers' unit tests
(`make e2e-unit`, no Docker needed). `make e2e-lint` lints the module.

Each suite starts its own network and removes its containers, volumes and
Docker network when it ends. Suites do not share state.

### Environment variables

| Variable | Default | Effect |
|---|---|---|
| `E2E_BZE_IMAGE_TAG` | `local` | Tag of the `bze` image to run |
| `E2E_TIMEOUT` (make only) | `60m` | `go test -timeout` of the make targets |
| `KEEP_CONTAINERS` | unset | Any value keeps the containers after the test, for debugging (remove them by hand) |
| `ICTEST_SKIP_FAILURE_CLEANUP` | unset | Any value keeps Docker volumes and temp dirs of failed tests |
| `SHOW_CONTAINER_LOGS` | unset | `always` prints container logs after each test, otherwise only on failure |
| `CONTAINER_LOG_TAIL` | `50` | Number of container log lines printed |

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

- Start the network with `chain.Start(t, ctx, opts...)`, or with
  `chain.StartIbc` for bze plus an ibc-go `simd` counterparty connected by a
  Hermes relayer (pinned in `chain/ibc.go`; send and follow packets with
  `helpers.SendIbc`, `WaitForRecv`, `WaitForAck`, `WaitForTimeout`).
- Fund users with `interchaintest.GetAndFundTestUsers`.
- Send messages with `helpers.ExecTx` / `helpers.ExecTxWith` (a non-zero
  `Code` is a rejection, not an error).
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
