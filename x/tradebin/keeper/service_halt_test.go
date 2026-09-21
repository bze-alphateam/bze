package keeper_test

import (
	"cosmossdk.io/math"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectormoduletypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

// denomHalted is the denom the halted-denom tests list in halted_denoms. It sorts after both
// "stake" and "ubze", so pool ids are "stake_uhalt" and "ubze_uhalt".
const denomHalted = "uhalt"

// setHaltedDenoms replaces the halted_denoms list, keeping every other param as stored.
func (suite *IntegrationTestSuite) setHaltedDenoms(denoms ...string) {
	params := suite.k.GetParams(suite.ctx)
	params.HaltedDenoms = denoms
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
}

// haltedBaseMarket is a market whose base is the halted denom (uhalt/ubze).
func haltedBaseMarket() types.Market {
	return types.Market{Base: denomHalted, Quote: denomBze, Creator: getTestAddress()}
}

// haltedQuoteMarket is a market whose quote is the halted denom (stake/uhalt).
func haltedQuoteMarket() types.Market {
	return types.Market{Base: denomStake, Quote: denomHalted, Creator: getTestAddress()}
}

func marketIdOf(m types.Market) string {
	return m.Base + "/" + m.Quote
}

// deepNativePool returns a native/other pool whose native reserve is above the default
// min_native_liquidity_for_module_swap, so every liquidity answer is "yes" unless the denom is halted.
func deepNativePool(other string) types.LiquidityPool {
	base, quote, id := (keeper.Keeper{}).CreatePoolId(denomBze, other)
	pool := types.LiquidityPool{
		Id:      id,
		Base:    base,
		Quote:   quote,
		LpDenom: "ulp_" + id,
		Creator: getTestAddress(),
		Fee:     math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
	native := math.NewInt(200_000_000_000)
	otherReserve := math.NewInt(100_000_000_000)
	if base == denomBze {
		pool.ReserveBase, pool.ReserveQuote = native, otherReserve
	} else {
		pool.ReserveBase, pool.ReserveQuote = otherReserve, native
	}

	return pool
}

func (suite *IntegrationTestSuite) TestServiceHalt_IsDenomHalted() {
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomHalted), "nothing is halted by default")

	suite.setHaltedDenoms(denomHalted)

	suite.Require().True(suite.k.IsDenomHalted(suite.ctx, denomHalted))
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomBze))
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomStake))
	// ids that merely contain the denom never match
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, marketIdOf(haltedQuoteMarket())))
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, "stake_"+denomHalted))

	suite.setHaltedDenoms()
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomHalted), "un-halting clears the answer")
}

func (suite *IntegrationTestSuite) TestServiceHalt_IsMarketHalted() {
	suite.k.SetMarket(suite.ctx, market)
	suite.k.SetMarket(suite.ctx, haltedBaseMarket())
	suite.k.SetMarket(suite.ctx, haltedQuoteMarket())

	suite.Require().False(suite.k.IsMarketHalted(suite.ctx, marketIdOf(haltedBaseMarket())), "not halted before the param is set")

	suite.setHaltedDenoms(denomHalted)

	suite.Require().True(suite.k.IsMarketHalted(suite.ctx, marketIdOf(haltedBaseMarket())), "halted denom as base")
	suite.Require().True(suite.k.IsMarketHalted(suite.ctx, marketIdOf(haltedQuoteMarket())), "halted denom as quote")
	suite.Require().False(suite.k.IsMarketHalted(suite.ctx, getMarketId()), "unrelated market")
	suite.Require().False(suite.k.IsMarketHalted(suite.ctx, "nope/"+denomHalted), "unknown market is not halted, it is not found")
}

// The three liquidity answers the ante handler, the fee collector and the burner rely on say "no"
// for a halted denom even with a deep pool, and keep saying "yes" for everything else.
func (suite *IntegrationTestSuite) TestServiceHalt_LiquidityAnswers_FalseForHaltedDenom() {
	suite.k.SetLiquidityPool(suite.ctx, deepNativePool(denomHalted))
	suite.k.SetLiquidityPool(suite.ctx, deepNativePool(denomStake))
	coin := sdk.NewInt64Coin(denomHalted, 1_000_000)

	// before the halt every answer is yes (proves the pools are deep enough)
	suite.Require().True(suite.k.HasLiquidityWithNativeDenom(suite.ctx, denomHalted))
	suite.Require().True(suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, denomHalted))
	suite.Require().True(suite.k.CanSwapForNativeDenom(suite.ctx, coin))

	suite.setHaltedDenoms(denomHalted)

	suite.Require().False(suite.k.HasLiquidityWithNativeDenom(suite.ctx, denomHalted))
	suite.Require().False(suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, denomHalted))
	suite.Require().False(suite.k.CanSwapForNativeDenom(suite.ctx, coin))

	// the unrelated denom is unaffected
	suite.Require().True(suite.k.HasLiquidityWithNativeDenom(suite.ctx, denomStake))
	suite.Require().True(suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, denomStake))
	suite.Require().True(suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin(denomStake, 1_000_000)))
}

