package keeper_test

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// A coin whose swap panics (e.g. an arithmetic overflow on an oversized amount) only fails itself: the other coins of
// the page are still swapped and sent, the pass commits, and the next pass starts after the page.
func (suite *IntegrationTestSuite) TestConvertCollectedFees_FailingCoinDoesNotBlockOthers() {
	suite.setMaxBalanceIterations(3)
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	suite.accountMock.EXPECT().GetModuleAddress(types.ModuleName).Return(moduleAddr).AnyTimes()

	a, b, c := sdk.NewInt64Coin("ibc/A", 100), sdk.NewInt64Coin("ibc/B", 200), sdk.NewInt64Coin("ibc/C", 300)
	page := sdk.NewCoins(a, b, c)
	suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), moduleAddr, "", 3).Return(page, "ibc/C").Times(1)
	for _, coin := range page {
		suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), coin.Denom).Return(false).Times(1)
		suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), coin).Return(true).Times(1)
	}
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(a)).
		Return(sdk.NewInt64Coin(denomBze, 10), nil).
		Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(b)).
		DoAndReturn(func(_ sdk.Context, _ string, _ sdk.Coins) (sdk.Coin, error) {
			panic("integer overflow")
		}).
		Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(c)).
		Return(sdk.NewInt64Coin(denomBze, 30), nil).
		Times(1)

	sent := false
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(denomBze, 40))).
		DoAndReturn(func(_ sdk.Context, _, _ string, _ sdk.Coins) error {
			sent = true
			return nil
		}).
		Times(1)

	// run it the way EndBlock does
	err := bzeutils.ApplyFuncIfNoError(suite.ctx, suite.k.ConvertCollectedFeesToNativeDenom)
	suite.Require().NoError(err)
	suite.Require().True(sent, "the swapped coins must reach the fee collector")

	// the cursor moved past the page: the next pass reads after ibc/C, then wraps to the start
	gomock.InOrder(
		suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), moduleAddr, "ibc/C", 3).Return(nil, "").Times(1),
		suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), moduleAddr, "", 3).Return(nil, "").Times(1),
	)
	suite.Require().NoError(bzeutils.ApplyFuncIfNoError(suite.ctx, suite.k.ConvertCollectedFeesToNativeDenom))
}

// A coin whose swap returns an error is kept in the collector without failing the rest of the pass.
func (suite *IntegrationTestSuite) TestConvertCollectedFees_SwapErrorKeepsCoinAndConvertsOthers() {
	suite.setMaxBalanceIterations(3)
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	suite.accountMock.EXPECT().GetModuleAddress(types.ModuleName).Return(moduleAddr).AnyTimes()

	a, b := sdk.NewInt64Coin("ibc/A", 100), sdk.NewInt64Coin("ibc/B", 200)
	page := sdk.NewCoins(a, b)
	suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), moduleAddr, "", 3).Return(page, "ibc/B").Times(1)
	for _, coin := range page {
		suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), coin.Denom).Return(false).Times(1)
		suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), coin).Return(true).Times(1)
	}
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(a)).
		Return(sdk.Coin{}, errors.New("swap failed")).
		Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(b)).
		Return(sdk.NewInt64Coin(denomBze, 20), nil).
		Times(1)

	sent := false
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(denomBze, 20))).
		DoAndReturn(func(_ sdk.Context, _, _ string, _ sdk.Coins) error {
			sent = true
			return nil
		}).
		Times(1)

	suite.Require().NoError(suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx))
	suite.Require().True(sent)
}

// In the community pool pass, a coin whose handling panics only fails itself: the other coins are still swapped and
// fund the community pool, and the next pass starts after the page.
func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_FailingCoinDoesNotBlockOthers() {
	suite.setMaxBalanceIterations(3)
	suite.accountMock.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpModuleAddr).AnyTimes()

	a, b, c := sdk.NewInt64Coin("ibc/A", 100), sdk.NewInt64Coin("ibc/B", 200), sdk.NewInt64Coin("ibc/C", 300)
	page := sdk.NewCoins(a, b, c)
	suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), cpModuleAddr, "", 3).Return(page, "ibc/C").Times(1)
	for _, coin := range page {
		suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), coin.Denom).Return(false).Times(1)
	}
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), a).Return(true).Times(1)
	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(gomock.Any(), b).
		DoAndReturn(func(_ sdk.Context, _ sdk.Coin) bool {
			panic("integer overflow")
		}).
		Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), c).Return(true).Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(a)).
		Return(sdk.NewInt64Coin(denomBze, 10), nil).
		Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(c)).
		Return(sdk.NewInt64Coin(denomBze, 30), nil).
		Times(1)

	funded := false
	suite.distrMock.EXPECT().
		FundCommunityPool(gomock.Any(), sdk.NewCoins(sdk.NewInt64Coin(denomBze, 40)), cpModuleAddr).
		DoAndReturn(func(_ sdk.Context, _ sdk.Coins, _ sdk.AccAddress) error {
			funded = true
			return nil
		}).
		Times(1)

	suite.Require().NoError(bzeutils.ApplyFuncIfNoError(suite.ctx, suite.k.ConvertCommunityPoolFees))
	suite.Require().True(funded, "the swapped coins must fund the community pool")

	gomock.InOrder(
		suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), cpModuleAddr, "ibc/C", 3).Return(nil, "").Times(1),
		suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), cpModuleAddr, "", 3).Return(nil, "").Times(1),
	)
	suite.Require().NoError(bzeutils.ApplyFuncIfNoError(suite.ctx, suite.k.ConvertCommunityPoolFees))
}
