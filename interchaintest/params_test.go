package interchaintest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

const (
	paAdminFunds = 1_000_000_000_000 // 1,000,000 BZE: denoms, two pools, two markets, rewards, proposals
	paUserFunds  = 10_000_000_000    // 10,000 BZE each

	paMint     = 1_000_000_000_000 // of each factory denom, to the admin
	paUserMint = 1_000_000         // of tok, stk1 and stk3, to each user

	// tok/ubze: 1 tok = 1000 ubze, the ubze side above the default
	// MinNativeLiquidityForModuleSwap (1e11) so tok pays tx fees
	paTokPoolTok = 200_000_000
	paTokPoolBze = 200_000_000_000
	// pol/ubze, created at the changed create fee (step 4)
	paPolPoolPol = 1_000_000
	paPolPoolBze = 1_000_000_000

	// step 3: the trading fees by gov, paid by one sell and the buy that takes it
	paMakerFee    = 2_500
	paTakerFee    = 50_000
	paOrderAmount = 1_000
	paOrderPrice  = "2"

	paCreateMarketFee = 1_234_000_000 // step 4: 1,234 BZE

	paTokTxFee = 10 // tok, about 10,000 ubze: enough for paSendGas
	paSendGas  = 200_000
	// step 5 raises the min liquidity this far above the tok pool's ubze side
	paLiquidityMargin = 100_000_000_000

	paStakingRewardFee = 1_000_000_000 // step 6: 1,000 BZE
	paDenomRewardFee   = 2_000_000_000 // step 6: 2,000 BZE
	paSrPrize          = 1_000
	paSrDuration       = 100 // days of 30 s: the reward runs to the end of the suite
	paSrStake          = 100

	paDrMinStake = 1_000 // step 7
	paDrDaily    = 1_000
	paMaxPrizes  = 1 // step 8

	// step 9: both extra gas params rise by paExtraGasDelta; 1,000,000 and
	// 1,500,000 are both three-byte varints, so the params read costs the same
	paExtraGas      = 1_000_000
	paExtraGasDelta = 500_000
	paExitGas       = 3_000_000

	paTfFee = 500 // tok, step 10

	paMsgGas = 500_000
)

// TestParams changes the tradebin, rewards and tokenfactory params through
// real gov proposals and proves each change on the next real tx: the gov
// authority, refused and failing proposals, the trading fees and their
// destination, the market and pool create fee, the liquidity a fee denom
// needs, the reward create fees, the denom reward lock and min stake frozen at
// creation, the prize denom cap, the extra gas of exits and the create-denom
// fee. Each module is restored to its genesis params. Subtests run in order
// and share state.
func TestParams(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupParams(t, ctx, net.Chain)

	t.Run("1 authority", s.authority)
	t.Run("2 invalid params fail the proposal", s.invalidParams)
	t.Run("3 tradebin trading fees", s.tradingFees)
	t.Run("4 tradebin create fee", s.createFee)
	t.Run("5 tradebin min native liquidity", s.minNativeLiquidity)
	t.Run("6 rewards create fees", s.rewardFees)
	t.Run("7 rewards lock and min stake", s.lockAndMinStake)
	t.Run("8 rewards prize denom cap", s.prizeCap)
	t.Run("9 rewards extra gas of exits", s.exitGas)
	t.Run("10 tokenfactory create-denom fee", s.createDenomFee)
	t.Run("11 params back at genesis", s.backAtGenesis)
}

// paramsSuite reuses the cointrunk suite's proposal and query helpers and,
// through it, the fees suite's balance, transfer and refusal helpers.
type paramsSuite struct {
	*cointrunkSuite

	maker, taker, staker, user ibc.Wallet
	tok, stk1, stk2, stk3      string
	prz1, prz2, pol, mkt       string
	tokPool, market            string
	rewardsAcc                 string
	srID                       string

	// the params the chain must hold now, and at genesis
	tb, tbGenesis paTradebin
	rw, rwGenesis paRewards
	tf, tfGenesis paTokenfactory
}

func setupParams(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *paramsSuite {
	s := &paramsSuite{cointrunkSuite: &cointrunkSuite{feesSuite: &feesSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}}}}

	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "pa-admin", sdkmath.NewInt(paAdminFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "pa", sdkmath.NewInt(paUserFunds), bze, bze, bze, bze)
	s.maker, s.taker, s.staker, s.user = users[0], users[1], users[2], users[3]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	for name, addr := range map[string]*string{
		"tradebin":              &s.tradebin,
		"txfeecollector_burner": &s.burnerFees,
		"txfeecollector_cp":     &s.cpFees,
		"distribution":          &s.distr,
		"rewards":               &s.rewardsAcc,
		"tokenfactory":          &s.tfAcc,
	} {
		var err error
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	var err error
	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)

	s.tbGenesis, s.rwGenesis, s.tfGenesis = s.queryTradebin(t), s.queryRewards(t), s.queryTokenfactory(t)
	s.tb, s.rw, s.tf = s.tbGenesis, s.rwGenesis, s.tfGenesis
	require.Equal(t, "burner", s.tb.MakerFeeDestination, "genesis maker fee destination")
	require.Equal(t, "burner", s.tb.TakerFeeDestination, "genesis taker fee destination")
	require.EqualValues(t, chain.DenomRewardLock, s.rw.DenomRewardLock, "genesis denom reward lock")
	require.EqualValues(t, paExtraGas, s.rw.ExtraGasForExitStake, "genesis extra gas for exit stake")
	require.EqualValues(t, paExtraGas, s.rw.ExtraGasForDenomExit, "genesis extra gas for denom exit")
	require.Equal(t, chain.Denom, s.tf.CreateDenomFee.Denom, "genesis create-denom fee")

	factory := "factory/" + s.admin.FormattedAddress() + "/"
	s.tok, s.stk1, s.stk2, s.stk3 = factory+"tok", factory+"stk1", factory+"stk2", factory+"stk3"
	s.prz1, s.prz2, s.pol, s.mkt = factory+"prz1", factory+"prz2", factory+"pol", factory+"mkt"
	for _, d := range []string{s.tok, s.stk1, s.stk2, s.stk3, s.prz1, s.prz2, s.pol, s.mkt} {
		sub := d[len(factory):]
		s.mustExec(t, s.admin, "tokenfactory", "create-denom", sub)
		s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(paMint, d))
	}
	for _, u := range []ibc.Wallet{s.maker, s.staker, s.user} {
		send := sdk.NewCoins(sdk.NewInt64Coin(s.tok, paUserMint), sdk.NewInt64Coin(s.stk1, paUserMint), sdk.NewInt64Coin(s.stk3, paUserMint))
		s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), u.FormattedAddress(), send.String())
	}
	// every user signs once, so the step 9 gas measurements see a stored pubkey
	for _, u := range []ibc.Wallet{s.maker, s.taker, s.staker, s.user} {
		s.mustExec(t, u, "bank", "send", u.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
	}

	// "factory/…" sorts before "ubze": tok is the base of its pool and market
	s.tokPool, s.market = s.tok+"_"+chain.Denom, s.tok+"/"+chain.Denom
	s.mustExec(t, s.admin, "tradebin", "create-liquidity-pool", s.tok, chain.Denom, ammPoolFee, ammFeeDest, "false",
		strconv.Itoa(paTokPoolTok), strconv.Itoa(paTokPoolBze))
	s.mustExec(t, s.admin, "tradebin", "create-market", s.tok, chain.Denom)
	return s
}

