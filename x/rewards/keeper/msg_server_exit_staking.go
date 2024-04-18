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

	//send pending rewards
	_, err = k.claimPending(ctx, stakingReward, &participation)
	if err != nil {
		return nil, err
	}

	k.beginUnlock(ctx, participation, stakingReward)
	k.RemoveStakingRewardParticipant(ctx, participation.Address, participation.RewardId)
	remainingStakedAmount := stakedAmountInt.Sub(partCoins.AmountOf(stakingReward.StakingDenom))
	if remainingStakedAmount.IsPositive() {
		stakingReward.StakedAmount = remainingStakedAmount.String()
		k.SetStakingReward(ctx, stakingReward)
	} else {
		//if this staking reward is finished (all funds were distributed and payouts executed) we should remove it
		if stakingReward.Payouts >= stakingReward.Duration {
			k.RemoveStakingReward(ctx, stakingReward.RewardId)
		}
	}

	return &types.MsgExitStakingResponse{}, nil
}

func (k msgServer) beginUnlock(ctx sdk.Context, p types.StakingRewardParticipant, sr types.StakingReward) {
	lockedUntil := k.epochKeeper.GetEpochCountByIdentifier(ctx, expirationEpoch)
	lockedUntil += int64(sr.Lock) * 24
	pending := types.PendingUnlockParticipant{
		Index:   types.CreatePendingUnlockParticipantKey(lockedUntil, fmt.Sprintf("%s/%s", sr.RewardId, p.Address)),
		Address: p.Address,
		Amount:  p.Amount,
		Denom:   sr.StakingDenom,
	}

	k.SetPendingUnlockParticipant(ctx, pending)
}
