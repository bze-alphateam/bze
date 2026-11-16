package v830

import (
	"fmt"

	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channelv10 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v10"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// channelUpgradeRestoredMessage is the message of the error receipt written for every channel whose
// unfinished channel upgrade is restored to OPEN. A counterparty cancelling its side proves the
// receipt as stored, so any text works: the cancellation checks only the receipt sequence.
const channelUpgradeRestoredMessage = "channel upgrade cancelled by the v8.3.0 software upgrade"

// legacyChannel is a channel end read in the ibc-go v8 encoding, with the key it is stored under.
type legacyChannel struct {
	portID    string
	channelID string
	channel   channelv10.Channel
}

// restoreUnfinishedChannelUpgrades puts every channel with an unfinished channel upgrade (state FLUSHING or
// FLUSHCOMPLETE) back to OPEN before the IBC store migration, which accepts no channel in those states.
//
// It does what an ibc-go v8 upgrade cancellation does: the channel keeps its ordering, counterparty, connection
// hops, version and upgrade sequence, its upgrade and counterparty upgrade entries are deleted, and an error
// receipt is written at its upgrade sequence. The receipt is the proof a counterparty needs to cancel its side
// of the same upgrade. Channels in any other state are left untouched.
func restoreUnfinishedChannelUpgrades(ctx sdk.Context, ibcStore corestore.KVStoreService) error {
	store := runtime.KVStoreAdapter(ibcStore.OpenKVStore(ctx))

	channels, err := readLegacyChannels(store)
	if err != nil {
		return err
	}

	restored := 0
	for _, c := range channels {
		if c.channel.State != channelv10.FLUSHING && c.channel.State != channelv10.FLUSHCOMPLETE {
			continue
		}
		if err := restoreChannel(store, c); err != nil {
			return err
		}
		restored++
		ctx.Logger().Info("restored channel with an unfinished channel upgrade to OPEN",
			"port", c.portID, "channel", c.channelID, "previous_state", c.channel.State.String(),
			"upgrade_sequence", c.channel.UpgradeSequence)
	}

	ctx.Logger().Info("checked channels for unfinished channel upgrades", "channels", len(channels), "restored", restored)

	return nil
}

// readLegacyChannels reads every channel end in key order. The iterator is closed before anything is written.
func readLegacyChannels(store storetypes.KVStore) ([]legacyChannel, error) {
	iterator := storetypes.KVStorePrefixIterator(store, []byte(host.KeyChannelEndPrefix))
	defer iterator.Close()

	var channels []legacyChannel
	for ; iterator.Valid(); iterator.Next() {
		portID, channelID, err := host.ParseChannelPath(string(iterator.Key()))
		if err != nil {
			return nil, fmt.Errorf("parse channel key %q: %w", iterator.Key(), err)
		}

		var channel channelv10.Channel
		if err := channel.Unmarshal(iterator.Value()); err != nil {
			return nil, fmt.Errorf("decode channel %s/%s: %w", portID, channelID, err)
		}

		channels = append(channels, legacyChannel{portID: portID, channelID: channelID, channel: channel})
	}

	return channels, nil
}

// restoreChannel sets one channel back to OPEN, deletes its upgrade entries and writes its error receipt.
func restoreChannel(store storetypes.KVStore, c legacyChannel) error {
	channel := c.channel
	channel.State = channelv10.OPEN

	bz, err := channel.Marshal()
	if err != nil {
		return fmt.Errorf("encode channel %s/%s: %w", c.portID, c.channelID, err)
	}
	store.Set(host.ChannelKey(c.portID, c.channelID), bz)

	store.Delete(channelv10.ChannelUpgradeKey(c.portID, c.channelID))
	store.Delete(channelv10.ChannelCounterpartyUpgradeKey(c.portID, c.channelID))

	return writeErrorReceipt(store, c.portID, c.channelID, channel.UpgradeSequence)
}

// writeErrorReceipt writes an error receipt at the given upgrade sequence. An existing receipt at the same or a
// higher sequence already lets the counterparty cancel, so it is kept.
func writeErrorReceipt(store storetypes.KVStore, portID, channelID string, sequence uint64) error {
	key := channelv10.ChannelUpgradeErrorKey(portID, channelID)

	if bz := store.Get(key); bz != nil {
		var existing channelv10.ErrorReceipt
		if err := existing.Unmarshal(bz); err != nil {
			return fmt.Errorf("decode error receipt %s/%s: %w", portID, channelID, err)
		}
		if existing.Sequence >= sequence {
			return nil
		}
	}

	receipt := channelv10.ErrorReceipt{Sequence: sequence, Message: channelUpgradeRestoredMessage}
	bz, err := receipt.Marshal()
	if err != nil {
		return fmt.Errorf("encode error receipt %s/%s: %w", portID, channelID, err)
	}
	store.Set(key, bz)

	return nil
}