// 1. MsgUpdateParams of each module signed by a user is refused: only the
// gov authority may change params.
func (s *paramsSuite) authority(t *testing.T) {
	for _, c := range []struct {
		params    paParams
		codespace string
		code      uint32
	}{
		{s.tb, "tradebin", 1100},
		{s.rw, "rewards", 5011},
		{s.tf, "tokenfactory", 1100},
	} {
		t.Run(c.params.module(), func(t *testing.T) {
			before := s.addrBalances(t, s.user.FormattedAddress(), 0)
			res, err := helpers.ExecMsgs(s.ctx, s.bze, s.user, []helpers.TxOption{helpers.WithGas(paMsgGas)}, paramsMsg(c.params, s.user.FormattedAddress()))
			s.requireRefused(t, res, err, c.codespace, c.code, "invalid authority")
			require.Equal(t, before.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.addrBalances(t, s.user.FormattedAddress(), 0), "the user pays only the tx fee")
		})
	}
	s.requireParams(t)
}

// 2. Invalid params never apply. ValidateBasic refuses most of them at
// submission; a native denom without supply passes it and fails the proposal
// when it executes.
func (s *paramsSuite) invalidParams(t *testing.T) {
	negative := func(denom string) sdk.Coin { return sdk.Coin{Denom: denom, Amount: sdkmath.NewInt(-5)} }
	tb := func(edit func(p *paTradebin)) paParams { p := s.tb; edit(&p); return p }
	rw := func(edit func(p *paRewards)) paParams { p := s.rw; edit(&p); return p }
	tf := func(edit func(p *paTokenfactory)) paParams { p := s.tf; edit(&p); return p }

	for _, c := range []struct {
		name   string
		params paParams
		reason string
	}{
		{"tradebin negative maker fee", tb(func(p *paTradebin) { p.MarketMakerFee = negative(chain.Denom) }), "invalid MarketMakerFee"},
		{"tradebin negative taker fee", tb(func(p *paTradebin) { p.MarketTakerFee = negative(chain.Denom) }), "invalid MarketTakerFee"},
		{"tradebin negative create fee", tb(func(p *paTradebin) { p.CreateMarketFee = negative(chain.Denom) }), "invalid CreateMarketFee"},
		{"tradebin maker fee to the validators", tb(func(p *paTradebin) { p.MakerFeeDestination = "validators" }), "invalid MakerFeeDestination: invalid fee destination: validators"},
		{"tradebin taker fee to the validators", tb(func(p *paTradebin) { p.TakerFeeDestination = "validators" }), "invalid TakerFeeDestination: invalid fee destination: validators"},
		{"tradebin empty native denom", tb(func(p *paTradebin) { p.NativeDenom = "" }), "native denom cannot be an empty string"},
		{"tradebin zero min native liquidity", tb(func(p *paTradebin) { p.MinNativeLiquidityForModuleSwap = sdkmath.ZeroInt() }), "min native liquidity for module swap must be positive"},
		{"rewards negative staking reward fee", rw(func(p *paRewards) { p.CreateStakingRewardFee = negative(chain.Denom) }), "invalid CreateStakingRewardFee"},
		{"rewards negative trading reward fee", rw(func(p *paRewards) { p.CreateTradingRewardFee = negative(chain.Denom) }), "invalid createTradingRewardFee"},
		{"rewards invalid denom reward fee denom", rw(func(p *paRewards) { p.CreateDenomRewardFee = sdk.Coin{Denom: "1bad", Amount: sdkmath.NewInt(1)} }), "invalid fee coin"},
		{"tokenfactory negative create-denom fee", tf(func(p *paTokenfactory) { p.CreateDenomFee = negative(chain.Denom) }), "invalid CreateDenomFee"},
		{"tokenfactory invalid create-denom fee denom", tf(func(p *paTokenfactory) { p.CreateDenomFee = sdk.Coin{Denom: "1bad", Amount: sdkmath.NewInt(1)} }), "invalid CreateDenomFee"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := s.submitProposal("invalid params", paramsMsg(c.params, s.gov))
			require.ErrorContains(t, err, c.reason)
			s.requireParams(t)
		})
	}

	t.Run("tradebin native denom without supply", func(t *testing.T) {
		_, err := s.submitProposal("native denom without supply", paramsMsg(tb(func(p *paTradebin) { p.NativeDenom = "unobody" }), s.gov))
		require.ErrorContains(t, err, "PROPOSAL_STATUS_FAILED")
		require.ErrorContains(t, err, "invalid native denom provided: unobody")
		s.requireParams(t)
	})
}

