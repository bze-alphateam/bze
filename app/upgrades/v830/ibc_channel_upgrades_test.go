package v830_test

import (
	"context"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkruntime "github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	channelv10 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v10"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"

	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
)

const (
	fakeIBCModuleName = "ibc"
	transferPort      = "transfer"
	feeVersion        = `{"fee_version":"ics29-1","app_version":"ics20-1"}`
)

// memKVStore adapts an in-memory database to the core KVStore interface.
type memKVStore struct{ *dbm.MemDB }

func (s memKVStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	return s.MemDB.Iterator(start, end)
}

func (s memKVStore) ReverseIterator(start, end []byte) (corestore.Iterator, error) {
	return s.MemDB.ReverseIterator(start, end)
}

// memStoreService is an IBC store that holds no channels, for the handler tests that are not about IBC.
type memStoreService struct{ store memKVStore }

func (s memStoreService) OpenKVStore(context.Context) corestore.KVStore { return s.store }

func emptyIBCStore() corestore.KVStoreService {
	return memStoreService{store: memKVStore{dbm.NewMemDB()}}
}

// fakeChannelKeeper writes channels the way the ibc-go v10 channel keeper does.
type fakeChannelKeeper struct {
	store corestore.KVStoreService
	cdc   codec.BinaryCodec
}

func (k fakeChannelKeeper) Logger(ctx sdk.Context) log.Logger { return ctx.Logger() }

func (k fakeChannelKeeper) SetChannel(ctx sdk.Context, portID, channelID string, channel channeltypes.Channel) {
	if err := k.store.OpenKVStore(ctx).Set(host.ChannelKey(portID, channelID), k.cdc.MustMarshal(&channel)); err != nil {
		panic(err)
	}
}

// fakeIBCModule stands in for the IBC core module: its 7 -> 8 migration is the real ibc-go v10 channel store
// migration, run on the same store the handler gets.
type fakeIBCModule struct {
	store corestore.KVStoreService
}

func (fakeIBCModule) RegisterLegacyAminoCodec(*codec.LegacyAmino)                 {}
func (fakeIBCModule) RegisterInterfaces(codectypes.InterfaceRegistry)             {}
func (fakeIBCModule) RegisterGRPCGatewayRoutes(client.Context, *runtime.ServeMux) {}
func (fakeIBCModule) IsOnePerModuleType()                                         {}
func (fakeIBCModule) IsAppModule()                                                {}
func (fakeIBCModule) Name() string                                                { return fakeIBCModuleName }
func (fakeIBCModule) ConsensusVersion() uint64                                    { return 8 }

func (m fakeIBCModule) RegisterServices(cfg module.Configurator) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	err := cfg.RegisterMigration(fakeIBCModuleName, 7, func(ctx sdk.Context) error {
		return channelv10.MigrateStore(ctx, m.store, cdc, fakeChannelKeeper{store: m.store, cdc: cdc})
	})
	if err != nil {
		panic(err)
	}
}

type ibcFixture struct {
	t       *testing.T
	ctx     sdk.Context
	store   corestore.KVStoreService
	handler upgradetypes.UpgradeHandler
}

func newIBCFixture(t *testing.T) *ibcFixture {
	t.Helper()

	key := storetypes.NewKVStoreKey("ibc")
	ctx := testutil.DefaultContext(key, storetypes.NewTransientStoreKey("ibc_t"))
	store := sdkruntime.NewKVStoreService(key)

	mm := module.NewManager(fakeIBCModule{store: store})
	cfg := module.NewConfigurator(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), baseapp.NewMsgServiceRouter(), baseapp.NewGRPCQueryRouter())
	require.NoError(t, mm.RegisterServices(cfg))

	return &ibcFixture{
		t:       t,
		ctx:     ctx,
		store:   store,
		handler: v830.CreateUpgradeHandler(cfg, mm, &fakeGuardian{}, acceptingRateLimits(t), emptyBank(t), noPools{}, nil, &fakeStakingRewards{}, noTraces{}, nil, store),
	}
}

// upgrade runs the handler with the IBC module at the given consensus version: 7 runs the channel store
// migration, 8 runs only the handler's own steps.
func (f *ibcFixture) upgrade(ibcVersion uint64) (module.VersionMap, error) {
	return f.handler(f.ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{fakeIBCModuleName: ibcVersion})
}

func (f *ibcFixture) kv() corestore.KVStore { return f.store.OpenKVStore(f.ctx) }

func (f *ibcFixture) set(key []byte, msg interface{ Marshal() ([]byte, error) }) {
	f.t.Helper()
	bz, err := msg.Marshal()
	require.NoError(f.t, err)
	require.NoError(f.t, f.kv().Set(key, bz))
}

