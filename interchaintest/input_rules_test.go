package interchaintest_test

import (
	"context"
	"strconv"
	"strings"
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

const (
	irAdminFunds   = 1_000_000_000_000 // 1,000,000 BZE: a denom, a pool, a market, a staking reward
	irUserFunds    = 10_000_000_000    // 10,000 BZE
	irGranterFunds = 40_000_000_000    // 40,000 BZE: one create-denom fee (25,000 BZE) in ubze

	irMint = 1_000_000_000_000
	// tok/ubze: 1 tok = 1000 ubze, the ubze side above the default
	// minNativeLiquidityForModuleSwap (1e11) so tok pays tx and protocol fees
	irPoolTok = 200_000_000
	irPoolBze = 200_000_000_000

	irUserTok = 1_000_000 // tok to each user: about 1,000 BZE, for tx fees and orders
	// tok to the granter: enough to buy a 25,000 BZE create-denom fee out of
	// the pool (about 29,000,000 tok), so a fee taken in ubze is the fee payer
	// rule, not a lack of tok
	irGranterTok = 50_000_000
	irUserStk    = 1_000
	irOrderPrice = "1000" // ubze per tok: the pool price; there is no buy on the book, so the sell rests
	irTokTxFee   = 20     // tok, about 20,000 ubze: above the minimum at irTxGas
	irTxGas      = 500_000

	irCanonicalMsg = "must be a canonical base-10 integer"
	irRafflePot    = "100000" // the burner's minimum raffle pot
)

// irNonCanonical are integers the chain could parse but that are not
// written in the one canonical base-10 form: every amount field refuses them.
var irNonCanonical = []string{"0100", "+100", "0x64", "1_00"}

// TestInputRules covers v8.3.0 input rules a user meets through the real
// binary:
//   - amounts in tradebin, burner and rewards messages must be canonical
//     base-10 integers; other spellings of the same integer are refused in
//     CheckTx and charge nothing, the canonical spelling is accepted;
//   - x/rewards treats the uppercase and lowercase bech32 spellings of a
//     sender as one account: a join in uppercase and a top-up in lowercase
//     make one position, found by either spelling;
//   - a protocol fee follows the tx fee denom only for the account paying
//     the tx fee: a create-denom executed for a granter through authz MsgExec
//     charges the granter in ubze although the grantee pays the tx fee in
//     tok, while the granter's own create-denom with a tok tx fee pays its
//     fee in tok.
func TestInputRules(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupInputRules(t, ctx, net.Chain)

	t.Run("1 non-canonical amounts are refused", s.nonCanonicalAmounts)
	t.Run("2 one rewards position per account, whatever the address case", s.addressCase)
	t.Run("3 a protocol fee inside authz exec is charged in ubze", s.authzProtocolFee)
}

type inputRulesSuite struct {
	*feesSuite

	user, granter, grantee ibc.Wallet
	stk                    string
	market                 string
	srID                   string
}

func setupInputRules(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *inputRulesSuite {
	s := &inputRulesSuite{feesSuite: &feesSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}}}
	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "ir-admin", sdkmath.NewInt(irAdminFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "ir", sdkmath.NewInt(irUserFunds), bze, bze)
	s.user, s.grantee = users[0], users[1]
	s.granter = interchaintest.GetAndFundTestUsers(t, ctx, "ir-granter", sdkmath.NewInt(irGranterFunds), bze)[0]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	for name, addr := range map[string]*string{
		"tradebin":          &s.tradebin,
		"txfeecollector_cp": &s.cpFees,
		"tokenfactory":      &s.tfAcc,
	} {
		var err error
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	s.params = s.queryParams(t)

	factory := "factory/" + s.admin.FormattedAddress() + "/"
	s.tok, s.stk = factory+"tok", factory+"stk"
	for _, d := range []string{s.tok, s.stk} {
		s.mustExec(t, s.admin, "tokenfactory", "create-denom", strings.TrimPrefix(d, factory))
		s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(irMint, d))
	}
	for _, u := range []ibc.Wallet{s.user, s.granter, s.grantee} {
		send := sdk.NewCoins(sdk.NewInt64Coin(s.tok, irUserTok), sdk.NewInt64Coin(s.stk, irUserStk))
		s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), u.FormattedAddress(), send.String())
	}
	s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), s.granter.FormattedAddress(), coinStr(irGranterTok, s.tok))

	// "factory/…" sorts before "ubze": tok is the base of its pool and market
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(irTxGas)},
		"tradebin", "create-liquidity-pool", s.tok, chain.Denom, ammPoolFee, ammFeeDest, "false",
		strconv.Itoa(irPoolTok), strconv.Itoa(irPoolBze))
	s.market = s.tok + "/" + chain.Denom
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(irTxGas)}, "tradebin", "create-market", s.tok, chain.Denom)

	// a staking reward on stk, paying ubze
	res := s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(irTxGas)},
		"rewards", "create-staking-reward", "1000", chain.Denom, s.stk, "100", "1", "0")
	created := helpers.EventsWith(helpers.TxEvents(res, drEventPrefix+"StakingRewardCreateEvent"), map[string]string{"staking_denom": s.stk})
	require.Len(t, created, 1, "StakingRewardCreateEvent")
	s.srID = created[0].Attributes["reward_id"]
	require.NotEmpty(t, s.srID)
	return s
}