// 3. The trading fees and their destination follow the params: moved to the
// community pool, a sell pays the new maker fee and the buy that takes it the
// new taker fee, both through the community-pool fee collector (the genesis
// destination is the burner). With zero fees nothing is captured.
func (s *paramsSuite) tradingFees(t *testing.T) {
	p := s.tb
	p.MarketMakerFee, p.MarketTakerFee = sdk.NewInt64Coin(chain.Denom, paMakerFee), sdk.NewInt64Coin(chain.Denom, paTakerFee)
	p.MakerFeeDestination, p.TakerFeeDestination = "community_pool", "community_pool"
	s.propose(t, "trading fees to the community pool", p)
	t.Run("to the community pool", func(t *testing.T) { s.requireTrade(t) })

	p.MarketMakerFee, p.MarketTakerFee = sdk.NewInt64Coin(chain.Denom, 0), sdk.NewInt64Coin(chain.Denom, 0)
	s.propose(t, "zero trading fees", p)
	t.Run("zero fees", func(t *testing.T) { s.requireTrade(t) })
}

// 4. A changed CreateMarketFee is charged by the next market and the next
// pool, and goes to the community-pool fee collector.
func (s *paramsSuite) createFee(t *testing.T) {
	p := s.tb
	p.CreateMarketFee = sdk.NewInt64Coin(chain.Denom, paCreateMarketFee)
	s.propose(t, "create market fee", p)
	fee := sdk.NewCoins(p.CreateMarketFee)
	admin := s.admin.FormattedAddress()

	t.Run("market", func(t *testing.T) {
		before := s.addrBalances(t, admin, 0)
		res := s.mustExec(t, s.admin, "tradebin", "create-market", s.mkt, chain.Denom)
		require.Equal(t, fee.String(), s.transfers(t, res.Events, admin, s.tradebin).String(), "fee captured")
		require.Equal(t, fee.String(), s.transfers(t, res.Events, s.tradebin, s.cpFees).String(), "fee to the community-pool fee collector")
		require.Equal(t, before.Sub(fee.Add(sdk.NewCoin(chain.Denom, s.txFee(t, res)))...), s.addrBalances(t, admin, 0), "admin pays the fee and the tx fee")
	})
	t.Run("pool", func(t *testing.T) {
		before := s.addrBalances(t, admin, 0)
		res := s.mustExec(t, s.admin, "tradebin", "create-liquidity-pool", s.pol, chain.Denom, ammPoolFee, ammFeeDest, "false",
			strconv.Itoa(paPolPoolPol), strconv.Itoa(paPolPoolBze))
		reserves := sdk.NewCoins(sdk.NewInt64Coin(s.pol, paPolPoolPol), sdk.NewInt64Coin(chain.Denom, paPolPoolBze))
		require.Equal(t, fee.Add(reserves...).String(), s.transfers(t, res.Events, admin, s.tradebin).String(), "fee and reserves captured")
		require.Equal(t, fee.String(), s.transfers(t, res.Events, s.tradebin, s.cpFees).String(), "fee to the community-pool fee collector")
		require.Equal(t, before.Sub(fee.Add(reserves...).Add(sdk.NewCoin(chain.Denom, s.txFee(t, res)))...), s.addrBalances(t, admin, 0),
			"admin pays the fee, the reserves and the tx fee")
	})
}

// 5. A denom pays tx fees only while its ubze pool side reaches
// MinNativeLiquidityForModuleSwap: raised above the tok pool's ubze side, a
// tx fee in tok is refused; back at genesis it is accepted again.
func (s *paramsSuite) minNativeLiquidity(t *testing.T) {
	s.requireTokFee(t, true)

	p := s.tb
	p.MinNativeLiquidityForModuleSwap = s.pool(t, s.tokPool).reserveQuote().AddRaw(paLiquidityMargin)
	s.propose(t, "min native liquidity above the tok pool", p)
	s.requireTokFee(t, false)

	s.propose(t, "tradebin params back at genesis", s.tbGenesis)
	s.requireTokFee(t, true)
}

// 6. The staking reward and denom reward create fees follow the params:
// changed, each create pays the new fee to the community-pool fee collector;
// at zero, creating is free.
func (s *paramsSuite) rewardFees(t *testing.T) {
	p := s.rw
	p.CreateStakingRewardFee = sdk.NewInt64Coin(chain.Denom, paStakingRewardFee)
	p.CreateDenomRewardFee = sdk.NewInt64Coin(chain.Denom, paDenomRewardFee)
	s.propose(t, "reward create fees", p)
	t.Run("changed", func(t *testing.T) {
		s.srID = s.createStakingReward(t, s.stk1, p.CreateStakingRewardFee)
		s.createDenomReward(t, s.stk1, p.CreateDenomRewardFee)
	})

	p.CreateStakingRewardFee, p.CreateDenomRewardFee = sdk.NewInt64Coin(chain.Denom, 0), sdk.NewInt64Coin(chain.Denom, 0)
	s.propose(t, "free reward creation", p)
	t.Run("zero", func(t *testing.T) {
		s.createStakingReward(t, s.stk2, p.CreateStakingRewardFee)
		s.createDenomReward(t, s.stk2, p.CreateDenomRewardFee)
	})
}

