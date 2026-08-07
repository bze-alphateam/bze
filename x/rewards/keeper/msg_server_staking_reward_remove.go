package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// RemoveStakingReward - permissionless cleanup of a finished, emptied staking
// reward record. It applies the same condition + hook + event as ExitStaking's
// final-exit cleanup, but here a hook veto fails the message instead of being
// silently skipped: there is no exit to preserve, and an explicit error is more
// informative for the caller.
func (k msgServer) RemoveStakingReward(goCtx context.Context, msg *types.MsgRemoveStakingReward) (*types.MsgRemoveStakingRewardResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	stakingReward, found := k.GetStakingReward(ctx, msg.RewardId)
	if !found {
		return nil, errors.Wrapf(types.ErrInvalidRewardId, "reward with provided id not found")
	}

	if stakingReward.Payouts < stakingReward.Duration {
		return nil, errors.Wrapf(types.ErrStakingRewardNotFinished, "payouts %d of %d executed", stakingReward.Payouts, stakingReward.Duration)
	}

	stakedAmountInt, ok := math.NewIntFromString(stakingReward.StakedAmount)
	if !ok {
		return nil, fmt.Errorf("could not transform staked amount from storage into int")
	}
	if !stakedAmountInt.IsZero() {
		return nil, types.ErrStakingRewardNotEmpty
	}

	if err = k.beforeStakingRewardRemoval(ctx, stakingReward.RewardId); err != nil {
		return nil, err
	}

	k.Keeper.RemoveStakingReward(ctx, stakingReward.RewardId)

	err = ctx.EventManager().EmitTypedEvent(
		&types.StakingRewardFinishEvent{
			RewardId: stakingReward.RewardId,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgRemoveStakingRewardResponse{}, nil
}