// requireRefusedInCheckTx sends msgs signed as user and requires CheckTx to
// refuse them with reason in the log: never included, nothing charged.
func (s *inputRulesSuite) requireRefusedInCheckTx(t *testing.T, user ibc.Wallet, reason string, msgs ...any) {
	t.Helper()
	before := s.addrBalances(t, user.FormattedAddress(), 0)
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(irTxGas)}, msgs...)
	require.NoError(t, err)
	require.NotZero(t, res.Code, "refused: %+v", msgs)
	require.Zero(t, res.Height, "refused in CheckTx: %s", res.RawLog)
	require.Contains(t, res.RawLog, reason)
	require.Equal(t, before, s.addrBalances(t, user.FormattedAddress(), 0), "nothing charged")
}

// 1. Every amount field of the order, raffle and reward messages refuses an
// integer written in another form than its canonical one.
func (s *inputRulesSuite) nonCanonicalAmounts(t *testing.T) {
	u := s.user.FormattedAddress()
	order := func(amount string) map[string]any {
		return map[string]any{"@type": "/bze.tradebin.MsgCreateOrder", "creator": u, "order_type": "sell", "amount": amount, "price": irOrderPrice, "market_id": s.market}
	}
	fill := func(amount string) map[string]any {
		return map[string]any{"@type": "/bze.tradebin.MsgFillOrders", "creator": u, "market_id": s.market, "order_type": "buy",
			"orders": []map[string]string{{"price": irOrderPrice, "amount": amount}}}
	}
	raffle := func(pot, ticket string) map[string]any {
		return map[string]any{"@type": "/bze.burner.MsgStartRaffle", "creator": u, "pot": pot, "duration": "1", "chances": "1",
			"ratio": "0.5", "ticket_price": ticket, "denom": s.tok}
	}
	stakingReward := func(prize, minStake string) map[string]any {
		return map[string]any{"@type": "/bze.rewards.MsgCreateStakingReward", "creator": u, "prize_amount": prize, "prize_denom": chain.Denom,
			"staking_denom": s.stk, "duration": "1", "min_stake": minStake, "lock": "0"}
	}
	join := func(amount string) map[string]any {
		return map[string]any{"@type": "/bze.rewards.MsgJoinStaking", "creator": u, "reward_id": s.srID, "amount": amount}
	}
	distribute := func(amount string) map[string]any {
		return map[string]any{"@type": "/bze.rewards.MsgDistributeStakingRewards", "creator": u, "reward_id": s.srID, "amount": amount}
	}
	tradingReward := func(prize string) map[string]any {
		return map[string]any{"@type": "/bze.rewards.MsgCreateTradingReward", "creator": u, "prize_amount": prize, "prize_denom": chain.Denom,
			"duration": "1", "market_id": s.market, "slots": "1"}
	}
	for _, v := range irNonCanonical {
		for _, c := range []struct {
			name string
			msg  map[string]any
		}{
			{"order amount", order(v)},
			{"fill orders amount", fill(v)},
			{"raffle pot", raffle(v, "10")},
			// the pot is checked first and must be at least 100,000
			{"raffle ticket price", raffle(irRafflePot, v)},
			{"staking reward prize", stakingReward(v, "1")},
			{"staking reward min stake", stakingReward("100", v)},
			{"join staking amount", join(v)},
			{"distribute staking rewards amount", distribute(v)},
			{"trading reward prize", tradingReward(v)},
		} {
			t.Run(c.name+" "+v, func(t *testing.T) {
				s.requireRefusedInCheckTx(t, s.user, irCanonicalMsg, c.msg)
			})
		}
	}

	// the canonical spelling of the same amount goes through
	t.Run("canonical order amount", func(t *testing.T) {
		before := s.balance(t, u, s.tok)
		res, err := helpers.ExecMsgs(s.ctx, s.bze, s.user, []helpers.TxOption{helpers.WithGas(irTxGas)}, order("100"))
		require.NoError(t, err)
		require.Zero(t, res.Code, res.RawLog)
		require.Equal(t, before.SubRaw(100), s.balance(t, u, s.tok), "100 tok locked in the order")
	})
}

