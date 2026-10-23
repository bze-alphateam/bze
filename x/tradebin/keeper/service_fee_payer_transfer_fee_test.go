package keeper_test

import (
	"errors"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/testutil"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectormoduletypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

const denomFactoryToken = "factory/bze1creator/token"

var (
	transferFeePayer   = sdk.AccAddress("transfer_fee_payer__")
	tokenCommunityPool = sdk.AccAddress("token_cp____________")
)

func factoryTokenPool() types.LiquidityPool {
	return types.LiquidityPool{
		Id:           denomFactoryToken + "_" + denomBze,
		Base:         denomFactoryToken,
		Quote:        denomBze,
		ReserveBase:  math.NewInt(50_000_000_000),
		ReserveQuote: math.NewInt(100_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
	}
}

// useDenomFeeRates sets a strict provider mock: every expectation must be met and no other call is allowed.
func (suite *IntegrationTestSuite) useDenomFeeRates() *testutil.MockDenomFeeRatesProvider {
	provider := testutil.NewMockDenomFeeRatesProvider(gomock.NewController(suite.T()))
	suite.k.SetDenomFeeRatesProvider(provider)

	return provider
}

func expectTransferFeeRate(provider *testutil.MockDenomFeeRatesProvider, rate string) {
	provider.EXPECT().
		GetDenomFeeRates(gomock.Any(), denomFactoryToken).
		Return(math.LegacyMustNewDecFromStr(rate), math.LegacyZeroDec()).
		Times(1)
}

// requiredTokenFee sets the factory token pool and returns the context paying fees in the token and the token amount
// the native fee converts to.
func (suite *IntegrationTestSuite) requiredTokenFee(nativeFee sdk.Coin) (sdk.Context, sdk.Coin) {
	pool := factoryTokenPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)

	required, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().NoError(err)

	return suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomFactoryToken), required
}

// ceilFee is ceil(amount × rate) of the factory token.
func ceilFee(paid sdk.Coin, rate string) sdk.Coin {
	return sdk.NewCoin(paid.Denom, paid.Amount.ToLegacyDec().Mul(math.LegacyMustNewDecFromStr(rate)).Ceil().TruncateInt())
}

func (suite *IntegrationTestSuite) expectSpendable(coins sdk.Coins) {
	suite.bankMock.EXPECT().SpendableCoins(gomock.Any(), transferFeePayer).Return(coins).Times(1)
}

// The user pays the protocol fee in the token plus ceil(required × rate) on top, sent to txfeecollector_cp for a
// token with a community pool. The capture and the swap are unchanged.
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_TransferFeeToCommunityPoolCollector() {
	provider := suite.useDenomFeeRates()
	nativeFee := sdk.NewInt64Coin(denomBze, 100_000_000)
	ctx, required := suite.requiredTokenFee(nativeFee)
	transferFee := ceilFee(required, "0.01")

	expectTransferFeeRate(provider, "0.01")
	provider.EXPECT().GetDenomCommunityPool(gomock.Any(), denomFactoryToken).Return(tokenCommunityPool, true).Times(1)
	suite.expectSpendable(sdk.NewCoins(required.Add(transferFee)))
	gomock.InOrder(
		suite.bankMock.EXPECT().
			SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, txfeecollectormoduletypes.CpFeeCollector, sdk.NewCoins(transferFee)).
			Return(nil).
			Times(1),
		suite.bankMock.EXPECT().
			SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, types.ModuleName, sdk.NewCoins(required)).
			Return(nil).
			Times(1),
	)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	captured, err := suite.k.CaptureAndSwapUserFee(ctx, transferFeePayer, sdk.NewCoins(nativeFee), types.ModuleName)
	suite.Require().NoError(err)
	suite.Require().Len(captured, 1)
	suite.Require().Equal(denomBze, captured[0].Denom)
}

