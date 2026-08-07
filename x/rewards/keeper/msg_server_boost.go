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

	//check the budget only: the fee may be paid in the user's preferred denom via the trade keeper's swap,
	//so requiring it in the params denom here would wrongly reject valid users
	err = k.checkUserBalances(ctx, toCapture, acc)
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

	//a stale cursor on a re-armed boost would make a later cleanup resume
	//mid-way, skip the already-swept participants' new accrual segment and
	//then delete the record — silent loss. Restart the sweep instead:
	//already-stamped participants just settle the delta (stamps only advance)
	boost.CleanupCursor = ""

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

// CleanupBoost pays out a finished boost's remaining entitlements in batches
// and deletes its record when the sweep completes, freeing a boost slot while
// the parent reward is still running. Permissionless and fee-free: gas is the
// payment, and a fee would disincentivize the cleanup the module wants.
func (k msgServer) CleanupBoost(goCtx context.Context, msg *types.MsgCleanupBoost) (*types.MsgCleanupBoostResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return nil, err
	}

	boost, found := k.GetBoost(ctx, msg.RewardId, msg.BoostId)
	if !found {
		return nil, errors.Wrapf(types.ErrInvalidBoostId, "boost with provided id not found")
	}

	//an active boost may never be cleaned up: deleting it would strand its
	//future accrual and escrow
	if boost.Payouts < boost.Duration {
		return nil, errors.Wrapf(types.ErrBoostNotFinished, "the boost still has scheduled payouts")
	}

	//the param is the ceiling, never the caller: an unclamped huge limit
	//would run out of block gas and revert, wasting the whole call's work
	limit := k.GetParams(ctx).CleanupBatchSize
	if msg.Limit != 0 && msg.Limit < limit {
		limit = msg.Limit
	}

	sFinal, err := math.LegacyNewDecFromStr(boost.DistributedStake)
	if err != nil {
		return nil, err
	}

	addresses := k.GetStakingRewardParticipantIndexAddresses(ctx, msg.RewardId, boost.CleanupCursor, limit)
	for _, address := range addresses {
		if err = k.sweepBoostParticipant(ctx, boost, sFinal, address); err != nil {
			return nil, err
		}
	}

	//fewer addresses than asked for means the iteration is exhausted; an
	//exact-limit batch ending on the last entry completes on the next call
	//(a no-op success) — idempotent by construction
	completed := uint32(len(addresses)) < limit
	if completed {
		k.RemoveBoost(ctx, boost.RewardId, boost.Id)

		err = ctx.EventManager().EmitTypedEvent(
			&types.BoostCleanupEvent{
				RewardId: boost.RewardId,
				BoostId:  boost.Id,
			},
		)

		if err != nil {
			k.Logger().Error(err.Error())
		}
	} else {
		boost.CleanupCursor = addresses[len(addresses)-1]
		k.SetBoost(ctx, boost)
	}

	return &types.MsgCleanupBoostResponse{Processed: uint32(len(addresses)), Completed: completed}, nil
}
