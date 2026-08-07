package v5

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate sets the boost parameters added in consensus version 5 to their
// default values and backfills the staking reward participant reverse index
// from the existing participant store.
func Migrate(
	_ sdk.Context,
	store prefix.Store,
	cdc codec.BinaryCodec,
) error {
	var params types.Params
	bz := store.Get(types.ParamsKey)
	if bz != nil {
		cdc.MustUnmarshal(bz, &params)
	}

	params.CreateBoostFee = types.DefaultCreateBoostFee
	params.MaxBoostsPerReward = types.DefaultMaxBoostsPerReward
	params.CleanupBatchSize = types.DefaultCleanupBatchSize

	if err := params.Validate(); err != nil {
		return err
	}

	bz = cdc.MustMarshal(&params)
	store.Set(types.ParamsKey, bz)

	backfillParticipantIndex(store, cdc)

	return nil
}

// backfillParticipantIndex writes one reverse-index entry per existing
// StakingRewardParticipant. Completeness is consensus-critical: a participant
// missing from the index is silently skipped by the cleanup sweep and their
// accrual stranded (exploit C2). Idempotent — the index has set semantics, so
// re-writing an entry is a no-op overwrite.
func backfillParticipantIndex(store prefix.Store, cdc codec.BinaryCodec) {
	indexStore := prefix.NewStore(store, types.KeyPrefix(types.StakingRewardParticipantIndexKeyPrefix))

	// collect first: the store must not be mutated while the iterator is open
	for _, key := range collectParticipantIndexKeys(store, cdc) {
		indexStore.Set(key, types.StakingRewardParticipantIndexValue)
	}
}

func collectParticipantIndexKeys(store prefix.Store, cdc codec.BinaryCodec) (keys [][]byte) {
	participantStore := prefix.NewStore(store, types.KeyPrefix(types.StakingRewardParticipantKeyPrefix))
	iterator := participantStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var participant types.StakingRewardParticipant
		cdc.MustUnmarshal(iterator.Value(), &participant)
		keys = append(keys, types.StakingRewardParticipantIndexKey(participant.RewardId, participant.Address))
	}

	return
}
