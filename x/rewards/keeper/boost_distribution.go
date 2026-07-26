package keeper

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// distributeBoosts advances the boost accumulators of a staking reward for one
// distributed day: s_boost += daily_amount / T and days_left -= 1 for every
// active boost. It must be called only on days the base reward actually
// distributed (skip parity): a skipped day neither advances s_boost nor
// consumes a day of the boost budget.
func (k Keeper) distributeBoosts(ctx sdk.Context, sr *types.StakingReward) {
	logger := k.Logger().With("staking_reward", sr.RewardId)

	stakedAmount, err := math.LegacyNewDecFromStr(sr.StakedAmount)
	if err != nil {
		logger.Error("could not parse staked amount for boost distribution", "error", err)
		return
	}

	// the call site guarantees T > 0, but never divide by a non-positive T;
	// skip without decrementing days_left so no budget is stranded
	if !stakedAmount.IsPositive() {
		return
	}

	for _, boost := range k.GetRewardBoosts(ctx, sr.RewardId) {
		if boost.DaysLeft == 0 {
			// finalizing boost: emissions done, awaiting cleanup
			continue
		}

		daily, err := math.LegacyNewDecFromStr(boost.DailyAmount)
		if err != nil {
			logger.Error("could not parse boost daily amount", "denom", boost.Denom, "error", err)
			continue
		}

		sBoost, err := math.LegacyNewDecFromStr(boost.SBoost)
		if err != nil {
			logger.Error("could not parse boost accumulator", "denom", boost.Denom, "error", err)
			continue
		}

		//s_boost = s_boost + daily_amount / T
		sBoost = sBoost.Add(daily.Quo(stakedAmount))
		boost.SBoost = sBoost.String()
		boost.DaysLeft--
		k.SetBoost(ctx, boost)

		err = ctx.EventManager().EmitTypedEvent(
			&types.BoostDistributionEvent{
				RewardId: boost.RewardId,
				Denom:    boost.Denom,
				Uid:      boost.Uid,
				Amount:   boost.DailyAmount,
			},
		)

		if err != nil {
			k.Logger().Error(err.Error())
		}
	}
}
