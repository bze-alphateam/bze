package keeper

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) DistributeAllStakingRewards(ctx sdk.Context) {
	k.IterateAllStakingRewards(ctx, k.getDistributeRewardHandler())
}

func (k Keeper) getDistributeRewardHandler() func(ctx sdk.Context, reward types.StakingReward) (stop bool) {
	return func(ctx sdk.Context, sr types.StakingReward) (stop bool) {
		logger := k.Logger(ctx).With("staking_reward", sr)

		logger.Debug("preparing to distribute staking reward")

		if sr.Payouts >= sr.Duration {
			logger.Debug("staking reward finished. skipping distribution")
			stop = false
			return
		}

		err := k.distributeStakingRewards(ctx, &sr, sr.PrizeAmount)
		if err != nil {
			logger.Error(err.Error())
			stop = false
			return
		}

		//increment payouts to know when the reward finished (a.k.a. all payouts calculated)
		sr.Payouts++
		k.SetStakingReward(ctx, sr)

		logger.Debug("staking reward distributed")

		return
	}
}

func (k Keeper) distributeStakingRewards(ctx sdk.Context, sr *types.StakingReward, rewardAmount string) error {
	stakedAmount, ok := sdk.NewIntFromString(sr.StakedAmount)
	if !ok {
		return fmt.Errorf("could not transform staked amount from storage into int")
	}

	if !stakedAmount.IsPositive() {
		return fmt.Errorf("no stakers found")
	}

	reward, ok := sdk.NewIntFromString(rewardAmount)
	if !ok {
		return fmt.Errorf("could not transform reward amount to int")
	}

	if !reward.IsPositive() {
		return fmt.Errorf("reward amount should be positive")
	}

	sFloat, err := sdk.NewDecFromStr(sr.DistributedStake)
	if err != nil {
		return err
	}

	//S = S + r / T;
	sFloat = sFloat.Add(reward.ToDec().Quo(stakedAmount.ToDec()))
	sr.DistributedStake = sFloat.String()

	err = ctx.EventManager().EmitTypedEvent(
		&types.StakingRewardDistributionEvent{
			RewardId: sr.RewardId,
			Amount:   rewardAmount,
		},
	)

	if err != nil {
		k.Logger(ctx).Error(err.Error())
	}

	return nil
}
