package ante_test

import (
	sdkmath "cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/txfeecollector/ante"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_NativeDenom() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Setup: Native denom should pass IsNativeDenom check
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomBze).
		Return(true).
		Times(1)

	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas: 100000,
	}

	newCtx, err := decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)

	// Verify fee denom was set in context
	feeDenom := newCtx.Value(ante.FeeDenomKey)
	suite.Require().Equal(denomBze, feeDenom)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_NonNativeDenomWithDeepLiquidity() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Setup: Non-native denom with deep liquidity should pass
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomUsd).
		Return(false).
		Times(1)
	suite.tradeMock.EXPECT().
		HasDeepLiquidityWithNativeDenom(gomock.Any(), denomUsd).
		Return(true).
		Times(1)

	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(1000))),
		gas: 100000,
	}

	newCtx, err := decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)

	// Verify fee denom was set in context
	feeDenom := newCtx.Value(ante.FeeDenomKey)
	suite.Require().Equal(denomUsd, feeDenom)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_NonNativeDenomWithoutDeepLiquidity() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Setup: Non-native denom without deep liquidity should fail
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomOther).
		Return(false).
		Times(1)
	suite.tradeMock.EXPECT().
		HasDeepLiquidityWithNativeDenom(gomock.Any(), denomOther).
		Return(false).
		Times(1)

	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomOther, sdkmath.NewInt(1000))),
		gas: 100000,
	}

	_, err := decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "can be used to pay for fees only if enough liquidity is available")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_MultipleDenominations() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Multiple denominations should fail
	tx := &mockFeeTx{
		fee: sdk.NewCoins(
			sdk.NewCoin(denomBze, sdkmath.NewInt(1000)),
			sdk.NewCoin(denomUsd, sdkmath.NewInt(500)),
		),
		gas: 100000,
	}

	_, err := decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "multiple denominations for same transaction fee are not supported")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_EmptyFee() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Empty fee should fail in normal conditions (not genesis, not simulation)
	tx := &mockFeeTx{
		fee: sdk.NewCoins(),
		gas: 100000,
	}

	ctx := suite.ctx.WithBlockHeight(1)

	_, err := decorator.AnteHandle(ctx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "no fee supplied")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_EmptyFee_Genesis() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Empty fee should pass during genesis (block height 0)
	genesisCtx := suite.ctx.WithBlockHeight(0)
	tx := &mockFeeTx{
		fee: sdk.NewCoins(),
		gas: 100000,
	}

	_, err := decorator.AnteHandle(genesisCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_EmptyFee_Simulation() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Empty fee should pass during simulation
	tx := &mockFeeTx{
		fee: sdk.NewCoins(),
		gas: 100000,
	}

	_, err := decorator.AnteHandle(suite.ctx, tx, true, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ZeroFee() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Zero fee coins are filtered out by sdk.NewCoins, resulting in empty fee
	// So this test actually validates the empty fee scenario in normal conditions
	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(0))),
		gas: 100000,
	}

	_, err := decorator.AnteHandle(suite.ctx.WithBlockHeight(10), tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "no fee supplied")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ZeroFee_Genesis() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Zero fee should pass during genesis
	genesisCtx := suite.ctx.WithBlockHeight(0)
	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(0))),
		gas: 100000,
	}

	_, err := decorator.AnteHandle(genesisCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ZeroFee_Simulation() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Zero fee should pass during simulation
	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(0))),
		gas: 100000,
	}

	_, err := decorator.AnteHandle(suite.ctx, tx, true, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_NilTradeKeeper() {
	// Create decorator with nil trade keeper
	decorator := ante.NewValidateTxFeeDenomsDecorator(nil)

	// Non-native denom with nil trade keeper should fail
	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(1000))),
		gas: 100000,
	}

	_, err := decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid fee supplied")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ReCheckTx() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// ReCheckTx should bypass validation but still set FeeDenomKey in context
	recheckCtx := suite.ctx.WithIsReCheckTx(true)

	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomOther, sdkmath.NewInt(1000))),
		gas: 100000,
	}

	// No trade keeper mocks should be called for ReCheckTx (validation is skipped)
	_, err := decorator.AnteHandle(recheckCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)

	// Verify FeeDenomKey was set in the context passed to the next handler
	feeDenom := suite.nextCalledWith.Value(ante.FeeDenomKey)
	suite.Require().Equal(denomOther, feeDenom)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ReCheckTx_EmptyFee() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// ReCheckTx with empty fee should still pass without setting FeeDenomKey
	recheckCtx := suite.ctx.WithIsReCheckTx(true)

	tx := &mockFeeTx{
		fee: sdk.NewCoins(),
		gas: 100000,
	}

	_, err := decorator.AnteHandle(recheckCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)

	// FeeDenomKey should not be set when fee is empty
	feeDenom := suite.nextCalledWith.Value(ante.FeeDenomKey)
	suite.Require().Nil(feeDenom)
}

// nonFeeTxImpl implements sdk.Tx but not sdk.FeeTx
type nonFeeTxImpl struct{}

func (n nonFeeTxImpl) GetMsgs() []sdk.Msg {
	return nil
}

func (n nonFeeTxImpl) GetMsgsV2() ([]proto.Message, error) {
	return nil, nil
}

func (n nonFeeTxImpl) ValidateBasic() error {
	return nil
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_NotFeeTx() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	_, err := decorator.AnteHandle(suite.ctx, nonFeeTxImpl{}, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "requires tx to be a FeeTx")
	suite.Require().False(suite.nextCalled)
}
