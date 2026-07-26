package v5

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate performs the x/rewards consensus version 4 -> 5 migration for the
// staking reward boosts feature:
//
//   - it backfills the boost participant reverse index (bpi/v/) for every
//     existing staking reward participant so that participants who joined before
//     the upgrade are reachable by the boost finalization sweep (Security A5 —
//     without this backfill they would silently lose boost payouts);
//   - it sets the two new boost params (create_boost_fee, max_boosts_per_reward)
//     to their defaults.
//
// Participant records themselves are left untouched: the new boost_snapshots map
// field is absent in pre-upgrade bytes and decodes as a nil/empty map, which is
// exactly the desired "no snapshot yet" state.
//
// The migration is idempotent: re-running it Sets the same index keys and params
// values again, which is a no-op in effect.
func Migrate(
	ctx sdk.Context,
	store prefix.Store,
	cdc codec.BinaryCodec,
) error {
	if err := backfillBoostParticipantIndex(store, cdc); err != nil {
		return err
	}

	return migrateParams(store, cdc)
}

// backfillBoostParticipantIndex iterates every staking reward participant and
// writes the reverse-index entry (bpi/v/{reward_id}/{address}/) the finalization
// sweep walks. Keys are derived from the record's own Address/RewardId fields so
// the index format stays identical to the keeper's SetBoostParticipantIndex.
func backfillBoostParticipantIndex(store prefix.Store, cdc codec.BinaryCodec) error {
	participantStore := prefix.NewStore(store, types.KeyPrefix(types.StakingRewardParticipantKeyPrefix))
	indexStore := prefix.NewStore(store, types.KeyPrefix(types.BoostParticipantIndexKeyPrefix))

	iterator := storetypes.KVStorePrefixIterator(participantStore, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var participant types.StakingRewardParticipant
		cdc.MustUnmarshal(iterator.Value(), &participant)

		indexStore.Set(
			types.BoostParticipantIndexKey(participant.RewardId, participant.Address),
			types.BoostParticipantIndexValue,
		)
	}

	return nil
}

// migrateParams sets the boost params introduced in consensus version 5 to their
// default values, preserving the existing (staking/trading/exit-gas) params.
func migrateParams(store prefix.Store, cdc codec.BinaryCodec) error {
	var params types.Params
	bz := store.Get(types.ParamsKey)
	if bz != nil {
		cdc.MustUnmarshal(bz, &params)
	}

	params.CreateBoostFee = types.DefaultCreateBoostFee
	params.MaxBoostsPerReward = types.DefaultMaxBoostsPerReward

	if err := params.Validate(); err != nil {
		return err
	}

	store.Set(types.ParamsKey, cdc.MustMarshal(&params))

	return nil
}
