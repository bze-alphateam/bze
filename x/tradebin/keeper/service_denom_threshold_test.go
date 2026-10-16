package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

// thresholdPool stores a ubze/uother pool whose ubze side is nativeReserve and returns the fee coins the
// txfeecollector would convert.
func (suite *IntegrationTestSuite) thresholdPool(nativeReserve math.Int) sdk.Coins {
	suite.Require().NoError(suite.k.SetParams(suite.ctx, denomTestParams("ubze")))
	suite.k.SetLiquidityPool(suite.ctx, types.LiquidityPool{
		Id:           "ubze_uother",
		Base:         "ubze",
		Quote:        "uother",
		ReserveBase:  nativeReserve,
		ReserveQuote: math.NewInt(1_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(5, 1),
			Providers: math.LegacyZeroDec(),
		},
	})

	return sdk.NewCoins(sdk.NewInt64Coin("uother", 1000))
}

func (suite *IntegrationTestSuite) mockModuleAccounts() {
	for _, name := range []string{"test_module", types.ModuleName} {
		acc := authtypes.ModuleAccount{BaseAccount: &authtypes.BaseAccount{Address: sdk.AccAddress(name).String()}, Name: name}
		suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), name).Return(&acc).AnyTimes()
	}
}

// A pool whose ubze side is exactly MinNativeLiquidityForModuleSwap does not qualify: the ante check
// (HasDeepLiquidityWithNativeDenom), the txfeecollector classification (CanSwapForNativeDenom) and the
// conversion itself (ModuleSwapForNativeDenom) all refuse it, so a fee is never accepted in a denom the
// EndBlock then fails to convert.
func (suite *IntegrationTestSuite) TestServiceDenom_LiquidityThreshold_ExactlyAtThreshold_AllRefuse() {
	coins := suite.thresholdPool(denomTestParams("ubze").MinNativeLiquidityForModuleSwap)
	suite.mockModuleAccounts()
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "test_module", types.ModuleName, coins).Return(nil).AnyTimes()

	suite.Require().False(suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, "uother"), "ante")
	suite.Require().False(suite.k.CanSwapForNativeDenom(suite.ctx, coins[0]), "txfeecollector classification")
	_, err := suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().ErrorContains(err, "not enough liquidity available to swap coin uother to native coin")
}

// One ubze above the threshold, all three accept and the conversion succeeds.
func (suite *IntegrationTestSuite) TestServiceDenom_LiquidityThreshold_OneAboveThreshold_AllAccept() {
	coins := suite.thresholdPool(denomTestParams("ubze").MinNativeLiquidityForModuleSwap.AddRaw(1))
	suite.mockModuleAccounts()
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	suite.Require().True(suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, "uother"), "ante")
	suite.Require().True(suite.k.CanSwapForNativeDenom(suite.ctx, coins[0]), "txfeecollector classification")
	out, err := suite.k.ModuleSwapForNativeDenom(suite.ctx, "test_module", coins)
	suite.Require().NoError(err)
	suite.Require().Equal("ubze", out.Denom)
	suite.Require().True(out.Amount.IsPositive())
}
