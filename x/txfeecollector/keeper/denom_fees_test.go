package keeper_test

import (

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

var feeSender = sdk.AccAddress("fee_sender__________")

func (suite *IntegrationTestSuite) expectRates(denom string, transferFee, gasSurcharge string) {
	suite.cpMock.EXPECT().
		GetDenomFeeRates(gomock.Any(), denom).
		Return(math.LegacyMustNewDecFromStr(transferFee), math.LegacyMustNewDecFromStr(gasSurcharge)).
		Times(1)
}

func (suite *IntegrationTestSuite) expectCommunityPool(denom string, has bool) {
	if has {
		suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denom).Return(tokenACp, true).Times(1)
		return
	}
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denom).Return(nil, false).Times(1)
}

func gasSurchargeEvents(ctx sdk.Context) []types.GasSurchargePaidEvent {
	var out []types.GasSurchargePaidEvent
	for _, e := range ctx.EventManager().Events() {
		if e.Type != proto.MessageName(&types.GasSurchargePaidEvent{}) {
			continue
		}
		msg, err := sdk.ParseTypedEvent(abci.Event(e))
		if err != nil {
			panic(err)
		}
		out = append(out, *msg.(*types.GasSurchargePaidEvent))
	}

	return out
}

func (suite *IntegrationTestSuite) TestGasSurcharge_FloorSplitToCpCollector() {
	suite.useDenomCpProvider()
	fees := sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 1_000))
	surcharge := sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 333)) // floor(1000 × 0.5 / 1.5)

	suite.expectRates(denomTokenA, "0.05", "0.5")
	suite.expectCommunityPool(denomTokenA, true)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.CpFeeCollector, surcharge).Return(nil).Times(1)

	suite.Require().NoError(suite.k.ChargeGasSurcharge(suite.ctx, feeSender, fees))
	suite.Require().Equal([]types.GasSurchargePaidEvent{{
		Payer:       feeSender.String(),
		Denom:       denomTokenA,
		Amount:      "333",
		Destination: types.CpFeeCollector,
	}}, gasSurchargeEvents(suite.ctx))
}

func (suite *IntegrationTestSuite) TestGasSurcharge_ExactSplitAtMaxRate() {
	suite.useDenomCpProvider()
	fees := sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 300))

	suite.expectRates(denomTokenA, "0", "2")
	suite.expectCommunityPool(denomTokenA, true)
	// 300 × 2 / 3 = 200 exactly
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.CpFeeCollector, sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 200))).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ChargeGasSurcharge(suite.ctx, feeSender, fees))
}

func (suite *IntegrationTestSuite) TestGasSurcharge_TokenWithoutCommunityPoolGoesToBurner() {
	suite.useDenomCpProvider()
	fees := sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 1_000))

	suite.expectRates(denomTokenA, "0", "1")
	suite.expectCommunityPool(denomTokenA, false)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.BurnerFeeCollector, sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 500))).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ChargeGasSurcharge(suite.ctx, feeSender, fees))
	events := gasSurchargeEvents(suite.ctx)
	suite.Require().Len(events, 1)
	suite.Require().Equal(types.BurnerFeeCollector, events[0].Destination)
}

func (suite *IntegrationTestSuite) TestGasSurcharge_NothingWithoutRateOrForTinyFee() {
	suite.useDenomCpProvider()
	suite.expectRates(denomTokenA, "0.05", "0")
	suite.expectRates(denomTokenB, "0", "0.5")

	// tokenA has no surcharge, 1 tokenB rounds down to 0, ubze never reaches the provider
	fees := sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 1_000), sdk.NewInt64Coin(denomTokenB, 1), sdk.NewInt64Coin(denomBze, 1_000))
	suite.Require().NoError(suite.k.ChargeGasSurcharge(suite.ctx, feeSender, fees))
	suite.Require().Empty(gasSurchargeEvents(suite.ctx))
}

func (suite *IntegrationTestSuite) TestGasSurcharge_NilProviderDoesNothing() {
	suite.Require().NoError(suite.k.ChargeGasSurcharge(suite.ctx, feeSender, sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 1_000))))
}

func (suite *IntegrationTestSuite) TestGetDenomFeeRates_ZerosWithoutProvider() {
	k, ctx := keepertest.TxfeecollectorKeeper(suite.T())
	transferFee, gasSurcharge := k.GetDenomFeeRates(ctx, denomTokenA)
	suite.Require().True(transferFee.IsZero())
	suite.Require().True(gasSurcharge.IsZero())
}

// After a successful routing, a coin past the routing bound whose denom has a community pool waits in
// txfeecollector_cp for the next block's routing: the conversion for the chain's community pool doesn't take it.
// Regression test for BZE-220: with three fee-bearing denoms and max_balance_iterations 2, the conversion sent the
// third (unswappable) to the burner.
func (suite *IntegrationTestSuite) TestConvertCommunityPoolFeesLeftAfterRouting_SkipsDenomsWithCommunityPool() {
	suite.useDenomCpProvider()
	balances := sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 30), sdk.NewInt64Coin(denomTokenB, 40))

	suite.accountMock.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpModuleAddr).AnyTimes()
	suite.bankMock.EXPECT().GetAllBalances(gomock.Any(), cpModuleAddr).Return(balances).Times(1)
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denomTokenA).Return(tokenACp, true).Times(1)
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denomTokenB).Return(nil, false).Times(1)
	// only tokenB, which has no community pool, is considered: unswappable, it goes to the burner
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), sdk.NewInt64Coin(denomTokenB, 40)).Return(false).Times(1)
	suite.tradeMock.EXPECT().HasDeepLiquidityWithNativeDenom(gomock.Any(), denomTokenB).Return(false).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.CpFeeCollector, burnermoduletypes.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(denomTokenB, 40))).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ConvertCommunityPoolFeesLeftAfterRouting(suite.ctx))
}

// After a failed routing, every coin takes the old path: ConvertCommunityPoolFeesToNativeDenom converts a coin whose
// denom has a community pool too, without reading the provider.
func (suite *IntegrationTestSuite) TestConvertCommunityPoolFeesToNativeDenom_ConvertsDenomsWithCommunityPool() {
	suite.useDenomCpProvider()
	coin := sdk.NewInt64Coin(denomTokenA, 30)

	suite.accountMock.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpModuleAddr).AnyTimes()
	suite.bankMock.EXPECT().GetAllBalances(gomock.Any(), cpModuleAddr).Return(sdk.NewCoins(coin)).Times(1)
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomTokenA).Return(false).Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), coin).Return(false).Times(1)
	suite.tradeMock.EXPECT().HasDeepLiquidityWithNativeDenom(gomock.Any(), denomTokenA).Return(false).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.CpFeeCollector, burnermoduletypes.ModuleName, sdk.NewCoins(coin)).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ConvertCommunityPoolFeesToNativeDenom(suite.ctx))
}
