package ante_test

import (
	sdkmath "cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/txfeecollector/ante"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

var (
	anteFeePayer   = sdk.AccAddress("tx_fee_payer________")
	anteFeeGranter = sdk.AccAddress("tx_fee_granter______")
)

// The fee payer is stored next to the fee denom, as an account address.
func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_StoresFeePayer() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomUsd).Return(false).Times(1)
	suite.tradeMock.EXPECT().HasDeepLiquidityWithNativeDenom(gomock.Any(), denomUsd).Return(true).Times(1)

	tx := &mockFeeTx{fee: sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(1000))), gas: 100000, feePayer: anteFeePayer}
	_, err := decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())

	suite.Require().NoError(err)
	suite.Require().Equal(denomUsd, suite.nextCalledWith.Value(ante.FeeDenomKey))
	suite.Require().Equal(anteFeePayer, suite.nextCalledWith.Value(ante.FeePayerKey))
}

// ReCheckTx skips the validation but stores the same two values.
func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ReCheckTx_StoresFeePayer() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	tx := &mockFeeTx{fee: sdk.NewCoins(sdk.NewCoin(denomOther, sdkmath.NewInt(1000))), gas: 100000, feePayer: anteFeePayer}
	_, err := decorator.AnteHandle(suite.ctx.WithIsReCheckTx(true), tx, false, suite.mockNext())

	suite.Require().NoError(err)
	suite.Require().Equal(denomOther, suite.nextCalledWith.Value(ante.FeeDenomKey))
	suite.Require().Equal(anteFeePayer, suite.nextCalledWith.Value(ante.FeePayerKey))
}

// A fee granter pays the tx fee in coins, but the denom stays the choice of the fee payer who signed it.
func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_FeeGranterIsNotStored() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomUsd).Return(false).Times(1)
	suite.tradeMock.EXPECT().HasDeepLiquidityWithNativeDenom(gomock.Any(), denomUsd).Return(true).Times(1)

	tx := &mockFeeTx{
		fee:        sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(1000))),
		gas:        100000,
		feePayer:   anteFeePayer,
		feeGranter: anteFeeGranter,
	}
	_, err := decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())

	suite.Require().NoError(err)
	suite.Require().Equal(anteFeePayer, suite.nextCalledWith.Value(ante.FeePayerKey))
}

// A tx without a fee denom stores no fee payer either.
func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ReCheckTx_EmptyFeeStoresNoFeePayer() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	tx := &mockFeeTx{fee: sdk.NewCoins(), gas: 100000, feePayer: anteFeePayer}
	_, err := decorator.AnteHandle(suite.ctx.WithIsReCheckTx(true), tx, false, suite.mockNext())

	suite.Require().NoError(err)
	suite.Require().Nil(suite.nextCalledWith.Value(ante.FeePayerKey))
}
