package app_test

import (
	"testing"

	corestore "cosmossdk.io/core/store"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channelv10 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v10"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
)

// ibcConsensusVersionV8 is the IBC core consensus version of ibc-go v8, the version the chain upgrades from.
const ibcConsensusVersionV8 = 6

func ibcStoreService(bApp *app.App) corestore.KVStoreService {
	return runtime.NewKVStoreService(bApp.GetKey(ibcexported.StoreKey))
}

// setLegacyChannel stores a transfer channel end in the ibc-go v8 encoding, with an upgrade in progress.
func setLegacyChannel(t *testing.T, bApp *app.App, ctx sdk.Context, channelID string, state channelv10.State, withCounterpartyUpgrade bool) {
	t.Helper()
	store := ctx.KVStore(bApp.GetKey(ibcexported.StoreKey))

	channel := channelv10.Channel{
		State:           state,
		Ordering:        channelv10.UNORDERED,
		Counterparty:    channelv10.Counterparty{PortId: ibctransfertypes.PortID, ChannelId: "channel-8"},
		ConnectionHops:  []string{"connection-0"},
		Version:         ibctransfertypes.V1,
		UpgradeSequence: 1,
	}
	bz, err := channel.Marshal()
	require.NoError(t, err)
	store.Set(host.ChannelKey(ibctransfertypes.PortID, channelID), bz)

	upgrade := channelv10.Upgrade{
		Fields: channelv10.UpgradeFields{
			Ordering:       channelv10.UNORDERED,
			ConnectionHops: []string{"connection-0"},
			Version:        `{"fee_version":"ics29-1","app_version":"ics20-1"}`,
		},
		NextSequenceSend: 1,
	}
	bz, err = upgrade.Marshal()
	require.NoError(t, err)
	store.Set(channelv10.ChannelUpgradeKey(ibctransfertypes.PortID, channelID), bz)
	if withCounterpartyUpgrade {
		store.Set(channelv10.ChannelCounterpartyUpgradeKey(ibctransfertypes.PortID, channelID), bz)
	}
}

// The real v8.3.0 handler, on the real app, migrates the IBC store from the ibc-go v8 consensus version while
// channel upgrades are unfinished: the channels end OPEN on their current version, without upgrade entries.
func TestIBCChannelUpgrade_UpgradeHandlerRestoresUnfinishedUpgrades(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)
	setLegacyChannel(t, bApp, ctx, "channel-1", channelv10.FLUSHING, false)
	setLegacyChannel(t, bApp, ctx, "channel-2", channelv10.FLUSHCOMPLETE, true)
	healthy, found := bApp.IBCKeeper.ChannelKeeper.GetChannel(ctx, ibctransfertypes.PortID, rlChannel)
	require.True(t, found)

	vm := bApp.ModuleManager.GetVersionMap()
	vm[ibcexported.ModuleName] = ibcConsensusVersionV8

	newVM, err := v830.CreateUpgradeHandler(bApp.Configurator(), bApp.ModuleManager, bApp.GuardianKeeper, bApp.RatelimitKeeper, bApp.BankKeeper, bApp.TradebinKeeper, bApp.BankKeeper, bApp.RewardsKeeper, ibcStoreService(bApp))(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, vm,
	)
	require.NoError(t, err)
	require.Equal(t, uint64(8), newVM[ibcexported.ModuleName])

	store := ctx.KVStore(bApp.GetKey(ibcexported.StoreKey))
	for _, channelID := range []string{"channel-1", "channel-2"} {
		channel, found := bApp.IBCKeeper.ChannelKeeper.GetChannel(ctx, ibctransfertypes.PortID, channelID)
		require.True(t, found, channelID)
		require.Equal(t, channeltypes.OPEN, channel.State, channelID)
		require.Equal(t, ibctransfertypes.V1, channel.Version, channelID)
		require.Equal(t, []string{"connection-0"}, channel.ConnectionHops, channelID)
		require.False(t, store.Has(channelv10.ChannelUpgradeKey(ibctransfertypes.PortID, channelID)), channelID)
		require.False(t, store.Has(channelv10.ChannelCounterpartyUpgradeKey(ibctransfertypes.PortID, channelID)), channelID)
		require.True(t, store.Has(channelv10.ChannelUpgradeErrorKey(ibctransfertypes.PortID, channelID)), channelID)
	}

	after, found := bApp.IBCKeeper.ChannelKeeper.GetChannel(ctx, ibctransfertypes.PortID, rlChannel)
	require.True(t, found)
	require.Equal(t, healthy, after)
}
