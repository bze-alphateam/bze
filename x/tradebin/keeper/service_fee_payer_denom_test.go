package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

var (
	protocolFeePayer = sdk.AccAddress("protocol_fee_payer__")
	txFeePayer       = sdk.AccAddress("tx_fee_payer________")
)

type captureFee func(ctx sdk.Context, payer sdk.AccAddress, fee sdk.Coins, toModule string) (sdk.Coins, error)

func (suite *IntegrationTestSuite) feeEntryPoints() map[string]captureFee {
	return map[string]captureFee{
		"CaptureAndSwapUserFee":                   suite.k.CaptureAndSwapUserFee,
		"CaptureAndTryToSwapUserFeesOrSendItAsIs": suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs,
	}
}

// requireNativeFeeOnly captures the native fee with ctx through both entry points and checks that nothing but the
// native fee moved: the bank mock has no SpendableCoins expectation, so an attempt to pay in the preferred denom fails.
func (suite *IntegrationTestSuite) requireNativeFeeOnly(ctx sdk.Context, payer sdk.AccAddress) {
	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)
	fee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(100_000_000)))

	for name, capture := range suite.feeEntryPoints() {
		suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), payer, types.ModuleName, fee).Return(nil).Times(1)

		captured, err := capture(ctx, payer, fee, types.ModuleName)
		suite.Require().NoError(err, name)
		suite.Require().Equal(fee, captured, name)

		stored, found := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
		suite.Require().True(found, name)
		suite.Require().Equal(pool.ReserveBase, stored.ReserveBase, name)
		suite.Require().Equal(pool.ReserveQuote, stored.ReserveQuote, name)
	}
}

// The account paying the tx fee keeps paying protocol fees in the tx fee denom.
func (suite *IntegrationTestSuite) TestFeePayerDenom_TxFeePayerUsesPreferredDenom() {
	pool := trySwapDeepPool()
	nativeFee := sdk.NewCoin(denomBze, math.NewInt(100_000_000))
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake).WithValue(keeper.CtxFeePayerKey, txFeePayer)

	for name, capture := range suite.feeEntryPoints() {
		suite.k.SetLiquidityPool(suite.ctx, pool)
		required, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
		suite.Require().NoError(err, name)

		suite.bankMock.EXPECT().SpendableCoins(gomock.Any(), txFeePayer).
			Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(50_000_000_000)))).Times(1)
		suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), txFeePayer, types.ModuleName, sdk.NewCoins(required)).
			Return(nil).Times(1)
		suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).AnyTimes()

		captured, err := capture(ctx, txFeePayer, sdk.NewCoins(nativeFee), types.ModuleName)
		suite.Require().NoError(err, name)
		suite.Require().Len(captured, 1, name)
		suite.Require().Equal(denomBze, captured[0].Denom, name)

		stored, found := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
		suite.Require().True(found, name)
		suite.Require().True(stored.ReserveBase.GT(pool.ReserveBase), name)
	}
}

// Another account charged in the same tx pays in native: the tx fee denom was not its choice.
func (suite *IntegrationTestSuite) TestFeePayerDenom_OtherPayerPaysNative() {
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake).WithValue(keeper.CtxFeePayerKey, txFeePayer)

	suite.requireNativeFeeOnly(ctx, protocolFeePayer)
}

// A preferred denom with no fee payer next to it speaks for nobody.
func (suite *IntegrationTestSuite) TestFeePayerDenom_NoStoredFeePayerPaysNative() {
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)

	suite.requireNativeFeeOnly(ctx, protocolFeePayer)
}

// An empty stored fee payer matches nobody, not even an empty payer.
func (suite *IntegrationTestSuite) TestFeePayerDenom_EmptyStoredFeePayerPaysNative() {
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake).WithValue(keeper.CtxFeePayerKey, sdk.AccAddress{})

	suite.requireNativeFeeOnly(ctx, sdk.AccAddress{})
}

// A stored fee payer of an unexpected type is ignored.
func (suite *IntegrationTestSuite) TestFeePayerDenom_MalformedStoredFeePayerPaysNative() {
	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake).WithValue(keeper.CtxFeePayerKey, protocolFeePayer.String())

	suite.requireNativeFeeOnly(ctx, protocolFeePayer)
}

// Without a preferred denom the fee stays native, whoever pays the tx fee.
func (suite *IntegrationTestSuite) TestFeePayerDenom_NoPreferredDenomPaysNative() {
	ctx := suite.ctx.WithValue(keeper.CtxFeePayerKey, protocolFeePayer)

	suite.requireNativeFeeOnly(ctx, protocolFeePayer)
}
