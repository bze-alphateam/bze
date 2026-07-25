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

// CreateBoost creates a scheduled extra-reward program (boost) on top of an
// existing staking reward. It enforces every creation rule from the Business
// Logic page: the reward must exist, the boost denom must have on-chain supply,
// days must fit inside the reward's remaining emissions, no record may already
// exist for (reward_id, denom), and the reward must be below the concurrent cap.
// It charges create_boost_fee (same capture-and-swap flow as the base reward
// creation fee) and escrows daily_amount × days to the module account.
func (k msgServer) CreateBoost(goCtx context.Context, msg *types.MsgCreateBoost) (*types.MsgCreateBoostResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	// the reward the boost is attached to must exist
	stakingReward, found := k.GetStakingReward(ctx, msg.RewardId)
	if !found {
		return nil, errors.Wrapf(types.ErrInvalidRewardId, "reward with provided id not found")
	}

	// the boost denom must have on-chain supply, like base rewards
	if !k.bankKeeper.HasSupply(ctx, msg.Denom) {
		return nil, types.ErrInvalidBoostDenom
	}

	// days must be a positive integer and must fit inside the reward's remaining
	// emissions so the boost never outlives the reward that carries it (I3).
	days, err := strconv.Atoi(msg.Days)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidBoostDays, "could not convert days to int: %s", err.Error())
	}
	if days < 1 {
		return nil, errors.Wrap(types.ErrInvalidBoostDays, "days should be greater than 0")
	}
	remainingDays := int(stakingReward.Duration) - int(stakingReward.Payouts)
	if days > remainingDays {
		return nil, errors.Wrapf(types.ErrInvalidBoostDays, "days (%d) exceed the reward's remaining emissions (%d)", days, remainingDays)
	}

	// uniqueness: any existing record for (reward_id, denom) blocks creation,
	// whether it is still active or finalizing. Finalize first, then recreate.
	if _, exists := k.GetBoost(ctx, msg.RewardId, msg.Denom); exists {
		return nil, errors.Wrapf(types.ErrBoostAlreadyExists, "a boost already exists for reward %s and denom %s", msg.RewardId, msg.Denom)
	}

	// concurrent cap: the reward may hold at most max_boosts_per_reward records.
	params := k.GetParams(ctx)
	existing := k.GetRewardBoosts(ctx, msg.RewardId)
	if uint32(len(existing)) >= params.MaxBoostsPerReward {
		return nil, errors.Wrapf(types.ErrBoostCapReached, "reward %s already has the maximum of %d boosts", msg.RewardId, params.MaxBoostsPerReward)
	}

	// daily_amount must be a positive integer. ValidateBasic already enforces
	// this at the ante stage; we re-check defensively so a malformed amount
	// returns a graceful error instead of panicking in getAmountToCapture.
	if dailyAmount, ok := math.NewIntFromString(msg.DailyAmount); !ok || !dailyAmount.IsPositive() {
		return nil, errors.Wrap(types.ErrInvalidAmount, "daily_amount must be a positive integer")
	}

	// escrow = daily_amount × days, computed with math.Int (overflow-safe).
	toEscrow, err := k.getAmountToCapture(msg.Denom, msg.DailyAmount, int64(days))
	if err != nil {
		return nil, errors.Wrap(types.ErrInvalidAmount, err.Error())
	}

	fee := k.getRewardCreationFee(ctx, params.CreateBoostFee)

	neededBalance := toEscrow
	if fee != nil {
		neededBalance = neededBalance.Add(fee...)
	}

	if err = k.checkUserBalances(ctx, neededBalance, acc); err != nil {
		return nil, err
	}

	// escrow the full boost budget to the shared module account
	if err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toEscrow); err != nil {
		return nil, err
	}

	if fee != nil {
		// boosts with a creation fee can only be created if the trade keeper is
		// available to capture and swap that fee (mirrors CreateStakingReward).
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

	// assign a fresh uid from the global monotonic counter and persist the run.
	uid := k.ReserveBoostUid(ctx)
	boost := types.Boost{
		RewardId:       msg.RewardId,
		Denom:          msg.Denom,
		Uid:            uid,
		DailyAmount:    msg.DailyAmount,
		DaysLeft:       uint32(days),
		SBoost:         "0",
		FinalizeCursor: "",
		Creator:        msg.Creator,
	}
	k.SetBoost(ctx, boost)

	err = ctx.EventManager().EmitTypedEvent(
		&types.BoostCreateEvent{
			RewardId:    boost.RewardId,
			Denom:       boost.Denom,
			Uid:         boost.Uid,
			DailyAmount: boost.DailyAmount,
			Days:        boost.DaysLeft,
			Creator:     boost.Creator,
		},
	)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgCreateBoostResponse{Uid: uid}, nil
}