func (f *ibcFixture) get(key []byte) []byte {
	f.t.Helper()
	bz, err := f.kv().Get(key)
	require.NoError(f.t, err)

	return bz
}

// setChannel stores a channel end in the ibc-go v8 encoding.
func (f *ibcFixture) setChannel(channelID string, state channelv10.State, version string, upgradeSequence uint64) channelv10.Channel {
	f.t.Helper()
	channel := channelv10.Channel{
		State:           state,
		Ordering:        channelv10.UNORDERED,
		Counterparty:    channelv10.Counterparty{PortId: transferPort, ChannelId: "channel-9" + channelID[len("channel-"):]},
		ConnectionHops:  []string{"connection-0"},
		Version:         version,
		UpgradeSequence: upgradeSequence,
	}
	f.set(host.ChannelKey(transferPort, channelID), &channel)

	return channel
}

func proposedUpgrade(sequence uint64) *channelv10.Upgrade {
	return &channelv10.Upgrade{
		Fields: channelv10.UpgradeFields{
			Ordering:       channelv10.UNORDERED,
			ConnectionHops: []string{"connection-0"},
			Version:        feeVersion,
		},
		Timeout:          channelv10.Timeout{Timestamp: 1_900_000_000_000_000_000},
		NextSequenceSend: sequence,
	}
}

// setUpgrade stores the upgrade proposal an upgrade handshake step leaves for the channel.
func (f *ibcFixture) setUpgrade(channelID string) {
	f.set(channelv10.ChannelUpgradeKey(transferPort, channelID), proposedUpgrade(7))
}

// setCounterpartyUpgrade stores the counterparty proposal that the ACK and CONFIRM steps leave for the channel.
func (f *ibcFixture) setCounterpartyUpgrade(channelID string) {
	f.set(channelv10.ChannelCounterpartyUpgradeKey(transferPort, channelID), proposedUpgrade(9))
}

func (f *ibcFixture) setErrorReceipt(channelID string, sequence uint64) {
	f.set(channelv10.ChannelUpgradeErrorKey(transferPort, channelID), &channelv10.ErrorReceipt{Sequence: sequence, Message: "earlier attempt"})
}

func (f *ibcFixture) legacyChannel(channelID string) channelv10.Channel {
	f.t.Helper()
	var channel channelv10.Channel
	require.NoError(f.t, channel.Unmarshal(f.get(host.ChannelKey(transferPort, channelID))))

	return channel
}

// migratedChannel reads a channel end in the ibc-go v10 encoding the migration writes.
func (f *ibcFixture) migratedChannel(channelID string) channeltypes.Channel {
	f.t.Helper()
	var channel channeltypes.Channel
	require.NoError(f.t, channel.Unmarshal(f.get(host.ChannelKey(transferPort, channelID))))

	return channel
}

func (f *ibcFixture) errorReceipt(channelID string) (channelv10.ErrorReceipt, bool) {
	f.t.Helper()
	bz := f.get(channelv10.ChannelUpgradeErrorKey(transferPort, channelID))
	if bz == nil {
		return channelv10.ErrorReceipt{}, false
	}
	var receipt channelv10.ErrorReceipt
	require.NoError(f.t, receipt.Unmarshal(bz))

	return receipt, true
}

// snapshot copies every entry in the IBC store.
func (f *ibcFixture) snapshot() map[string][]byte {
	f.t.Helper()
	it, err := f.kv().Iterator(nil, nil)
	require.NoError(f.t, err)
	defer it.Close()

	entries := map[string][]byte{}
	for ; it.Valid(); it.Next() {
		entries[string(it.Key())] = append([]byte(nil), it.Value()...)
	}

	return entries
}

// seedMixedChannels stores the channels of a chain where some channel upgrades are unfinished:
//   - channel-0: OPEN, never upgraded;
//   - channel-1: OPEN with an upgrade proposed (the state INIT leaves);
//   - channel-2: FLUSHING after TRY (upgrade only);
//   - channel-3: FLUSHING after ACK (upgrade and counterparty upgrade);
//   - channel-4: FLUSHCOMPLETE after CONFIRM, with an error receipt from an earlier attempt.
func (f *ibcFixture) seedMixedChannels() {
	f.setChannel("channel-0", channelv10.OPEN, "ics20-1", 0)

	f.setChannel("channel-1", channelv10.OPEN, "ics20-1", 1)
	f.setUpgrade("channel-1")

	f.setChannel("channel-2", channelv10.FLUSHING, "ics20-1", 1)
	f.setUpgrade("channel-2")

	f.setChannel("channel-3", channelv10.FLUSHING, "ics20-1", 3)
	f.setUpgrade("channel-3")
	f.setCounterpartyUpgrade("channel-3")

	f.setChannel("channel-4", channelv10.FLUSHCOMPLETE, "ics20-1", 2)
	f.setUpgrade("channel-4")
	f.setCounterpartyUpgrade("channel-4")
	f.setErrorReceipt("channel-4", 1)
}

