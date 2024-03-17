package keeper

import (
	"context"
	"fmt"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) JoinStaking(goCtx context.Context, msg *types.MsgJoinStaking) (*types.MsgJoinStakingResponse, error) {
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

	stakedAmount := sdk.NewInt(int64(0))
	if stakingReward.StakedAmount != "" {
		ok := false
		stakedAmount, ok = sdk.NewIntFromString(stakingReward.StakedAmount)
		if !ok {
			return nil, fmt.Errorf("could not transform staked amount from storage into int")
		}
	}

	toCapture, err := k.getAmountToCapture("", stakingReward.StakingDenom, msg.Amount, int64(1))
	if err != nil {
		return nil, err
	}

	if err = k.checkUserBalances(ctx, toCapture, acc); err != nil {
		return nil, err
	}

	participant, found := k.GetStakingRewardParticipant(ctx, msg.Creator, msg.RewardId)
	if found {
		//TODO: claim already pending rewards here
	} else {
		participant = types.StakingRewardParticipant{
			Address:  msg.Creator,
			RewardId: msg.RewardId,
			Amount:   "0",
		}
	}

	amtInt, ok := sdk.NewIntFromString(participant.Amount)
	if !ok {
		return nil, fmt.Errorf("could not transform amount from storage into int")
	}
	amtInt = amtInt.Add(toCapture.AmountOf(stakingReward.StakingDenom))
	participant.Amount = amtInt.String()

	stakedAmount = stakedAmount.Add(toCapture.AmountOf(stakingReward.StakingDenom))
	stakingReward.StakedAmount = stakedAmount.String()

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture)
	if err != nil {
		return nil, err
	}
	k.SetStakingRewardParticipant(ctx, participant)
	k.SetStakingReward(ctx, stakingReward)

	return &types.MsgJoinStakingResponse{}, nil
}
