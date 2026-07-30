package keeper

import (
	"context"
	"strconv"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) CreateBoost(goCtx context.Context, msg *types.MsgCreateBoost) (*types.MsgCreateBoostResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	days, err := strconv.ParseInt(msg.Days, 10, 32)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidBoostDays, "could not convert days to int: %s", err.Error())
	}
	if days <= 0 {
		return nil, types.ErrInvalidBoostDays
	}

	stakingReward, found := k.GetStakingReward(ctx, msg.RewardId)
	if !found {
		return nil, errors.Wrapf(types.ErrInvalidRewardId, "reward with provided id not found")
	}

	if !k.bankKeeper.HasSupply(ctx, msg.Denom) {
		return nil, types.ErrInvalidBoostDenom
	}

	//a boost may never outlive its parent reward's remaining emissions
	if days > int64(stakingReward.Duration)-int64(stakingReward.Payouts) {
		return nil, errors.Wrapf(types.ErrInvalidBoostDays, "days exceed the parent reward's remaining payouts")
	}

	params := k.GetParams(ctx)
	//active and finished boosts both count towards the cap: it bounds settle gas
	if uint32(len(k.GetRewardBoosts(ctx, msg.RewardId))) >= params.MaxBoostsPerReward {
		return nil, types.ErrBoostCapReached
	}

	//guard before getAmountToCapture: sdk.NewCoin panics on negative amounts
	dailyAmount, ok := math.NewIntFromString(msg.DailyAmount)
	if !ok || !dailyAmount.IsPositive() {
		return nil, errors.Wrapf(types.ErrInvalidAmount, "daily amount must be a positive integer")
	}

	toCapture, err := k.getAmountToCapture(msg.Denom, msg.DailyAmount, days)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidAmount, "could not calculate the boost budget: %s", err.Error())
	}
	fee := k.getRewardCreationFee(ctx, params.CreateBoostFee)

	neededBalance := toCapture
	if fee != nil {
		neededBalance = neededBalance.Add(fee...)
	}

	err = k.checkUserBalances(ctx, neededBalance, acc)
	if err != nil {
		return nil, err
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture)
	if err != nil {
		return nil, err
	}

	if fee != nil {
		//boosts with fees can be created only if the trade keeper is available to capture that fee
		if k.tradeKeeper == nil {
			return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "trade keeper is not available")
		}
		capturedFee, err := k.tradeKeeper.CaptureAndSwapUserFee(ctx, acc, fee, types.ModuleName)
		if err != nil {
			return nil, err
		}

		err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, capturedFee)
		if err != nil {
			return nil, err
		}
	}

	boost := types.Boost{
		Id:               k.ReserveBoostId(ctx),
		RewardId:         msg.RewardId,
		Denom:            msg.Denom,
		DailyAmount:      msg.DailyAmount,
		Duration:         uint32(days),
		Payouts:          0,
		DistributedStake: "0",
		Creator:          msg.Creator,
	}
	k.SetBoost(ctx, boost)

	err = ctx.EventManager().EmitTypedEvent(
		&types.BoostCreateEvent{
			RewardId:    boost.RewardId,
			BoostId:     boost.Id,
			Denom:       boost.Denom,
			DailyAmount: boost.DailyAmount,
			Days:        boost.Duration,
			Creator:     boost.Creator,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgCreateBoostResponse{BoostId: boost.Id}, nil
}

func (k msgServer) UpdateBoost(goCtx context.Context, msg *types.MsgUpdateBoost) (*types.MsgUpdateBoostResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	days, err := strconv.ParseInt(msg.Days, 10, 32)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidBoostDays, "could not convert days to int: %s", err.Error())
	}
	if days <= 0 {
		return nil, types.ErrInvalidBoostDays
	}

	stakingReward, found := k.GetStakingReward(ctx, msg.RewardId)
	if !found {
		return nil, errors.Wrapf(types.ErrInvalidRewardId, "reward with provided id not found")
	}

	boost, found := k.GetBoost(ctx, msg.RewardId, msg.BoostId)
	if !found {
		return nil, errors.Wrapf(types.ErrInvalidBoostId, "boost with provided id not found")
	}

	//escrow the extra budget before mutating the schedule
	toCapture, err := k.getAmountToCapture(boost.Denom, boost.DailyAmount, days)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidAmount, "could not calculate the boost budget: %s", err.Error())
	}

	err = k.checkUserBalances(ctx, toCapture, acc)
	if err != nil {
		return nil, err
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture)
	if err != nil {
		return nil, err
	}

	boost.Duration += uint32(days)
	//the extended remainder may never exceed the parent reward's remaining
	//emissions (a finished boost re-arms here: its accumulator continues)
	if int64(boost.Duration)-int64(boost.Payouts) > int64(stakingReward.Duration)-int64(stakingReward.Payouts) {
		return nil, errors.Wrapf(types.ErrInvalidBoostDays, "the extended schedule exceeds the parent reward's remaining payouts")
	}

	k.SetBoost(ctx, boost)

	err = ctx.EventManager().EmitTypedEvent(
		&types.BoostUpdateEvent{
			RewardId: boost.RewardId,
			BoostId:  boost.Id,
			Duration: boost.Duration,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgUpdateBoostResponse{}, nil
}
