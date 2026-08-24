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

// scheduleDaysMaxBitLen is the bit length that bounds every valid day count (duration or extension):
// HundredYearsInDays = 36500 < 2^16.
const scheduleDaysMaxBitLen = 16

// getScheduleBudget computes the escrow for `days` payouts of dailyInt in the prize denom.
// math.Int multiplication PANICS past 256 bits instead of erroring, and the daily amount is
// user-supplied and may be parsed right up to that limit — so the product is bounds-checked first:
// days is at most HundredYearsInDays (< 2^16), making the product provably safe whenever
// dailyInt fits in the remaining bits.
func (k msgServer) getScheduleBudget(prizeDenom string, dailyInt math.Int, days int64) (sdk.Coins, error) {
	if dailyInt.IsNil() || !dailyInt.IsPositive() {
		return nil, errors.Wrapf(types.ErrInvalidAmount, "daily_amount should be greater than 0")
	}

	if dailyInt.BigInt().BitLen()+scheduleDaysMaxBitLen > math.MaxBitLen {
		return nil, errors.Wrapf(types.ErrInvalidAmount, "daily_amount is too large")
	}

	return denomAmountToCapture(prizeDenom, dailyInt, days)
}

// CreateDenomRewardSchedule attaches a daily-payout campaign to an existing denom reward.
//
// Attaching is permissionless (Business Logic rule 13): anyone may fund any DR, and no creator
// identity is stored on the schedule. The caller pays, in one tx: the DR-prize creation fee IF the
// prize denom is new to the DR (via ensureDenomRewardPrize — the paid, capped slot event), the flat
// schedule fee, and the FULL budget daily_amount × duration, escrowed up front so every future
// payout is already funded. The daily distribution pass that consumes the schedule arrives in a
// later story; the schedule is stored with payouts = 0 until then.
func (k msgServer) CreateDenomRewardSchedule(goCtx context.Context, msg *types.MsgCreateDenomRewardSchedule) (*types.MsgCreateDenomRewardScheduleResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	dr, found := k.GetDenomReward(ctx, msg.Denom)
	if !found {
		return nil, types.ErrDenomRewardNotFound
	}

	if !k.bankKeeper.HasSupply(ctx, msg.PrizeDenom) {
		return nil, types.ErrInvalidPrizeDenom
	}

	durationInt, err := strconv.ParseInt(msg.Duration, 10, 32)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}
	if durationInt <= 0 || durationInt > types.HundredYearsInDays {
		return nil, errors.Wrapf(types.ErrInvalidDuration, "duration should be between 1 and %d days", types.HundredYearsInDays)
	}

	budget, err := k.getScheduleBudget(msg.PrizeDenom, msg.DailyAmount, durationInt)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	fee := k.getRewardCreationFee(ctx, params.AddDenomRewardScheduleFee)

	// the balance is pre-checked for budget + schedule fee together (mirrors CreateStakingReward);
	// ensureDenomRewardPrize pre-checks the prize fee on its own when the prize denom is new
	neededBalance := budget
	if fee != nil {
		neededBalance = neededBalance.Add(fee...)
	}
	if err = k.checkUserBalances(ctx, neededBalance, acc); err != nil {
		return nil, err
	}

	// existing prize denom → returned free; new prize denom → cap check + prize creation fee
	if _, err = k.ensureDenomRewardPrize(ctx, dr.StakingDenom, msg.PrizeDenom, acc); err != nil {
		return nil, err
	}

	if fee != nil {
		// the fee can be captured only if the trade keeper is available to swap it
		if k.tradeKeeper == nil {
			return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "trade keeper is not available")
		}
		capturedFee, err := k.tradeKeeper.CaptureAndSwapUserFee(ctx, acc, fee, types.ModuleName)
		if err != nil {
			return nil, err
		}

		if err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, capturedFee); err != nil {
			return nil, err
		}
	}

	// escrow the full budget up front
	if err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, budget); err != nil {
		return nil, err
	}

	schedule := types.DenomRewardSchedule{
		ScheduleId:   k.smallZeroFillId(k.GetDenomRewardScheduleCounter(ctx)),
		StakingDenom: dr.StakingDenom,
		PrizeDenom:   msg.PrizeDenom,
		DailyAmount:  msg.DailyAmount,
		Duration:     uint32(durationInt),
		Payouts:      0,
	}
	k.SetDenomRewardSchedule(ctx, schedule)
	k.incrementDenomRewardScheduleCounter(ctx)

	err = ctx.EventManager().EmitTypedEvent(
		&types.DenomRewardScheduleCreateEvent{
			ScheduleId:  schedule.ScheduleId,
			Denom:       schedule.StakingDenom,
			PrizeDenom:  schedule.PrizeDenom,
			DailyAmount: schedule.DailyAmount.String(),
			Duration:    schedule.Duration,
		},
	)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgCreateDenomRewardScheduleResponse{ScheduleId: schedule.ScheduleId}, nil
}

// UpdateDenomRewardSchedule extends a schedule by `duration` EXTRA days (mirrors
// UpdateStakingReward). Extension is permissionless and pays no fee beyond the extra budget
// daily_amount × extra_days, which is escrowed BEFORE the duration is mutated; the post-extension
// duration is still bound by HundredYearsInDays (Business Logic rule 16), and a bound violation
// aborts the tx so the escrow transfer is rolled back with it.
func (k msgServer) UpdateDenomRewardSchedule(goCtx context.Context, msg *types.MsgUpdateDenomRewardSchedule) (*types.MsgUpdateDenomRewardScheduleResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	extraDays, err := strconv.ParseInt(msg.Duration, 10, 32)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}
	if extraDays <= 0 {
		return nil, types.ErrInvalidDuration
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	schedule, found := k.GetDenomRewardSchedule(ctx, msg.Denom, msg.ScheduleId)
	if !found {
		return nil, errors.Wrap(sdkerrors.ErrKeyNotFound, "denom reward schedule not found")
	}

	toCapture, err := k.getScheduleBudget(schedule.PrizeDenom, schedule.DailyAmount, extraDays)
	if err != nil {
		return nil, err
	}

	if err = k.checkUserBalances(ctx, toCapture, acc); err != nil {
		return nil, err
	}

	// escrow the extra budget BEFORE mutating the duration
	if err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture); err != nil {
		return nil, err
	}

	schedule.Duration += uint32(extraDays)
	if schedule.Duration > types.HundredYearsInDays {
		return nil, errors.Wrapf(types.ErrInvalidDuration, "the new duration exceeds the maximum allowed of %d days", types.HundredYearsInDays)
	}

	k.SetDenomRewardSchedule(ctx, schedule)

	err = ctx.EventManager().EmitTypedEvent(
		&types.DenomRewardScheduleUpdateEvent{
			ScheduleId: schedule.ScheduleId,
			Duration:   schedule.Duration,
		},
	)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgUpdateDenomRewardScheduleResponse{}, nil
}