// swapTokens is the single choke point every swap goes through; it refuses a halted pool and leaves
// the reserves untouched, and swaps normally once the denom is un-halted.
func (suite *IntegrationTestSuite) TestServiceHalt_SwapTokens_RefusesHaltedPool() {
	pool := deepNativePool(denomHalted)
	suite.k.SetLiquidityPool(suite.ctx, pool)
	suite.setHaltedDenoms(denomHalted)

	// no bank expectation: a swap that reached collectSwapFee would call the bank and panic the mock
	for _, input := range []sdk.Coin{sdk.NewInt64Coin(denomHalted, 1_000_000), sdk.NewInt64Coin(denomBze, 1_000_000)} {
		p := pool
		out, err := suite.k.SwapTokens(suite.ctx, input, &p)
		suite.Require().Error(err, "input %s", input)
		suite.Require().ErrorIs(err, types.ErrDenomHalted)
		suite.Require().True(out.IsNil() || out.IsZero())

		stored, found := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
		suite.Require().True(found)
		suite.Require().Equal(pool.ReserveBase, stored.ReserveBase)
		suite.Require().Equal(pool.ReserveQuote, stored.ReserveQuote)
	}

	// un-halted: the same swap goes through (regression for "nothing stays frozen")
	suite.setHaltedDenoms()
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, txfeecollectormoduletypes.CpFeeCollector, gomock.Any()).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnermoduletypes.ModuleName, gomock.Any()).Return(nil).Times(1)

	p := pool
	out, err := suite.k.SwapTokens(suite.ctx, sdk.NewInt64Coin(denomHalted, 1_000_000), &p)
	suite.Require().NoError(err)
	suite.Require().Equal(denomBze, out.Denom)
	suite.Require().True(out.IsPositive())

	stored, _ := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
	suite.Require().NotEqual(pool.ReserveBase, stored.ReserveBase)
}

// The txfeecollector EndBlock conversion path: a module swap of a halted denom fails and no coin
// leaves the cache context (nothing is sent back to the caller, the pool is unchanged).
func (suite *IntegrationTestSuite) TestServiceHalt_ModuleSwapForNativeDenom_RefusesHaltedDenom() {
	pool := deepNativePool(denomHalted)
	suite.k.SetLiquidityPool(suite.ctx, pool)
	suite.setHaltedDenoms(denomHalted)

	coins := sdk.NewCoins(sdk.NewInt64Coin(denomHalted, 1_000_000))
	moduleAcc := authtypes.NewEmptyModuleAccount("test_module")
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), "test_module").Return(moduleAcc).Times(1)
	// the capture happens inside the cache context that is discarded on error; nothing else is sent
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)

	result, err := suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrDenomHalted)
	suite.Require().Equal(sdk.Coin{}, result)

	stored, _ := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
	suite.Require().Equal(pool.ReserveBase, stored.ReserveBase)
	suite.Require().Equal(pool.ReserveQuote, stored.ReserveQuote)
}

// The burner add-liquidity path: a halted coin is refunded to the caller module, nothing is added.
func (suite *IntegrationTestSuite) TestServiceHalt_ModuleAddLiquidityWithNativeDenom_RefundsHaltedCoin() {
	pool := deepNativePool(denomHalted)
	suite.k.SetLiquidityPool(suite.ctx, pool)
	suite.setHaltedDenoms(denomHalted)

	coins := sdk.NewCoins(sdk.NewInt64Coin(denomHalted, 1_000_000))
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, "test_module", coins).Return(nil).Times(1)

	added, refunded, err := suite.k.ModuleAddLiquidityWithNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)
	suite.Require().Empty(added)
	suite.Require().Equal(coins, refunded)

	stored, _ := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
	suite.Require().Equal(pool.ReserveBase, stored.ReserveBase)
	suite.Require().Equal(pool.ReserveQuote, stored.ReserveQuote)
}

// A fee preferred in a halted denom is never swapped: both fee-payer entry points fall back to
// capturing the native fee as-is (the ante handler refuses the denom up front; this pins the keeper).
func (suite *IntegrationTestSuite) TestServiceHalt_FeePayer_PreferredHaltedDenom_FallsBackToNative() {
	pool := trySwapDeepPool() // stake/ubze, deep
	suite.k.SetLiquidityPool(suite.ctx, pool)
	suite.setHaltedDenoms(denomStake)

	addr1 := sdk.AccAddress("addr1_______________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(100_000_000)))
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	// the native fee is captured directly, once per entry point; no SpendableCoins lookup, no swap
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, fee).Return(nil).Times(2)

	captured, err := suite.k.CaptureAndSwapUserFee(ctx, addr1, fee, types.ModuleName)
	suite.Require().NoError(err)
	suite.Require().Equal(fee, captured)

	captured, err = suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, addr1, fee, types.ModuleName)
	suite.Require().NoError(err)
	suite.Require().Equal(fee, captured)

	stored, _ := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
	suite.Require().Equal(pool.ReserveBase, stored.ReserveBase)
	suite.Require().Equal(pool.ReserveQuote, stored.ReserveQuote)
}
