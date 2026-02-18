package keeper

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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

	toCapture, err := k.getAmountToCapture(stakingReward.PrizeDenom, stakingReward.PrizeAmount, int64(stakingReward.Duration))
	if err != nil {
		return nil, errors.Wrapf(err, "could not calculate amount needed to create the reward")
	}
	fee := k.getRewardCreationFee(ctx, k.GetParams(ctx).CreateStakingRewardFee)

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
		capturedFee, err := k.tradeKeeper.CaptureAndSwapUserFee(ctx, acc, fee, types.ModuleName)
		if err != nil {
			return nil, err
		}

		err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, capturedFee)
		if err != nil {
			return nil, err
		}
	}

	//add ID
	stakingReward.RewardId = k.smallZeroFillId(k.GetStakingRewardsCounter(ctx))
	k.SetStakingReward(
		ctx,
		stakingReward,
	)
	k.incrementStakingRewardsCounter(ctx)

	err = ctx.EventManager().EmitTypedEvent(
		&types.StakingRewardCreateEvent{
			RewardId:     stakingReward.RewardId,
			PrizeAmount:  stakingReward.PrizeAmount,
			PrizeDenom:   stakingReward.PrizeDenom,
			StakingDenom: stakingReward.StakingDenom,
			Duration:     stakingReward.Duration,
			MinStake:     stakingReward.MinStake,
			Lock:         stakingReward.Lock,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

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
		return nil, errors.Wrapf(types.ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}

	if durationInt <= 0 {
		return nil, types.ErrInvalidDuration
	}

	stakingReward, isFound := k.GetStakingReward(ctx, msg.RewardId)
	if !isFound {
		return nil, errors.Wrap(sdkerrors.ErrKeyNotFound, "staking reward not found")
	}

	toCapture, err := k.getAmountToCapture(stakingReward.PrizeDenom, stakingReward.PrizeAmount, durationInt)
	if err != nil {
		return nil, errors.Wrapf(err, "could not calculate amount needed to create the reward")
	}

	err = k.checkUserBalances(ctx, toCapture, acc)
	if err != nil {
		return nil, err
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture)
	if err != nil {
		return nil, err
	}

	stakingReward.Duration += uint32(durationInt)
	if stakingReward.Duration > types.HundredYearsInDays {
		return nil, errors.Wrapf(types.ErrInvalidDuration, "the new duration exceeds the maximum allowed of %d days", types.HundredYearsInDays)
	}

	k.SetStakingReward(ctx, stakingReward)

	err = ctx.EventManager().EmitTypedEvent(
		&types.StakingRewardUpdateEvent{
			RewardId: stakingReward.RewardId,
			Duration: stakingReward.Duration,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgUpdateStakingRewardResponse{}, nil
}

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
		return nil, errors.Wrapf(types.ErrInvalidRewardId, "reward with provided id not found")
	}

	stakedAmount := math.ZeroInt()
	if stakingReward.StakedAmount != "" {
		ok := false
		stakedAmount, ok = math.NewIntFromString(stakingReward.StakedAmount)
		if !ok {
			return nil, fmt.Errorf("could not transform staked amount from storage into int")
		}
	}

	toCapture, err := k.getAmountToCapture(stakingReward.StakingDenom, msg.Amount, int64(1))
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

	amtInt, ok := math.NewIntFromString(participant.Amount)
	if !ok {
		return nil, fmt.Errorf("could not transform amount from storage into int")
	}
	amtInt = amtInt.Add(toCapture.AmountOf(stakingReward.StakingDenom))

	//check if min stake requirement is met
	if amtInt.LT(math.NewIntFromUint64(stakingReward.MinStake)) {
		return nil, fmt.Errorf("amount is smaller than staking reward min stake")
	}

	participant.Amount = amtInt.String()

	stakedAmount = stakedAmount.Add(toCapture.AmountOf(stakingReward.StakingDenom))
	stakingReward.StakedAmount = stakedAmount.String()

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture)
	if err != nil {
		return nil, err
	}
	k.SetStakingRewardParticipant(ctx, participant)
	k.SetStakingReward(ctx, stakingReward)

	err = ctx.EventManager().EmitTypedEvent(
		&types.StakingRewardJoinEvent{
			RewardId: stakingReward.RewardId,
			Address:  msg.Creator,
			Amount:   toCapture.AmountOf(stakingReward.StakingDenom).String(),
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgJoinStakingResponse{}, nil
}

func (k msgServer) ExitStaking(goCtx context.Context, msg *types.MsgExitStaking) (*types.MsgExitStakingResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := k.GetParams(ctx)
	ctx.GasMeter().ConsumeGas(params.ExtraGasForExitStake, "exit_stake_extra_gas")

	stakingReward, found := k.GetStakingReward(ctx, msg.RewardId)
	if !found {
		return nil, errors.Wrapf(types.ErrInvalidRewardId, "reward with provided id not found")
	}

	participation, found := k.GetStakingRewardParticipant(ctx, msg.Creator, msg.RewardId)
	if !found {
		return nil, errors.Wrapf(types.ErrInvalidRewardId, "you are not a participant in this staking reward")
	}

	partCoins, err := k.getAmountToCapture(stakingReward.StakingDenom, participation.Amount, int64(1))
	if err != nil {
		return nil, err
	}
	stakedAmountInt, ok := math.NewIntFromString(stakingReward.StakedAmount)
	if !ok {
		return nil, fmt.Errorf("could not transform amount from storage into int")
	}
	if !stakedAmountInt.IsPositive() {
		//disaster in this case
		return nil, fmt.Errorf("no staked amount left")
	}

	//send pending rewards
	_, err = k.claimPending(ctx, stakingReward, &participation)
	if err != nil {
		return nil, err
	}

	err = k.beginUnlock(ctx, participation, stakingReward)
	if err != nil {
		return nil, err
	}

	k.RemoveStakingRewardParticipant(ctx, participation.Address, participation.RewardId)

	remainingStakedAmount := stakedAmountInt.Sub(partCoins.AmountOf(stakingReward.StakingDenom))
	stakingReward.StakedAmount = remainingStakedAmount.String()
	k.SetStakingReward(ctx, stakingReward)

	//if this staking reward is finished (all funds were distributed and payouts executed) we should remove it
	if remainingStakedAmount.IsZero() && stakingReward.Payouts >= stakingReward.Duration {
		k.RemoveStakingReward(ctx, stakingReward.RewardId)
		err = ctx.EventManager().EmitTypedEvent(
			&types.StakingRewardFinishEvent{
				RewardId: stakingReward.RewardId,
			},
		)

		if err != nil {
			k.Logger().Error(err.Error())
		}
	}

	err = ctx.EventManager().EmitTypedEvent(
		&types.StakingRewardExitEvent{
			RewardId: stakingReward.RewardId,
			Address:  msg.Creator,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgExitStakingResponse{}, nil
}

func (k msgServer) ClaimStakingRewards(goCtx context.Context, msg *types.MsgClaimStakingRewards) (*types.MsgClaimStakingRewardsResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	stakingReward, found := k.GetStakingReward(ctx, msg.RewardId)
	if !found {
		return nil, errors.Wrapf(types.ErrInvalidRewardId, "reward with provided id not found")
	}

	participant, found := k.GetStakingRewardParticipant(ctx, msg.Creator, msg.RewardId)
	if !found {
		return nil, errors.Wrapf(types.ErrInvalidRewardId, "you are not a participant in this staking reward")
	}

	paid, err := k.claimPending(ctx, stakingReward, &participant)
	if err != nil {
		return nil, err
	}

	k.SetStakingRewardParticipant(ctx, participant)

	err = ctx.EventManager().EmitTypedEvent(
		&types.StakingRewardClaimEvent{
			RewardId: stakingReward.RewardId,
			Address:  msg.Creator,
			Amount:   paid.Amount.String(),
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgClaimStakingRewardsResponse{Amount: paid.Amount.String()}, nil
}

func (k msgServer) DistributeStakingRewards(goCtx context.Context, msg *types.MsgDistributeStakingRewards) (*types.MsgDistributeStakingRewardsResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	amtInt, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, errors.Wrapf(types.ErrInvalidAmount, "could not convert order amount")
	}

	if !amtInt.IsPositive() {
		return nil, errors.Wrapf(types.ErrInvalidAmount, "amount should be greater than 0")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	stakingReward, isFound := k.GetStakingReward(ctx, msg.RewardId)
	if !isFound {
		return nil, errors.Wrap(sdkerrors.ErrKeyNotFound, "staking reward not found")
	}

	toCapture, err := k.getAmountToCapture(stakingReward.PrizeDenom, msg.Amount, 1)
	if err != nil {
		return nil, errors.Wrap(types.ErrInvalidAmount, "could not create capture amount")
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
		k.Logger().Error(err.Error())
	}

	return &types.MsgDistributeStakingRewardsResponse{}, nil
}

func (k msgServer) getRewardCreationFee(_ sdk.Context, feeParam sdk.Coin) sdk.Coins {
	if !feeParam.IsPositive() {
		return nil
	}

	//just avoid any accidental panic
	if !feeParam.IsValid() {
		k.Logger().Error("invalid reward creation fee", "feeParam", feeParam)

		return nil
	}

	return sdk.NewCoins(feeParam)
}

func (k msgServer) checkUserBalances(ctx sdk.Context, neededCoins sdk.Coins, address sdk.AccAddress) error {
	spendable := k.bankKeeper.SpendableCoins(ctx, address)
	if !spendable.IsAllGTE(neededCoins) {
		return fmt.Errorf("user balance is too low")
	}

	return nil
}

// claimPending - sends the pending rewards to the participant and updates the participant.JoinedAt field with current
// StakingReward.DistributedStake
func (k msgServer) claimPending(ctx sdk.Context, sr types.StakingReward, participant *types.StakingRewardParticipant) (*sdk.Coin, error) {
	deposited, err := math.LegacyNewDecFromStr(participant.Amount)
	if err != nil {
		return nil, err
	}
	distributedStake, err := math.LegacyNewDecFromStr(sr.DistributedStake)
	if err != nil {
		return nil, err
	}
	joinedAt, err := math.LegacyNewDecFromStr(participant.JoinedAt)
	if err != nil {
		return nil, err
	}

	zeroCoins := sdk.NewCoin(sr.PrizeDenom, math.NewInt(0))
	//user has nothing to claim
	if distributedStake.Equal(joinedAt) {
		return &zeroCoins, nil
	}

	//the user might have a small amount to claim, like 0.01 ubze. We can't send him this reward, but we must NOT
	// update his JoinedAt because that will make him lose funds.
	// 1. so we check if the decimal is positive, if not return an error
	rewardDec := deposited.Mul(distributedStake.Sub(joinedAt))
	if !rewardDec.IsPositive() {
		return nil, fmt.Errorf("no rewards to claim")
	}

	reward := rewardDec.TruncateInt()
	//2. if the previous "if" statement was false, it means the reward is bigger than 0.
	//we truncate it to get the amount we can actually send, which should be an int.
	if !reward.IsPositive() {
		//truncation of the decimal resulted in a number <= 0.
		//this means he has nothing to claim.
		return &zeroCoins, nil
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

func (k msgServer) beginUnlock(ctx sdk.Context, p types.StakingRewardParticipant, sr types.StakingReward) error {
	lockedUntil := k.epochKeeper.GetEpochCountByIdentifier(ctx, expirationEpoch)
	lockedUntil += int64(sr.Lock) * 24
	pendingKey := types.CreatePendingUnlockParticipantKey(lockedUntil, fmt.Sprintf("%s/%s", sr.RewardId, p.Address))
	pending := types.PendingUnlockParticipant{
		Index:   pendingKey,
		Address: p.Address,
		Amount:  p.Amount,
		Denom:   sr.StakingDenom,
	}

	inStore, found := k.GetPendingUnlockParticipant(ctx, pendingKey)
	if found {
		//we already have a pending unlock for this reward and participant at the same epoch
		//update the amount, so it can all be unlocked at once
		inStoreAmount, _ := math.NewIntFromString(inStore.Amount)
		pendingAmount, _ := math.NewIntFromString(pending.Amount)
		pending.Amount = pendingAmount.Add(inStoreAmount).String()
	}

	//in case the lock is 0 send the funds immediately
	if sr.Lock == 0 {
		return k.performUnlock(ctx, &pending)
	}

	k.SetPendingUnlockParticipant(ctx, pending)

	return nil
}
