package interchaintest_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

const (
	// step 22: abc sold and bought on the abc/ubze pool of step 15 (fee 0.001,
	// treasury 0.3, burner 0.2): a sell of 10,000,000 abc takes a fee of 10,000
	// abc, 3,000 of it for the community pool
	ammTokenCpSell     = 10_000_000
	ammTokenCpBuy      = 40_000_000 // ubze
	ammTokenCpFunds    = 1_000_000  // ubze, the cp account only receives
	communityPoolPaidT = "bze.txfeecollector.CommunityPoolFeePaidEvent"
)

// 22. abc's admin (the lp) sets abc's community pool to a fresh account with
// zero rates. From then on the txfeecollector forwards, in the EndBlock of the
// same block, the abc that txfeecollector_cp holds to that account, with a
// CommunityPoolFeePaidEvent: the treasury share of an abc sell plus anything
// in abc waiting there. A buy pays its treasury share in ubze, which funds
// the chain's community pool as before; the account gets nothing. Once the
// config is back to the defaults, an abc sell's share takes today's path
// again: the pool's ubze side is below min_native_liquidity_for_module_swap,
// so the txfeecollector cannot swap it and sends it to the burner.
func (s *ammSuite) tokenCommunityPool(t *testing.T) {
	cp := interchaintest.GetAndFundTestUsers(t, s.ctx, "amm-token-cp", sdkmath.NewInt(ammTokenCpFunds), s.bze)[0].FormattedAddress()
	s.mustExec(t, s.lp, "tokenfactory", "set-denom-config", s.abc, cp, "0", "0")
	s.mustExec(t, s.lp, "bank", "send", s.lp.FormattedAddress(), s.trader.FormattedAddress(), coinStr(2*ammTokenCpSell, s.abc))

	// 1. a sell: the treasury share reaches the token's community pool
	pending := s.balance(t, s.cpFees, s.abc)
	cpBefore := s.balance(t, cp, s.abc)
	pool := s.pool(t, s.cpPool)
	q := ammQuoteSwap(sdkmath.NewInt(ammTokenCpSell), pool.reserveBase(), pool.reserveQuote(), ammMinPoolFee, "0.3", "0.2")
	require.Equal(t, sdkmath.NewInt(3_000), q.treasury)

	res := s.mustExec(t, s.trader, "tradebin", "multi-swap", s.cpPool, coinStr(ammTokenCpSell, s.abc), coinStr(1, chain.Denom))
	require.Equal(t, q.treasury, s.transferred(t, res.Events, s.tradebin, s.cpFees, s.abc), "treasury share to txfeecollector_cp")

	forwarded := pending.Add(q.treasury)
	block := s.blockEvents(t, res.Height)
	require.Equal(t, forwarded, s.transferred(t, block, s.cpFees, cp, s.abc), "abc forwarded to the token's community pool in EndBlock")
	s.requireCommunityPoolPaid(t, res.Height, []map[string]string{{"denom": s.abc, "recipient": cp, "amount": forwarded.String()}})
	require.Equal(t, cpBefore.Add(forwarded), s.balance(t, cp, s.abc), "community pool balance")
	require.True(t, s.balance(t, s.cpFees, s.abc).IsZero(), "txfeecollector_cp keeps no abc")

	// 2. a buy: the ubze share funds the chain's community pool, the token's gets nothing
	pool = s.pool(t, s.cpPool)
	q = ammQuoteSwap(sdkmath.NewInt(ammTokenCpBuy), pool.reserveQuote(), pool.reserveBase(), ammMinPoolFee, "0.3", "0.2")
	res = s.mustExec(t, s.trader, "tradebin", "multi-swap", s.cpPool, coinStr(ammTokenCpBuy, chain.Denom), coinStr(1, s.abc))
	require.Equal(t, q.treasury, s.transferred(t, res.Events, s.tradebin, s.cpFees, chain.Denom), "treasury share to txfeecollector_cp")
	block = s.blockEvents(t, res.Height)
	require.Equal(t, q.treasury, s.transferred(t, block, s.cpFees, s.distribution, chain.Denom), "ubze share to the chain's community pool")
	s.requireCommunityPoolPaid(t, res.Height, nil)
	require.Equal(t, cpBefore.Add(forwarded), s.balance(t, cp, s.abc), "the token's community pool gets nothing on a buy")
	require.True(t, s.balance(t, cp, chain.Denom).Equal(sdkmath.NewInt(ammTokenCpFunds)), "nor any ubze")

	// 3. config back to the defaults: today's path again
	s.mustExec(t, s.lp, "tokenfactory", "set-denom-config", s.abc, "", "0", "0")
	pool = s.pool(t, s.cpPool)
	require.True(t, pool.reserveQuote().LTE(s.minNativeLiquidity(t)), "test setup: the pool is too shallow for a module swap")
	q = ammQuoteSwap(sdkmath.NewInt(ammTokenCpSell), pool.reserveBase(), pool.reserveQuote(), ammMinPoolFee, "0.3", "0.2")
	res = s.mustExec(t, s.trader, "tradebin", "multi-swap", s.cpPool, coinStr(ammTokenCpSell, s.abc), coinStr(1, chain.Denom))
	require.Equal(t, q.treasury, s.transferred(t, res.Events, s.tradebin, s.cpFees, s.abc), "treasury share to txfeecollector_cp")
	block = s.blockEvents(t, res.Height)
	require.True(t, s.transferred(t, block, s.cpFees, cp, s.abc).IsZero(), "nothing forwarded once the address is cleared")
	require.Equal(t, q.treasury, s.transferred(t, block, s.cpFees, s.burner, s.abc), "unswappable abc share to the burner")
	s.requireCommunityPoolPaid(t, res.Height, nil)
	require.Equal(t, cpBefore.Add(forwarded), s.balance(t, cp, s.abc), "community pool balance unchanged")
}

// requireCommunityPoolPaid asserts the CommunityPoolFeePaidEvents of the block
// at height, in order.
func (s *ammSuite) requireCommunityPoolPaid(t *testing.T, height int64, want []map[string]string) {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, height, communityPoolPaidT)
	require.NoError(t, err)
	got := make([]map[string]string, 0, len(events))
	for _, e := range events {
		got = append(got, map[string]string{"denom": e.Attributes["denom"], "recipient": e.Attributes["recipient"], "amount": e.Attributes["amount"]})
	}
	if want == nil {
		want = []map[string]string{}
	}
	require.Equal(t, want, got, "CommunityPoolFeePaidEvents")
}

// minNativeLiquidity is tradebin's min_native_liquidity_for_module_swap.
func (s *ammSuite) minNativeLiquidity(t *testing.T) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Params struct {
			MinNativeLiquidityForModuleSwap string `json:"minNativeLiquidityForModuleSwap"`
		} `json:"params"`
	}](s.ctx, s.bze, "tradebin", "params")
	require.NoError(t, err)
	return parseInt(res.Params.MinNativeLiquidityForModuleSwap)
}
