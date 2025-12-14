package keeper_test

import (
	"cosmossdk.io/math"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestConvertCollectedFeesToNativeDenom_NoBalance() {
	moduleAddr := sdk.AccAddress("txfeecollector_addr")

	suite.accountMock.EXPECT().
		GetModuleAddress(types.ModuleName).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(sdk.NewCoins()).
		Times(1)

	err := suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestConvertCollectedFeesToNativeDenom_OnlyNativeCoins() {
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	nativeBalance := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 1000))

	suite.accountMock.EXPECT().
		GetModuleAddress(types.ModuleName).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(nativeBalance).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, denomBze).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(suite.ctx, types.ModuleName, sdk.NewCoins()).
		Return(sdk.NewCoin(denomBze, math.ZeroInt()), nil).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, authtypes.FeeCollectorName, nativeBalance).
		Return(nil).
		Times(1)

	err := suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestConvertCollectedFeesToNativeDenom_SwapSuccess() {
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	balances := sdk.NewCoins(
		sdk.NewInt64Coin(denomStake, 1000),
		sdk.NewInt64Coin(denomOther, 500),
	)
	swappedAmount := sdk.NewInt64Coin(denomBze, 1500)

	suite.accountMock.EXPECT().
		GetModuleAddress(types.ModuleName).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(balances).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, denomStake).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, denomOther).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(suite.ctx, balances[0]).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(suite.ctx, balances[1]).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(suite.ctx, types.ModuleName, balances).
		Return(swappedAmount, nil).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, authtypes.FeeCollectorName, sdk.NewCoins(swappedAmount)).
		Return(nil).
		Times(1)

	err := suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestConvertCollectedFeesToNativeDenom_MixedCoins() {
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	nativeCoin := sdk.NewInt64Coin(denomBze, 500)
	swappableCoin := sdk.NewInt64Coin(denomStake, 1000)
	balances := sdk.NewCoins(nativeCoin, swappableCoin)
	swappedAmount := sdk.NewInt64Coin(denomBze, 1000)
	expectedTotal := sdk.NewInt64Coin(denomBze, 1500)

	suite.accountMock.EXPECT().
		GetModuleAddress(types.ModuleName).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(balances).
		Times(1)

	// Expectations for iteration - order may vary
	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, gomock.Any()).
		DoAndReturn(func(_ sdk.Context, denom string) bool {
			return denom == denomBze
		}).
		Times(2)

	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(suite.ctx, swappableCoin).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(suite.ctx, types.ModuleName, sdk.NewCoins(swappableCoin)).
		Return(swappedAmount, nil).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, authtypes.FeeCollectorName, sdk.NewCoins(expectedTotal)).
		Return(nil).
		Times(1)

	err := suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestConvertCollectedFeesToNativeDenom_OnlyUnswappableCoins() {
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	balances := sdk.NewCoins(sdk.NewInt64Coin(denomStake, 1000))

	suite.accountMock.EXPECT().
		GetModuleAddress(types.ModuleName).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(balances).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, denomStake).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(suite.ctx, balances[0]).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		HasDeepLiquidityWithNativeDenom(suite.ctx, denomStake).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(suite.ctx, types.ModuleName, sdk.NewCoins()).
		Return(sdk.NewCoin(denomBze, math.ZeroInt()), nil).
		Times(1)

	err := suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestConvertCollectedFeesToNativeDenom_ZeroAmountCoins() {
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	zeroCoin := sdk.NewInt64Coin(denomStake, 0)
	positiveCoin := sdk.NewInt64Coin(denomOther, 100)
	// Note: NewCoins filters out zero coins
	balances := sdk.Coins{zeroCoin, positiveCoin}
	swappedAmount := sdk.NewInt64Coin(denomBze, 100)

	suite.accountMock.EXPECT().
		GetModuleAddress(types.ModuleName).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(balances).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, gomock.Any()).
		DoAndReturn(func(_ sdk.Context, denom string) bool {
			return false
		}).
		Times(2)

	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(suite.ctx, positiveCoin).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(suite.ctx, types.ModuleName, sdk.NewCoins(positiveCoin)).
		Return(swappedAmount, nil).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, authtypes.FeeCollectorName, sdk.NewCoins(swappedAmount)).
		Return(nil).
		Times(1)

	err := suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestConvertCollectedFeesToNativeDenom_SwapReturnsZero() {
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	balances := sdk.NewCoins(sdk.NewInt64Coin(denomStake, 1000))

	suite.accountMock.EXPECT().
		GetModuleAddress(types.ModuleName).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(balances).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, denomStake).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(suite.ctx, balances[0]).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(suite.ctx, types.ModuleName, balances).
		Return(sdk.NewCoin(denomBze, math.ZeroInt()), nil).
		Times(1)

	err := suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestConvertCollectedFeesToNativeDenom_SwapError() {
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	balances := sdk.NewCoins(sdk.NewInt64Coin(denomStake, 1000))

	suite.accountMock.EXPECT().
		GetModuleAddress(types.ModuleName).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(balances).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, denomStake).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(suite.ctx, balances[0]).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(suite.ctx, types.ModuleName, balances).
		Return(sdk.Coin{}, sdkerrors.ErrInvalidCoins).
		Times(1)

	err := suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidCoins)
}

