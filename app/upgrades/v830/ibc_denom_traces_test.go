package v830_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/runtime"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/bze-alphateam/bze/app"
	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
)

// noTraces is a transfer store with no denom traces, for the handler tests that do not exercise them.
type noTraces struct{}

func (noTraces) OpenKVStore(context.Context) corestore.KVStore { return emptyKVStore{} }

type emptyKVStore struct{}

func (emptyKVStore) Get([]byte) ([]byte, error) { return nil, nil }
func (emptyKVStore) Has([]byte) (bool, error)   { return false, nil }
func (emptyKVStore) Set([]byte, []byte) error   { return nil }
func (emptyKVStore) Delete([]byte) error        { return nil }
func (emptyKVStore) Iterator([]byte, []byte) (corestore.Iterator, error) {
	return emptyIterator{}, nil
}
func (emptyKVStore) ReverseIterator([]byte, []byte) (corestore.Iterator, error) {
	return emptyIterator{}, nil
}

type emptyIterator struct{}

func (emptyIterator) Domain() ([]byte, []byte) { return nil, nil }
func (emptyIterator) Valid() bool              { return false }
func (emptyIterator) Next()                    {}
func (emptyIterator) Key() []byte              { return nil }
func (emptyIterator) Value() []byte            { return nil }
func (emptyIterator) Error() error             { return nil }
func (emptyIterator) Close() error             { return nil }

// v8Trace is a denom trace as ibc-go v8 stored it: the port/channel path and the base denom.
type v8Trace struct {
	path string
	base string
}

func (tr v8Trace) fullPath() string {
	if tr.path == "" {
		return tr.base
	}

	return tr.path + "/" + tr.base
}

func (tr v8Trace) hash() []byte {
	h := sha256.Sum256([]byte(tr.fullPath()))

	return h[:]
}

// encode is the protobuf encoding of ibc.applications.transfer.v1.DenomTrace (path = 1, base_denom = 2).
func (tr v8Trace) encode() []byte {
	var bz []byte
	if tr.path != "" {
		bz = protowire.AppendTag(bz, 1, protowire.BytesType)
		bz = protowire.AppendString(bz, tr.path)
	}
	if tr.base != "" {
		bz = protowire.AppendTag(bz, 2, protowire.BytesType)
		bz = protowire.AppendString(bz, tr.base)
	}

	return bz
}

func traceStoreKey(hash []byte) []byte {
	return append(append([]byte{}, transfertypes.DenomTraceKey...), hash...)
}

// Traces in the shapes ibc-go v8 stores for vouchers received over a transfer channel, which the
// transfer store migration converts on its own.
var validTraces = []v8Trace{
	{path: "transfer/channel-7", base: "uatom"},
	{path: "transfer/channel-7", base: "gamm/pool/1"},
	{path: "transfer/channel-7", base: "factory/bze1qqqqqq/token"},
	{path: "transfer/channel-7", base: "xx/abcdefg-0/tok"},
	{path: "transfer/channel-7/transfer/channel-3", base: "uosmo"},
}

// convertedTrace is a trace the migration cannot parse on its own and the Denom the handler stores for it.
type convertedTrace struct {
	name  string
	trace v8Trace
	want  transfertypes.Denom
}

var convertedTraces = []convertedTrace{
	{
		name:  "base with a short client-shaped segment",
		trace: v8Trace{path: "transfer/channel-7", base: "ab/foo-1/x"},
		want:  transfertypes.NewDenom("ab/foo-1/x", transfertypes.NewHop("transfer", "channel-7")),
	},
	{
		name:  "base ending in a short client-shaped segment",
		trace: v8Trace{path: "transfer/channel-7", base: "ab/foo-1"},
		want:  transfertypes.NewDenom("ab/foo-1", transfertypes.NewHop("transfer", "channel-7")),
	},
	{
		name:  "base ending in a client-shaped segment",
		trace: v8Trace{path: "transfer/channel-7", base: "xx/abcdefg-0"},
		want:  transfertypes.NewDenom("xx/abcdefg-0", transfertypes.NewHop("transfer", "channel-7")),
	},
	{
		name:  "base with a one-character segment before a client-shaped segment",
		trace: v8Trace{path: "transfer/channel-7", base: "x/aaa-0/tok"},
		want:  transfertypes.NewDenom("x/aaa-0/tok", transfertypes.NewHop("transfer", "channel-7")),
	},
	{
		name:  "blank base after port/channel pairs",
		trace: v8Trace{path: "transfer/channel-7/xx/channel-0", base: ""},
		want:  transfertypes.NewDenom("xx/channel-0/", transfertypes.NewHop("transfer", "channel-7")),
	},
	{
		name:  "multi-hop path with a short client-shaped segment in the base",
		trace: v8Trace{path: "transfer/channel-7/transfer/channel-3", base: "ab/foo-1/x"},
		want: transfertypes.NewDenom("ab/foo-1/x",
			transfertypes.NewHop("transfer", "channel-7"), transfertypes.NewHop("transfer", "channel-3")),
	},
	{
		// ibc-go v8 refuses this path on receive; the handler still converts it.
		name:  "path with a one-character port",
		trace: v8Trace{path: "transfer/channel-7/a/channel-1", base: "tok"},
		want:  transfertypes.NewDenom("a/channel-1/tok", transfertypes.NewHop("transfer", "channel-7")),
	},
}

