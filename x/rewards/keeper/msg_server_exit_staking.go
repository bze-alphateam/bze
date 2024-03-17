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
	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
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

	//TODO: Pay pending rewards here
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, partCoins)
	if err != nil {
		return nil, err
	}

	k.RemoveStakingRewardParticipant(ctx, participation.Address, participation.RewardId)
	stakingReward.StakedAmount = stakedAmountInt.Sub(partCoins.AmountOf(stakingReward.StakingDenom)).String()
	k.SetStakingReward(ctx, stakingReward)

	return &types.MsgExitStakingResponse{}, nil
}
