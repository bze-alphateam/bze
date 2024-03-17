package keeper

import (
	"context"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) ClaimStakingRewards(goCtx context.Context, msg *types.MsgClaimStakingRewards) (*types.MsgClaimStakingRewardsResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	stakingReward, found := k.GetStakingReward(ctx, msg.RewardId)
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrInvalidRewardId, "reward with provided id not found")
	}

	participant, found := k.GetStakingRewardParticipant(ctx, msg.Creator, msg.RewardId)
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrInvalidRewardId, "you are not a participant in this staking reward")
	}

	paid, err := k.claimPending(ctx, stakingReward, &participant)
	if err != nil {
		return nil, err
	}

	k.SetStakingRewardParticipant(ctx, participant)

	return &types.MsgClaimStakingRewardsResponse{Amount: paid.Amount.String()}, nil
}
