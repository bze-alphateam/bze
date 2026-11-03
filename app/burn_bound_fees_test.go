package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	tradebinkeeper "github.com/bze-alphateam/bze/x/tradebin/keeper"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// A factory token without a community pool has its transfer fee and its gas surcharge sent straight to the burner
// module, as the token itself (the burner burns it as is).
func TestBurnBoundFees_TokenWithoutCommunityPoolGoStraightToBurner(t *testing.T) {
	bApp, ctx, token, pool := transferFeeToken(t)
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomConfig(ctx, token, tokenfactorytypes.DenomConfig{
		TransferFee:  math.LegacyMustNewDecFromStr("0.01"),
		GasSurcharge: math.LegacyOneDec(),
	}))

	burner := authtypes.NewModuleAddress(burnertypes.ModuleName)
	burnerBefore := bApp.BankKeeper.GetBalance(ctx, burner, token).Amount

	// transfer fee on a protocol fee paid in the token
	nativeFee := sdk.NewInt64Coin(pool.Quote, 100_000_000)
	payer := sdk.AccAddress("burn_bound_payer____")
	required := fundTokenPayer(t, bApp, ctx, pool, nativeFee, payer)
	_, err := bApp.TradebinKeeper.CaptureAndSwapUserFee(ctx.WithValue(tradebinkeeper.CtxFeeDenomKey, token), payer, sdk.NewCoins(nativeFee), tradebintypes.ModuleName)
	require.NoError(t, err)
	transferFee := required.Amount.ToLegacyDec().Mul(math.LegacyMustNewDecFromStr("0.01")).Ceil().TruncateInt()
	require.True(t, transferFee.IsPositive())
	require.Equal(t, burnerBefore.Add(transferFee), bApp.BankKeeper.GetBalance(ctx, burner, token).Amount, "the transfer fee reaches the burner module")

	// gas surcharge on a tx fee paid in the token: at 100%, half of the fee in the txfeecollector module
	txFee := sdk.NewCoins(sdk.NewInt64Coin(token, 10_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, tokenfactorytypes.ModuleName, txFee))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToModule(ctx, tokenfactorytypes.ModuleName, txfeecollectortypes.ModuleName, txFee))
	require.NoError(t, bApp.TxfeecollectorKeeper.ChargeGasSurcharge(ctx, payer, txFee))
	require.Equal(t, burnerBefore.Add(transferFee).AddRaw(5_000), bApp.BankKeeper.GetBalance(ctx, burner, token).Amount, "the gas surcharge reaches the burner module")
	require.Equal(t, int64(5_000), bApp.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(txfeecollectortypes.ModuleName), token).Amount.Int64())

}
