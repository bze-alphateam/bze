package keeper

import (
	"context"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) DistributeStakingRewards(goCtx context.Context, msg *types.MsgDistributeStakingRewards) (*types.MsgDistributeStakingRewardsResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	amtInt, ok := sdk.NewIntFromString(msg.Amount)
	if !ok {
		return nil, sdkerrors.Wrapf(types.ErrInvalidAmount, "could not convert order amount")
	}

	if !amtInt.IsPositive() {
		return nil, sdkerrors.Wrapf(types.ErrInvalidAmount, "amount should be greater than 0")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	stakingReward, isFound := k.GetStakingReward(ctx, msg.RewardId)
	if !isFound {
		return nil, sdkerrors.Wrap(sdkerrors.ErrKeyNotFound, "staking reward not found")
	}

	err = k.distributeStakingRewards(ctx, &stakingReward, msg.Amount)
	if err != nil {
		return nil, err
	}

	k.SetStakingReward(ctx, stakingReward)

	k.Logger(ctx).Debug("staking reward distributed")

	return &types.MsgDistributeStakingRewardsResponse{}, nil
}