// The IBC channel store migration refuses channels in FLUSHING or FLUSHCOMPLETE; the handler restores them
// before it runs, so the migration succeeds.
func TestHandler_ChannelStoreMigrationSucceedsWithUnfinishedChannelUpgrades(t *testing.T) {
	f := newIBCFixture(t)
	f.seedMixedChannels()

	newVM, err := f.upgrade(7)
	require.NoError(t, err)
	require.Equal(t, uint64(8), newVM[fakeIBCModuleName])
}

// After the migration every restored channel is OPEN with the fields it had before the upgrade attempt.
func TestHandler_RestoredChannelsAreOpenWithTheirCurrentFields(t *testing.T) {
	f := newIBCFixture(t)
	f.seedMixedChannels()

	_, err := f.upgrade(7)
	require.NoError(t, err)

	for channelID, counterpartyID := range map[string]string{"channel-2": "channel-92", "channel-3": "channel-93", "channel-4": "channel-94"} {
		channel := f.migratedChannel(channelID)
		require.Equal(t, channeltypes.OPEN, channel.State, channelID)
		require.Equal(t, channeltypes.UNORDERED, channel.Ordering, channelID)
		require.Equal(t, "ics20-1", channel.Version, channelID)
		require.Equal(t, []string{"connection-0"}, channel.ConnectionHops, channelID)
		require.Equal(t, channeltypes.Counterparty{PortId: transferPort, ChannelId: counterpartyID}, channel.Counterparty, channelID)
		require.NoError(t, channel.ValidateBasic(), channelID)
	}
}

// The restored channels lose their upgrade and counterparty upgrade entries, also after the migration (which
// leaves per-channel upgrade entries in place).
func TestHandler_RestoredChannelsHaveNoUpgradeEntries(t *testing.T) {
	f := newIBCFixture(t)
	f.seedMixedChannels()

	_, err := f.upgrade(7)
	require.NoError(t, err)

	for _, channelID := range []string{"channel-2", "channel-3", "channel-4"} {
		require.Nil(t, f.get(channelv10.ChannelUpgradeKey(transferPort, channelID)), channelID)
		require.Nil(t, f.get(channelv10.ChannelCounterpartyUpgradeKey(transferPort, channelID)), channelID)
	}
}

// Every restored channel gets an error receipt at its upgrade sequence, the proof the counterparty needs to cancel
// its side of the upgrade. It replaces a receipt from an earlier attempt.
func TestHandler_RestoredChannelsGetAnErrorReceiptAtTheirUpgradeSequence(t *testing.T) {
	f := newIBCFixture(t)
	f.seedMixedChannels()

	_, err := f.upgrade(7)
	require.NoError(t, err)

	for channelID, sequence := range map[string]uint64{"channel-2": 1, "channel-3": 3, "channel-4": 2} {
		receipt, found := f.errorReceipt(channelID)
		require.True(t, found, channelID)
		require.Equal(t, sequence, receipt.Sequence, channelID)
		require.NotEmpty(t, receipt.Message, channelID)
	}
}

// A receipt already at or above the upgrade sequence lets the counterparty cancel; it is kept as it is.
func TestHandler_KeepsAHigherErrorReceipt(t *testing.T) {
	f := newIBCFixture(t)
	f.setChannel("channel-0", channelv10.FLUSHING, "ics20-1", 2)
	f.setUpgrade("channel-0")
	f.setErrorReceipt("channel-0", 5)
	before := f.get(channelv10.ChannelUpgradeErrorKey(transferPort, "channel-0"))

	_, err := f.upgrade(7)
	require.NoError(t, err)
	require.Equal(t, before, f.get(channelv10.ChannelUpgradeErrorKey(transferPort, "channel-0")))
	require.Equal(t, channeltypes.OPEN, f.migratedChannel("channel-0").State)
}