// 7. A denom reward freezes denomRewardLock and denomRewardMinStake at
// creation: after a change the existing denom rewards keep theirs and a new
// one takes the new values. The new one has lock 0, so its exit returns the
// stake at once, and refuses a join below its min stake; the old one accepts
// any stake and queues the exit for its lock.
func (s *paramsSuite) lockAndMinStake(t *testing.T) {
	p := s.rw
	p.DenomRewardLock, p.DenomRewardMinStake = 0, paDrMinStake
	s.propose(t, "denom reward lock 0 and min stake", p)

	for _, d := range []string{s.stk1, s.stk2} {
		dr := s.denomReward(t, d)
		require.EqualValues(t, s.rwGenesis.DenomRewardLock, dr.Lock, "lock of the existing denom reward of %s", d)
		require.EqualValues(t, s.rwGenesis.DenomRewardMinStake, dr.MinStake, "min stake of the existing denom reward of %s", d)
	}
	s.createDenomReward(t, s.stk3, p.CreateDenomRewardFee)
	dr := s.denomReward(t, s.stk3)
	require.Zero(t, int64(dr.Lock), "lock of the new denom reward")
	require.EqualValues(t, paDrMinStake, dr.MinStake, "min stake of the new denom reward")

	staker := s.staker.FormattedAddress()
	t.Run("new: min stake", func(t *testing.T) {
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.staker, []helpers.TxOption{helpers.WithGas(paMsgGas)},
			"rewards", "join-denom-reward", s.stk3, strconv.Itoa(paDrMinStake-1))
		reason, err := helpers.RejectionReason(res, err)
		require.NoError(t, err)
		require.Contains(t, reason, "amount is smaller than denom reward min stake")
		s.mustExec(t, s.staker, "rewards", "join-denom-reward", s.stk3, strconv.Itoa(paDrMinStake))
	})
	t.Run("new: lock 0 returns the stake at once", func(t *testing.T) {
		res := s.mustExec(t, s.staker, "rewards", "exit-denom-reward", s.stk3)
		require.Equal(t, coins(paDrMinStake, s.stk3).String(), s.transfers(t, res.Events, s.rewardsAcc, staker).String(), "stake returned in the exit tx")
		require.Empty(t, s.pendingUnlocks(t, staker), "nothing queued")
	})
	t.Run("old: any stake, the exit waits for the lock", func(t *testing.T) {
		s.mustExec(t, s.staker, "rewards", "join-denom-reward", s.stk1, "1")
		res := s.mustExec(t, s.staker, "rewards", "exit-denom-reward", s.stk1)
		require.Empty(t, s.transfers(t, res.Events, s.rewardsAcc, staker), "nothing returned in the exit tx")
		at := s.hourEpoch(t, res.Height) + int64(s.rwGenesis.DenomRewardLock)*24
		require.Equal(t, []pendingUnlockInfo{{
			Index:   fmt.Sprintf("%d/dr/%s/%s", at, s.stk1, staker),
			Address: staker,
			Amount:  "1",
			Denom:   s.stk1,
		}}, s.pendingUnlocks(t, staker), "the stake unlocks %d hour epochs later", s.rwGenesis.DenomRewardLock*24)
	})
}

// 8. maxPrizeDenomsPerDr caps the prize denoms of a denom reward: lowered to
// one, the first prize denom is accepted, a second refused, and a further
// schedule in the first prize denom still accepted (it takes no new slot and
// pays no prize fee).
func (s *paramsSuite) prizeCap(t *testing.T) {
	p := s.rw
	p.MaxPrizeDenomsPerDr = paMaxPrizes
	s.propose(t, "prize denom cap", p)

	admin := s.admin.FormattedAddress()
	schedule := func(t *testing.T, prize string, fees sdk.Coins) {
		t.Helper()
		before := s.addrBalances(t, admin, 0)
		res := s.mustExec(t, s.admin, "rewards", "create-denom-reward-schedule", s.stk3, prize, strconv.Itoa(paDrDaily), "1")
		require.Equal(t, fees.String(), s.transfers(t, res.Events, s.rewardsAcc, s.cpFees).String(), "fees to the community-pool fee collector")
		require.Equal(t, before.Sub(fees.Add(sdk.NewInt64Coin(prize, paDrDaily), sdk.NewCoin(chain.Denom, s.txFee(t, res)))...), s.addrBalances(t, admin, 0),
			"admin pays the fees, the schedule and the tx fee")
	}
	t.Run("first prize denom", func(t *testing.T) {
		schedule(t, s.prz1, sdk.NewCoins(p.CreateDenomRewardPrizeFee).Add(p.AddDenomRewardScheduleFee))
	})
	t.Run("second prize denom refused", func(t *testing.T) {
		before := s.addrBalances(t, admin, 0)
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.admin, []helpers.TxOption{helpers.WithGas(paMsgGas)},
			"rewards", "create-denom-reward-schedule", s.stk3, s.prz2, strconv.Itoa(paDrDaily), "1")
		s.requireRefused(t, res, err, "rewards", 5017, "prize denom cap reached for this denom reward")
		require.Equal(t, before.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.addrBalances(t, admin, 0), "admin pays only the tx fee")
	})
	t.Run("first prize denom again", func(t *testing.T) {
		schedule(t, s.prz1, sdk.NewCoins(p.AddDenomRewardScheduleFee))
	})
}

