package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// settleBoosts is the single shared settle routine used by every settle moment
// (join, claim, exit). It implements the snapshot semantics table from the
// Business Logic page verbatim:
//
//   - For each existing boost record of the reward, the participant's accounted
//     value S0 is: the stored snapshot S0 when the snapshot's uid matches the
//     record's uid; otherwise 0 (absent entry, or an entry left over from an
//     older run — a "stale" uid). A stale/absent entry ⇒ S0 = 0 means the user
//     is entitled to the whole run so far, which is correct precisely because a
//     record only exists for a currently-running or finalizing boost.
//   - pending = amount × (s_boost − S0), truncated to an integer, is paid via
//     SendCoinsFromModuleToAccount.
//   - The snapshot is advanced to (uid, s_boost) ONLY when a positive payout was
//     made. When the payout truncates to zero the snapshot is left untouched so
//     small stakers keep accumulating instead of being repeatedly erased
//     (truncation guard, mirrors the base reward). Leaving a stale/absent entry
//     untouched is safe: S0 stays 0 next time, so nothing is double-paid.
//   - Finally, snapshot entries whose denom no longer has a record are dropped
//     (lazy cleanup — I4).
//
// It never reads or writes joined_at (boost settle is independent of the base
// reward accumulator). It mutates participant in place but does NOT persist it;
// the caller owns persistence. It returns whether any positive payout was made.
func (k Keeper) settleBoosts(ctx sdk.Context, participant *types.StakingRewardParticipant, rewardId string) (bool, error) {
	boosts := k.GetRewardBoosts(ctx, rewardId)
	if len(boosts) == 0 {
		// no active or finalizing records for this reward: every snapshot entry
		// the participant might hold is an orphan of a finalized run — drop them.
		if len(participant.BoostSnapshots) > 0 {
			participant.BoostSnapshots = nil
		}
		return false, nil
	}

	amount, ok := math.NewIntFromString(participant.Amount)
	if !ok {
		return false, fmt.Errorf("could not transform participant amount into int: %s", participant.Amount)
	}

	acc, err := sdk.AccAddressFromBech32(participant.Address)
	if err != nil {
		return false, err
	}

	recordDenoms := make(map[string]struct{}, len(boosts))
	paidSomething := false

	for _, boost := range boosts {
		recordDenoms[boost.Denom] = struct{}{}

		sBoost, err := math.LegacyNewDecFromStr(boost.SBoost)
		if err != nil {
			return false, fmt.Errorf("could not parse boost s_boost %q: %w", boost.SBoost, err)
		}

		// S0 = stored value only when the snapshot uid matches the record uid.
		s0 := math.LegacyZeroDec()
		if snap, exists := participant.BoostSnapshots[boost.Denom]; exists && snap.Uid == boost.Uid {
			s0, err = math.LegacyNewDecFromStr(snap.S0)
			if err != nil {
				return false, fmt.Errorf("could not parse snapshot s0 %q: %w", snap.S0, err)
			}
		}

		diff := sBoost.Sub(s0)
		if !diff.IsPositive() {
			// nothing accrued since the accounted point — keep the snapshot.
			continue
		}

		pending := amount.ToLegacyDec().Mul(diff).TruncateInt()
		if !pending.IsPositive() {
			// truncation guard: do not advance the snapshot, so the sub-unit
			// accrual keeps accumulating until it is worth at least one unit.
			continue
		}

		coin := sdk.NewCoin(boost.Denom, pending)
		if err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, sdk.NewCoins(coin)); err != nil {
			return false, err
		}
		paidSomething = true

		if participant.BoostSnapshots == nil {
			participant.BoostSnapshots = make(map[string]*types.BoostSnapshot)
		}
		participant.BoostSnapshots[boost.Denom] = &types.BoostSnapshot{Uid: boost.Uid, S0: boost.SBoost}

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

	// lazy cleanup: drop snapshot entries whose denom no longer has a record.
	for denom := range participant.BoostSnapshots {
		if _, hasRecord := recordDenoms[denom]; !hasRecord {
			delete(participant.BoostSnapshots, denom)
		}
	}

	return paidSomething, nil
}

// writeJoinBoostSnapshots resets the participant's snapshot to (uid, current
// s_boost) for EVERY existing boost record of the reward — active AND finalizing.
// It is the second half of the join settle moment: after settleBoosts has paid
// out accrual on the pre-join amount, this prevents the freshly added stake from
// earning retroactively. Covering finalizing records too is mandatory — omitting
// them lets a joiner during finalization collect amount × s_final for zero
// staking time (exploit A3).
func (k Keeper) writeJoinBoostSnapshots(ctx sdk.Context, participant *types.StakingRewardParticipant, rewardId string) {
	boosts := k.GetRewardBoosts(ctx, rewardId)
	if len(boosts) == 0 {
		return
	}

	if participant.BoostSnapshots == nil {
		participant.BoostSnapshots = make(map[string]*types.BoostSnapshot, len(boosts))
	}
	for _, boost := range boosts {
		participant.BoostSnapshots[boost.Denom] = &types.BoostSnapshot{Uid: boost.Uid, S0: boost.SBoost}
	}
}
