package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// sweepFinalizingBoost pays out the outstanding accrual of a finalizing boost
// (days_left == 0) to up to limit participants, walking the reward's reverse
// index from the record's persisted finalize_cursor. Per participant the boost
// is settled against the final s_boost and the snapshot is set to
// (uid, s_final) unconditionally — no truncation guard here: a sub-unit
// remainder becomes dust by design, and the entry is NEVER deleted while the
// record exists (deleting would reset S0 to 0 and double-pay on the next
// settle — exploit A4). The cursor is persisted after each participant so
// concurrent cranks each continue where the last one stopped. Returns whether
// the record's iteration completed and how many participants were processed.
func (k Keeper) sweepFinalizingBoost(ctx sdk.Context, boost types.Boost, limit int) (done bool, processed int, err error) {
	sFinal, err := math.LegacyNewDecFromStr(boost.SBoost)
	if err != nil {
		return false, 0, fmt.Errorf("could not parse boost s_boost %q: %w", boost.SBoost, err)
	}

	addresses := k.GetBoostParticipantsFromCursor(ctx, boost.RewardId, boost.FinalizeCursor, limit)
	for _, address := range addresses {
		// a participant who exited mid-finalization was removed from the index;
		// one present in the index but without a record is skipped defensively.
		participant, found := k.GetStakingRewardParticipant(ctx, address, boost.RewardId)
		if found {
			if err = k.settleFinalizedBoost(ctx, &participant, boost, sFinal); err != nil {
				return false, processed, err
			}
			k.SetStakingRewardParticipant(ctx, participant)
		}

		boost.FinalizeCursor = address
		k.SetBoost(ctx, boost)
		processed++
	}

	// the record is done when nothing remains after the last processed cursor
	remaining := k.GetBoostParticipantsFromCursor(ctx, boost.RewardId, boost.FinalizeCursor, 1)
	return len(remaining) == 0, processed, nil
}

// settleFinalizedBoost settles a single boost record against its final s_boost
// for one participant: pays amount × (s_final − S0) (truncated) when positive
// and stamps the snapshot to (uid, s_final) unconditionally. It mutates the
// participant in place; the caller owns persistence.
func (k Keeper) settleFinalizedBoost(ctx sdk.Context, participant *types.StakingRewardParticipant, boost types.Boost, sFinal math.LegacyDec) error {
	amount, ok := math.NewIntFromString(participant.Amount)
	if !ok {
		return fmt.Errorf("could not transform participant amount into int: %s", participant.Amount)
	}

	// S0 = stored value only when the snapshot uid matches the record uid;
	// absent or stale entries mean the whole run is owed (same rule as settleBoosts).
	s0 := math.LegacyZeroDec()
	if snap, exists := participant.BoostSnapshots[boost.Denom]; exists && snap.Uid == boost.Uid {
		var err error
		s0, err = math.LegacyNewDecFromStr(snap.S0)
		if err != nil {
			return fmt.Errorf("could not parse snapshot s0 %q: %w", snap.S0, err)
		}
	}

	diff := sFinal.Sub(s0)
	if diff.IsPositive() {
		pending := amount.ToLegacyDec().Mul(diff).TruncateInt()
		if pending.IsPositive() {
			acc, err := sdk.AccAddressFromBech32(participant.Address)
			if err != nil {
				return err
			}

			coin := sdk.NewCoin(boost.Denom, pending)
			if err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, sdk.NewCoins(coin)); err != nil {
				return err
			}

			if err = ctx.EventManager().EmitTypedEvent(&types.BoostClaimEvent{
				RewardId: boost.RewardId,
				Denom:    boost.Denom,
				Uid:      boost.Uid,
				Address:  participant.Address,
				Amount:   pending.String(),
			}); err != nil {
				k.Logger().Error(err.Error())
			}
		}
	}

	if participant.BoostSnapshots == nil {
		participant.BoostSnapshots = make(map[string]*types.BoostSnapshot)
	}
	participant.BoostSnapshots[boost.Denom] = &types.BoostSnapshot{Uid: boost.Uid, S0: boost.SBoost}

	return nil
}

// removeFinalizedBoost deletes a boost record whose finalization is complete
// (or whose reward no longer exists) and emits the BoostFinalizeEvent.
func (k Keeper) removeFinalizedBoost(ctx sdk.Context, boost types.Boost) {
	k.RemoveBoost(ctx, boost.RewardId, boost.Denom)

	if err := ctx.EventManager().EmitTypedEvent(&types.BoostFinalizeEvent{
		RewardId: boost.RewardId,
		Denom:    boost.Denom,
		Uid:      boost.Uid,
	}); err != nil {
		k.Logger().Error(err.Error())
	}
}