// A token without a community pool has its transfer fee sent to txfeecollector_burner.
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_TransferFeeWithoutCommunityPoolToBurner() {
	provider := suite.useDenomFeeRates()
	nativeFee := sdk.NewInt64Coin(denomBze, 100_000_000)
	ctx, required := suite.requiredTokenFee(nativeFee)
	transferFee := ceilFee(required, "0.05")

	expectTransferFeeRate(provider, "0.05")
	provider.EXPECT().GetDenomCommunityPool(gomock.Any(), denomFactoryToken).Return(nil, false).Times(1)
	suite.expectSpendable(sdk.NewCoins(required.Add(transferFee)))
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, txfeecollectormoduletypes.BurnerFeeCollector, sdk.NewCoins(transferFee)).
		Return(nil).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, types.ModuleName, sdk.NewCoins(required)).
		Return(nil).
		Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	_, err := suite.k.CaptureAndSwapUserFee(ctx, transferFeePayer, sdk.NewCoins(nativeFee), types.ModuleName)
	suite.Require().NoError(err)
}

// A balance covering the converted fee but not the transfer fee on top takes the existing fallback: the fee is
// captured in native denom and no transfer fee is charged.
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_BalanceShortOfTransferFeeFallsBackToNative() {
	provider := suite.useDenomFeeRates()
	nativeFee := sdk.NewInt64Coin(denomBze, 100_000_000)
	ctx, required := suite.requiredTokenFee(nativeFee)
	transferFee := ceilFee(required, "0.01")

	expectTransferFeeRate(provider, "0.01")
	suite.expectSpendable(sdk.NewCoins(required.Add(transferFee).SubAmount(math.OneInt()), nativeFee))
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, types.ModuleName, sdk.NewCoins(nativeFee)).
		Return(nil).
		Times(1)

	captured, err := suite.k.CaptureAndSwapUserFee(ctx, transferFeePayer, sdk.NewCoins(nativeFee), types.ModuleName)
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(nativeFee), captured)
}

// A zero rate charges nothing and leaves the capture unchanged.
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_ZeroTransferFeeChargesNothing() {
	provider := suite.useDenomFeeRates()
	nativeFee := sdk.NewInt64Coin(denomBze, 100_000_000)
	ctx, required := suite.requiredTokenFee(nativeFee)

	expectTransferFeeRate(provider, "0")
	suite.expectSpendable(sdk.NewCoins(required))
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, types.ModuleName, sdk.NewCoins(required)).
		Return(nil).
		Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	_, err := suite.k.CaptureAndSwapUserFee(ctx, transferFeePayer, sdk.NewCoins(nativeFee), types.ModuleName)
	suite.Require().NoError(err)
}

// Without a provider the keeper charges nothing and behaves as before.
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_NoProviderChargesNothing() {
	nativeFee := sdk.NewInt64Coin(denomBze, 100_000_000)
	ctx, required := suite.requiredTokenFee(nativeFee)

	suite.expectSpendable(sdk.NewCoins(required))
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, types.ModuleName, sdk.NewCoins(required)).
		Return(nil).
		Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	_, err := suite.k.CaptureAndSwapUserFee(ctx, transferFeePayer, sdk.NewCoins(nativeFee), types.ModuleName)
	suite.Require().NoError(err)
}

// A preferred denom that is not a factory token never reads the provider (the strict mock fails on any call).
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_NonFactoryDenomReadsNoRate() {
	suite.useDenomFeeRates()
	nativeFee := sdk.NewInt64Coin(denomBze, 100_000_000)
	pool := trySwapDeepPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)
	required, err := suite.k.CalculateOptimalInputForOutput(pool, nativeFee)
	suite.Require().NoError(err)

	suite.expectSpendable(sdk.NewCoins(required))
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, types.ModuleName, sdk.NewCoins(required)).
		Return(nil).
		Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	ctx := suite.ctx.WithValue(keeper.CtxFeeDenomKey, denomStake)
	_, err = suite.k.CaptureAndSwapUserFee(ctx, transferFeePayer, sdk.NewCoins(nativeFee), types.ModuleName)
	suite.Require().NoError(err)
}

