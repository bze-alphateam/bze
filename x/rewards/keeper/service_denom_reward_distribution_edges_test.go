package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// A scheduled payout whose accumulator bump fails (the epoch keeper cannot serve the day count)
// is logged and skipped: the schedule keeps its payouts and budget, the accumulator is untouched,
// no finish event fires, and the queue still drains so the next day tick can retry.
func (suite *IntegrationTestSuite) TestProcessDenomRewardsDistributionQueue_EpochError_SkipsWithoutAdvancing() {
	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)
	// two of three payouts done: a successful pass would have been the finishing one
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId:   "000000000001",
		StakingDenom: "ubze",
		PrizeDenom:   "uprize",
		DailyAmount:  math.NewInt(100),
		Duration:     3,
		Payouts:      2,
	})

	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "day").
		Return(int64(0), fmt.Errorf("epoch keeper unavailable")).
		Times(1)

	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)

	schedule, found := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000001")
	suite.Require().True(found, "a failed payout must not delete the schedule")
	suite.Require().Equal(uint32(2), schedule.Payouts)
	suite.requirePrizeS("ubze", "uprize", "0")
	prize, _ := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().Equal(int64(0), prize.LastDistributionEpoch)

	_, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardScheduleFinishEvent{}))
	suite.Require().False(ok)

	_, found = suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found, "queue must drain even when a payout failed")

	// the next day tick retries and finishes the schedule
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "day").
		Return(int64(9), nil).
		Times(1)
	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)

	_, found = suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000001")
	suite.Require().False(found)
	suite.requirePrizeS("ubze", "uprize", "1")
	_, ok = suite.findTypedEvent(proto.MessageName(&types.DenomRewardScheduleFinishEvent{}))
	suite.Require().True(ok)
}

// Exactly MaxDenomRewardDistributionsPerBlock schedules: the first block processes the whole
// batch but cannot know it was the last one (len == limit), so the queue survives with a cursor;
// the second block finds nothing past the cursor and removes the queue. Every schedule is paid
// exactly once. Mirrors TestProcessStakingDistributionQueue_ExactlyAtBatchLimit.
func (suite *IntegrationTestSuite) TestProcessDenomRewardsDistributionQueue_ExactlyAtBatchLimit() {
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "day").
		Return(int64(7), nil).
		AnyTimes()

	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)
	total := types.MaxDenomRewardDistributionsPerBlock
	for i := 1; i <= total; i++ {
		suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
			ScheduleId:   fmt.Sprintf("%012d", i),
			StakingDenom: "ubze",
			PrizeDenom:   "uprize",
			DailyAmount:  math.NewInt(100),
			Duration:     5,
		})
	}

	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)

	// block 1: len(batch) == limit -> not finished, cursor at the last schedule
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	queue, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
	suite.Require().Equal(string(types.DenomRewardScheduleKey("ubze", fmt.Sprintf("%012d", total))), queue.Cursor)

	count := 0
	suite.k.IterateDenomSchedules(suite.ctx, "ubze", func(_ sdk.Context, s types.DenomRewardSchedule) bool {
		count++
		suite.Require().Equal(uint32(1), s.Payouts, "schedule %s", s.ScheduleId)
		return false
	})
	suite.Require().Equal(total, count)
	suite.requirePrizeS("ubze", "uprize", fmt.Sprintf("%d", total))

	// block 2: nothing after the cursor -> queue removed, nothing paid twice
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	_, found = suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
	suite.k.IterateDenomSchedules(suite.ctx, "ubze", func(_ sdk.Context, s types.DenomRewardSchedule) bool {
		suite.Require().Equal(uint32(1), s.Payouts, "schedule %s", s.ScheduleId)
		return false
	})
	suite.requirePrizeS("ubze", "uprize", fmt.Sprintf("%d", total))
}

// A stored queue record that is not pending (e.g. left behind by a genesis import) does not
// block the day tick: the enqueue re-arms it from the beginning with an empty cursor.
func (suite *IntegrationTestSuite) TestEnqueueDenomRewardsDistribution_NotPendingRecord_ReArmsFromStart() {
	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId:   "000000000001",
		StakingDenom: "ubze",
		PrizeDenom:   "uprize",
		DailyAmount:  math.NewInt(100),
		Duration:     5,
	})
	suite.k.SetDenomRewardsDistributionQueue(suite.ctx, types.DenomRewardsDistributionQueue{
		Pending: false,
		Cursor:  "stale-cursor",
	})

	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)

	queue, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
	suite.Require().Equal("", queue.Cursor)
}
