package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
)

func (k msgServer) CreateStakingReward(goCtx context.Context, msg *types.MsgCreateStakingReward) (*types.MsgCreateStakingRewardResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	stakingReward, err := msg.ToStakingReward()
	if err != nil {
		return nil, err
	}

	//check denoms
	ok := k.bankKeeper.HasSupply(ctx, stakingReward.StakingDenom)
	if !ok {
		return nil, types.ErrInvalidStakingDenom
	}
	ok = k.bankKeeper.HasSupply(ctx, stakingReward.PrizeDenom)
	if !ok {
		return nil, types.ErrInvalidPrizeDenom
	}

	feeParam := k.GetParams(ctx).CreateStakingRewardFee
	toCapture, err := k.getAmountToCapture(feeParam, stakingReward.PrizeDenom, stakingReward.PrizeAmount, int64(stakingReward.Duration))
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "could not calculate amount needed to create the reward")
	}

	err = k.checkUserBalances(ctx, toCapture, acc)
	if err != nil {
		return nil, err
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture)
	if err != nil {
		return nil, err
	}

	//add ID
	stakingReward.RewardId = k.smallZeroFillId(k.GetStakingRewardsCounter(ctx))
	k.SetStakingReward(
		ctx,
		stakingReward,
	)

	return &types.MsgCreateStakingRewardResponse{RewardId: stakingReward.RewardId}, nil
}

func (k msgServer) UpdateStakingReward(goCtx context.Context, msg *types.MsgUpdateStakingReward) (*types.MsgUpdateStakingRewardResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	durationInt, err := strconv.ParseInt(msg.Duration, 10, 32)
	if err != nil {
		return nil, sdkerrors.Wrapf(types.ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}

	if durationInt <= 0 {
		return nil, types.ErrInvalidDuration
	}

	stakingReward, isFound := k.GetStakingReward(ctx, msg.RewardId)
	if !isFound {
		return nil, sdkerrors.Wrap(sdkerrors.ErrKeyNotFound, "staking reward not found")
	}

	toCapture, err := k.getAmountToCapture("", stakingReward.PrizeDenom, stakingReward.PrizeAmount, durationInt)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "could not calculate amount needed to create the reward")
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture)
	if err != nil {
		return nil, err
	}

	stakingReward.Duration += uint32(durationInt)
	k.SetStakingReward(ctx, stakingReward)

	return &types.MsgUpdateStakingRewardResponse{}, nil
}