// The order fee path charges the transfer fee too when it swaps.
func (suite *IntegrationTestSuite) TestTrySwap_TransferFeeOnSwapPath() {
	provider := suite.useDenomFeeRates()
	nativeFee := sdk.NewInt64Coin(denomBze, 100_000_000)
	ctx, required := suite.requiredTokenFee(nativeFee)
	transferFee := ceilFee(required, "0.01")

	expectTransferFeeRate(provider, "0.01")
	provider.EXPECT().GetDenomCommunityPool(gomock.Any(), denomFactoryToken).Return(tokenCommunityPool, true).Times(1)
	suite.expectSpendable(sdk.NewCoins(required.Add(transferFee)))
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, txfeecollectormoduletypes.CpFeeCollector, sdk.NewCoins(transferFee)).
		Return(nil).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, types.ModuleName, sdk.NewCoins(required)).
		Return(nil).
		Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, transferFeePayer, sdk.NewCoins(nativeFee), types.ModuleName)
	suite.Require().NoError(err)
	suite.Require().Equal(denomBze, captured[0].Denom)
}

// The order fee path charges the transfer fee too when the amount is too small to swap and the token is taken as is.
// A 1 unit fee at 0.1% still pays 1 unit: the fee is rounded up.
func (suite *IntegrationTestSuite) TestTrySwap_TransferFeeOnTooSmallPath() {
	provider := suite.useDenomFeeRates()
	nativeFee := sdk.NewInt64Coin(denomBze, 1)
	ctx, required := suite.requiredTokenFee(nativeFee)
	transferFee := ceilFee(required, "0.001")
	suite.Require().Equal(int64(1), transferFee.Amount.Int64())

	expectTransferFeeRate(provider, "0.001")
	provider.EXPECT().GetDenomCommunityPool(gomock.Any(), denomFactoryToken).Return(tokenCommunityPool, true).Times(1)
	suite.expectSpendable(sdk.NewCoins(required.Add(transferFee)))
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, txfeecollectormoduletypes.CpFeeCollector, sdk.NewCoins(transferFee)).
		Return(nil).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, types.ModuleName, sdk.NewCoins(required)).
		Return(nil).
		Times(1)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, transferFeePayer, sdk.NewCoins(nativeFee), types.ModuleName)
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(required), captured)
}

// A failed transfer fee send returns its error before anything is captured or swapped (the bank mock fails on the
// capture call, which is not expected).
func (suite *IntegrationTestSuite) TestCaptureAndSwapUserFee_FailedTransferFeeSendCapturesNothing() {
	provider := suite.useDenomFeeRates()
	nativeFee := sdk.NewInt64Coin(denomBze, 100_000_000)
	ctx, required := suite.requiredTokenFee(nativeFee)
	transferFee := ceilFee(required, "0.01")
	sendErr := errors.New("transfer fee send refused")

	expectTransferFeeRate(provider, "0.01")
	provider.EXPECT().GetDenomCommunityPool(gomock.Any(), denomFactoryToken).Return(tokenCommunityPool, true).Times(1)
	suite.expectSpendable(sdk.NewCoins(required.Add(transferFee)))
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, txfeecollectormoduletypes.CpFeeCollector, sdk.NewCoins(transferFee)).
		Return(sendErr).
		Times(1)

	captured, err := suite.k.CaptureAndSwapUserFee(ctx, transferFeePayer, sdk.NewCoins(nativeFee), types.ModuleName)
	suite.Require().ErrorIs(err, sendErr)
	suite.Require().Nil(captured)
}

// The order fee path stops on a failed transfer fee send too.
func (suite *IntegrationTestSuite) TestTrySwap_FailedTransferFeeSendCapturesNothing() {
	provider := suite.useDenomFeeRates()
	nativeFee := sdk.NewInt64Coin(denomBze, 100_000_000)
	ctx, required := suite.requiredTokenFee(nativeFee)
	transferFee := ceilFee(required, "0.01")
	sendErr := errors.New("transfer fee send refused")

	expectTransferFeeRate(provider, "0.01")
	provider.EXPECT().GetDenomCommunityPool(gomock.Any(), denomFactoryToken).Return(nil, false).Times(1)
	suite.expectSpendable(sdk.NewCoins(required.Add(transferFee)))
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), transferFeePayer, txfeecollectormoduletypes.BurnerFeeCollector, sdk.NewCoins(transferFee)).
		Return(sendErr).
		Times(1)

	captured, err := suite.k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, transferFeePayer, sdk.NewCoins(nativeFee), types.ModuleName)
	suite.Require().ErrorIs(err, sendErr)
	suite.Require().Nil(captured)
}
