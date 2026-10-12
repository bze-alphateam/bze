package interchaintest_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// The ubze side of hlt/ubze and usd/ubze is above the tradebin
// minNativeLiquidityForModuleSwap param (100,000 BZE by default), so both
// tokens qualify as fee denoms while they are not halted.
const (
	haltAdminFunds  = 1_000_000_000_000 // 1,000,000 BZE: three denoms, three pools, a market, proposals
	haltTraderFunds = 50_000_000_000    // 50,000 BZE: enough for the create fees it is refused before paying

	haltMint = 1_000_000_000_000
	// hlt/ubze and usd/ubze: 4 ubze per token
	haltPoolToken = 50_000_000_000
	haltPoolBze   = 200_000_000_000
	// hlt/usd: 1 usd per hlt
	haltPoolPair = 10_000_000_000

	haltTraderTokens = 1_000_000_000 // hlt, usd and oth sent to trader
	// trader's deposit to hlt/ubze, at the pool price
	haltLpHlt = 100_000_000
	haltLpBze = 400_000_000

	haltOrderAmount = 1_000
	haltOrderPrice  = "10" // ubze per hlt, far above the pool price: the order rests
	haltSwapInput   = 1_000_000
	haltFee         = 20_000 // a tx fee in hlt or usd
	haltTxGas       = 500_000
	haltSendGas     = 200_000

	haltCodespace = "tradebin"
	haltCode      = 4017 // ErrDenomHalted
)

// TestHaltedDenoms halts a factory denom through a real governance proposal
// carrying MsgHaltDenoms and proves the tradebin msg servers refuse every
// trading path that touches it (code 4017, nothing moves but the tx fee),
// while cancelling an order, removing liquidity and bank sends keep working,
// and the ante handler no longer takes it as a fee denom. A second proposal
// unhalts it and trading works again. Subtests run in order and share state;
// the step numbers match the scenario of the suite's design.
func TestHaltedDenoms(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupHalt(t, ctx, net.Chain)

	t.Run("1 before the halt", s.beforeHalt)
	t.Run("2 halt by governance", s.halt)
	t.Run("3 trading refused", s.tradingRefused)
	t.Run("4 exits still work", s.exits)
	t.Run("5 fee denom", s.feeDenom)
	t.Run("6 unhalt by governance", s.unhalt)
	t.Run("7 native denom can not be halted", s.haltNative)
}

type haltSuite struct {
	// the AMM suite's query and tx helpers (pool, balances, supply, events)
	*ammSuite

	admin, trader ibc.Wallet
	gov           string

	hlt, usd, oth           string
	hltPool, usdPool, pair  string // hlt/ubze, usd/ubze, hlt/usd
	hltLp                   string
	market                  string // hlt/ubze
	orderID                 string // trader's resting sell order, placed before the halt
	minNativeLiquidityParam sdkmath.Int
}

