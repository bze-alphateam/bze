package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"go.uber.org/mock/gomock"
)

// A factory token with a transfer fee: the payer holds exactly the converted fee, so the spendable check passes but
// x/bank's send restriction refuses the capture (fee on top). The fee must be paid in native denom instead of failing
// the tx. Regression test for BZE-220: the old code returned the capture error.
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_TransferFeeOnPreferredDenom_FallsBackToNative() {
	payer := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100_000_000))
	fee := sdk.NewCoins(nativeFee)

	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	required, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().NoError(err)

	// exactly the converted fee in the preferred denom, plus some BZE
	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), payer).
		Return(sdk.NewCoins(required, sdk.NewCoin(denomBze, math.NewInt(500_000_000)))).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), payer, types.ModuleName, sdk.NewCoins(required)).
		Return(sdkerrors.ErrInsufficientFunds.Wrap("spendable balance is smaller than the amount plus the transfer fee")).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), payer, types.ModuleName, fee).
		Return(nil).
		Times(1)

	captured, err := suite.k.CaptureAndSwapUserFee(ctx, payer, fee, types.ModuleName)
	suite.Require().NoError(err)
	suite.Require().Equal(fee, captured)

	// nothing was swapped
	stored, found := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
	suite.Require().True(found)
	suite.Require().Equal(pool.ReserveBase, stored.ReserveBase)
	suite.Require().Equal(pool.ReserveQuote, stored.ReserveQuote)
}

// The same capture failure in the try-swap variant also falls back to the native fee.
func (suite *IntegrationTestSuite) TestTrySwap_TransferFeeOnPreferredDenom_FallsBackToNative() {
	payer := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100_000_000))
	fee := sdk.NewCoins(nativeFee)

	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	required, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().NoError(err)

	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), payer).
		Return(sdk.NewCoins(required, sdk.NewCoin(denomBze, math.NewInt(500_000_000)))).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), payer, types.ModuleName, sdk.NewCoins(required)).
		Return(sdkerrors.ErrInsufficientFunds).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), payer, types.ModuleName, fee).
		Return(nil).
		Times(1)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, payer, fee, types.ModuleName)
	suite.Require().NoError(err)
	suite.Require().Equal(fee, captured)
}

// When neither denom can be captured the error is returned.
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_PreferredAndNativeCaptureFail() {
	payer := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100_000_000))
	fee := sdk.NewCoins(nativeFee)

	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	required, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().NoError(err)

	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), payer).
		Return(sdk.NewCoins(required)).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), payer, types.ModuleName, sdk.NewCoins(required)).
		Return(sdkerrors.ErrInsufficientFunds).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), payer, types.ModuleName, fee).
		Return(fmt.Errorf("no native balance")).
		Times(1)

	_, err = suite.k.CaptureAndSwapUserFee(ctx, payer, fee, types.ModuleName)
	suite.Require().ErrorContains(err, "no native balance")
}

// The preferred-denom capture and swap run in a cached context: a failure after the swap returns the error and
// leaves the pool as it was.
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_FailureAfterSwap_WritesNothing() {
	payer := sdk.AccAddress("addr1_______________")
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100_000_000))
	fee := sdk.NewCoins(nativeFee)
	targetModule := "rewards"

	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	required, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().NoError(err)

	suite.bankMock.EXPECT().
		SpendableCoins(gomock.Any(), payer).
		Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000)))).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), payer, types.ModuleName, sdk.NewCoins(required)).
		Return(nil).
		Times(1)
	// the send of the swapped fee to the caller module fails
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, targetModule, gomock.Any()).
		Return(fmt.Errorf("send to caller failed")).
		Times(1)
	// the swap's own fee distribution
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	_, err = suite.k.CaptureAndSwapUserFee(ctx, payer, fee, targetModule)
	suite.Require().ErrorContains(err, "send to caller failed")

	stored, found := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
	suite.Require().True(found)
	suite.Require().Equal(pool.ReserveBase, stored.ReserveBase)
	suite.Require().Equal(pool.ReserveQuote, stored.ReserveQuote)
}
