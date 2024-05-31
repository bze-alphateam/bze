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

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
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

	toCapture, err := k.getAmountToCapture("", stakingReward.PrizeDenom, msg.Amount, 1)
	if err != nil {
		return nil, sdkerrors.Wrap(types.ErrInvalidAmount, "could not create capture amount")
	}

	err = k.checkUserBalances(ctx, toCapture, acc)
	if err != nil {
		return nil, sdkerrors.ErrInsufficientFunds
	}

	err = k.distributeStakingRewards(&stakingReward, msg.Amount)
	if err != nil {
		return nil, err
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture)
	if err != nil {
		return nil, err
	}

	k.SetStakingReward(ctx, stakingReward)

	err = ctx.EventManager().EmitTypedEvent(
		&types.StakingRewardDistributionEvent{
			RewardId: stakingReward.RewardId,
			Amount:   msg.Amount,
		},
	)

	if err != nil {
		k.Logger(ctx).Error(err.Error())
	}

	return &types.MsgDistributeStakingRewardsResponse{}, nil
}
