package keeper_test

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"

	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// A coin whose pool price drifted is kept in the collector without failing the rest of the page; the next pass, with
// the price back in range, swaps it.
func (suite *IntegrationTestSuite) TestConvertCollectedFees_PriceDriftKeepsCoinAndRetriesLater() {
	suite.setMaxBalanceIterations(3)
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	suite.accountMock.EXPECT().GetModuleAddress(types.ModuleName).Return(moduleAddr).AnyTimes()

	a, b := sdk.NewInt64Coin("ibc/A", 100), sdk.NewInt64Coin("ibc/B", 200)
	page := sdk.NewCoins(a, b)
	suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), moduleAddr, "", 3).Return(page, "").Times(1)
	for _, coin := range page {
		suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), coin.Denom).Return(false).Times(1)
		suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), coin).Return(true).Times(1)
	}
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(a)).
		Return(sdk.Coin{}, tradebintypes.ErrPoolPriceDrift).
		Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(b)).
		Return(sdk.NewInt64Coin(denomBze, 20), nil).
		Times(1)
	// only b's result is sent: a is still in the collector
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(denomBze, 20))).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx))

	// next pass: only a is left and its pool price is back in range
	suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), moduleAddr, "", 3).Return(sdk.NewCoins(a), "").Times(1)
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), a.Denom).Return(false).Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), a).Return(true).Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(a)).
		Return(sdk.NewInt64Coin(denomBze, 10), nil).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(denomBze, 10))).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx))
}

// Same for the community pool collector: the drifted coin is not funded and not lost, the other coin is.
func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_PriceDriftKeepsCoin() {
	drifted, ok := sdk.NewInt64Coin(denomTokenA, 1000), sdk.NewInt64Coin(denomStake, 500)
	suite.expectCpBalances(sdk.NewCoins(drifted, ok))

	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).Times(2)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), drifted).Return(true).Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), ok).Return(true).Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(drifted)).
		Return(sdk.Coin{}, errors.Join(tradebintypes.ErrPoolPriceDrift)).
		Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(ok)).
		Return(sdk.NewInt64Coin(denomBze, 90), nil).
		Times(1)
	suite.distrMock.EXPECT().
		FundCommunityPool(gomock.Any(), sdk.NewCoins(sdk.NewInt64Coin(denomBze, 90)), cpModuleAddr).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ConvertCommunityPoolFees(suite.ctx))
}
