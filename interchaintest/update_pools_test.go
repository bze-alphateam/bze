package interchaintest_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

const (
	updLpFunds     = 100_000_000_000 // 100,000 BZE: the create fee, the pool and the proposals
	updTraderFunds = 10_000_000_000  // 10,000 BZE

	updMint    = 100_000_000_000
	updPoolUpd = 10_000_000_000 // upd/ubze: 4 ubze per upd
	updPoolBze = 40_000_000_000

	updSwapInput = 1_234_567_890 // ubze swapped after the update

	// the pool is created with ammPoolFee / ammFeeDest and changed to these
	updNewFee       = "0.01"
	updNewTreasury  = "0.3"
	updNewBurner    = "0.2"
	updNewProviders = "0.5"
	// the second proposal's fee for the existing pool, never applied
	updRefusedFee = "0.02"
)

// TestUpdatePools changes a pool's swap fee and fee split through a real
// governance proposal carrying MsgUpdatePools, checks that nothing else about
// the pool moved and that the next swap charges the new fee and splits it
// with the new destinations. A second proposal listing the same pool and an
// unknown one fails at execution and changes nothing. Subtests run in order
// and share state.
func TestUpdatePools(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupUpdatePools(t, ctx, net.Chain)

	t.Run("1 update fee and split by governance", s.updateByGov)
	t.Run("2 swap uses the new fee and split", s.swapAfterUpdate)
	t.Run("3 unknown pool fails the whole proposal", s.unknownPoolRefused)
}

type updatePoolsSuite struct {
	// the AMM suite's query and tx helpers (pool, balances, events)
	*ammSuite

	gov    string
	upd    string
	poolID string
	before poolInfo // the pool as created
}

func setupUpdatePools(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *updatePoolsSuite {
	s := &updatePoolsSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}}

	s.lp = interchaintest.GetAndFundTestUsers(t, ctx, "upd-lp", sdkmath.NewInt(updLpFunds), bze)[0]
	s.trader = interchaintest.GetAndFundTestUsers(t, ctx, "upd-trader", sdkmath.NewInt(updTraderFunds), bze)[0]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	var err error
	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)
	for name, addr := range map[string]*string{
		"tradebin":              &s.tradebin,
		"burner":                &s.burner,
		"burner_black_hole":     &s.blackHole,
		"txfeecollector":        &s.txFees,
		"txfeecollector_burner": &s.burnerFees,
		"txfeecollector_cp":     &s.cpFees,
		"distribution":          &s.distribution,
	} {
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	s.params = s.queryParams(t)

	s.upd = "factory/" + s.lp.FormattedAddress() + "/upd"
	s.mustExec(t, s.lp, "tokenfactory", "create-denom", "upd")
	s.mustExec(t, s.lp, "tokenfactory", "mint", coinStr(updMint, s.upd))

	// pool ids order the denoms alphabetically: "factory/…" sorts before "ubze"
	s.poolID = s.upd + "_" + chain.Denom
	s.before = s.createPool(t, ammPoolArgs{
		base: s.upd, quote: chain.Denom, initBase: updPoolUpd, initQuote: updPoolBze,
		fee: ammPoolFee, feeDest: ammFeeDest,
		wantID: s.poolID, wantBase: s.upd, wantQuote: chain.Denom, wantReserveBase: updPoolUpd, wantReserveQuote: updPoolBze,
	})
	return s
}

func (s *updatePoolsSuite) updateMsg(updates ...map[string]any) map[string]any {
	return map[string]any{
		"@type":     "/bze.tradebin.MsgUpdatePools",
		"authority": s.gov,
		"updates":   updates,
	}
}

func poolUpdateEntry(poolID, fee, treasury, burner, providers string) map[string]any {
	return map[string]any{
		"pool_id": poolID,
		"fee":     fee,
		"fee_dest": map[string]string{
			"treasury":  treasury,
			"burner":    burner,
			"providers": providers,
		},
	}
}

