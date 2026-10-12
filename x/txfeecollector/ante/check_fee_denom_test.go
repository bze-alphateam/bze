package ante_test

import (
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
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

// The three tests below build the fee as a Coins literal on purpose: sdk.NewCoins strips
// zero-amount coins, but a decoded tx does not (AuthInfo.Fee.Amount is used verbatim).
// The SDK tx factory produces exactly such a fee when simulating with `--gas auto` and
// `--gas-prices`: gas is 0 at that point, so the derived fee is a single `0ubze` coin.

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ZeroAmountCoin_Simulation() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// A zero-amount fee coin must be tolerated during simulation, and its denom is validated
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomBze).Return(true).Times(1)

	tx := &mockFeeTx{
		fee: sdk.Coins{sdk.NewCoin(denomBze, sdkmath.NewInt(0))},
		gas: 0,
	}
	suite.Require().False(tx.fee.Empty(), "the literal must keep the zero coin for this test to be meaningful")

	_, err := decorator.AnteHandle(suite.ctx.WithBlockHeight(10), tx, true, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
	suite.Require().Equal(denomBze, suite.nextCalledWith.Value(ante.FeeDenomKey))
}

// gasConsumingTradeMock makes the denom checks consume gas like the real keeper's store reads,
// so a test can compare the gas a simulation consumes with the gas of the real fee.
func (suite *AnteTestSuite) gasConsumingTradeMock(denom string, native, deep bool) {
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denom).
		DoAndReturn(func(ctx sdk.Context, _ string) bool {
			ctx.GasMeter().ConsumeGas(1000, "is native denom")
			return native
		}).Times(1)
	if native {
		return
	}
	suite.tradeMock.EXPECT().HasDeepLiquidityWithNativeDenom(gomock.Any(), denom).
		DoAndReturn(func(ctx sdk.Context, _ string) bool {
			ctx.GasMeter().ConsumeGas(3000, "has deep liquidity")
			return deep
		}).Times(1)
}

// Regression (--gas auto with a non-native fee): the CLI simulates with gas 0, so the fee is
// a zero coin in the denom the tx will pay in. The simulation must run the same denom checks
// as DeliverTx (same gas) and pass the denom on, or the estimate is too low to execute.
func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ZeroAmountCoin_Simulation_NonNative() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)
	ctx := suite.ctx.WithBlockHeight(10)

	// gas of the real fee
	suite.gasConsumingTradeMock(denomUsd, false, true)
	realCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	realTx := &mockFeeTx{fee: sdk.Coins{sdk.NewCoin(denomUsd, sdkmath.NewInt(1000))}, gas: 100000}
	_, err := decorator.AnteHandle(realCtx, realTx, false, suite.mockNext())
	suite.Require().NoError(err)
	realGas := realCtx.GasMeter().GasConsumed()
	suite.Require().Positive(realGas)

	// gas of the simulation with the zero coin
	suite.nextCalled = false
	suite.gasConsumingTradeMock(denomUsd, false, true)
	simCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	simTx := &mockFeeTx{fee: sdk.Coins{sdk.NewCoin(denomUsd, sdkmath.NewInt(0))}, gas: 0}
	_, err = decorator.AnteHandle(simCtx, simTx, true, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
	suite.Require().Equal(realGas, simCtx.GasMeter().GasConsumed())
	suite.Require().Equal(denomUsd, suite.nextCalledWith.Value(ante.FeeDenomKey))
}

// A simulated zero coin in a denom DeliverTx would refuse is refused by the simulation too.
func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ZeroAmountCoin_Simulation_NoLiquidity() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)
	suite.gasConsumingTradeMock(denomOther, false, false)

	tx := &mockFeeTx{fee: sdk.Coins{sdk.NewCoin(denomOther, sdkmath.NewInt(0))}, gas: 0}
	_, err := decorator.AnteHandle(suite.ctx.WithBlockHeight(10), tx, true, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "can be used to pay for fees only if enough liquidity is available")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ZeroAmountCoin_Genesis() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// A zero-amount fee coin must be tolerated at genesis, like an empty fee
	tx := &mockFeeTx{
		fee: sdk.Coins{sdk.NewCoin(denomBze, sdkmath.NewInt(0))},
		gas: 100000,
	}

	_, err := decorator.AnteHandle(suite.ctx.WithBlockHeight(0), tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestValidateTxFeeDenomsDecorator_ZeroAmountCoin() {
	decorator := ante.NewValidateTxFeeDenomsDecorator(suite.tradeMock)

	// Outside genesis and simulation a zero-amount fee coin is still refused, as no fee
	tx := &mockFeeTx{
		fee: sdk.Coins{sdk.NewCoin(denomBze, sdkmath.NewInt(0))},
		gas: 100000,
	}

	_, err := decorator.AnteHandle(suite.ctx.WithBlockHeight(10), tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "no fee supplied")
	suite.Require().False(suite.nextCalled)
}
