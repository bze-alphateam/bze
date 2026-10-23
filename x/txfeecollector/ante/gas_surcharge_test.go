package ante_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/txfeecollector/ante"
	"github.com/bze-alphateam/bze/x/txfeecollector/testutil"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

const denomToken = "factory/bze1creator/token"

var tokenCp = sdk.AccAddress("token_cp____________")

// useDenomConfig rebuilds the keeper with a denom config provider mock, as the app wires the tokenfactory keeper.
func (suite *AnteTestSuite) useDenomConfig() *testutil.MockDenomCommunityPoolProvider {
	provider := testutil.NewMockDenomCommunityPoolProvider(suite.mockCtrl)
	suite.k, suite.ctx = keepertest.TxfeecollectorKeeperWithMocks(suite.T(), suite.bankMock, suite.accountMock, suite.tradeMock, suite.distrMock, provider)

	return provider
}

// expectTokenFeeDeduction sets the mocks for a tx paying fee in denomToken from payer.
func (suite *AnteTestSuite) expectTokenFeeDeduction(payer sdk.AccAddress, fee sdk.Coins) {
	suite.accountMock.EXPECT().GetModuleAddress(authtypes.FeeCollectorName).Return(sdk.AccAddress("feecollector________")).Times(1)
	suite.accountMock.EXPECT().GetAccount(gomock.Any(), payer).Return(authtypes.NewBaseAccountWithAddress(payer)).Times(1)
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomToken).Return(false).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), payer, types.ModuleName, fee).
		Return(nil).
		Times(1)
}

// Spot price 0.1 ubze per token and 0.01 ubze/gas give 0.1 token/gas; a 50% surcharge makes it 0.15 token/gas,
// so 100,000 gas needs 15,000 tokens.
func (suite *AnteTestSuite) TestGasSurcharge_MinGasPriceIncludesSurcharge() {
	provider := suite.useDenomConfig()
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)
	feePayer := sdk.AccAddress("feepayer____________")

	provider.EXPECT().GetDenomFeeRates(gomock.Any(), denomToken).
		Return(sdkmath.LegacyZeroDec(), sdkmath.LegacyMustNewDecFromStr("0.5")).
		AnyTimes()
	suite.tradeMock.EXPECT().GetDenomSpotPriceInNativeCoin(gomock.Any(), denomToken).
		Return(sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 1)), nil).
		AnyTimes()

	checkCtx := suite.ctx.WithIsCheckTx(true).
		WithValue(ante.FeeDenomKey, denomToken).
		WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2))))

	// what the price was without the surcharge is now too little
	tooLow := &mockFeeTx{fee: sdk.NewCoins(sdk.NewInt64Coin(denomToken, 14_999)), gas: 100_000, feePayer: feePayer}
	_, err := decorator.AnteHandle(checkCtx, tooLow, false, suite.mockNext())
	suite.Require().ErrorContains(err, "insufficient fees")
	suite.Require().ErrorContains(err, "15000"+denomToken)
	suite.Require().False(suite.nextCalled)

	fee := sdk.NewCoins(sdk.NewInt64Coin(denomToken, 15_000))
	suite.expectTokenFeeDeduction(feePayer, fee)
	provider.EXPECT().GetDenomCommunityPool(gomock.Any(), denomToken).Return(tokenCp, true).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.CpFeeCollector, sdk.NewCoins(sdk.NewInt64Coin(denomToken, 5_000))).
		Return(nil).
		Times(1)

	_, err = decorator.AnteHandle(checkCtx, &mockFeeTx{fee: fee, gas: 100_000, feePayer: feePayer}, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

// A spot-price failure keeps today's fallback: the native min gas price, surcharge ignored.
func (suite *AnteTestSuite) TestGasSurcharge_SpotPriceFailureKeepsFallback() {
	suite.useDenomConfig()
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)

	suite.tradeMock.EXPECT().GetDenomSpotPriceInNativeCoin(gomock.Any(), denomToken).
		Return(sdk.DecCoin{}, fmt.Errorf("no pool")).
		Times(1)

	checkCtx := suite.ctx.WithIsCheckTx(true).
		WithValue(ante.FeeDenomKey, denomToken).
		WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoinFromDec(denomBze, sdkmath.LegacyNewDecWithPrec(1, 2))))

	tx := &mockFeeTx{fee: sdk.NewCoins(sdk.NewInt64Coin(denomToken, 1_000_000)), gas: 100_000}
	_, err := decorator.AnteHandle(checkCtx, tx, false, suite.mockNext())
	suite.Require().ErrorContains(err, "1000"+denomBze)
}

