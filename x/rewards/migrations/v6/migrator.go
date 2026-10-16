package v6

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TradingRewardStore is the part of the rewards keeper the migration reads and writes.
type TradingRewardStore interface {
	GetAllActiveTradingReward(ctx sdk.Context) []types.TradingReward
	SetActiveTradingReward(ctx sdk.Context, tradingReward types.TradingReward)
	GetTradingRewardLeaderboard(ctx sdk.Context, rewardId string) (types.TradingRewardLeaderboard, bool)
	RemoveActiveTradingRewardExpiration(ctx sdk.Context, expireAt uint32, rewardId string)
}

// Migrate moves the active trading rewards nobody traded on to the consensus version 6 rule: the countdown
// starts at the first trade, not at activation. Such a reward has no leaderboard (the first trade creates it).
// Its expiration entry is removed and its ExpireAt is set to 0, so it waits for its first trade like a reward
// activated after the upgrade. This also covers a reward whose expiry already passed with no trades, which the
// old payout skipped forever. Rewards with a leaderboard already had a real trade and are left as they are.
// Returns the ids of the reset rewards.
func Migrate(ctx sdk.Context, k TradingRewardStore) []string {
	var reset []string
	for _, tr := range k.GetAllActiveTradingReward(ctx) {
		if tr.ExpireAt == 0 {
			continue
		}

		if _, found := k.GetTradingRewardLeaderboard(ctx, tr.RewardId); found {
			continue
		}

		k.RemoveActiveTradingRewardExpiration(ctx, tr.ExpireAt, tr.RewardId)
		tr.ExpireAt = 0
		k.SetActiveTradingReward(ctx, tr)
		reset = append(reset, tr.RewardId)
	}

	return reset
}
