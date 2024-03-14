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

	//capture the total value of prizes
	//the prize amount needs to be multiplied by duration
	err = k.captureStakingRewardCoins(ctx, msg.Creator, stakingReward.PrizeDenom, stakingReward.PrizeAmount, int64(stakingReward.Duration))
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

	stakingReward, isFound := k.GetStakingReward(ctx, msg.RewardId)
	if !isFound {
		return nil, sdkerrors.Wrap(sdkerrors.ErrKeyNotFound, "staking reward not found")
	}

	durationInt, err := strconv.ParseInt(msg.Duration, 10, 32)
	if err != nil {
		return nil, sdkerrors.Wrapf(types.ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}

	if durationInt <= 0 {
		return nil, types.ErrInvalidDuration
	}

	err = k.captureStakingRewardCoins(ctx, msg.Creator, stakingReward.PrizeDenom, stakingReward.PrizeAmount, durationInt)
	if err != nil {
		return nil, err
	}

	stakingReward.Duration += uint32(durationInt)
	k.SetStakingReward(ctx, stakingReward)

	return &types.MsgUpdateStakingRewardResponse{}, nil
}

func (k msgServer) captureStakingRewardCoins(ctx sdk.Context, creator, denom string, amount, duration int64) error {
	acc, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return err
	}

	toCapture := sdk.NewCoin(denom, sdk.NewInt(amount))
	toCapture.Amount.MulRaw(duration)
	if !toCapture.IsPositive() {
		//should never happen
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "calculated amount to capture is not positive")
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, sdk.NewCoins(toCapture))
	if err != nil {
		return err
	}

	return nil
}
