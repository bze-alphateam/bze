package interchaintest_test

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// A trading reward nobody activates expires 30 days of hour epochs after the
// block that created it (720 hour epochs, fixed in the rewards module). The
// suite runs its own network with 2 s hour epochs, so the expiry comes about
// 24 minutes after the create. The epoch can't be shorter than a block: an
// epoch that falls behind the block time is "catching up", and the rewards
// module refuses to create a reward until it has caught up.
const (
	trExpiryHour = "2s"
	trExpiryWait = 45 * time.Minute
)

var (
	trRewardX = trSpec{prize: 600_000, slots: 2} // the tr/ubze market, never activated: expires
	trRewardY = trSpec{prize: 800_000, slots: 1} // the tr_ubze pool, activated before its pending expiry
)

// TestTradingRewardExpiry runs the expiry of a pending trading reward: at the
// end of its expire_at hour epoch the reward is removed and its whole escrow
// (prize × slots) goes from the rewards module to the burner. A reward
// activated by gov before then drops its pending expiry and keeps its escrow.
func TestTradingRewardExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx,
		chain.WithoutGenesis(chain.EpochsKey),
		chain.WithGenesis(chain.Epochs(chain.DayEpochDuration, trExpiryHour, chain.WeekEpochDuration)),
	)
	s := &trExpirySuite{tradingRewardsSuite: setupTradingRewards(t, ctx, net.Chain)}

	t.Run("1 create two pending rewards", s.createPending)
	t.Run("2 activate one by gov", s.activateOne)
	t.Run("3 the other expires after 30 days", s.expire)
}

type trExpirySuite struct {
	*tradingRewardsSuite

	rewardX, rewardY     string
	expireAtX, expireAtY int64 // pending expire_at hour epochs
}

// 1. Both rewards are pending with expire_at 720 hour epochs after the block
// that created them (createReward asserts it), their escrow in the rewards
// module.
func (s *trExpirySuite) createPending(t *testing.T) {
	s.rewardX = s.createReward(t, trRewardX, s.trMarket)
	s.rewardY = s.createReward(t, trRewardY, s.trPool)
	s.expireAtX = int64(s.reward(t, s.rewardX).ExpireAt)
	s.expireAtY = int64(s.reward(t, s.rewardY).ExpireAt)
	require.Equal(t, []string{s.rewardX, s.rewardY}, s.rewardIDs(t, "pending"), "pending rewards")
	s.requireEscrow(t)
}

// 2. Gov activates Y long before its pending expiry: it is active with no
// expire_at (no trade started it) and the pool maps to it.
func (s *trExpirySuite) activateOne(t *testing.T) {
	_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.creator, helpers.Proposal{
		Messages: []any{s.activateMsg(s.rewardY)},
		Title:    "activate a trading reward",
		Summary:  "e2e: activate the pool trading reward before its pending expiry",
	})
	require.NoError(t, err)

	height := s.activationHeight(t, s.rewardY)
	require.Less(t, s.hourAt(t, height), s.expireAtY, "Y is activated before its pending expiry")
	require.Equal(t, s.wantReward(s.rewardY, trRewardY, s.trPool, 0), s.reward(t, s.rewardY), "active reward Y")
	require.Equal(t, []string{s.rewardX}, s.rewardIDs(t, "pending"), "pending rewards")
	require.Equal(t, []string{s.rewardY}, s.rewardIDs(t, "active"), "active rewards")
	s.requireEscrow(t)
}

// 3. The block that ends X's expire_at hour epoch removes X and sends its
// whole escrow to the burner, with one TradingRewardExpireEvent. Y keeps its
// escrow and stays active after its own former pending expiry.
func (s *trExpirySuite) expire(t *testing.T) {
	end := s.expireAtX + 1
	ctx, cancel := context.WithTimeout(s.ctx, trExpiryWait)
	defer cancel()
	require.NoError(t, helpers.WaitForEpoch(ctx, s.bze, "hour", max(end, s.expireAtY+1)))

	height := s.epochStartHeight(t, end)
	require.Equal(t, []string{s.rewardX}, s.pendingAt(t, height-1), "X is pending until the end of its expire_at epoch")

	expired := s.rewardEvents(t, height, "TradingRewardExpireEvent")
	require.Len(t, expired, 1, "TradingRewardExpireEvent: %+v", expired)
	require.Equal(t, s.rewardX, expired[0].Attributes["reward_id"])
	require.Equal(t, sdkmath.NewInt(trRewardX.escrow()), s.transferred(t, s.blockEvents(t, height), s.rewards, s.burner, s.prize),
		"X's escrow to the burner")

	require.Empty(t, s.pendingAt(t, height), "no pending reward left")
	_, err := s.queryReward(s.rewardX)
	require.Error(t, err, "X is removed")
	delete(s.live, s.rewardX)
	s.requireEscrow(t)

	require.Equal(t, s.wantReward(s.rewardY, trRewardY, s.trPool, 0), s.reward(t, s.rewardY), "Y is untouched")
	require.Equal(t, []string{s.rewardY}, s.rewardIDs(t, "active"), "active rewards")
	mapped, found := s.marketReward(t, s.trPool)
	require.True(t, found, "the pool stays mapped to Y")
	require.Equal(t, s.rewardY, mapped)
}

// pendingAt lists the ids of the pending trading rewards in the state of
// height, sorted.
func (s *trExpirySuite) pendingAt(t *testing.T, height int64) []string {
	t.Helper()
	res, err := helpers.Query[struct {
		List []trReward `json:"list"`
	}](s.ctx, s.bze, "rewards", "all-trading-rewards", "--state", "pending", "--height", strconv.FormatInt(height, 10))
	require.NoError(t, err)
	ids := []string{}
	for _, r := range res.List {
		ids = append(ids, r.RewardID)
	}
	slices.Sort(ids)
	return ids
}
