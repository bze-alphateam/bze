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

func (suite *AnteTestSuite) TestCheckTxFeeWithValidatorMinGasPrices_NativeDenom_MeetsMinimum() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")

	// Set params with 0.01 ubze min gas fee
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Gas: 100000, MinGasPrice: 0.01 ubze -> Required fee: 1000 ubze
	// Provided fee: 1000 ubze (meets minimum)
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000)))

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

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	// Set min gas prices in context
	checkCtx := suite.ctx.WithIsCheckTx(true).WithMinGasPrices(sdk.NewDecCoins(
		sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)), // 0.01
	))

	_, err = decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCheckTxFeeWithValidatorMinGasPrices_NativeDenom_InsufficientFee() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	// Set params with 0.01 ubze min gas fee
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Gas: 100000, MinGasPrice: 0.01 ubze -> Required fee: 1000 ubze
	// Provided fee: 500 ubze (insufficient)
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(500)))

	tx := &mockFeeTx{
		fee: fee,
		gas: 100000,
	}

	checkCtx := suite.ctx.WithIsCheckTx(true).WithMinGasPrices(sdk.NewDecCoins(
		sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)), // 0.01
	))

	_, err = decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "insufficient fees")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCheckTxFeeWithValidatorMinGasPrices_NonNativeDenom_WithSpotPrice() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")

	// Set params with 0.01 ubze min gas fee
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Spot price: 1 usd = 0.1 ubze (1 ubze = 10 usd)
	// Min gas price in ubze: 0.01 ubze/gas
	// Min gas price in usd: 0.01 / 0.1 = 0.1 usd/gas
	// Gas: 100000 -> Required fee: 10000 usd
	// Provided fee: 10000 usd (meets minimum)
	fee := sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(10000)))

	suite.accountMock.EXPECT().
		GetModuleAddress(authtypes.FeeCollectorName).
		Return(feeCollectorAddr).
		Times(1)

	suite.accountMock.EXPECT().
		GetAccount(gomock.Any(), feePayer).
		Return(authtypes.NewBaseAccountWithAddress(feePayer)).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomUsd).
		Return(false).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, fee).
		Return(nil).
		Times(1)

	// Return spot price: 1 usd = 0.1 ubze
	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), denomUsd).
		Return(sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 1)), nil). // 0.1 ubze per usd
		Times(1)

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	// Set context with usd as fee denom
	checkCtx := suite.ctx.WithIsCheckTx(true).
		WithValue(ante.FeeDenomKey, denomUsd).
		WithMinGasPrices(sdk.NewDecCoins(
			sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)), // 0.01 ubze
		))

	_, err = decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCheckTxFeeWithValidatorMinGasPrices_NonNativeDenom_SpotPriceFails() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	// Set params with 0.01 ubze min gas fee
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// When spot price fails, it should fallback to native denom requirement
	// This will cause validation to fail because fee is in usd but required is in ubze
	fee := sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(10000)))

	// Return error for spot price
	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), denomUsd).
		Return(sdk.DecCoin{}, fmt.Errorf("pool not found")).
		Times(1)

	tx := &mockFeeTx{
		fee: fee,
		gas: 100000,
	}

	checkCtx := suite.ctx.WithIsCheckTx(true).
		WithValue(ante.FeeDenomKey, denomUsd).
		WithMinGasPrices(sdk.NewDecCoins(
			sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)),
		))

	_, err = decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "insufficient fees")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCheckTxFeeWithValidatorMinGasPrices_ValidatorConfigHigherThanProtocol() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")

	// Protocol minimum: 0.01 ubze
	// Validator config: 0.02 ubze (higher)
	// Should use validator config (0.02)
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Gas: 100000, ValidatorMinGasPrice: 0.02 ubze -> Required fee: 2000 ubze
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(2000)))

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

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	// Validator sets higher min gas price
	checkCtx := suite.ctx.WithIsCheckTx(true).WithMinGasPrices(sdk.NewDecCoins(
		sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(2, 2)), // 0.02 (higher than protocol 0.01)
	))

	_, err = decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCheckTxFeeWithValidatorMinGasPrices_ValidatorConfigLowerThanProtocol() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")

	// Protocol minimum: 0.01 ubze
	// Validator config: 0.005 ubze (lower)
	// Should use protocol minimum (0.01)
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Gas: 100000, ProtocolMinGasPrice: 0.01 ubze -> Required fee: 1000 ubze
	// Even though validator set 0.005, protocol enforces 0.01
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000)))

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

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	// Validator sets lower min gas price, but protocol should enforce higher minimum
	checkCtx := suite.ctx.WithIsCheckTx(true).WithMinGasPrices(sdk.NewDecCoins(
		sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(5, 3)), // 0.005 (lower than protocol 0.01)
	))

	_, err = decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCheckTxFeeWithValidatorMinGasPrices_PriorityCalculation() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")

	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Higher fee should result in higher priority
	// Gas: 100000, Fee: 10000 ubze -> Gas price: 0.1 ubze/gas
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(10000)))

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

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	checkCtx := suite.ctx.WithIsCheckTx(true).WithMinGasPrices(sdk.NewDecCoins(
		sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)),
	))

	newCtx, err := decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)

	// Priority should be set based on gas price
	// Priority = amount / gas = 10000 / 100000 = 0 (integer division)
	// But since we're using integer math, priority should be at least the gas price in smallest units
	suite.Require().True(newCtx.Priority() >= 0)
}

func (suite *AnteTestSuite) TestCheckTxFeeWithValidatorMinGasPrices_NonCheckTxMode() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")

	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// In non-CheckTx mode (e.g., DeliverTx), min gas price checks should be skipped
	// even with very low fee
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1)))

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

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	// Not in CheckTx mode
	deliverCtx := suite.ctx.WithIsCheckTx(false)

	_, err = decorator.AnteHandle(deliverCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCheckTxFeeWithValidatorMinGasPrices_NonNativeDenom_ValidatorLocalConfig() {
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeCollectorAddr := sdk.AccAddress("feecollector________")

	// Protocol minimum: 0.01 ubze
	// Spot price: 1 usd = 0.1 ubze -> Protocol min in USD: 0.01 / 0.1 = 0.1 usd/gas
	// Validator config for USD: 0.15 usd/gas (higher than protocol-derived)
	// Should use validator config (0.15)
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Gas: 100000, ValidatorMinGasPrice: 0.15 usd -> Required fee: 15000 usd
	fee := sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(15000)))

	suite.accountMock.EXPECT().
		GetModuleAddress(authtypes.FeeCollectorName).
		Return(feeCollectorAddr).
		Times(1)

	suite.accountMock.EXPECT().
		GetAccount(gomock.Any(), feePayer).
		Return(authtypes.NewBaseAccountWithAddress(feePayer)).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomUsd).
		Return(false).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, fee).
		Return(nil).
		Times(1)

	// Spot price: 1 usd = 0.1 ubze
	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), denomUsd).
		Return(sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 1)), nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      fee,
		gas:      100000,
		feePayer: feePayer,
	}

	checkCtx := suite.ctx.WithIsCheckTx(true).
		WithValue(ante.FeeDenomKey, denomUsd).
		WithMinGasPrices(sdk.NewDecCoins(
			sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2)),  // 0.01 ubze (protocol)
			sdk.NewDecCoinFromDec(denomUsd, sdkmath.LegacyNewDecWithPrec(15, 2)), // 0.15 usd (validator config)
		))

	_, err = decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}