func (suite *IntegrationTestSuite) TestConvertCollectedFeesToNativeDenom_SendCoinsError() {
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	balances := sdk.NewCoins(sdk.NewInt64Coin(denomStake, 1000))
	swappedAmount := sdk.NewInt64Coin(denomBze, 1000)

	suite.accountMock.EXPECT().
		GetModuleAddress(types.ModuleName).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(balances).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, denomStake).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(suite.ctx, balances[0]).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(suite.ctx, types.ModuleName, balances).
		Return(swappedAmount, nil).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, authtypes.FeeCollectorName, sdk.NewCoins(swappedAmount)).
		Return(sdkerrors.ErrInsufficientFunds).
		Times(1)

	err := suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds)
}

func (suite *IntegrationTestSuite) TestConvertBurnerFeesToNativeDenom_Success() {
	moduleAddr := sdk.AccAddress("burner_collector_addr")
	balances := sdk.NewCoins(sdk.NewInt64Coin(denomStake, 1000))
	swappedAmount := sdk.NewInt64Coin(denomBze, 1000)

	suite.accountMock.EXPECT().
		GetModuleAddress(types.BurnerFeeCollector).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(balances).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, denomStake).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(suite.ctx, balances[0]).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(suite.ctx, types.BurnerFeeCollector, balances).
		Return(swappedAmount, nil).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.BurnerFeeCollector, burnermoduletypes.ModuleName, sdk.NewCoins(swappedAmount)).
		Return(nil).
		Times(1)

	err := suite.k.ConvertBurnerFeesToNativeDenom(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestConvertBurnerFeesToNativeDenom_NoBalance() {
	moduleAddr := sdk.AccAddress("burner_collector_addr")

	suite.accountMock.EXPECT().
		GetModuleAddress(types.BurnerFeeCollector).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(sdk.NewCoins()).
		Times(1)

	err := suite.k.ConvertBurnerFeesToNativeDenom(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestConvertCommunityPoolFeesToNativeDenom_Success() {
	moduleAddr := sdk.AccAddress("cp_collector_addr")
	balances := sdk.NewCoins(sdk.NewInt64Coin(denomStake, 1000))
	swappedAmount := sdk.NewInt64Coin(denomBze, 1000)

	suite.accountMock.EXPECT().
		GetModuleAddress(types.CpFeeCollector).
		Return(moduleAddr).
		Times(2)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(balances).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, denomStake).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(suite.ctx, balances[0]).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(suite.ctx, types.CpFeeCollector, balances).
		Return(swappedAmount, nil).
		Times(1)

	suite.distrMock.EXPECT().
		FundCommunityPool(suite.ctx, sdk.NewCoins(swappedAmount), moduleAddr).
		Return(nil).
		Times(1)

	err := suite.k.ConvertCommunityPoolFeesToNativeDenom(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestConvertCommunityPoolFeesToNativeDenom_NoBalance() {
	moduleAddr := sdk.AccAddress("cp_collector_addr")

	suite.accountMock.EXPECT().
		GetModuleAddress(types.CpFeeCollector).
		Return(moduleAddr).
		Times(1)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(sdk.NewCoins()).
		Times(1)

	err := suite.k.ConvertCommunityPoolFeesToNativeDenom(suite.ctx)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestConvertCommunityPoolFeesToNativeDenom_FundCommunityPoolError() {
	moduleAddr := sdk.AccAddress("cp_collector_addr")
	balances := sdk.NewCoins(sdk.NewInt64Coin(denomStake, 1000))
	swappedAmount := sdk.NewInt64Coin(denomBze, 1000)

	suite.accountMock.EXPECT().
		GetModuleAddress(types.CpFeeCollector).
		Return(moduleAddr).
		Times(2)

	suite.bankMock.EXPECT().
		GetAllBalances(suite.ctx, moduleAddr).
		Return(balances).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(suite.ctx, denomStake).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		CanSwapForNativeDenom(suite.ctx, balances[0]).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(suite.ctx, types.CpFeeCollector, balances).
		Return(swappedAmount, nil).
		Times(1)

	suite.distrMock.EXPECT().
		FundCommunityPool(suite.ctx, sdk.NewCoins(swappedAmount), moduleAddr).
		Return(sdkerrors.ErrInsufficientFunds).
		Times(1)

	err := suite.k.ConvertCommunityPoolFeesToNativeDenom(suite.ctx)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds)
}