// 9. The exits charge their extra gas param before anything else, so an exit
// refused for an unknown reward uses exactly the param plus a fixed cost:
// raised by gov, the same refused exit uses exactly the raise more. A real
// exit after the raise uses more gas than the new param.
func (s *paramsSuite) exitGas(t *testing.T) {
	exitStaking := map[string]any{"@type": "/bze.rewards.MsgExitStaking", "creator": s.user.FormattedAddress(), "reward_id": "999999"}
	exitDenom := map[string]any{"@type": "/bze.rewards.MsgExitDenomReward", "creator": s.user.FormattedAddress(), "denom": s.prz2}
	refusedGas := func(t *testing.T, msg any, code uint32, reason string) int64 {
		t.Helper()
		res, err := helpers.ExecMsgs(s.ctx, s.bze, s.user, []helpers.TxOption{helpers.WithGas(paExitGas)}, msg)
		s.requireRefused(t, res, err, "rewards", code, reason)
		return res.GasUsed
	}
	stakingBefore := refusedGas(t, exitStaking, 5009, "reward with provided id not found")
	denomBefore := refusedGas(t, exitDenom, 5015, "denom reward not found")
	s.mustExec(t, s.staker, "rewards", "join-staking", s.srID, strconv.Itoa(paSrStake))

	p := s.rw
	p.ExtraGasForExitStake += paExtraGasDelta
	p.ExtraGasForDenomExit += paExtraGasDelta
	s.propose(t, "extra gas of exits", p)

	require.Equal(t, stakingBefore+paExtraGasDelta, refusedGas(t, exitStaking, 5009, "reward with provided id not found"), "refused staking exit")
	require.Equal(t, denomBefore+paExtraGasDelta, refusedGas(t, exitDenom, 5015, "denom reward not found"), "refused denom exit")
	res := s.mustExecWith(t, s.staker, []helpers.TxOption{helpers.WithGas(paExitGas)}, "rewards", "exit-staking", s.srID)
	require.Greater(t, res.GasUsed, int64(p.ExtraGasForExitStake), "a real exit pays the new extra gas")
}

// 10. A changed create_denom_fee is charged by the next create-denom, also in
// a factory denom (forwarded as-is to the community-pool fee collector); at
// zero, creating a denom is free. The rewards params go back to genesis in
// the same proposal.
func (s *paramsSuite) createDenomFee(t *testing.T) {
	p := s.tf
	p.CreateDenomFee = sdk.NewInt64Coin(s.tok, paTfFee)
	s.propose(t, "create-denom fee in tok, rewards params back at genesis", p, s.rwGenesis)
	t.Run("in a factory denom", func(t *testing.T) { s.requireCreateDenom(t, "pa1", sdk.NewCoins(p.CreateDenomFee)) })

	p.CreateDenomFee = sdk.NewInt64Coin(chain.Denom, 0)
	s.propose(t, "free create-denom", p)
	t.Run("zero", func(t *testing.T) { s.requireCreateDenom(t, "pa2", sdk.NewCoins()) })
}

// 11. After the last restore every module holds its genesis params again.
func (s *paramsSuite) backAtGenesis(t *testing.T) {
	s.propose(t, "tokenfactory params back at genesis", s.tfGenesis)
	require.Equal(t, s.tbGenesis.msgParams(), s.queryTradebin(t).msgParams(), "tradebin")
	require.Equal(t, s.rwGenesis.msgParams(), s.queryRewards(t).msgParams(), "rewards")
	require.Equal(t, s.tfGenesis.msgParams(), s.queryTokenfactory(t).msgParams(), "tokenfactory")
}

// requireTrade rests a sell from the maker, then takes it with a buy from the
// taker, and asserts each tx moved the order funds and exactly the current
// trading fee, and the fee went on to its destination.
func (s *paramsSuite) requireTrade(t *testing.T) {
	t.Helper()
	s.requireOrder(t, s.maker, "sell", sdk.NewInt64Coin(s.tok, paOrderAmount), s.tb.MarketMakerFee, s.tb.MakerFeeDestination, sdk.NewCoins())
	// the buy fills in the EndBlock of its own block: the taker already holds the tok
	s.requireOrder(t, s.taker, "buy", sdk.NewCoin(chain.Denom, mulPrice(paOrderAmount, paOrderPrice)), s.tb.MarketTakerFee, s.tb.TakerFeeDestination,
		coins(paOrderAmount, s.tok))
	s.requireBookEmpty(t)
}

func (s *paramsSuite) requireOrder(t *testing.T, u ibc.Wallet, typ string, locked, fee sdk.Coin, dest string, filled sdk.Coins) {
	t.Helper()
	addr := u.FormattedAddress()
	fees := sdk.NewCoins(fee)
	before := s.addrBalances(t, addr, 0)
	res := s.mustExec(t, u, "tradebin", "create-order", typ, strconv.Itoa(paOrderAmount), paOrderPrice, s.market)

	require.Equal(t, fees.Add(locked).String(), s.transfers(t, res.Events, addr, s.tradebin).String(), "%s: order funds and fee to the tradebin module", typ)
	toCp, toBurner := s.transfers(t, res.Events, s.tradebin, s.cpFees), s.transfers(t, res.Events, s.tradebin, s.burnerFees)
	require.Equal(t, "community_pool", dest, "the suite only moves the fees to the community pool")
	require.Equal(t, fees.String(), toCp.String(), "%s: fee to the community-pool fee collector", typ)
	require.Empty(t, toBurner, "%s: nothing to the burner fee collector", typ)
	require.Equal(t, fees.String(), s.transfers(t, s.blockEvents(t, res.Height), s.cpFees, s.distr).String(), "%s: fee to the community pool in EndBlock", typ)
	require.Equal(t, before.Add(filled...).Sub(fees.Add(locked, sdk.NewCoin(chain.Denom, s.txFee(t, res)))...), s.addrBalances(t, addr, 0), "%s: balances", typ)
}

// requireBookEmpty asserts the buy took the whole sell: nothing rests on the
// market once the taker's block executed.
func (s *paramsSuite) requireBookEmpty(t *testing.T) {
	t.Helper()
	for _, typ := range []string{"buy", "sell"} {
		res, err := helpers.Query[struct {
			List []map[string]any `json:"list"`
		}](s.ctx, s.bze, "tradebin", "market-aggregated-orders", s.market, typ)
		require.NoError(t, err)
		require.Empty(t, res.List, "no %s order rests", typ)
	}
}