// Before the migration the restored channels are stored in the ibc-go v8 encoding: OPEN, with the upgrade sequence
// they had, the way an ibc-go v8 upgrade cancellation leaves them.
func TestHandler_RestoredChannelsKeepTheirUpgradeSequence(t *testing.T) {
	f := newIBCFixture(t)
	f.seedMixedChannels()
	want := map[string]channelv10.Channel{}
	for _, channelID := range []string{"channel-2", "channel-3", "channel-4"} {
		channel := f.legacyChannel(channelID)
		channel.State = channelv10.OPEN
		want[channelID] = channel
	}

	_, err := f.upgrade(8)
	require.NoError(t, err)

	for channelID, channel := range want {
		require.Equal(t, channel, f.legacyChannel(channelID), channelID)
	}
}

// Channels without an unfinished upgrade keep every byte, including an upgrade proposal still waiting in OPEN.
func TestHandler_LeavesHealthyChannelsUntouched(t *testing.T) {
	f := newIBCFixture(t)
	f.seedMixedChannels()
	before := f.snapshot()

	_, err := f.upgrade(8)
	require.NoError(t, err)

	after := f.snapshot()
	for _, key := range [][]byte{
		host.ChannelKey(transferPort, "channel-0"),
		host.ChannelKey(transferPort, "channel-1"),
		channelv10.ChannelUpgradeKey(transferPort, "channel-1"),
	} {
		require.Equal(t, before[string(key)], after[string(key)], string(key))
	}
	_, found := f.errorReceipt("channel-0")
	require.False(t, found)
	_, found = f.errorReceipt("channel-1")
	require.False(t, found)
}

// A store with only healthy channels is left exactly as it was, and the migration succeeds.
func TestHandler_StoreWithOnlyHealthyChannelsIsUnchanged(t *testing.T) {
	f := newIBCFixture(t)
	f.setChannel("channel-0", channelv10.OPEN, "ics20-1", 0)
	f.setChannel("channel-1", channelv10.CLOSED, "ics20-1", 0)
	f.setChannel("channel-2", channelv10.OPEN, "ics20-1", 1)
	f.setUpgrade("channel-2")
	before := f.snapshot()

	_, err := f.upgrade(8)
	require.NoError(t, err)
	require.Equal(t, before, f.snapshot())

	_, err = f.upgrade(7)
	require.NoError(t, err)
	require.Equal(t, channeltypes.OPEN, f.migratedChannel("channel-0").State)
	require.Equal(t, channeltypes.CLOSED, f.migratedChannel("channel-1").State)
}

// A chain without channels upgrades without writing anything to the IBC store.
func TestHandler_EmptyIBCStore(t *testing.T) {
	f := newIBCFixture(t)

	_, err := f.upgrade(7)
	require.NoError(t, err)
	require.Empty(t, f.snapshot())
}

// A channel end that does not decode fails the upgrade before anything is written.
func TestHandler_UndecodableChannelFailsTheUpgrade(t *testing.T) {
	f := newIBCFixture(t)
	f.setChannel("channel-1", channelv10.FLUSHING, "ics20-1", 1)
	f.setUpgrade("channel-1")
	// A length-delimited field that claims more bytes than follow.
	require.NoError(t, f.kv().Set(host.ChannelKey(transferPort, "channel-0"), []byte{0x2a, 0x05}))
	before := f.snapshot()

	_, err := f.upgrade(8)
	require.ErrorContains(t, err, "decode channel transfer/channel-0")
	require.Equal(t, before, f.snapshot())
}

// A key under the channel end prefix that is not a channel path fails the upgrade before anything is written.
func TestHandler_UnparsableChannelKeyFailsTheUpgrade(t *testing.T) {
	f := newIBCFixture(t)
	f.setChannel("channel-1", channelv10.FLUSHING, "ics20-1", 1)
	f.setUpgrade("channel-1")
	bz, err := (&channelv10.Channel{State: channelv10.OPEN}).Marshal()
	require.NoError(t, err)
	require.NoError(t, f.kv().Set([]byte(host.KeyChannelEndPrefix+"/not-a-channel-path"), bz))
	before := f.snapshot()

	_, err = f.upgrade(8)
	require.ErrorContains(t, err, "parse channel key")
	require.Equal(t, before, f.snapshot())
}

// An error receipt that does not decode fails the upgrade.
func TestHandler_UndecodableErrorReceiptFailsTheUpgrade(t *testing.T) {
	f := newIBCFixture(t)
	f.setChannel("channel-0", channelv10.FLUSHING, "ics20-1", 1)
	f.setUpgrade("channel-0")
	require.NoError(t, f.kv().Set(channelv10.ChannelUpgradeErrorKey(transferPort, "channel-0"), []byte{0x12, 0x05}))

	_, err := f.upgrade(8)
	require.ErrorContains(t, err, "decode error receipt transfer/channel-0")
}
