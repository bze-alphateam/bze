package ante_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/txfeecollector/ante"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

func (suite *AnteTestSuite) TestDeductFeeDecorator_NativeFee() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000)))

	// Set params with validator min gas fee
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Setup mocks
	suite.accountMock.EXPECT().
		GetModuleAddress(authtypes.FeeCollectorName).
		Return(feeCollectorAddr).
		Times(1)

	suite.accountMock.EXPECT().
		GetAccount(gomock.Any(), feePayer).
		Return(authtypes.NewBaseAccountWithAddress(feePayer)).
		Times(1)

	// Native fee goes to fee collector
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomBze).
		Return(true).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, authtypes.FeeCollectorName, fee).
		Return(nil).
		Times(1)

	// For checkTxFeeWithValidatorMinGasPrices - need to mock GetDenomSpotPriceInNativeCoin
	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), gomock.Any()).
		Return(sdk.DecCoin{}, fmt.Errorf("not needed")).
		AnyTimes()

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	// Set context to CheckTx mode and add min gas prices
	checkCtx := suite.ctx.WithIsCheckTx(true).WithMinGasPrices(sdk.NewDecCoins(
		sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)), // 0.01
	))

	newCtx, err := decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
	suite.Require().True(newCtx.Priority() >= 0)
}

func (suite *AnteTestSuite) TestDeductFeeDecorator_NonNativeFee() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")
	fee := sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(10000)))

	// Set params
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Setup mocks
	suite.accountMock.EXPECT().
		GetModuleAddress(authtypes.FeeCollectorName).
		Return(feeCollectorAddr).
		Times(1)

	suite.accountMock.EXPECT().
		GetAccount(gomock.Any(), feePayer).
		Return(authtypes.NewBaseAccountWithAddress(feePayer)).
		Times(1)

	// Non-native fee goes to txfeecollector module
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomUsd).
		Return(false).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, fee).
		Return(nil).
		Times(1)

	// For checkTxFeeWithValidatorMinGasPrices
	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), denomUsd).
		Return(sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 1)), nil). // 0.1 ubze per usd
		AnyTimes()

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	// Set context with fee denom
	checkCtx := suite.ctx.WithIsCheckTx(true).
		WithValue(ante.FeeDenomKey, denomUsd).
		WithMinGasPrices(sdk.NewDecCoins(
			sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)), // 0.01 ubze
		))

	newCtx, err := decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
	suite.Require().True(newCtx.Priority() >= 0)
}

func (suite *AnteTestSuite) TestDeductFeeDecorator_MixedFees() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")
	nativeFee := sdk.NewCoin(denomBze, sdkmath.NewInt(1000))
	nonNativeFee := sdk.NewCoin(denomUsd, sdkmath.NewInt(500))
	totalFee := sdk.NewCoins(nativeFee, nonNativeFee)

	// Set params
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Setup mocks
	suite.accountMock.EXPECT().
		GetModuleAddress(authtypes.FeeCollectorName).
		Return(feeCollectorAddr).
		Times(1)

	suite.accountMock.EXPECT().
		GetAccount(gomock.Any(), feePayer).
		Return(authtypes.NewBaseAccountWithAddress(feePayer)).
		Times(1)

	// Check for native denom - first coin
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomBze).
		Return(true).
		Times(1)

	// Check for non-native denom - second coin
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomUsd).
		Return(false).
		Times(1)

	// Native fee goes to fee collector
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, authtypes.FeeCollectorName, sdk.NewCoins(nativeFee)).
		Return(nil).
		Times(1)

	// Non-native fee goes to txfeecollector module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, sdk.NewCoins(nonNativeFee)).
		Return(nil).
		Times(1)

	// For checkTxFeeWithValidatorMinGasPrices
	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), gomock.Any()).
		Return(sdk.DecCoin{}, fmt.Errorf("not needed")).
		AnyTimes()

	tx := &mockFeeTx{
		fee:      totalFee,
		gas:      100000,
		feePayer: feePayer,
	}

	checkCtx := suite.ctx.WithIsCheckTx(true).WithMinGasPrices(sdk.NewDecCoins(
		sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)),
	))

	newCtx, err := decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
	suite.Require().True(newCtx.Priority() >= 0)
}