// requireTokFee sends a bank tx paying its fee in tok and asserts it is
// accepted (and charged in tok) or refused by the fee-denom check.
func (s *paramsSuite) requireTokFee(t *testing.T, accepted bool) {
	t.Helper()
	before := s.addrBalances(t, s.user.FormattedAddress(), 0)
	res, err := helpers.ExecTxWith(s.ctx, s.bze, s.user, []helpers.TxOption{helpers.WithFees(coinStr(paTokTxFee, s.tok)), helpers.WithGas(paSendGas)},
		"bank", "send", s.user.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
	if !accepted {
		s.requireRefused(t, res, err, "store", 7, "can be used to pay for fees only if enough liquidity is available")
		require.Equal(t, before, s.addrBalances(t, s.user.FormattedAddress(), 0), "nothing charged")
		return
	}
	require.NoError(t, err)
	require.Zero(t, res.Code, res.RawLog)
	require.Equal(t, before.Sub(sdk.NewCoins(sdk.NewInt64Coin(s.tok, paTokTxFee), sdk.NewInt64Coin(chain.Denom, 1))...), s.addrBalances(t, s.user.FormattedAddress(), 0), "fee paid in tok")
}

// createStakingReward creates a staking reward of staking and asserts the
// creator paid fee to the community-pool fee collector; it returns its id.
func (s *paramsSuite) createStakingReward(t *testing.T, staking string, fee sdk.Coin) string {
	t.Helper()
	res := s.requireRewardFee(t, sdk.NewCoins(sdk.NewInt64Coin(s.prz1, paSrPrize*paSrDuration)), fee,
		"rewards", "create-staking-reward", strconv.Itoa(paSrPrize), s.prz1, staking, strconv.Itoa(paSrDuration), "1", "1")
	events := helpers.EventsWith(helpers.TxEvents(res, drEventPrefix+"StakingRewardCreateEvent"), map[string]string{"staking_denom": staking})
	require.Len(t, events, 1, "staking reward created")
	id := events[0].Attributes["reward_id"]
	require.NotEmpty(t, id)
	return id
}

func (s *paramsSuite) createDenomReward(t *testing.T, staking string, fee sdk.Coin) {
	t.Helper()
	res := s.requireRewardFee(t, sdk.NewCoins(), fee, "rewards", "create-denom-reward", staking)
	require.Len(t, helpers.EventsWith(helpers.TxEvents(res, drEventPrefix+"DenomRewardCreateEvent"), map[string]string{"denom": staking}), 1, "denom reward created")
}

// requireRewardFee runs a reward create tx and asserts the admin paid escrow
// and the fee (none when zero) and the fee reached the community-pool fee
// collector through the rewards module.
func (s *paramsSuite) requireRewardFee(t *testing.T, escrow sdk.Coins, fee sdk.Coin, args ...string) helpers.TxResult {
	t.Helper()
	admin := s.admin.FormattedAddress()
	fees := sdk.NewCoins(fee)
	before := s.addrBalances(t, admin, 0)
	res := s.mustExec(t, s.admin, args...)
	require.Equal(t, escrow.Add(fees...).String(), s.transfers(t, res.Events, admin, s.rewardsAcc).String(), "escrow and fee to the rewards module")
	require.Equal(t, fees.String(), s.transfers(t, res.Events, s.rewardsAcc, s.cpFees).String(), "fee to the community-pool fee collector")
	require.Equal(t, before.Sub(escrow.Add(fees...).Add(sdk.NewCoin(chain.Denom, s.txFee(t, res)))...), s.addrBalances(t, admin, 0), "admin balances")
	return res
}

// requireCreateDenom creates factory/admin/sub and asserts the admin paid
// fees (none when empty) to the community-pool fee collector, through the
// tokenfactory module.
func (s *paramsSuite) requireCreateDenom(t *testing.T, sub string, fees sdk.Coins) {
	t.Helper()
	admin := s.admin.FormattedAddress()
	before := s.addrBalances(t, admin, 0)
	res := s.mustExec(t, s.admin, "tokenfactory", "create-denom", sub)
	require.Equal(t, fees.String(), s.transfers(t, res.Events, admin, s.tfAcc).String(), "fee to the tokenfactory module")
	require.Equal(t, fees.String(), s.transfers(t, res.Events, s.tfAcc, s.cpFees).String(), "fee to the community-pool fee collector")
	require.Equal(t, before.Sub(fees.Add(sdk.NewCoin(chain.Denom, s.txFee(t, res)))...), s.addrBalances(t, admin, 0), "admin balances")
	require.True(t, s.supply(t, "factory/"+admin+"/"+sub).IsZero(), "a new denom has no supply")
}

func (s *paramsSuite) denomReward(t *testing.T, denom string) denomRewardInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		DenomReward denomRewardInfo `json:"denom_reward"`
	}](s.ctx, s.bze, "rewards", "denom-reward", denom)
	require.NoError(t, err)
	require.Equal(t, denom, res.DenomReward.StakingDenom)
	return res.DenomReward
}