func setupHalt(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *haltSuite {
	s := &haltSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}}

	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "halt-admin", sdkmath.NewInt(haltAdminFunds), bze)[0]
	s.trader = interchaintest.GetAndFundTestUsers(t, ctx, "halt-trader", sdkmath.NewInt(haltTraderFunds), bze)[0]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	var err error
	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)
	for name, addr := range map[string]*string{"tradebin": &s.tradebin, "txfeecollector": &s.txFees} {
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	s.params = s.queryParams(t)
	s.minNativeLiquidityParam = s.minNativeLiquidity(t)
	require.True(t, sdkmath.NewInt(haltPoolBze).GTE(s.minNativeLiquidityParam), "the pools must qualify their tokens as fee denoms")

	factory := "factory/" + s.admin.FormattedAddress() + "/"
	s.hlt, s.usd, s.oth = factory+"hlt", factory+"usd", factory+"oth"
	for _, sub := range []string{"hlt", "usd", "oth"} {
		s.mustExec(t, s.admin, "tokenfactory", "create-denom", sub)
		s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(haltMint, factory+sub))
		s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), s.trader.FormattedAddress(), coinStr(haltTraderTokens, factory+sub))
	}

	// pool ids order the denoms alphabetically: "factory/…/hlt" < "factory/…/usd" < "ubze"
	s.hltPool = s.hlt + "_" + chain.Denom
	s.usdPool = s.usd + "_" + chain.Denom
	s.pair = s.hlt + "_" + s.usd
	s.market = s.hlt + "/" + chain.Denom
	for _, p := range []struct {
		base, quote         string
		initBase, initQuote int64
	}{
		{s.hlt, chain.Denom, haltPoolToken, haltPoolBze},
		{s.usd, chain.Denom, haltPoolToken, haltPoolBze},
		{s.hlt, s.usd, haltPoolPair, haltPoolPair},
	} {
		s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(haltTxGas)},
			"tradebin", "create-liquidity-pool", p.base, p.quote, ammPoolFee, ammFeeDest, "false",
			sdkmath.NewInt(p.initBase).String(), sdkmath.NewInt(p.initQuote).String())
	}
	s.hltLp = s.pool(t, s.hltPool).LpDenom
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(haltTxGas)}, "tradebin", "create-market", s.hlt, chain.Denom)

	// trader: LP shares of hlt/ubze and one resting sell order
	s.mustExec(t, s.trader, "tradebin", "add-liquidity", s.hltPool,
		sdkmath.NewInt(haltLpHlt).String(), sdkmath.NewInt(haltLpBze).String(), "1")
	require.True(t, s.balance(t, s.trader.FormattedAddress(), s.hltLp).IsPositive(), "trader holds hlt/ubze LP")
	s.orderID = s.placeSellOrder(t)
	return s
}

// 1. hlt is not halted and, with a deep ubze pool, pays tx fees: the fee goes
// to the txfeecollector, which swaps it to ubze through hlt/ubze in EndBlock.
func (s *haltSuite) beforeHalt(t *testing.T) {
	s.requireHaltedDenoms(t)
	require.False(t, s.denomHalted(t, s.hlt))

	res := s.mustExecWith(t, s.trader, []helpers.TxOption{helpers.WithFees(coinStr(haltFee, s.hlt)), helpers.WithGas(haltSendGas)},
		"bank", "send", s.trader.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
	paid, err := helpers.TxFee(res, s.hlt)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(haltFee), paid, "fee paid in hlt")
	swaps := helpers.EventsWith(filterType(s.blockEvents(t, res.Height), tradebinEventPrefix+"SwapEvent"), map[string]string{"creator": s.txFees, "pool_id": s.hltPool})
	require.Len(t, swaps, 1, "the hlt fee is swapped to ubze in EndBlock")
	require.Equal(t, sdk.NewInt64Coin(s.hlt, haltFee), eventCoin(t, swaps[0], "in"))
}

// 2. A proposal halts hlt. Its second message halts the already-halted denom
// again: a no-op without a second event. The same message signed by a user
// is refused.
func (s *haltSuite) halt(t *testing.T) {
	halt := s.haltMsg("MsgHaltDenoms", s.gov, s.hlt)
	height := s.passProposal(t, "Halt hlt", halt, halt)

	events := s.blockTypedEvents(t, height, "DenomHaltedEvent")
	require.Len(t, events, 1, "one DenomHaltedEvent in the block that executed the proposal, none for the repeat: %+v", events)
	require.Equal(t, s.hlt, events[0].Attributes["denom"])
	require.True(t, s.denomHalted(t, s.hlt))
	s.requireHaltedDenoms(t, s.hlt)

	for _, typ := range []string{"MsgHaltDenoms", "MsgUnhaltDenoms"} {
		t.Run(typ+" signed by a user", func(t *testing.T) {
			res, err := helpers.ExecMsgs(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithGas(haltTxGas)},
				s.haltMsg(typ, s.trader.FormattedAddress(), s.hlt))
			reason, err := helpers.RejectionReason(res, err)
			require.NoError(t, err)
			require.Contains(t, reason, "invalid authority")
			require.True(t, s.denomHalted(t, s.hlt), "hlt still halted")
		})
	}
}