func (suite *AnteTestSuite) TestDeductFeeDecorator_WithFeeGranter() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeGranter := sdk.AccAddress("feegranter__________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000)))

	// Set params
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Setup mocks
	suite.accountMock.EXPECT().
		GetModuleAddress(authtypes.FeeCollectorName).
		Return(feeCollectorAddr).
		Times(1)

	// UseGrantedFees should be called with granter and grantee
	suite.feegrantMock.EXPECT().
		UseGrantedFees(gomock.Any(), feeGranter, feePayer, fee, gomock.Any()).
		Return(nil).
		Times(1)

	suite.accountMock.EXPECT().
		GetAccount(gomock.Any(), feeGranter).
		Return(authtypes.NewBaseAccountWithAddress(feeGranter)).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomBze).
		Return(true).
		Times(1)

	// Fee should be deducted from granter
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feeGranter, authtypes.FeeCollectorName, fee).
		Return(nil).
		Times(1)

	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), gomock.Any()).
		Return(sdk.DecCoin{}, fmt.Errorf("not needed")).
		AnyTimes()

	tx := &mockFeeTx{
		fee:        fee,
		gas:        100000,
		feePayer:   feePayer,
		feeGranter: feeGranter,
	}

	checkCtx := suite.ctx.WithIsCheckTx(true).WithMinGasPrices(sdk.NewDecCoins(
		sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)),
	))

	_, err = decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestDeductFeeDecorator_ZeroGas() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000)))

	tx := &mockFeeTx{
		fee: fee,
		gas: 0, // Zero gas should fail
	}

	// Use non-zero block height to trigger the check
	nonSimulateCtx := suite.ctx.WithBlockHeight(1)

	_, err := decorator.AnteHandle(nonSimulateCtx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "must provide positive gas")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestDeductFeeDecorator_InsufficientFunds() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000)))

	// Set params
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Setup mocks
	suite.accountMock.EXPECT().
		GetModuleAddress(authtypes.FeeCollectorName).
		Return(feeCollectorAddr).
		Times(1)

	suite.accountMock.EXPECT().
		GetAccount(gomock.Any(), feePayer).
		Return(authtypes.NewBaseAccountWithAddress(feePayer)).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomBze).
		Return(true).
		Times(1)

	// Simulate insufficient funds
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, authtypes.FeeCollectorName, fee).
		Return(fmt.Errorf("insufficient funds")).
		Times(1)

	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), gomock.Any()).
		Return(sdk.DecCoin{}, fmt.Errorf("not needed")).
		AnyTimes()

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	checkCtx := suite.ctx.WithIsCheckTx(true).WithMinGasPrices(sdk.NewDecCoins(
		sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)),
	))

	_, err = decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "insufficient funds")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestDeductFeeDecorator_FeeCollectorNotSet() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000)))

	// Set params
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Setup mocks - return nil for fee collector address
	suite.accountMock.EXPECT().
		GetModuleAddress(authtypes.FeeCollectorName).
		Return(nil).
		Times(1)

	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), gomock.Any()).
		Return(sdk.DecCoin{}, fmt.Errorf("not needed")).
		AnyTimes()

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	checkCtx := suite.ctx.WithIsCheckTx(true).WithMinGasPrices(sdk.NewDecCoins(
		sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)),
	))

	_, err = decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "fee collector module account")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestDeductFeeDecorator_SimulateMode() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000)))
	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	// Even in simulate mode, checkDeductFee still runs
	suite.accountMock.EXPECT().
		GetModuleAddress(authtypes.FeeCollectorName).
		Return(feeCollectorAddr).
		Times(1)

	suite.accountMock.EXPECT().
		GetAccount(gomock.Any(), feePayer).
		Return(authtypes.NewBaseAccountWithAddress(feePayer)).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomBze).
		Return(true).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, authtypes.FeeCollectorName, fee).
		Return(nil).
		Times(1)

	_, err := decorator.AnteHandle(suite.ctx, tx, true, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestDeductFeeDecorator_AccountNotFound() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000)))

	// Set params
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Setup mocks
	suite.accountMock.EXPECT().
		GetModuleAddress(authtypes.FeeCollectorName).
		Return(feeCollectorAddr).
		Times(1)

	// Return nil for account (account doesn't exist)
	suite.accountMock.EXPECT().
		GetAccount(gomock.Any(), feePayer).
		Return(nil).
		Times(1)

	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), gomock.Any()).
		Return(sdk.DecCoin{}, fmt.Errorf("not needed")).
		AnyTimes()

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	checkCtx := suite.ctx.WithIsCheckTx(true).WithMinGasPrices(sdk.NewDecCoins(
		sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)),
	))

	_, err = decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "does not exist")
	suite.Require().False(suite.nextCalled)
}