// newTraceApp is an app whose transfer module is at its ibc-go v8 consensus version, with the
// given traces in its transfer store, so the handler runs the transfer store migration over them.
func newTraceApp(t *testing.T, traces ...v8Trace) (*app.App, sdk.Context, map[string]uint64) {
	t.Helper()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()
	bApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)
	ctx := bApp.NewUncachedContext(false, cmtproto.Header{Time: time.Date(2026, 11, 2, 13, 30, 0, 0, time.UTC)})
	bApp.IBCKeeper.ClientKeeper.SetParams(ctx, clienttypes.DefaultParams())

	store := ctx.KVStore(bApp.GetKey(transfertypes.StoreKey))
	for _, tr := range traces {
		store.Set(traceStoreKey(tr.hash()), tr.encode())
	}

	vm := bApp.ModuleManager.GetVersionMap()
	vm[transfertypes.ModuleName] = 5

	return bApp, ctx, vm
}

func runTraceUpgrade(t *testing.T, bApp *app.App, ctx sdk.Context, vm map[string]uint64) {
	t.Helper()

	handler := v830.CreateUpgradeHandler(bApp.Configurator(), bApp.ModuleManager, bApp.GuardianKeeper,
		bApp.RatelimitKeeper, bApp.BankKeeper, bApp.TradebinKeeper, bApp.BankKeeper, bApp.RewardsKeeper,
		runtime.NewKVStoreService(bApp.GetKey(transfertypes.StoreKey)), bApp.TransferKeeper)

	var (
		newVM map[string]uint64
		err   error
	)
	require.NotPanics(t, func() {
		newVM, err = handler(ctx, upgradetypes.Plan{Name: v830.UpgradeName}, vm)
	})
	require.NoError(t, err)
	require.Equal(t, uint64(6), newVM[transfertypes.ModuleName])
}

// requireNoTracesLeft checks that the v8 trace records are all gone after the upgrade.
func requireNoTracesLeft(t *testing.T, bApp *app.App, ctx sdk.Context) {
	t.Helper()

	it := storetypes.KVStorePrefixIterator(ctx.KVStore(bApp.GetKey(transfertypes.StoreKey)), transfertypes.DenomTraceKey)
	defer it.Close()
	require.False(t, it.Valid(), "denom trace records left after the upgrade")
}

// requireDenom checks that the voucher keeps its bank denom ibc/{sha256(full path)} and that the stored
// Denom is the wanted one.
func requireDenom(t *testing.T, bApp *app.App, ctx sdk.Context, tr v8Trace, want transfertypes.Denom) {
	t.Helper()

	got, found := bApp.TransferKeeper.GetDenom(ctx, tr.hash())
	require.True(t, found, "no denom for %q", tr.fullPath())
	require.Equal(t, want, got)
	require.NoError(t, got.Validate())
	require.Equal(t, tr.fullPath(), got.Path())
	require.Equal(t, "ibc/"+cmtbytes.HexBytes(tr.hash()).String(), got.IBCDenom())
}

// requireValidExport checks that the stored denoms pass the transfer genesis validation.
func requireValidExport(t *testing.T, bApp *app.App, ctx sdk.Context) {
	t.Helper()

	require.NoError(t, bApp.TransferKeeper.GetAllDenoms(ctx).Validate())
}

func TestHandler_DenomTraces_EmptyStore(t *testing.T) {
	bApp, ctx, vm := newTraceApp(t)

	runTraceUpgrade(t, bApp, ctx, vm)

	require.Empty(t, bApp.TransferKeeper.GetAllDenoms(ctx))
	requireNoTracesLeft(t, bApp, ctx)
}

