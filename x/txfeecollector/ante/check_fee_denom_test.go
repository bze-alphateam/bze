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

	// Empty fee should fail
	tx := &mockFeeTx{
		fee: sdk.NewCoins(),
		gas: 100000,
	}

	_, err := decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "no fee supplied")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ZeroFee() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Zero fee coins are filtered out by sdk.NewCoins, resulting in empty fee
	// So this test actually validates the empty fee scenario
	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(0))),
		gas: 100000,
	}

	_, err := decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "no fee supplied")
	suite.Require().False(suite.nextCalled)
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

	// ReCheckTx should bypass validation and call next handler directly
	recheckCtx := suite.ctx.WithIsReCheckTx(true)

	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomOther, sdkmath.NewInt(1000))),
		gas: 100000,
	}

	// No mocks should be called for ReCheckTx
	_, err := decorator.AnteHandle(recheckCtx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
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
