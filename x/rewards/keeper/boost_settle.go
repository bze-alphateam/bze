package keeper

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// settleBoosts pays the participant's pending entitlement for every boost of
// their reward — active AND finished — and advances the paid entries to the
// boost's current accumulator. It is the single settle routine used by every
// settle moment (join, claim, exit); call sites never re-implement the math.
//
// Per boost the baseline S0 is the entry's joined_at when an entry exists and
// 0 otherwise: an absent entry provably means "staked since before the boost
// was created" (join stamps entries for every existing boost), so the
// participant is entitled from the boost's start. When the pending amount
// truncates to zero the entry is left untouched/absent so small stakers keep
// accruing instead of being repeatedly zeroed (mirrors claimPending).
//
// It returns whether at least one boost paid out.
func (k Keeper) settleBoosts(ctx sdk.Context, participant *types.StakingRewardParticipant) (bool, error) {
	boosts := k.GetRewardBoosts(ctx, participant.RewardId)
	if len(boosts) == 0 {
		return false, nil
	}

	amount, err := math.LegacyNewDecFromStr(participant.Amount)
	if err != nil {
		return false, err
	}

	paid := false
	for _, boost := range boosts {
		sBoost, err := math.LegacyNewDecFromStr(boost.DistributedStake)
		if err != nil {
			return paid, err
		}

		s0 := math.LegacyZeroDec()
		entry, found := k.GetBoostParticipant(ctx, participant.Address, boost.RewardId, boost.Id)
		if found {
			s0, err = math.LegacyNewDecFromStr(entry.JoinedAt)
			if err != nil {
				return paid, err
			}
		}

		if sBoost.Equal(s0) {
			continue
		}

		pending := amount.Mul(sBoost.Sub(s0)).TruncateInt()
		if !pending.IsPositive() {
			continue
		}

		acc, err := sdk.AccAddressFromBech32(participant.Address)
		if err != nil {
			return paid, err
		}

		toSend := sdk.NewCoin(boost.Denom, pending)
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, sdk.NewCoins(toSend))
		if err != nil {
			return paid, err
		}

		k.SetBoostParticipant(ctx, types.BoostParticipant{
			Address:  participant.Address,
			RewardId: boost.RewardId,
			BoostId:  boost.Id,
			JoinedAt: boost.DistributedStake,
		})
		paid = true

		err = ctx.EventManager().EmitTypedEvent(
			&types.BoostClaimEvent{
				RewardId: boost.RewardId,
				BoostId:  boost.Id,
				Denom:    boost.Denom,
				Address:  participant.Address,
				Amount:   pending.String(),
			},
		)

		if err != nil {
			k.Logger().Error(err.Error())
		}
	}

	return paid, nil
}

// stampBoostParticipants writes an entry (joined_at = current accumulator)
// for EVERY existing boost of the reward, finished included: a participant
// joining while a finished boost is still stored must read a final baseline,
// not an absent entry, or they would be paid for time they never staked.
// Mirrors the base handler's unconditional JoinedAt refresh on join, with the
// same sub-unit-dust-on-top-up semantics.
func (k Keeper) stampBoostParticipants(ctx sdk.Context, address, rewardId string) {
	for _, boost := range k.GetRewardBoosts(ctx, rewardId) {
		k.SetBoostParticipant(ctx, types.BoostParticipant{
			Address:  address,
			RewardId: rewardId,
			BoostId:  boost.Id,
			JoinedAt: boost.DistributedStake,
		})
	}
}
