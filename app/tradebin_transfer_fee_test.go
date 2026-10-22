package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	tradebinkeeper "github.com/bze-alphateam/bze/x/tradebin/keeper"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// The app wires the tokenfactory keeper as tradebin's denom fee rates provider: a protocol fee paid in a factory token
// with a transfer fee pays ceil(fee in the token × rate) on top, into txfeecollector_cp.
func TestTradebin_ProtocolFeeInFactoryTokenPaysTransferFee(t *testing.T) {
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

	nativeFee := sdk.NewInt64Coin(nativeDenom, 100_000_000)
	required, err := bApp.TradebinKeeper.CalculateOptimalInputForOutput(pool, nativeFee)
	require.NoError(t, err)

	payer := sdk.AccAddress("protocol_fee_payer__")
	userTokens := sdk.NewCoins(sdk.NewCoin(token, required.Amount.MulRaw(2)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, tokenfactorytypes.ModuleName, userTokens))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, tokenfactorytypes.ModuleName, payer, userTokens))

	feeCtx := ctx.WithValue(tradebinkeeper.CtxFeeDenomKey, token)
	captured, err := bApp.TradebinKeeper.CaptureAndSwapUserFee(feeCtx, payer, sdk.NewCoins(nativeFee), tradebintypes.ModuleName)
	require.NoError(t, err)
	require.Equal(t, nativeDenom, captured[0].Denom)

	transferFee := required.Amount.ToLegacyDec().Mul(math.LegacyMustNewDecFromStr("0.01")).Ceil().TruncateInt()
	cpCollector := authtypes.NewModuleAddress(txfeecollectortypes.CpFeeCollector)
	require.Equal(t, transferFee, bApp.BankKeeper.GetBalance(ctx, cpCollector, token).Amount, "the transfer fee reaches txfeecollector_cp")
	require.Equal(t, required.Amount.MulRaw(2).Sub(required.Amount).Sub(transferFee), bApp.BankKeeper.GetBalance(ctx, payer, token).Amount)
}