// 3. Every trading entry point that touches hlt is refused by its msg server
// with ErrDenomHalted. The tx is delivered (fixed gas) so the code is the
// msg server's; pools, markets, the order book and the trader's balances are
// unchanged apart from the ubze tx fee.
func (s *haltSuite) tradingRefused(t *testing.T) {
	cli := func(args ...string) func() (helpers.TxResult, error) {
		return func() (helpers.TxResult, error) {
			return helpers.ExecTxWith(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithGas(haltTxGas)}, args...)
		}
	}
	amount := sdkmath.NewInt(haltOrderAmount).String()
	for _, c := range []struct {
		name string
		tx   func() (helpers.TxResult, error)
	}{
		{"create order", cli("tradebin", "create-order", "buy", amount, haltOrderPrice, s.market)},
		{"swap hlt to ubze", cli("tradebin", "multi-swap", s.hltPool, coinStr(haltSwapInput, s.hlt), coinStr(1, chain.Denom))},
		{"swap ubze to hlt", cli("tradebin", "multi-swap", s.hltPool, coinStr(haltSwapInput, chain.Denom), coinStr(1, s.hlt))},
		{"multi-hop through hlt", cli("tradebin", "multi-swap", s.pair+","+s.hltPool, coinStr(haltSwapInput, s.usd), coinStr(1, chain.Denom))},
		{"add liquidity", cli("tradebin", "add-liquidity", s.hltPool, sdkmath.NewInt(haltLpHlt).String(), sdkmath.NewInt(haltLpBze).String(), "1")},
		{"create pool", cli("tradebin", "create-liquidity-pool", s.hlt, s.oth, ammPoolFee, ammFeeDest, "false",
			sdkmath.NewInt(haltSwapInput).String(), sdkmath.NewInt(haltSwapInput).String())},
		{"create market", cli("tradebin", "create-market", s.hlt, s.usd)},
		{"fill orders", func() (helpers.TxResult, error) {
			return helpers.ExecMsgs(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithGas(haltTxGas)}, map[string]any{
				"@type":      "/bze.tradebin.MsgFillOrders",
				"creator":    s.trader.FormattedAddress(),
				"market_id":  s.market,
				"order_type": "sell",
				"orders":     []map[string]string{{"price": haltOrderPrice, "amount": amount}},
			})
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := s.snapshot(t)
			res, err := c.tx()
			require.NoError(t, err)
			require.NotZero(t, res.Height, "refused by CheckTx instead of the msg server: %s", res.RawLog)
			require.Equal(t, haltCodespace, res.Codespace, res.RawLog)
			require.EqualValues(t, haltCode, res.Code, res.RawLog)
			require.Contains(t, res.RawLog, "halted")

			after := s.snapshot(t)
			require.Equal(t, before.pools, after.pools, "pools changed")
			require.Equal(t, before.markets, after.markets, "markets changed")
			require.Equal(t, before.orders, after.orders, "order book changed")
			require.Equal(t, before.balances.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), after.balances, "a refused tx only costs the tx fee")
		})
	}
	require.Equal(t, s.orderID, s.restingOrder(t).ID, "the resting order is still on the book")
}

