package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	tradebinkeeper "github.com/bze-alphateam/bze/x/tradebin/keeper"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// transferFeeToken sets up a factory token with a community pool, a 1% transfer fee and a deep pool against the
// native denom, so protocol fees can be paid in it.
func transferFeeToken(t *testing.T) (*app.App, sdk.Context, string, tradebintypes.LiquidityPool) {
	t.Helper()
	bApp, ctx, admin, treasury := newAdminDaoApp(t)

	token := "factory/" + admin.String() + "/fee"
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomAuthority(ctx, token, tokenfactorytypes.DenomAuthority{Admin: admin.String(), Mintable: true}))
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomConfig(ctx, token, tokenfactorytypes.DenomConfig{
		CommunityPool: treasury.String(),
		TransferFee:   math.LegacyMustNewDecFromStr("0.01"),
		GasSurcharge:  math.LegacyZeroDec(),
	}))

	nativeDenom := bApp.TradebinKeeper.GetParams(ctx).NativeDenom
	reserves := sdk.NewCoins(sdk.NewInt64Coin(token, 200_000_000_000), sdk.NewInt64Coin(nativeDenom, 200_000_000_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, tokenfactorytypes.ModuleName, reserves))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToModule(ctx, tokenfactorytypes.ModuleName, tradebintypes.ModuleName, reserves))
	pool := tradebintypes.LiquidityPool{
		Id:           token + "_" + nativeDenom,
		Base:         token,
		Quote:        nativeDenom,
		LpDenom:      "ulp_fee",
		ReserveBase:  reserves.AmountOf(token),
		ReserveQuote: reserves.AmountOf(nativeDenom),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &tradebintypes.FeeDestination{
			Treasury:  math.LegacyZeroDec(),
			Burner:    math.LegacyZeroDec(),
			Providers: math.LegacyOneDec(),
		},
	}
	bApp.TradebinKeeper.SetLiquidityPool(ctx, pool)

	return bApp, ctx, token, pool
}

// fundTokenPayer gives the payer twice the token amount the native fee converts to and returns that amount.
func fundTokenPayer(t *testing.T, bApp *app.App, ctx sdk.Context, pool tradebintypes.LiquidityPool, nativeFee sdk.Coin, payer sdk.AccAddress) sdk.Coin {
	t.Helper()
	required, err := bApp.TradebinKeeper.CalculateOptimalInputForOutput(pool, nativeFee)
	require.NoError(t, err)

	userTokens := sdk.NewCoins(sdk.NewCoin(pool.Base, required.Amount.MulRaw(2)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, tokenfactorytypes.ModuleName, userTokens))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, tokenfactorytypes.ModuleName, payer, userTokens))

	return required
}

// requireTransferFeePaid checks that ceil(required × 1%) reached txfeecollector_cp and left the payer.
func requireTransferFeePaid(t *testing.T, bApp *app.App, ctx sdk.Context, payer sdk.AccAddress, required sdk.Coin) {
	t.Helper()
	transferFee := required.Amount.ToLegacyDec().Mul(math.LegacyMustNewDecFromStr("0.01")).Ceil().TruncateInt()
	cpCollector := authtypes.NewModuleAddress(txfeecollectortypes.CpFeeCollector)
	require.Equal(t, transferFee, bApp.BankKeeper.GetBalance(ctx, cpCollector, required.Denom).Amount, "the transfer fee reaches txfeecollector_cp")
	require.Equal(t, required.Amount.MulRaw(2).Sub(required.Amount).Sub(transferFee), bApp.BankKeeper.GetBalance(ctx, payer, required.Denom).Amount)
}

// The app wires the tokenfactory keeper as tradebin's denom fee rates provider: a protocol fee paid in a factory token
// with a transfer fee pays ceil(fee in the token × rate) on top, into txfeecollector_cp.
func TestTradebin_ProtocolFeeInFactoryTokenPaysTransferFee(t *testing.T) {
	bApp, ctx, token, pool := transferFeeToken(t)

	nativeFee := sdk.NewInt64Coin(pool.Quote, 100_000_000)
	payer := sdk.AccAddress("protocol_fee_payer__")
	required := fundTokenPayer(t, bApp, ctx, pool, nativeFee, payer)

	feeCtx := ctx.WithValue(tradebinkeeper.CtxFeeDenomKey, token)
	captured, err := bApp.TradebinKeeper.CaptureAndSwapUserFee(feeCtx, payer, sdk.NewCoins(nativeFee), tradebintypes.ModuleName)
	require.NoError(t, err)
	require.Equal(t, pool.Quote, captured[0].Denom)

	requireTransferFeePaid(t, bApp, ctx, payer, required)
}

// The tokenfactory msg server registered on the app router sees the same tradebin keeper: a create-denom fee paid in
// the token through the real message handler pays the transfer fee.
func TestTokenfactory_CreateDenomFeeInFactoryTokenPaysTransferFee(t *testing.T) {
	bApp, ctx, token, pool := transferFeeToken(t)

	createDenomFee := sdk.NewInt64Coin(pool.Quote, tokenfactorytypes.DefaultCreateDenomFee)
	require.NoError(t, bApp.TokenfactoryKeeper.SetParams(ctx, tokenfactorytypes.NewParams(createDenomFee)))
	creator := sdk.AccAddress("create_denom_payer__")
	required := fundTokenPayer(t, bApp, ctx, pool, createDenomFee, creator)

	msg := &tokenfactorytypes.MsgCreateDenom{Creator: creator.String(), Subdenom: "paidintoken"}
	handler := bApp.MsgServiceRouter().Handler(msg)
	require.NotNil(t, handler)
	_, err := handler(ctx.WithValue(tradebinkeeper.CtxFeeDenomKey, token), msg)
	require.NoError(t, err)

	requireTransferFeePaid(t, bApp, ctx, creator, required)
}
