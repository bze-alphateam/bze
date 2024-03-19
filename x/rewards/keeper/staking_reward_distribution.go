package keeper

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetDistributeAllStakingRewardsHook() types.EpochHook {
	hookName := "distribution_hook"
	return types.NewBeforeEpochHook(hookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		k.Logger(ctx).
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", hookName).
			Debug("preparing to execute hook")

		k.DistributeAllStakingRewards(ctx)

		return nil
	})
}

func (k Keeper) DistributeAllStakingRewards(ctx sdk.Context) {
	k.IterateAllStakingRewards(ctx, k.getDistributeRewardHandler())
}

func (k Keeper) getDistributeRewardHandler() func(ctx sdk.Context, reward types.StakingReward) (stop bool) {
	return func(ctx sdk.Context, sr types.StakingReward) (stop bool) {
		logger := k.Logger(ctx).With("reward_id", sr.RewardId, "type", "staking")
		if sr.Payouts >= sr.Duration {
			logger.Debug("stake finished. skipping distribution")
			stop = false
			return
		}

		stakedAmount, ok := sdk.NewIntFromString(sr.StakedAmount)
		if !ok {
			logger.Error("could not transform staked amount from storage into int")
			stop = true
			return
		}

		if !stakedAmount.IsPositive() {
			logger.Debug("no stakers found. skipping distribution")
			stop = false
			return
		}

		err := k.distributeStakingRewards(&sr, sr.PrizeAmount)
		if err != nil {
			logger.Error(err.Error())
			stop = true
			return
		}

		//increment payouts to skip when the reward finished (a.k.a. all payouts calculated)
		sr.Payouts++
		k.SetStakingReward(ctx, sr)

		return
	}
}

func (k Keeper) distributeStakingRewards(sr *types.StakingReward, rewardAmount string) error {
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

	return nil
}
