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
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnermoduletypes.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 500))).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ChargeGasSurcharge(suite.ctx, feeSender, fees))
	events := gasSurchargeEvents(suite.ctx)
	suite.Require().Len(events, 1)
	suite.Require().Equal(burnermoduletypes.ModuleName, events[0].Destination)
}

func (suite *IntegrationTestSuite) TestGasSurcharge_NothingWithoutRateOrForTinyFee() {
	suite.useDenomCpProvider()
	suite.expectRates(denomTokenA, "0.05", "0")
	suite.expectRates(denomTokenB, "0", "0.5")

	// tokenA has no surcharge, a 1 tokenB fee stays whole with the validators, ubze never reaches the provider
	fees := sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 1_000), sdk.NewInt64Coin(denomTokenB, 1), sdk.NewInt64Coin(denomBze, 1_000))
	suite.Require().NoError(suite.k.ChargeGasSurcharge(suite.ctx, feeSender, fees))
	suite.Require().Empty(gasSurchargeEvents(suite.ctx))
}

// A split that floors to 0 takes the smallest unit when the fee is above 1: 2 tokenA at 50% (floor(0.66)) and
// 3 tokenB at 10% (floor(0.27)) each send 1.
func (suite *IntegrationTestSuite) TestGasSurcharge_SplitFlooredToZeroTakesOneUnit() {
	suite.useDenomCpProvider()
	suite.expectRates(denomTokenA, "0", "0.5")
	suite.expectRates(denomTokenB, "0", "0.1")
	suite.expectCommunityPool(denomTokenA, true)
	suite.expectCommunityPool(denomTokenB, false)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.CpFeeCollector, sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 1))).
		Return(nil).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnermoduletypes.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(denomTokenB, 1))).
		Return(nil).
		Times(1)

	fees := sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 2), sdk.NewInt64Coin(denomTokenB, 3))
	suite.Require().NoError(suite.k.ChargeGasSurcharge(suite.ctx, feeSender, fees))
	events := gasSurchargeEvents(suite.ctx)
	suite.Require().Len(events, 2)
	for _, e := range events {
		suite.Require().Equal("1", e.Amount, "surcharge of %s", e.Denom)
	}
}

// At the top rate a fee of 2 floors to 1 (floor(2 × 2 / 3)), so the minimum changes nothing and the validators keep 1.
func (suite *IntegrationTestSuite) TestGasSurcharge_SmallFeeAtMaxRateKeepsOneForValidators() {
	suite.useDenomCpProvider()
	suite.expectRates(denomTokenA, "0", "2")
	suite.expectCommunityPool(denomTokenA, true)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.CpFeeCollector, sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 1))).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ChargeGasSurcharge(suite.ctx, feeSender, sdk.NewCoins(sdk.NewInt64Coin(denomTokenA, 2))))
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
