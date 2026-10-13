package interchaintest_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// TestSmoke proves the harness: the image runs the expected bzed, genesis has
// the fast epochs, and txs, queries, balances and block events work.
func TestSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	bze := net.Chain

	t.Run("bzed version matches the image", func(t *testing.T) {
		image := chain.BzeImage("")
		inspect, _, err := net.Docker.ImageInspectWithRaw(ctx, image.Ref())
		require.NoError(t, err)
		want := inspect.Config.Labels[chain.ImageVersionLabel]
		require.NotEmpty(t, want, "image %s has no %s label; build it with scripts/e2e/build-image.sh", image.Ref(), chain.ImageVersionLabel)

		stdout, stderr, err := bze.GetNode().ExecBin(ctx, "version")
		require.NoError(t, err)
		got := strings.TrimSpace(string(stdout) + string(stderr))
		require.Equal(t, want, got)
	})

	t.Run("bank send deducts amount and fee", func(t *testing.T) {
		funds := sdkmath.NewInt(10_000_000_000)
		users := interchaintest.GetAndFundTestUsers(t, ctx, "smoke", funds, bze, bze)
		alice, bob := users[0], users[1]
		require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

		aliceBefore, err := helpers.Balance(ctx, bze, alice.FormattedAddress(), chain.Denom)
		require.NoError(t, err)
		bobBefore, err := helpers.Balance(ctx, bze, bob.FormattedAddress(), chain.Denom)
		require.NoError(t, err)

		amount := sdkmath.NewInt(1_000_000) // 1 BZE
		res, err := helpers.ExecTx(ctx, bze, alice,
			"bank", "send", alice.FormattedAddress(), bob.FormattedAddress(), amount.String()+chain.Denom)
		require.NoError(t, err)
		require.Zero(t, res.Code, res.RawLog)
		require.Positive(t, res.Height)

		txEvents := helpers.TxEvents(res, "tx")
		var fee sdkmath.Int
		for _, e := range txEvents {
			if v, ok := e.Attributes["fee"]; ok {
				coins, err := sdk.ParseCoinsNormalized(v)
				require.NoError(t, err)
				fee = coins.AmountOf(chain.Denom)
			}
		}
		require.False(t, fee.IsNil(), "no fee attribute in tx events")
		require.True(t, fee.IsPositive(), "fee must be paid in %s", chain.Denom)

		aliceAfter, err := helpers.Balance(ctx, bze, alice.FormattedAddress(), chain.Denom)
		require.NoError(t, err)
		bobAfter, err := helpers.Balance(ctx, bze, bob.FormattedAddress(), chain.Denom)
		require.NoError(t, err)
		require.Equal(t, aliceBefore.Sub(amount).Sub(fee).String(), aliceAfter.String())
		require.Equal(t, bobBefore.Add(amount).String(), bobAfter.String())
	})

	t.Run("rejected tx: simulation error with gas auto, code with fixed gas", func(t *testing.T) {
		funds := sdkmath.NewInt(10_000_000)
		users := interchaintest.GetAndFundTestUsers(t, ctx, "reject", funds, bze, bze)
		carol, dave := users[0], users[1]
		require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

		tooMuch := funds.MulRaw(2).String() + chain.Denom
		send := []string{"bank", "send", carol.FormattedAddress(), dave.FormattedAddress(), tooMuch}

		_, err := helpers.ExecTx(ctx, bze, carol, send...)
		var simErr *helpers.SimulationError
		require.True(t, errors.As(err, &simErr), "want a SimulationError, got %v", err)
		require.Contains(t, simErr.Output, "insufficient funds")

		// Fixed gas skips simulation: the tx is included and fails with
		// the SDK's ErrInsufficientFunds (code 5).
		res, err := helpers.ExecTxWith(ctx, bze, carol, []helpers.TxOption{helpers.WithGas(200_000)}, send...)
		require.NoError(t, err)
		require.Equal(t, uint32(5), res.Code, res.RawLog)
		require.Equal(t, "sdk", res.Codespace)
		require.Positive(t, res.Height)
	})

	t.Run("day epoch ticks fast and emits its start event", func(t *testing.T) {
		start, err := helpers.CurrentEpoch(ctx, bze, "day")
		require.NoError(t, err)

		waitCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		require.NoError(t, helpers.WaitForEpoch(waitCtx, bze, "day", start+1))

		day, err := helpers.Epoch(ctx, bze, "day")
		require.NoError(t, err)
		require.Equal(t, chain.DayEpochDuration, day.Duration)

		events, err := helpers.BlockEvents(ctx, bze, int64(day.CurrentEpochStartHeight), "bze.epochs.EpochStartEvent")
		require.NoError(t, err)
		var found bool
		for _, e := range events {
			if e.Attributes["identifier"] == "day" {
				require.Equal(t, strconv.FormatInt(int64(day.CurrentEpoch), 10), e.Attributes["epoch"])
				found = true
			}
		}
		require.True(t, found, "no day EpochStartEvent at height %d: %+v", day.CurrentEpochStartHeight, events)
	})

	t.Run("every epoch starts at 1 in the first block", func(t *testing.T) {
		// genesis leaves the epochs unstarted: the first block initialises
		// them, emits their start events and no end event
		events := smokeEpochEvents(t, ctx, bze, 1)
		for _, id := range []string{"day", "hour", "week"} {
			require.Contains(t, events, smokeEpochEvent{"start", id, "1"}, "no %s start event in block 1", id)
		}
		for _, e := range events {
			require.NotEqual(t, "end", e.kind, "block 1 must not end an epoch: %+v", events)
		}
	})

	t.Run("week epoch ticks, its end event before its start event", func(t *testing.T) {
		waitCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
		defer cancel()
		require.NoError(t, helpers.WaitForEpoch(waitCtx, bze, "week", 2))

		week, err := helpers.Epoch(ctx, bze, "week")
		require.NoError(t, err)
		require.Equal(t, int64(2), int64(week.CurrentEpoch))
		require.Equal(t, chain.WeekEpochDuration, week.Duration)
		require.Greater(t, int64(week.CurrentEpochStartHeight), int64(1))

		var order []smokeEpochEvent
		for _, e := range smokeEpochEvents(t, ctx, bze, int64(week.CurrentEpochStartHeight)) {
			if e.identifier == "week" {
				order = append(order, e)
			}
		}
		require.Equal(t, []smokeEpochEvent{{"end", "week", "1"}, {"start", "week", "2"}}, order,
			"the tick must end week 1 and then start week 2")
	})
}

// smokeEpochEvent is an epochs module event: kind "start" or "end".
type smokeEpochEvent struct {
	kind, identifier, epoch string
}

// smokeEpochEvents returns the epoch start and end events of a block in the
// order the block emitted them (helpers.BlockEvents filters by type, which
// loses the order between the two types).
func smokeEpochEvents(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain, height int64) []smokeEpochEvent {
	t.Helper()
	res, err := bze.GetNode().Client.BlockResults(ctx, &height)
	require.NoError(t, err)
	var out []smokeEpochEvent
	for _, e := range res.FinalizeBlockEvents {
		kind := map[string]string{
			"bze.epochs.EpochStartEvent": "start",
			"bze.epochs.EpochEndEvent":   "end",
		}[e.Type]
		if kind == "" {
			continue
		}
		ev := smokeEpochEvent{kind: kind}
		for _, a := range e.Attributes {
			switch a.Key {
			case "identifier":
				ev.identifier = helpers.Unquote(a.Value)
			case "epoch":
				ev.epoch = helpers.Unquote(a.Value)
			}
		}
		out = append(out, ev)
	}
	return out
}
