package keeper

import (
	"context"
	"fmt"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) ExitStaking(goCtx context.Context, msg *types.MsgExitStaking) (*types.MsgExitStakingResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	stakingReward, found := k.GetStakingReward(ctx, msg.RewardId)
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrInvalidRewardId, "reward with provided id not found")
	}

	participation, found := k.GetStakingRewardParticipant(ctx, msg.Creator, msg.RewardId)
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrInvalidRewardId, "you are not a participant in this staking reward")
	}

	partCoins, err := k.getAmountToCapture("", stakingReward.StakingDenom, participation.Amount, int64(1))
	if err != nil {
		return nil, err
	}
	stakedAmountInt, ok := sdk.NewIntFromString(stakingReward.StakedAmount)
	if !ok {
		return nil, fmt.Errorf("could not transform amount from storage into int")
	}
	if !stakedAmountInt.IsPositive() {
		//disaster in this case
		return nil, fmt.Errorf("no staked amount left")
	}

	//send pending rewards
	_, err = k.claimPending(ctx, stakingReward, &participation)
	if err != nil {
		return nil, err
	}

	err = k.beginUnlock(ctx, participation, stakingReward)
	if err != nil {
		return nil, err
	}

	k.RemoveStakingRewardParticipant(ctx, participation.Address, participation.RewardId)

	remainingStakedAmount := stakedAmountInt.Sub(partCoins.AmountOf(stakingReward.StakingDenom))
	stakingReward.StakedAmount = remainingStakedAmount.String()
	k.SetStakingReward(ctx, stakingReward)

	//if this staking reward is finished (all funds were distributed and payouts executed) we should remove it
	if remainingStakedAmount.IsZero() && stakingReward.Payouts >= stakingReward.Duration {
		k.RemoveStakingReward(ctx, stakingReward.RewardId)
		err = ctx.EventManager().EmitTypedEvent(
			&types.StakingRewardFinishEvent{
				RewardId: stakingReward.RewardId,
			},
		)

		if err != nil {
			k.Logger(ctx).Error(err.Error())
		}
	}

	err = ctx.EventManager().EmitTypedEvent(
		&types.StakingRewardExitEvent{
			RewardId: stakingReward.RewardId,
			Address:  msg.Creator,
		},
	)

	if err != nil {
		k.Logger(ctx).Error(err.Error())
	}

	return &types.MsgExitStakingResponse{}, nil
}

func (k msgServer) beginUnlock(ctx sdk.Context, p types.StakingRewardParticipant, sr types.StakingReward) error {
	lockedUntil := k.epochKeeper.GetEpochCountByIdentifier(ctx, expirationEpoch)
	lockedUntil += int64(sr.Lock) * 24
	pendingKey := types.CreatePendingUnlockParticipantKey(lockedUntil, fmt.Sprintf("%s/%s", sr.RewardId, p.Address))
	pending := types.PendingUnlockParticipant{
		Index:   pendingKey,
		Address: p.Address,
		Amount:  p.Amount,
		Denom:   sr.StakingDenom,
	}

	inStore, found := k.GetPendingUnlockParticipant(ctx, pendingKey)
	if found {
		//we already have a pending unlock for this reward and participant at the same epoch
		//update the amount, so it can all be unlocked at once
		inStoreAmount, _ := sdk.NewIntFromString(inStore.Amount)
		pendingAmount, _ := sdk.NewIntFromString(pending.Amount)
		pendingAmount.Add(inStoreAmount)
		pending.Amount = pendingAmount.String()
	}

	//in case the lock is 0 send the funds immediately
	if sr.Lock == 0 {
		return k.performUnlock(ctx, &pending)
	}

	k.SetPendingUnlockParticipant(ctx, pending)

	return nil
}
