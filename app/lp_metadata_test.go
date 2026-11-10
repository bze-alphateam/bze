package app_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

// requireBankGenesisValid is what `bzed genesis validate` checks for the bank module on an export.
func requireBankGenesisValid(t *testing.T, bApp *app.App, ctx sdk.Context) {
	t.Helper()
	require.NoError(t, bApp.BankKeeper.ExportGenesis(ctx).Validate())
}

// A pool created through the real message router stores LP metadata that the bank module accepts, so an export of a
// chain with pools passes genesis validation.
func TestLpMetadata_NewPoolExportsValidBankGenesis(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)
	require.NoError(t, bApp.TradebinKeeper.SetParams(ctx, v2types.DefaultParams()))
	creator := sdk.AccAddress("lp_pool_creator_____")
	funds := sdk.NewCoins(sdk.NewInt64Coin("ubze", 100_000_000_000), sdk.NewInt64Coin("uquote", 1_000_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, funds))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, creator, funds))

	require.NoError(t, handle(bApp, ctx, &tradebintypes.MsgCreateLiquidityPool{
		Creator:      creator.String(),
		Base:         "ubze",
		Quote:        "uquote",
		Fee:          "0.003",
		FeeDest:      `{"treasury":"0.25","burner":"0.25","providers":"0.5"}`,
		InitialBase:  math.NewInt(10_000_000),
		InitialQuote: math.NewInt(100_000),
	}))

	pools := bApp.TradebinKeeper.GetAllLiquidityPool(ctx)
	require.Len(t, pools, 1)
	md, found := bApp.BankKeeper.GetDenomMetaData(ctx, pools[0].GetLpDenom())
	require.True(t, found)
	require.NoError(t, md.Validate())
	require.Equal(t, "LP share of pool "+pools[0].GetId(), md.Name)
	require.Equal(t, wantLpSymbol(pools[0].GetId()), md.Symbol)
	requireBankGenesisValid(t, bApp, ctx)
}

// Pools created before v8.3.0 left their LP metadata without a name or symbol, which made exported genesis fail
// validation. With the real bank and tradebin keepers: the upgrade fills the blank entries of both denom schemes,
// keeps an entry that was already valid, and the exported bank genesis validates afterwards.
func TestLpMetadata_UpgradeBackfillsBlankEntries(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)

	const (
		legacyId = "factory/bze1creator/token_ubze"
		hashId   = "ubze_uquote"
		fineId   = "uatom_ubze"
	)
	blank := func(base, display string) banktypes.Metadata {
		return banktypes.Metadata{
			DenomUnits: []*banktypes.DenomUnit{{Denom: base, Exponent: 0}, {Denom: display, Exponent: 6}},
			Base:       base,
			Display:    display,
		}
	}
	legacy := blank("ulp_"+legacyId, "lp_"+legacyId)
	hashed := blank(tradebintypes.GetLpDenom(hashId), tradebintypes.GetLpScaledDenom(hashId))
	fine := blank(tradebintypes.GetLpDenom(fineId), tradebintypes.GetLpScaledDenom(fineId))
	fine.Name, fine.Symbol, fine.Description = "ATOM/BZE LP", "ATOMBZE", "kept as is"
	for id, md := range map[string]banktypes.Metadata{legacyId: legacy, hashId: hashed, fineId: fine} {
		bApp.BankKeeper.SetDenomMetaData(ctx, md)
		bApp.TradebinKeeper.SetLiquidityPool(ctx, tradebintypes.LiquidityPool{Id: id, LpDenom: md.Base})
	}
	require.ErrorContains(t, bApp.BankKeeper.ExportGenesis(ctx).Validate(), "name field cannot be blank")

	_, err := v830.CreateUpgradeHandler(bApp.Configurator(), bApp.ModuleManager, bApp.GuardianKeeper, bApp.RatelimitKeeper, bApp.BankKeeper, bApp.TradebinKeeper, bApp.BankKeeper, bApp.RewardsKeeper, runtime.NewKVStoreService(bApp.GetKey(ibctransfertypes.StoreKey)), bApp.TransferKeeper)(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, bApp.ModuleManager.GetVersionMap(),
	)
	require.NoError(t, err)

	for id, before := range map[string]banktypes.Metadata{legacyId: legacy, hashId: hashed} {
		got, found := bApp.BankKeeper.GetDenomMetaData(ctx, before.Base)
		require.True(t, found)
		require.NoError(t, got.Validate(), id)
		require.Equal(t, "LP share of pool "+id, got.Name)
		require.Equal(t, wantLpSymbol(id), got.Symbol)
		require.Equal(t, before.DenomUnits, got.DenomUnits)
		require.Equal(t, before.Display, got.Display)
	}
	got, _ := bApp.BankKeeper.GetDenomMetaData(ctx, fine.Base)
	require.Equal(t, fine, got)
	requireBankGenesisValid(t, bApp, ctx)
}

// wantLpSymbol is the expected LP share symbol, derived here independently of the code under test.
func wantLpSymbol(poolId string) string {
	sum := sha256.Sum256([]byte(poolId))
	return "LP-" + strings.ToUpper(hex.EncodeToString(sum[:]))[:6]
}