// 4. Exits are never halted: the resting order is cancelled and refunded,
// the trader's LP shares are redeemed for both sides, and hlt still moves by
// bank send.
func (s *haltSuite) exits(t *testing.T) {
	t.Run("cancel order", func(t *testing.T) {
		before := s.balance(t, s.trader.FormattedAddress(), s.hlt)
		res := s.mustExec(t, s.trader, "tradebin", "cancel-order", s.market, s.orderID, "sell")
		canceled := helpers.EventsWith(filterType(s.blockEvents(t, res.Height), tradebinEventPrefix+"OrderCanceledEvent"), map[string]string{"id": s.orderID})
		require.Len(t, canceled, 1, "the cancel executes in EndBlock")
		require.Equal(t, before.AddRaw(haltOrderAmount), s.balance(t, s.trader.FormattedAddress(), s.hlt), "order amount refunded")
		require.Empty(t, s.traderOrders(t), "no order left on the book")
	})

	t.Run("remove liquidity", func(t *testing.T) {
		pool := s.pool(t, s.hltPool)
		supply := s.supply(t, s.hltLp)
		burn := s.balance(t, s.trader.FormattedAddress(), s.hltLp)
		// out = ⌊reserve × burn / supply⌋
		wantHlt := pool.reserveBase().Mul(burn).Quo(supply)
		wantBze := pool.reserveQuote().Mul(burn).Quo(supply)
		require.True(t, wantHlt.IsPositive() && wantBze.IsPositive())
		before := s.balances(t, s.trader, chain.Denom, s.hlt, s.hltLp)

		res := s.mustExec(t, s.trader, "tradebin", "remove-liquidity", s.hltPool, burn.String(), wantHlt.String(), wantBze.String())

		s.requireTxEvent(t, res, "LiquidityRemovedEvent", map[string]string{
			"pool_id": s.hltPool, "baseAmount": wantHlt.String(), "quoteAmount": wantBze.String(),
		})
		after := s.pool(t, s.hltPool)
		require.Equal(t, pool.reserveBase().Sub(wantHlt), after.reserveBase())
		require.Equal(t, pool.reserveQuote().Sub(wantBze), after.reserveQuote())
		want := before.Add(sdk.NewCoin(s.hlt, wantHlt), sdk.NewCoin(chain.Denom, wantBze)).
			Sub(sdk.NewCoin(s.hltLp, burn), sdk.NewCoin(chain.Denom, s.txFee(t, res)))
		require.Equal(t, want, s.balances(t, s.trader, chain.Denom, s.hlt, s.hltLp), "trader balances")
	})

	t.Run("bank send", func(t *testing.T) {
		before := s.balance(t, s.admin.FormattedAddress(), s.hlt)
		s.mustExec(t, s.trader, "bank", "send", s.trader.FormattedAddress(), s.admin.FormattedAddress(), coinStr(haltOrderAmount, s.hlt))
		require.Equal(t, before.AddRaw(haltOrderAmount), s.balance(t, s.admin.FormattedAddress(), s.hlt))
	})
}

// 5. The ante handler refuses a fee in a halted denom (a halted denom never
// has usable liquidity), before the tx reaches a block: nothing is charged.
// A fee in usd, whose pool is not halted, still works.
func (s *haltSuite) feeDenom(t *testing.T) {
	t.Run("hlt refused", func(t *testing.T) {
		before := s.balances(t, s.trader, chain.Denom, s.hlt)
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithFees(coinStr(haltFee, s.hlt)), helpers.WithGas(haltSendGas)},
			"bank", "send", s.trader.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
		reason, err := helpers.RejectionReason(res, err)
		require.NoError(t, err)
		require.Contains(t, reason, s.hlt+" can be used to pay for fees only if enough liquidity is available")
		require.Zero(t, res.Height, "refused by CheckTx")
		require.Equal(t, before, s.balances(t, s.trader, chain.Denom, s.hlt), "nothing charged")
	})

	t.Run("usd accepted", func(t *testing.T) {
		res := s.mustExecWith(t, s.trader, []helpers.TxOption{helpers.WithFees(coinStr(haltFee, s.usd)), helpers.WithGas(haltSendGas)},
			"bank", "send", s.trader.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
		paid, err := helpers.TxFee(res, s.usd)
		require.NoError(t, err)
		require.Equal(t, sdkmath.NewInt(haltFee), paid, "fee paid in usd")
	})
}