// runProposal submits msgs as one proposal from the lp and returns the
// heights before submitting and after it ended, with its outcome.
func (s *updatePoolsSuite) runProposal(t *testing.T, title string, msgs ...any) (from, to int64, err error) {
	t.Helper()
	from, herr := s.bze.Height(s.ctx)
	require.NoError(t, herr)
	_, err = helpers.SubmitAndPassProposal(s.ctx, s.bze, s.lp, helpers.Proposal{
		Messages: msgs,
		Title:    title,
		Summary:  "e2e: " + title,
	})
	to, herr = s.bze.Height(s.ctx)
	require.NoError(t, herr)
	return from, to, err
}

func (s *updatePoolsSuite) feesUpdatedEvents(t *testing.T, from, to int64) []helpers.Event {
	t.Helper()
	var out []helpers.Event
	for h := from; h <= to; h++ {
		events, err := helpers.BlockEvents(s.ctx, s.bze, h, tradebinEventPrefix+"PoolFeesUpdatedEvent")
		require.NoError(t, err)
		out = append(out, events...)
	}
	return out
}

// 1. A proposal changes the pool's fee and fee split: the pool keeps its id,
// denoms, LP denom, reserves and LP supply, only fee and fee_dest change, and
// the block that executed the proposal carries one PoolFeesUpdatedEvent with
// the old and new values.
func (s *updatePoolsSuite) updateByGov(t *testing.T) {
	supply := s.supply(t, s.before.LpDenom)

	from, to, err := s.runProposal(t, "Change the upd pool fees",
		s.updateMsg(poolUpdateEntry(s.poolID, updNewFee, updNewTreasury, updNewBurner, updNewProviders)))
	require.NoError(t, err)

	after := s.pool(t, s.poolID)
	ammRequireDec(t, updNewFee, after.Fee, "pool fee")
	ammRequireDec(t, updNewTreasury, after.FeeDest.Treasury, "treasury share")
	ammRequireDec(t, updNewBurner, after.FeeDest.Burner, "burner share")
	ammRequireDec(t, updNewProviders, after.FeeDest.Providers, "providers share")

	want := s.before
	want.Fee, want.FeeDest = after.Fee, after.FeeDest
	require.Equal(t, want, after, "only fee and fee_dest change")
	require.Equal(t, supply, s.supply(t, after.LpDenom), "LP supply")

	events := s.feesUpdatedEvents(t, from, to)
	require.Len(t, events, 1, "one PoolFeesUpdatedEvent: %+v", events)
	ev := events[0]
	require.Equal(t, s.poolID, ev.Attributes["pool_id"])
	updRequireEventDec(t, ammPoolFee, ev.Attributes["old_fee"], "old_fee")
	updRequireEventDec(t, updNewFee, ev.Attributes["new_fee"], "new_fee")
	var oldDest map[string]string
	require.NoError(t, json.Unmarshal([]byte(ammFeeDest), &oldDest))
	updRequireEventDest(t, oldDest, ev.Attributes["old_fee_dest"], "old_fee_dest")
	updRequireEventDest(t, map[string]string{"treasury": updNewTreasury, "burner": updNewBurner, "providers": updNewProviders}, ev.Attributes["new_fee_dest"], "new_fee_dest")
}