// pendingUnlocks returns the pending unlocks of addr, by index.
func (s *paramsSuite) pendingUnlocks(t *testing.T, addr string) []pendingUnlockInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		List []pendingUnlockInfo `json:"list"`
	}](s.ctx, s.bze, "rewards", "all-pending-unlock-participants")
	require.NoError(t, err)
	var out []pendingUnlockInfo
	for _, e := range res.List {
		if e.Address == addr {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}

func (s *paramsSuite) hourEpoch(t *testing.T, height int64) int64 {
	t.Helper()
	res, err := helpers.Query[struct {
		CurrentEpoch helpers.Int64 `json:"current_epoch"`
	}](s.ctx, s.bze, "epochs", "identifier", "hour", "--height", strconv.FormatInt(height, 10))
	require.NoError(t, err)
	return int64(res.CurrentEpoch)
}

// propose passes one MsgUpdateParams per params as one proposal, records them
// as the chain's params and asserts every module's params.
func (s *paramsSuite) propose(t *testing.T, title string, params ...paParams) {
	t.Helper()
	msgs := make([]any, 0, len(params))
	for _, p := range params {
		msgs = append(msgs, paramsMsg(p, s.gov))
	}
	s.passProposal(t, title, msgs...)
	for _, p := range params {
		switch v := p.(type) {
		case paTradebin:
			s.tb = v
		case paRewards:
			s.rw = v
		case paTokenfactory:
			s.tf = v
		}
	}
	s.requireParams(t)
}

// requireParams asserts the three modules hold exactly the params the suite
// expects.
func (s *paramsSuite) requireParams(t *testing.T) {
	t.Helper()
	require.Equal(t, s.tb.msgParams(), s.queryTradebin(t).msgParams(), "tradebin params")
	require.Equal(t, s.rw.msgParams(), s.queryRewards(t).msgParams(), "rewards params")
	require.Equal(t, s.tf.msgParams(), s.queryTokenfactory(t).msgParams(), "tokenfactory params")
}

// paParams are one module's full params: a MsgUpdateParams replaces them
// whole, so each change carries every field.
type paParams interface {
	module() string
	// msgParams are the params by proto field name, every value a string or a
	// coin, as MsgUpdateParams JSON carries them; equal params give equal maps.
	msgParams() map[string]any
}

func paramsMsg(p paParams, authority string) map[string]any {
	return map[string]any{
		"@type":     "/bze." + p.module() + ".MsgUpdateParams",
		"authority": authority,
		"params":    p.msgParams(),
	}
}

func coinJSON(c sdk.Coin) map[string]string {
	return map[string]string{"denom": c.Denom, "amount": c.Amount.String()}
}

func u64(v uint64) string { return strconv.FormatUint(v, 10) }

type paTradebin struct {
	CreateMarketFee, MarketMakerFee, MarketTakerFee sdk.Coin
	MakerFeeDestination, TakerFeeDestination        string
	NativeDenom                                     string
	OrderBookExtraGasWindow, OrderBookQueueExtraGas uint64
	FillOrdersExtraGas                              uint64
	MinNativeLiquidityForModuleSwap                 sdkmath.Int
	OrderBookPerBlockMessages                       uint64
	OrderBookQueueMessageScanExtraGas               uint64
}

func (paTradebin) module() string { return "tradebin" }

func (p paTradebin) msgParams() map[string]any {
	return map[string]any{
		"createMarketFee":                   coinJSON(p.CreateMarketFee),
		"marketMakerFee":                    coinJSON(p.MarketMakerFee),
		"marketTakerFee":                    coinJSON(p.MarketTakerFee),
		"makerFeeDestination":               p.MakerFeeDestination,
		"takerFeeDestination":               p.TakerFeeDestination,
		"native_denom":                      p.NativeDenom,
		"orderBookExtraGasWindow":           u64(p.OrderBookExtraGasWindow),
		"orderBookQueueExtraGas":            u64(p.OrderBookQueueExtraGas),
		"fillOrdersExtraGas":                u64(p.FillOrdersExtraGas),
		"minNativeLiquidityForModuleSwap":   p.MinNativeLiquidityForModuleSwap.String(),
		"orderBookPerBlockMessages":         u64(p.OrderBookPerBlockMessages),
		"orderBookQueueMessageScanExtraGas": u64(p.OrderBookQueueMessageScanExtraGas),
	}
}

type paRewards struct {
	CreateStakingRewardFee, CreateTradingRewardFee sdk.Coin
	ExtraGasForExitStake                           uint64
	CreateDenomRewardFee                           sdk.Coin
	CreateDenomRewardPrizeFee                      sdk.Coin
	AddDenomRewardScheduleFee                      sdk.Coin
	MaxPrizeDenomsPerDr                            uint64
	ExtraGasForDenomExit                           uint64
	DenomRewardLock                                uint64
	DenomRewardMinStake                            uint64
}

func (paRewards) module() string { return "rewards" }

func (p paRewards) msgParams() map[string]any {
	return map[string]any{
		"createStakingRewardFee":    coinJSON(p.CreateStakingRewardFee),
		"createTradingRewardFee":    coinJSON(p.CreateTradingRewardFee),
		"extraGasForExitStake":      u64(p.ExtraGasForExitStake),
		"createDenomRewardFee":      coinJSON(p.CreateDenomRewardFee),
		"createDenomRewardPrizeFee": coinJSON(p.CreateDenomRewardPrizeFee),
		"addDenomRewardScheduleFee": coinJSON(p.AddDenomRewardScheduleFee),
		"maxPrizeDenomsPerDr":       u64(p.MaxPrizeDenomsPerDr),
		"extraGasForDenomExit":      u64(p.ExtraGasForDenomExit),
		"denomRewardLock":           u64(p.DenomRewardLock),
		"denomRewardMinStake":       u64(p.DenomRewardMinStake),
	}
}

type paTokenfactory struct {
	CreateDenomFee sdk.Coin
}

func (paTokenfactory) module() string { return "tokenfactory" }

func (p paTokenfactory) msgParams() map[string]any {
	return map[string]any{"create_denom_fee": coinJSON(p.CreateDenomFee)}
}

// paCoin is a coin as the query prints it; an omitted amount is zero.
type paCoin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

func (c paCoin) coin() sdk.Coin { return sdk.Coin{Denom: c.Denom, Amount: parseInt(c.Amount)} }

// moduleParams queries a module's params with normalized keys (the CLI prints
// proto names or JSON names depending on the encoder) and fails on a param
// the suite does not know: MsgUpdateParams must carry all of them.
func (s *paramsSuite) moduleParams(t *testing.T, module string, known ...string) []byte {
	t.Helper()
	var res struct {
		Params map[string]json.RawMessage `json:"params"`
	}
	raw := s.query(t, module, "params")
	require.NoError(t, json.Unmarshal(raw, &res), "%s", raw)
	for k := range res.Params {
		require.Contains(t, known, k, "unknown %s param %s: the suite must carry it in MsgUpdateParams", module, k)
	}
	out, err := json.Marshal(res.Params)
	require.NoError(t, err)
	return out
}

func (s *paramsSuite) queryTradebin(t *testing.T) paTradebin {
	t.Helper()
	var p struct {
		CreateMarketFee                   paCoin        `json:"createmarketfee"`
		MarketMakerFee                    paCoin        `json:"marketmakerfee"`
		MarketTakerFee                    paCoin        `json:"markettakerfee"`
		MakerFeeDestination               string        `json:"makerfeedestination"`
		TakerFeeDestination               string        `json:"takerfeedestination"`
		NativeDenom                       string        `json:"nativedenom"`
		OrderBookExtraGasWindow           helpers.Int64 `json:"orderbookextragaswindow"`
		OrderBookQueueExtraGas            helpers.Int64 `json:"orderbookqueueextragas"`
		FillOrdersExtraGas                helpers.Int64 `json:"fillordersextragas"`
		MinNativeLiquidityForModuleSwap   string        `json:"minnativeliquidityformoduleswap"`
		OrderBookPerBlockMessages         helpers.Int64 `json:"orderbookperblockmessages"`
		OrderBookQueueMessageScanExtraGas helpers.Int64 `json:"orderbookqueuemessagescanextragas"`
	}
	raw := s.moduleParams(t, "tradebin", "createmarketfee", "marketmakerfee", "markettakerfee", "makerfeedestination", "takerfeedestination",
		"nativedenom", "orderbookextragaswindow", "orderbookqueueextragas", "fillordersextragas", "minnativeliquidityformoduleswap",
		"orderbookperblockmessages", "orderbookqueuemessagescanextragas")
	require.NoError(t, json.Unmarshal(raw, &p), "%s", raw)
	return paTradebin{
		CreateMarketFee:                   p.CreateMarketFee.coin(),
		MarketMakerFee:                    p.MarketMakerFee.coin(),
		MarketTakerFee:                    p.MarketTakerFee.coin(),
		MakerFeeDestination:               p.MakerFeeDestination,
		TakerFeeDestination:               p.TakerFeeDestination,
		NativeDenom:                       p.NativeDenom,
		OrderBookExtraGasWindow:           uint64(p.OrderBookExtraGasWindow),
		OrderBookQueueExtraGas:            uint64(p.OrderBookQueueExtraGas),
		FillOrdersExtraGas:                uint64(p.FillOrdersExtraGas),
		MinNativeLiquidityForModuleSwap:   parseInt(p.MinNativeLiquidityForModuleSwap),
		OrderBookPerBlockMessages:         uint64(p.OrderBookPerBlockMessages),
		OrderBookQueueMessageScanExtraGas: uint64(p.OrderBookQueueMessageScanExtraGas),
	}
}

func (s *paramsSuite) queryRewards(t *testing.T) paRewards {
	t.Helper()
	var p struct {
		CreateStakingRewardFee    paCoin        `json:"createstakingrewardfee"`
		CreateTradingRewardFee    paCoin        `json:"createtradingrewardfee"`
		ExtraGasForExitStake      helpers.Int64 `json:"extragasforexitstake"`
		CreateDenomRewardFee      paCoin        `json:"createdenomrewardfee"`
		CreateDenomRewardPrizeFee paCoin        `json:"createdenomrewardprizefee"`
		AddDenomRewardScheduleFee paCoin        `json:"adddenomrewardschedulefee"`
		MaxPrizeDenomsPerDr       helpers.Int64 `json:"maxprizedenomsperdr"`
		ExtraGasForDenomExit      helpers.Int64 `json:"extragasfordenomexit"`
		DenomRewardLock           helpers.Int64 `json:"denomrewardlock"`
		DenomRewardMinStake       helpers.Int64 `json:"denomrewardminstake"`
	}
	raw := s.moduleParams(t, "rewards", "createstakingrewardfee", "createtradingrewardfee", "extragasforexitstake", "createdenomrewardfee",
		"createdenomrewardprizefee", "adddenomrewardschedulefee", "maxprizedenomsperdr", "extragasfordenomexit", "denomrewardlock",
		"denomrewardminstake")
	require.NoError(t, json.Unmarshal(raw, &p), "%s", raw)
	return paRewards{
		CreateStakingRewardFee:    p.CreateStakingRewardFee.coin(),
		CreateTradingRewardFee:    p.CreateTradingRewardFee.coin(),
		ExtraGasForExitStake:      uint64(p.ExtraGasForExitStake),
		CreateDenomRewardFee:      p.CreateDenomRewardFee.coin(),
		CreateDenomRewardPrizeFee: p.CreateDenomRewardPrizeFee.coin(),
		AddDenomRewardScheduleFee: p.AddDenomRewardScheduleFee.coin(),
		MaxPrizeDenomsPerDr:       uint64(p.MaxPrizeDenomsPerDr),
		ExtraGasForDenomExit:      uint64(p.ExtraGasForDenomExit),
		DenomRewardLock:           uint64(p.DenomRewardLock),
		DenomRewardMinStake:       uint64(p.DenomRewardMinStake),
	}
}

func (s *paramsSuite) queryTokenfactory(t *testing.T) paTokenfactory {
	t.Helper()
	var p struct {
		CreateDenomFee paCoin `json:"createdenomfee"`
	}
	raw := s.moduleParams(t, "tokenfactory", "createdenomfee")
	require.NoError(t, json.Unmarshal(raw, &p), "%s", raw)
	return paTokenfactory{CreateDenomFee: p.CreateDenomFee.coin()}
}
