package keeper

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// distributeBoosts advances every unfinished boost of the reward by one day:
// distributed_stake += daily_amount / T, where T is the parent's LIVE staked
// amount read at call time — a boost stores no copy of T. Finished boosts
// (payouts == duration) stay in store untouched so accrued amounts remain
// claim-servable through settle.
//
// Skip parity (exploit A6): call sites invoke this ONLY on a day the base
// reward actually distributed, so a skipped base day means boosts neither
// advance nor consume a payout. The non-positive-T check here is defensive
// only — the caller already guarantees T > 0.
func (k Keeper) distributeBoosts(ctx sdk.Context, sr *types.StakingReward) {
	boosts := k.GetRewardBoosts(ctx, sr.RewardId)
	if len(boosts) == 0 {
		return
	}

	stakedAmount, err := math.LegacyNewDecFromStr(sr.StakedAmount)
	if err != nil {
		k.Logger().Error(err.Error())
		return
	}

	if !stakedAmount.IsPositive() {
		return
	}

	for _, boost := range boosts {
		if boost.Payouts >= boost.Duration {
			continue
		}

		dailyAmount, err := math.LegacyNewDecFromStr(boost.DailyAmount)
		if err != nil {
			k.Logger().Error(err.Error())
			continue
		}

		sFloat, err := math.LegacyNewDecFromStr(boost.DistributedStake)
		if err != nil {
			k.Logger().Error(err.Error())
			continue
		}

		//S = S + r / T;
		boost.DistributedStake = sFloat.Add(dailyAmount.Quo(stakedAmount)).String()
		boost.Payouts++
		k.SetBoost(ctx, boost)

		err = ctx.EventManager().EmitTypedEvent(
			&types.BoostDistributionEvent{
				RewardId: boost.RewardId,
				BoostId:  boost.Id,
				Denom:    boost.Denom,
				Amount:   boost.DailyAmount,
			},
		)

		if err != nil {
			k.Logger().Error(err.Error())
		}
	}
}
