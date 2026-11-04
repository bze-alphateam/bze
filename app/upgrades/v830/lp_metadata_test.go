package v830_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
)

// noPools is a tradebin keeper without liquidity pools: the LP metadata backfill touches nothing.
type noPools struct{}

func (noPools) GetAllLiquidityPool(sdk.Context) []tradebintypes.LiquidityPool { return nil }

type fakePools []tradebintypes.LiquidityPool

func (p fakePools) GetAllLiquidityPool(sdk.Context) []tradebintypes.LiquidityPool { return p }

// fakeMetadata is an in-memory bank metadata store that counts writes.
type fakeMetadata struct {
	byDenom map[string]banktypes.Metadata
	writes  int
}

func (f *fakeMetadata) GetDenomMetaData(_ context.Context, denom string) (banktypes.Metadata, bool) {
	md, found := f.byDenom[denom]
	return md, found
}

func (f *fakeMetadata) SetDenomMetaData(_ context.Context, md banktypes.Metadata) {
	f.writes++
	f.byDenom[md.Base] = md
}

// lpMetadata is the metadata pool creation stored before v8.3.0: units, base and display only.
func lpMetadata(base, display string) banktypes.Metadata {
	return banktypes.Metadata{
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: base, Exponent: 0},
			{Denom: display, Exponent: 6},
		},
		Base:    base,
		Display: display,
	}
}

func runLpBackfill(t *testing.T, pools v830.TradebinKeeper, md v830.MetadataKeeper) {
	t.Helper()
	cfg, mm, ctx := guardianSetup(t)
	_, err := v830.CreateUpgradeHandler(cfg, mm, &fakeGuardian{}, acceptingRateLimits(t), emptyBank(t), pools, md)(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{fakeModuleName: 2},
	)
	require.NoError(t, err)
}

const (
	legacyPoolId = "factory/bze1creator/token_ubze"
	hashPoolId   = "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4_ubze"
	finePoolId   = "uatom_ubze"
)

// Both denom schemes on mainnet: pre-hash pools keep "ulp_<id>" / "lp_<id>", newer ones "ulp/<hash>" / "lp/<hash>".
// Every blank entry gets a name, symbol and description and then validates; units, base and display stay as stored.
// An entry that already has a name and symbol is not rewritten.
func TestHandler_BackfillsBlankLpMetadata(t *testing.T) {
	legacy := lpMetadata("ulp_"+legacyPoolId, "lp_"+legacyPoolId)
	hashed := lpMetadata(tradebintypes.GetLpDenom(hashPoolId), tradebintypes.GetLpScaledDenom(hashPoolId))
	fine := lpMetadata(tradebintypes.GetLpDenom(finePoolId), tradebintypes.GetLpScaledDenom(finePoolId))
	fine.Name, fine.Symbol = "ATOM/BZE LP", "ATOMBZE"
	for _, md := range []banktypes.Metadata{legacy, hashed} {
		require.ErrorContains(t, md.Validate(), "name field cannot be blank")
	}

	store := &fakeMetadata{byDenom: map[string]banktypes.Metadata{
		legacy.Base: legacy, hashed.Base: hashed, fine.Base: fine,
	}}
	pools := fakePools{
		{Id: legacyPoolId, LpDenom: legacy.Base},
		{Id: hashPoolId, LpDenom: hashed.Base},
		{Id: finePoolId, LpDenom: fine.Base},
	}

	runLpBackfill(t, pools, store)

	require.Equal(t, 2, store.writes)
	for _, pool := range pools[:2] {
		got := store.byDenom[pool.LpDenom]
		require.NoError(t, got.Validate())
		require.Equal(t, "LP share of pool "+pool.Id, got.Name)
		require.Equal(t, wantLpSymbol(pool.Id), got.Symbol)
		require.Equal(t, tradebintypes.LpDenomDescription(pool.Id), got.Description)
	}
	for _, before := range []banktypes.Metadata{legacy, hashed} {
		got := store.byDenom[before.Base]
		require.Equal(t, before.DenomUnits, got.DenomUnits)
		require.Equal(t, before.Base, got.Base)
		require.Equal(t, before.Display, got.Display)
	}
	require.Equal(t, fine, store.byDenom[fine.Base])
	// each pool's shares get their own symbol
	require.NotEqual(t, store.byDenom[legacy.Base].Symbol, store.byDenom[hashed.Base].Symbol)
}

// Only the blank fields are filled: a stored name or description is kept.
func TestHandler_LpBackfillKeepsSetFields(t *testing.T) {
	md := lpMetadata("ulp_"+legacyPoolId, "lp_"+legacyPoolId)
	md.Name, md.Description = "Custom name", "Custom description"
	store := &fakeMetadata{byDenom: map[string]banktypes.Metadata{md.Base: md}}

	runLpBackfill(t, fakePools{{Id: legacyPoolId, LpDenom: md.Base}}, store)

	got := store.byDenom[md.Base]
	require.NoError(t, got.Validate())
	require.Equal(t, "Custom name", got.Name)
	require.Equal(t, "Custom description", got.Description)
	require.Equal(t, wantLpSymbol(legacyPoolId), got.Symbol)
}

// A pool whose LP denom has no metadata gets none invented, and metadata that would still be invalid after the fill
// is left as stored; neither fails the upgrade.
func TestHandler_LpBackfillSkipsMissingAndInvalidMetadata(t *testing.T) {
	broken := lpMetadata("ulp_"+legacyPoolId, "lp_"+legacyPoolId)
	broken.Display = "not_a_unit"
	store := &fakeMetadata{byDenom: map[string]banktypes.Metadata{broken.Base: broken}}
	pools := fakePools{
		{Id: legacyPoolId, LpDenom: broken.Base},
		{Id: hashPoolId, LpDenom: tradebintypes.GetLpDenom(hashPoolId)},
	}

	runLpBackfill(t, pools, store)

	require.Zero(t, store.writes)
	require.Equal(t, broken, store.byDenom[broken.Base])
	_, found := store.byDenom[tradebintypes.GetLpDenom(hashPoolId)]
	require.False(t, found)
}

// wantLpSymbol is the expected LP share symbol, derived here independently of the code under test.
func wantLpSymbol(poolId string) string {
	sum := sha256.Sum256([]byte(poolId))
	return "LP-" + strings.ToUpper(hex.EncodeToString(sum[:]))[:6]
}