// 2. A join in the uppercase spelling of the user's address and a top-up in
// the lowercase one make a single position under the lowercase address; the
// participant query finds it by either spelling.
func (s *inputRulesSuite) addressCase(t *testing.T) {
	lower := s.user.FormattedAddress()
	upper := strings.ToUpper(lower)
	require.NotEqual(t, lower, upper)

	join := func(creator string, amount int64) {
		t.Helper()
		res, err := helpers.ExecMsgs(s.ctx, s.bze, s.user, []helpers.TxOption{helpers.WithGas(irTxGas)},
			map[string]any{"@type": "/bze.rewards.MsgJoinStaking", "creator": creator, "reward_id": s.srID, "amount": strconv.FormatInt(amount, 10)})
		require.NoError(t, err)
		require.Zero(t, res.Code, res.RawLog)
	}
	before := s.balance(t, lower, s.stk)
	join(upper, 100)
	join(lower, 50)
	require.Equal(t, before.SubRaw(150), s.balance(t, lower, s.stk), "both joins staked from the same account")

	participants := func(addr string) []srParticipant {
		res, err := helpers.Query[struct {
			List []srParticipant `json:"list"`
		}](s.ctx, s.bze, "rewards", "staking-reward-participant", addr)
		require.NoError(t, err)
		return res.List
	}
	got := participants(lower)
	require.Len(t, got, 1, "one position: %+v", got)
	require.Equal(t, s.srID, got[0].RewardID)
	require.Equal(t, "150", got[0].Amount, "the top-up landed on the uppercase join")
	require.Equal(t, got, participants(upper), "the uppercase spelling finds the same position")
}

// 3. The granter lets the grantee execute MsgCreateDenom for it. The grantee
// pays the tx fee in tok; the create-denom fee is charged to the granter, who
// is not the fee payer, so it is taken in ubze even though the granter holds
// tok. The granter's own create-denom with a tok tx fee pays its fee in tok.
func (s *inputRulesSuite) authzProtocolFee(t *testing.T) {
	granter, grantee := s.granter.FormattedAddress(), s.grantee.FormattedAddress()
	fee := s.tokenfactoryFee(t)
	require.Equal(t, chain.Denom, fee.Denom)
	s.mustExec(t, s.granter, "authz", "grant", grantee, "generic", "--msg-type", "/bze.tokenfactory.MsgCreateDenom")

	tokFee := helpers.WithFees(coinStr(irTokTxFee, s.tok))
	t.Run("through authz exec: the granter pays in ubze", func(t *testing.T) {
		gr, ge := s.addrBalances(t, granter, 0), s.addrBalances(t, grantee, 0)
		res, err := helpers.ExecMsgs(s.ctx, s.bze, s.grantee, []helpers.TxOption{tokFee, helpers.WithGas(irTxGas)},
			map[string]any{"@type": "/cosmos.authz.v1beta1.MsgExec", "grantee": grantee,
				"msgs": []any{map[string]any{"@type": "/bze.tokenfactory.MsgCreateDenom", "creator": granter, "subdenom": "viaexec"}}})
		require.NoError(t, err)
		require.Zero(t, res.Code, res.RawLog)
		require.True(t, s.transfers(t, res.Events, granter, s.tradebin).IsZero(), "nothing captured from the granter in tok")
		require.True(t, s.transfers(t, res.Events, grantee, s.tradebin).IsZero(), "nothing captured from the grantee")
		require.Equal(t, sdk.NewCoins(fee).String(), s.transfers(t, res.Events, granter, s.tfAcc).String(), "the fee in ubze from the granter")
		require.Equal(t, gr.Sub(fee).String(), s.addrBalances(t, granter, 0).String(), "the granter pays the fee in ubze, nothing in tok")
		require.Equal(t, ge.Sub(sdk.NewInt64Coin(s.tok, irTokTxFee)).String(), s.addrBalances(t, grantee, 0).String(), "the grantee pays the tx fee only")
		require.Equal(t, granter, s.denomAdmin(t, "factory/"+granter+"/viaexec"))
	})

	t.Run("the granter's own tx: paid in tok", func(t *testing.T) {
		require.NoError(t, helpers.WaitForPoolPriceCalm(s.ctx, s.bze, ammCalmBlocks, s.tok+"_"+chain.Denom))
		gr := s.addrBalances(t, granter, 0)
		res := s.mustExecWith(t, s.granter, []helpers.TxOption{tokFee, helpers.WithGas(irTxGas)}, "tokenfactory", "create-denom", "own")
		captured := s.transfers(t, res.Events, granter, s.tradebin).AmountOf(s.tok)
		require.True(t, captured.IsPositive(), "the fee is captured in tok")
		require.Equal(t, gr.AmountOf(chain.Denom).String(), s.addrBalances(t, granter, 0).AmountOf(chain.Denom).String(), "no ubze spent")
		require.True(t, s.transfers(t, res.Events, s.tfAcc, s.cpFees).AmountOf(chain.Denom).GTE(fee.Amount), "the swapped fee reaches the community-pool fee collector")
	})
}

// denomAdmin is the admin of a factory denom.
func (s *inputRulesSuite) denomAdmin(t *testing.T, denom string) string {
	t.Helper()
	res, err := helpers.Query[struct {
		DenomAuthority struct {
			Admin string `json:"admin"`
		} `json:"denomAuthority"`
	}](s.ctx, s.bze, "tokenfactory", "denom-authority", denom)
	require.NoError(t, err)
	return res.DenomAuthority.Admin
}
