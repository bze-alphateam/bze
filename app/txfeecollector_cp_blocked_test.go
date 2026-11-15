package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	tradebinkeeper "github.com/bze-alphateam/bze/x/tradebin/keeper"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// storeBlockedCommunityPool stores a config whose community pool is a blocked (module) address, the state a denom is
// left in when its community pool becomes blocked after it was set: MsgSetDenomConfig refuses a blocked address, the
// keeper store doesn't.
func storeBlockedCommunityPool(t *testing.T, bApp *app.App, ctx sdk.Context, denom, transferFee, gasSurcharge string) sdk.AccAddress {
	t.Helper()
	blocked := authtypes.NewModuleAddress(txfeecollectortypes.ModuleName)
	require.True(t, bApp.BankKeeper.BlockedAddr(blocked), "test setup: the community pool address is blocked")
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomConfig(ctx, denom, tokenfactorytypes.DenomConfig{
		CommunityPool: blocked.String(),
		TransferFee:   math.LegacyMustNewDecFromStr(transferFee),
		GasSurcharge:  math.LegacyMustNewDecFromStr(gasSurcharge),
	}))

	return blocked
}

// A token whose stored community pool is blocked routes as a token without one, and the EndBlock pass goes on: another
// token's coins in the same pass still reach its community pool. The blocked token has no pool to be swapped through,
// so it waits in the collector like any token without a community pool; nothing is burned or sent to the blocked
// address.
func TestTxfeecollector_BlockedCommunityPoolRoutesAsNone(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	admin := sdk.AccAddress("token_admin_________")
	treasury := sdk.AccAddress("token_treasury______")
	blockedToken := "factory/" + admin.String() + "/blocked"
	healthyToken := "factory/" + admin.String() + "/healthy"
	for _, denom := range []string{blockedToken, healthyToken} {
		require.NoError(t, bApp.TokenfactoryKeeper.SetDenomAuthority(ctx, denom, tokenfactorytypes.DenomAuthority{Admin: admin.String(), Mintable: true}))
	}
	blocked := storeBlockedCommunityPool(t, bApp, ctx, blockedToken, "0", "0")
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomConfig(ctx, healthyToken, tokenfactorytypes.DenomConfig{
		CommunityPool: treasury.String(),
		TransferFee:   math.LegacyZeroDec(),
		GasSurcharge:  math.LegacyZeroDec(),
	}))

	_, ok := bApp.TokenfactoryKeeper.GetDenomCommunityPool(ctx, blockedToken)
	require.False(t, ok, "a blocked community pool reads as none")

	blockedCoin := sdk.NewInt64Coin(blockedToken, 1_000)
	healthyCoin := sdk.NewInt64Coin(healthyToken, 2_000)
	mintToModule(t, bApp, ctx, txfeecollectortypes.CpFeeCollector, sdk.NewCoins(blockedCoin, healthyCoin))

	txfeecollectorEndBlock(t, bApp, ctx)

	require.Equal(t, sdk.NewCoins(healthyCoin), bApp.BankKeeper.GetAllBalances(ctx, treasury), "the healthy token reaches its community pool")
	cpCollector := authtypes.NewModuleAddress(txfeecollectortypes.CpFeeCollector)
	require.Equal(t, sdk.NewCoins(blockedCoin), bApp.BankKeeper.GetAllBalances(ctx, cpCollector), "the blocked token waits like one without a community pool")
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(burnertypes.ModuleName)).IsZero(), "nothing burned")
	require.True(t, bApp.BankKeeper.GetBalance(ctx, blocked, blockedToken).IsZero(), "nothing reached the blocked address")
}

// The gas surcharge of a token whose stored community pool is blocked goes where a token without one sends it: the
// burner module, not txfeecollector_cp.
func TestTxfeecollector_GasSurchargeWithBlockedCommunityPoolGoesToBurner(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	admin := sdk.AccAddress("token_admin_________")
	token := "factory/" + admin.String() + "/blocked"
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomAuthority(ctx, token, tokenfactorytypes.DenomAuthority{Admin: admin.String(), Mintable: true}))
	storeBlockedCommunityPool(t, bApp, ctx, token, "0", "1")

	fee := sdk.NewInt64Coin(token, 1_000)
	mintToModule(t, bApp, ctx, txfeecollectortypes.ModuleName, sdk.NewCoins(fee))
	require.NoError(t, bApp.TxfeecollectorKeeper.ChargeGasSurcharge(ctx, sdk.AccAddress("gas_payer___________"), sdk.NewCoins(fee)))

	// surcharge rate 1: floor(1000 × 1 / 2) = 500
	burner := authtypes.NewModuleAddress(burnertypes.ModuleName)
	require.Equal(t, math.NewInt(500), bApp.BankKeeper.GetBalance(ctx, burner, token).Amount, "the surcharge goes to the burner")
	cpCollector := authtypes.NewModuleAddress(txfeecollectortypes.CpFeeCollector)
	require.True(t, bApp.BankKeeper.GetBalance(ctx, cpCollector, token).IsZero(), "nothing goes to txfeecollector_cp")
}

// A protocol fee paid in a token whose stored community pool is blocked pays the transfer fee where a token without
// one sends it: the burner module, not txfeecollector_cp.
func TestTradebin_TransferFeeWithBlockedCommunityPoolGoesToBurner(t *testing.T) {
	bApp, ctx, token, pool := transferFeeToken(t)
	storeBlockedCommunityPool(t, bApp, ctx, token, "0.01", "0")

	nativeFee := sdk.NewInt64Coin(pool.Quote, 100_000_000)
	payer := sdk.AccAddress("protocol_fee_payer__")
	required := fundTokenPayer(t, bApp, ctx, pool, nativeFee, payer)

	feeCtx := ctx.WithValue(tradebinkeeper.CtxFeeDenomKey, token).WithValue(tradebinkeeper.CtxFeePayerKey, payer)
	_, err := bApp.TradebinKeeper.CaptureAndSwapUserFee(feeCtx, payer, sdk.NewCoins(nativeFee), tradebintypes.ModuleName)
	require.NoError(t, err)

	transferFee := required.Amount.ToLegacyDec().Mul(math.LegacyMustNewDecFromStr("0.01")).Ceil().TruncateInt()
	burner := authtypes.NewModuleAddress(burnertypes.ModuleName)
	require.Equal(t, transferFee, bApp.BankKeeper.GetBalance(ctx, burner, token).Amount, "the transfer fee goes to the burner")
	cpCollector := authtypes.NewModuleAddress(txfeecollectortypes.CpFeeCollector)
	require.True(t, bApp.BankKeeper.GetBalance(ctx, cpCollector, token).IsZero(), "nothing goes to txfeecollector_cp")
}