// 6. A second proposal lifts the halt; its repeated message finds hlt no
// longer halted and emits nothing. Swaps, liquidity and orders work again.
func (s *haltSuite) unhalt(t *testing.T) {
	unhalt := s.haltMsg("MsgUnhaltDenoms", s.gov, s.hlt)
	height := s.passProposal(t, "Unhalt hlt", unhalt, unhalt)

	events := s.blockTypedEvents(t, height, "DenomUnhaltedEvent")
	require.Len(t, events, 1, "one DenomUnhaltedEvent, none for the repeat: %+v", events)
	require.Equal(t, s.hlt, events[0].Attributes["denom"])
	require.False(t, s.denomHalted(t, s.hlt))
	s.requireHaltedDenoms(t)

	t.Run("swap", func(t *testing.T) {
		pool := s.pool(t, s.hltPool)
		in := sdkmath.NewInt(haltSwapInput)
		q := quoteSwap(in, pool.reserveQuote(), pool.reserveBase())
		res := s.mustExec(t, s.trader, "tradebin", "multi-swap", s.hltPool, sdk.NewCoin(chain.Denom, in).String(), coinStr(1, s.hlt))
		s.requireSwap(t, res.Events, s.trader.FormattedAddress(), s.hltPool, sdk.NewCoin(chain.Denom, in), sdk.NewCoin(s.hlt, q.out))
	})

	t.Run("add liquidity", func(t *testing.T) {
		before := s.balance(t, s.trader.FormattedAddress(), s.hltLp)
		res := s.mustExec(t, s.trader, "tradebin", "add-liquidity", s.hltPool,
			sdkmath.NewInt(haltLpHlt).String(), sdkmath.NewInt(haltLpBze).String(), "1")
		s.requireTxEvent(t, res, "LiquidityAddedEvent", map[string]string{"pool_id": s.hltPool})
		require.True(t, s.balance(t, s.trader.FormattedAddress(), s.hltLp).GT(before), "LP minted")
	})

	t.Run("new order", func(t *testing.T) {
		id := s.placeSellOrder(t)
		require.Equal(t, id, s.restingOrder(t).ID)
	})
}

// 7. Halting the native denom would halt every market and pool: the keeper
// refuses it, so the proposal passes the vote but fails on execution and
// nothing is halted.
func (s *haltSuite) haltNative(t *testing.T) {
	_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: []any{s.haltMsg("MsgHaltDenoms", s.gov, chain.Denom)},
		Title:    "Halt ubze",
		Summary:  "e2e: halting the native denom must fail",
	})
	require.ErrorContains(t, err, "PROPOSAL_STATUS_FAILED")
	require.ErrorContains(t, err, "the native denom "+chain.Denom+" cannot be halted")
	require.False(t, s.denomHalted(t, chain.Denom))
	s.requireHaltedDenoms(t)
}

func (s *haltSuite) haltMsg(typ, authority string, denoms ...string) map[string]any {
	return map[string]any{
		"@type":     "/bze.tradebin." + typ,
		"authority": authority,
		"denoms":    denoms,
	}
}

// passProposal submits msgs as one proposal from admin, passes it and
// returns the height of the block whose EndBlock executed it.
func (s *haltSuite) passProposal(t *testing.T, title string, msgs ...any) int64 {
	t.Helper()
	from, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	id, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: msgs,
		Title:    title,
		Summary:  "e2e: " + title,
	})
	require.NoError(t, err)

	to, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	attrs := map[string]string{"proposal_id": strconv.FormatUint(id, 10), "proposal_result": "proposal_passed"}
	for h := from; h <= to; h++ {
		events, err := helpers.BlockEvents(s.ctx, s.bze, h, "active_proposal")
		require.NoError(t, err)
		if len(helpers.EventsWith(events, attrs)) > 0 {
			return h
		}
	}
	t.Fatalf("no active_proposal event for proposal %d between heights %d and %d", id, from, to)
	return 0
}

func (s *haltSuite) blockTypedEvents(t *testing.T, height int64, name string) []helpers.Event {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, height, tradebinEventPrefix+name)
	require.NoError(t, err)
	return events
}

