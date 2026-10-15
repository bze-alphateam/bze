package v6_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	v6 "github.com/bze-alphateam/bze/x/rewards/migrations/v6"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

type expirationKey struct {
	expireAt uint32
	rewardId string
}

// fakeStore is an in-memory TradingRewardStore.
type fakeStore struct {
	rewards      map[string]types.TradingReward
	leaderboards map[string]types.TradingRewardLeaderboard
	expirations  map[expirationKey]bool
}

func (f *fakeStore) GetAllActiveTradingReward(_ sdk.Context) []types.TradingReward {
	var list []types.TradingReward
	for _, id := range []string{"no-trades-expired", "no-trades-running", "traded", "waiting"} {
		if tr, ok := f.rewards[id]; ok {
			list = append(list, tr)
		}
	}

	return list
}

func (f *fakeStore) SetActiveTradingReward(_ sdk.Context, tr types.TradingReward) {
	f.rewards[tr.RewardId] = tr
}

func (f *fakeStore) GetTradingRewardLeaderboard(_ sdk.Context, rewardId string) (types.TradingRewardLeaderboard, bool) {
	l, ok := f.leaderboards[rewardId]
	return l, ok
}

func (f *fakeStore) RemoveActiveTradingRewardExpiration(_ sdk.Context, expireAt uint32, rewardId string) {
	delete(f.expirations, expirationKey{expireAt, rewardId})
}

func TestMigrate_ResetsOnlyUntradedRewards(t *testing.T) {
	reward := func(id string, expireAt uint32) types.TradingReward {
		return types.TradingReward{RewardId: id, PrizeAmount: "10", PrizeDenom: "ubze", Duration: 1, MarketId: id, Slots: 2, ExpireAt: expireAt}
	}
	f := &fakeStore{
		rewards: map[string]types.TradingReward{
			"no-trades-expired": reward("no-trades-expired", 50), // stuck: expiry passed, the old payout skipped it
			"no-trades-running": reward("no-trades-running", 900),
			"traded":            reward("traded", 900),
			"waiting":           reward("waiting", 0),
		},
		leaderboards: map[string]types.TradingRewardLeaderboard{
			"traded": {RewardId: "traded", List: []types.TradingRewardLeaderboardEntry{{Address: "bze1x", Amount: "1"}}},
		},
		expirations: map[expirationKey]bool{
			{50, "no-trades-expired"}:  true,
			{900, "no-trades-running"}: true,
			{900, "traded"}:            true,
		},
	}

	reset := v6.Migrate(sdk.Context{}, f)

	require.Equal(t, []string{"no-trades-expired", "no-trades-running"}, reset)
	require.Zero(t, f.rewards["no-trades-expired"].ExpireAt)
	require.Zero(t, f.rewards["no-trades-running"].ExpireAt)
	require.Equal(t, uint32(900), f.rewards["traded"].ExpireAt)
	require.Zero(t, f.rewards["waiting"].ExpireAt)
	require.Equal(t, map[expirationKey]bool{{900, "traded"}: true}, f.expirations)

	// the other fields are untouched
	want := reward("no-trades-running", 0)
	require.Equal(t, want, f.rewards["no-trades-running"])

	// running it again changes nothing
	require.Empty(t, v6.Migrate(sdk.Context{}, f))
}
