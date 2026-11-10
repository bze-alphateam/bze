package v830

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

// RewardsKeeper is the part of the rewards keeper the handler uses.
type RewardsKeeper interface {
	GetAllStakingReward(ctx sdk.Context) []rewardstypes.StakingReward
	SetStakingReward(ctx sdk.Context, stakingReward rewardstypes.StakingReward)
}

// normaliseStakingRewardAmounts rewrites every stored staking reward prize amount that is not a canonical base-10
// integer to canonical form. From v8.3.0 the rewards module accepts amounts only in that form. A value that is not an
// integer at all is logged and left as it is.
func normaliseStakingRewardAmounts(ctx sdk.Context, rewards RewardsKeeper) {
	rewritten, skipped := 0, 0
	for _, sr := range rewards.GetAllStakingReward(ctx) {
		if _, err := rewardstypes.ParseCanonicalInt(sr.PrizeAmount); err == nil {
			continue
		}

		amt, ok := math.NewIntFromString(sr.PrizeAmount)
		if !ok {
			ctx.Logger().Error("staking reward prize amount is not an integer, skipped", "reward_id", sr.RewardId, "prize_amount", sr.PrizeAmount)
			skipped++
			continue
		}

		previous := sr.PrizeAmount
		sr.PrizeAmount = amt.String()
		rewards.SetStakingReward(ctx, sr)
		ctx.Logger().Info("normalised staking reward prize amount", "reward_id", sr.RewardId, "from", previous, "to", sr.PrizeAmount)
		rewritten++
	}

	ctx.Logger().Info("staking reward prize amounts normalised", "rewritten", rewritten, "skipped", skipped)
}