// The split runs on the fee actually paid (DeliverTx checks no min gas price): floor(fee × s / (1 + s)) goes to the
// token's community pool collector, the rest stays with the module as before.
func (suite *AnteTestSuite) TestGasSurcharge_SplitToCpCollector() {
	provider := suite.useDenomConfig()
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)
	feePayer := sdk.AccAddress("feepayer____________")
	fee := sdk.NewCoins(sdk.NewInt64Coin(denomToken, 1_001))

	suite.expectTokenFeeDeduction(feePayer, fee)
	provider.EXPECT().GetDenomFeeRates(gomock.Any(), denomToken).
		Return(sdkmath.LegacyMustNewDecFromStr("0.05"), sdkmath.LegacyMustNewDecFromStr("0.25")).
		Times(1)
	provider.EXPECT().GetDenomCommunityPool(gomock.Any(), denomToken).Return(tokenCp, true).Times(1)
	// floor(1001 × 0.25 / 1.25) = floor(200.2)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.CpFeeCollector, sdk.NewCoins(sdk.NewInt64Coin(denomToken, 200))).
		Return(nil).
		Times(1)

	deliverCtx := suite.ctx.WithValue(ante.FeeDenomKey, denomToken)
	_, err := decorator.AnteHandle(deliverCtx, &mockFeeTx{fee: fee, gas: 100_000, feePayer: feePayer}, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

// A token without a community pool has its surcharge burned.
func (suite *AnteTestSuite) TestGasSurcharge_NoCommunityPoolGoesToBurner() {
	provider := suite.useDenomConfig()
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)
	feePayer := sdk.AccAddress("feepayer____________")
	fee := sdk.NewCoins(sdk.NewInt64Coin(denomToken, 3_000))

	suite.expectTokenFeeDeduction(feePayer, fee)
	provider.EXPECT().GetDenomFeeRates(gomock.Any(), denomToken).
		Return(sdkmath.LegacyZeroDec(), sdkmath.LegacyMustNewDecFromStr("2")).
		Times(1)
	provider.EXPECT().GetDenomCommunityPool(gomock.Any(), denomToken).Return(nil, false).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.BurnerFeeCollector, sdk.NewCoins(sdk.NewInt64Coin(denomToken, 2_000))).
		Return(nil).
		Times(1)

	_, err := decorator.AnteHandle(suite.ctx.WithValue(ante.FeeDenomKey, denomToken), &mockFeeTx{fee: fee, gas: 100_000, feePayer: feePayer}, false, suite.mockNext())
	suite.Require().NoError(err)
}

// A fee grant is charged with the full declared fee, surcharge included, and the granter pays the surcharge split.
func (suite *AnteTestSuite) TestGasSurcharge_FeeGrant() {
	provider := suite.useDenomConfig()
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)
	feePayer := sdk.AccAddress("feepayer____________")
	feeGranter := sdk.AccAddress("feegranter__________")
	fee := sdk.NewCoins(sdk.NewInt64Coin(denomToken, 1_500))

	suite.feegrantMock.EXPECT().UseGrantedFees(gomock.Any(), feeGranter, feePayer, fee, gomock.Any()).Return(nil).Times(1)
	suite.expectTokenFeeDeduction(feeGranter, fee)
	provider.EXPECT().GetDenomFeeRates(gomock.Any(), denomToken).
		Return(sdkmath.LegacyZeroDec(), sdkmath.LegacyMustNewDecFromStr("0.5")).
		Times(1)
	provider.EXPECT().GetDenomCommunityPool(gomock.Any(), denomToken).Return(tokenCp, true).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.CpFeeCollector, sdk.NewCoins(sdk.NewInt64Coin(denomToken, 500))).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{fee: fee, gas: 100_000, feePayer: feePayer, feeGranter: feeGranter}
	_, err := decorator.AnteHandle(suite.ctx.WithValue(ante.FeeDenomKey, denomToken), tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

// BZE fees are unchanged: no rate read, no split.
func (suite *AnteTestSuite) TestGasSurcharge_NativeFeeUnchanged() {
	suite.useDenomConfig()
	decorator := ante.NewDeductFeeDecorator(suite.tradeMock, suite.accountMock, suite.bankMock, suite.feegrantMock, &suite.k)
	feePayer := sdk.AccAddress("feepayer____________")
	fee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 1_000))

	suite.accountMock.EXPECT().GetModuleAddress(authtypes.FeeCollectorName).Return(sdk.AccAddress("feecollector________")).Times(1)
	suite.accountMock.EXPECT().GetAccount(gomock.Any(), feePayer).Return(authtypes.NewBaseAccountWithAddress(feePayer)).Times(1)
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomBze).Return(true).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, authtypes.FeeCollectorName, fee).
		Return(nil).
		Times(1)

	_, err := decorator.AnteHandle(suite.ctx, &mockFeeTx{fee: fee, gas: 100_000, feePayer: feePayer}, false, suite.mockNext())
	suite.Require().NoError(err)
}