// 2. The next swap charges the new fee: 30% of it to txfeecollector_cp (and
// on to the community pool in EndBlock), 20% to the burner, the providers'
// 50% stays in the ubze reserve.
func (s *updatePoolsSuite) swapAfterUpdate(t *testing.T) {
	pool := s.pool(t, s.poolID)
	in := sdkmath.NewInt(updSwapInput)
	q := ammQuoteSwap(in, pool.reserveQuote(), pool.reserveBase(), updNewFee, updNewTreasury, updNewBurner)
	require.True(t, q.treasury.IsPositive() && q.burned.IsPositive(), "the swap must be large enough for every share")
	before := s.balances(t, s.trader, chain.Denom, s.upd)

	res := s.mustExec(t, s.trader, "tradebin", "multi-swap", s.poolID, sdk.NewCoin(chain.Denom, in).String(), coinStr(1, s.upd))

	s.requireSwap(t, res.Events, s.trader.FormattedAddress(), s.poolID, sdk.NewCoin(chain.Denom, in), sdk.NewCoin(s.upd, q.out))
	require.Equal(t, q.treasury, s.transferred(t, res.Events, s.tradebin, s.cpFees, chain.Denom), "treasury share to txfeecollector_cp")
	require.Equal(t, q.burned, s.transferred(t, res.Events, s.tradebin, s.burner, chain.Denom), "burner share to the burner")
	s.requireTakerFeeBurned(t, res)

	after := s.pool(t, s.poolID)
	require.Equal(t, pool.reserveQuote().Add(q.inReserveAdd), after.reserveQuote(), "ubze reserve keeps the providers share")
	require.Equal(t, pool.reserveBase().Sub(q.out), after.reserveBase(), "upd reserve")
	want := before.Add(sdk.NewCoin(s.upd, q.out)).Sub(sdk.NewCoin(chain.Denom, in.Add(s.params.MarketTakerFee.Amount).Add(s.txFee(t, res))))
	require.Equal(t, want, s.balances(t, s.trader, chain.Denom, s.upd), "trader balances")

	block := s.blockEvents(t, res.Height)
	require.Equal(t, q.treasury, s.transferred(t, block, s.cpFees, s.distribution, chain.Denom), "treasury share to the community pool in EndBlock")
}

// 3. A proposal listing the pool and a pool that does not exist fails at
// execution: the existing pool keeps the fees of step 1 and no event is
// emitted.
func (s *updatePoolsSuite) unknownPoolRefused(t *testing.T) {
	state := s.state(t)
	unknown := "nope_" + chain.Denom

	from, to, err := s.runProposal(t, "Change an unknown pool",
		s.updateMsg(
			poolUpdateEntry(s.poolID, updRefusedFee, updNewTreasury, updNewBurner, updNewProviders),
			poolUpdateEntry(unknown, updRefusedFee, updNewTreasury, updNewBurner, updNewProviders),
		))
	require.ErrorContains(t, err, "PROPOSAL_STATUS_FAILED")
	require.ErrorContains(t, err, "pool "+unknown+" not found")

	require.Equal(t, state, s.state(t), "a failed proposal changes no pool")
	ammRequireDec(t, updNewFee, s.pool(t, s.poolID).Fee, "pool fee")
	require.Empty(t, s.feesUpdatedEvents(t, from, to), "no PoolFeesUpdatedEvent")
	s.requireNoPool(t, unknown)
}

// updRequireEventDec compares a decimal as sent with one in a typed event:
// a plain decimal string ("0.010000000000000000"), or the 18-decimal scaled
// integer the proto JSON of a LegacyDec field may carry.
func updRequireEventDec(t *testing.T, want, got, what string) {
	t.Helper()
	got = helpers.Unquote(got)
	if strings.Contains(got, ".") {
		w, err := sdkmath.LegacyNewDecFromStr(want)
		require.NoError(t, err)
		g, err := sdkmath.LegacyNewDecFromStr(got)
		require.NoError(t, err, "%s: %q", what, got)
		require.True(t, w.Equal(g), "%s: want %s, got %s", what, w, g)
		return
	}
	ammRequireDec(t, want, got, what)
}

func updRequireEventDest(t *testing.T, want map[string]string, got, what string) {
	t.Helper()
	var dest map[string]string
	require.NoError(t, json.Unmarshal([]byte(got), &dest), "%s: %q", what, got)
	for _, k := range []string{"treasury", "burner", "providers"} {
		updRequireEventDec(t, want[k], dest[k], what+"."+k)
	}
}