// Traces the migration converts on its own become the Denom it parses, unchanged by the handler.
func TestHandler_DenomTraces_ValidTracesMigrateUnchanged(t *testing.T) {
	bApp, ctx, vm := newTraceApp(t, validTraces...)

	runTraceUpgrade(t, bApp, ctx, vm)

	require.Len(t, bApp.TransferKeeper.GetAllDenoms(ctx), len(validTraces))
	for _, tr := range validTraces {
		requireDenom(t, bApp, ctx, tr, transfertypes.ExtractDenomFromPath(tr.fullPath()))
	}
	requireNoTracesLeft(t, bApp, ctx)
	requireValidExport(t, bApp, ctx)
}

// Each trace the migration cannot parse is converted to a valid Denom with the same path, so the
// voucher keeps its bank denom, and the upgrade completes.
func TestHandler_DenomTraces_ConvertsTracesTheMigrationCannotParse(t *testing.T) {
	for _, tc := range convertedTraces {
		t.Run(tc.name, func(t *testing.T) {
			bApp, ctx, vm := newTraceApp(t, tc.trace, validTraces[0])

			runTraceUpgrade(t, bApp, ctx, vm)

			require.Len(t, bApp.TransferKeeper.GetAllDenoms(ctx), 2)
			requireDenom(t, bApp, ctx, tc.trace, tc.want)
			requireDenom(t, bApp, ctx, validTraces[0], transfertypes.ExtractDenomFromPath(validTraces[0].fullPath()))
			requireNoTracesLeft(t, bApp, ctx)
			requireValidExport(t, bApp, ctx)
		})
	}
}

// Valid and converted traces in one store all come out as their Denom.
func TestHandler_DenomTraces_MixedStore(t *testing.T) {
	traces := append([]v8Trace{}, validTraces...)
	for _, tc := range convertedTraces {
		traces = append(traces, tc.trace)
	}
	bApp, ctx, vm := newTraceApp(t, traces...)

	runTraceUpgrade(t, bApp, ctx, vm)

	require.Len(t, bApp.TransferKeeper.GetAllDenoms(ctx), len(traces))
	for _, tr := range validTraces {
		requireDenom(t, bApp, ctx, tr, transfertypes.ExtractDenomFromPath(tr.fullPath()))
	}
	for _, tc := range convertedTraces {
		requireDenom(t, bApp, ctx, tc.trace, tc.want)
	}
	requireNoTracesLeft(t, bApp, ctx)
	requireValidExport(t, bApp, ctx)
}

// Records with no valid Denom of the same path are deleted; the other traces still migrate.
func TestHandler_DenomTraces_DeletesTracesWithoutAValidDenom(t *testing.T) {
	deleted := []struct {
		name string
		key  []byte
		bz   []byte
	}{
		{
			name: "blank base after a single hop",
			key:  v8Trace{path: "transfer/channel-7"}.hash(),
			bz:   v8Trace{path: "transfer/channel-7"}.encode(),
		},
		{
			name: "empty path with a short client-shaped segment",
			key:  v8Trace{base: "ab/foo-1/x"}.hash(),
			bz:   v8Trace{base: "ab/foo-1/x"}.encode(),
		},
		{
			name: "odd number of path segments",
			key:  v8Trace{path: "transfer", base: "ab/foo-1"}.hash(),
			bz:   v8Trace{path: "transfer", base: "ab/foo-1"}.encode(),
		},
		{
			name: "stored under a key that is not its hash",
			key:  v8Trace{path: "transfer/channel-7", base: "other"}.hash(),
			bz:   v8Trace{path: "transfer/channel-7", base: "ab/foo-1"}.encode(),
		},
		{
			name: "empty record",
			key:  sha256Of("empty"),
			bz:   []byte{},
		},
		{
			name: "undecodable record",
			key:  sha256Of("garbage"),
			bz:   []byte{0x0a, 0xff},
		},
	}

	for _, tc := range deleted {
		t.Run(tc.name, func(t *testing.T) {
			bApp, ctx, vm := newTraceApp(t, validTraces...)
			ctx.KVStore(bApp.GetKey(transfertypes.StoreKey)).Set(traceStoreKey(tc.key), tc.bz)

			runTraceUpgrade(t, bApp, ctx, vm)

			_, found := bApp.TransferKeeper.GetDenom(ctx, tc.key)
			require.False(t, found)
			require.Len(t, bApp.TransferKeeper.GetAllDenoms(ctx), len(validTraces))
			for _, tr := range validTraces {
				requireDenom(t, bApp, ctx, tr, transfertypes.ExtractDenomFromPath(tr.fullPath()))
			}
			requireNoTracesLeft(t, bApp, ctx)
			requireValidExport(t, bApp, ctx)
		})
	}
}

func sha256Of(s string) []byte {
	h := sha256.Sum256([]byte(s))

	return h[:]
}
