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
		_, err = k.claimPending(ctx, stakingReward, &participant)
		if err != nil {
			return nil, err
		}
	} else {
		participant = types.StakingRewardParticipant{
			Address:  msg.Creator,
			RewardId: msg.RewardId,
			Amount:   "0",
		}
	}
	participant.JoinedAt = stakingReward.DistributedStake

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

// claimPending - sends the pending rewards to the participant and updates the participant.JoinedAt field with current
// StakingReward.DistributedStake
func (k msgServer) claimPending(ctx sdk.Context, sr types.StakingReward, participant *types.StakingRewardParticipant) (*sdk.Coin, error) {
	deposited, ok := sdk.NewIntFromString(participant.Amount)
	if !ok {
		return nil, fmt.Errorf("could not transform participant amount from storage into int")
	}
	distributedStake, ok := sdk.NewIntFromString(sr.DistributedStake)
	if !ok {
		return nil, fmt.Errorf("could not transform distributed stake from storage into int")
	}
	joinedAt, ok := sdk.NewIntFromString(participant.JoinedAt)
	if !ok {
		return nil, fmt.Errorf("could not transform joined at from storage into int")
	}

	reward := deposited.Mul(distributedStake.Sub(joinedAt))
	if !reward.IsPositive() {
		return nil, fmt.Errorf("no rewards to claim")
	}

	acc, err := sdk.AccAddressFromBech32(participant.Address)
	if err != nil {
		return nil, err
	}

	toSend := sdk.NewCoin(sr.PrizeDenom, reward)
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, sdk.NewCoins(toSend))
	if err != nil {
		return nil, err
	}

	participant.JoinedAt = sr.DistributedStake

	return &toSend, nil
}