func (s *haltSuite) denomHalted(t *testing.T, denom string) bool {
	t.Helper()
	res, err := helpers.Query[struct {
		Halted bool `json:"halted"`
	}](s.ctx, s.bze, "tradebin", "denom-halted", denom)
	require.NoError(t, err)
	return res.Halted
}

func (s *haltSuite) requireHaltedDenoms(t *testing.T, want ...string) {
	t.Helper()
	res, err := helpers.Query[struct {
		Denoms []string `json:"denoms"`
	}](s.ctx, s.bze, "tradebin", "halted-denoms")
	require.NoError(t, err)
	require.ElementsMatch(t, want, res.Denoms, "halted denoms")
}

// placeSellOrder places the trader's sell order on the hlt market at a price
// nobody fills and returns the id the EndBlock saved it under.
func (s *haltSuite) placeSellOrder(t *testing.T) string {
	t.Helper()
	before := s.balance(t, s.trader.FormattedAddress(), s.hlt)
	res := s.mustExec(t, s.trader, "tradebin", "create-order", "sell", sdkmath.NewInt(haltOrderAmount).String(), haltOrderPrice, s.market)
	msgID := s.requireTxEvent(t, res, "OrderCreateMessageEvent", map[string]string{"market_id": s.market}).Attributes["message_id"]
	saved := helpers.EventsWith(filterType(s.blockEvents(t, res.Height), tradebinEventPrefix+"OrderSavedEvent"), map[string]string{"message_id": msgID})
	require.Len(t, saved, 1, "the order is saved by the EndBlock that processed message %s", msgID)
	require.Equal(t, before.SubRaw(haltOrderAmount), s.balance(t, s.trader.FormattedAddress(), s.hlt), "order amount locked")
	return saved[0].Attributes["id"]
}

type haltOrder struct {
	ID        string `json:"id"`
	MarketID  string `json:"market_id"`
	OrderType string `json:"order_type"`
}

func (s *haltSuite) traderOrders(t *testing.T) []haltOrder {
	t.Helper()
	res, err := helpers.Query[struct {
		List []haltOrder `json:"list"`
	}](s.ctx, s.bze, "tradebin", "user-market-orders", s.trader.FormattedAddress(), s.market)
	require.NoError(t, err)
	return res.List
}

// restingOrder is the trader's only order on the hlt market.
func (s *haltSuite) restingOrder(t *testing.T) haltOrder {
	t.Helper()
	orders := s.traderOrders(t)
	require.Len(t, orders, 1, "trader orders: %+v", orders)
	require.Equal(t, s.market, orders[0].MarketID)
	require.Equal(t, "sell", orders[0].OrderType)
	return orders[0]
}

// haltSnapshot is the state a refused trading tx must not touch.
type haltSnapshot struct {
	pools, markets, orders string
	balances               sdk.Coins
}

func (s *haltSuite) snapshot(t *testing.T) haltSnapshot {
	t.Helper()
	raw := func(args ...string) string {
		res, err := helpers.Query[json.RawMessage](s.ctx, s.bze, append([]string{"tradebin"}, args...)...)
		require.NoError(t, err)
		return string(res)
	}
	return haltSnapshot{
		pools:    raw("all-liquidity-pools"),
		markets:  raw("all-markets"),
		orders:   raw("market-aggregated-orders", s.market, "sell") + raw("market-aggregated-orders", s.market, "buy") + raw("user-market-orders", s.trader.FormattedAddress(), s.market),
		balances: s.balances(t, s.trader, chain.Denom, s.hlt, s.usd, s.oth, s.hltLp),
	}
}

func (s *haltSuite) minNativeLiquidity(t *testing.T) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Params struct {
			MinNativeLiquidityForModuleSwap string `json:"minNativeLiquidityForModuleSwap"`
		} `json:"params"`
	}](s.ctx, s.bze, "tradebin", "params")
	require.NoError(t, err)
	return parseInt(res.Params.MinNativeLiquidityForModuleSwap)
}
