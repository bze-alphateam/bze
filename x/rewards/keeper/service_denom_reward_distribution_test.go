package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// seedDenomRewardWithPrize writes a DenomReward with the given staked total and a fresh S = 0
// accumulator for the prize denom — the state a schedule created via the BZE-89 handlers relies on.
func (suite *IntegrationTestSuite) seedDenomRewardWithPrize(stakingDenom, prizeDenom string, staked int64) {
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{
		StakingDenom: stakingDenom,
		StakedAmount: math.NewInt(staked),
	})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{
		StakingDenom:     stakingDenom,
		PrizeDenom:       prizeDenom,
		DistributedStake: math.LegacyMustNewDecFromStr("0"),
	})
}

func (suite *IntegrationTestSuite) requirePrizeS(stakingDenom, prizeDenom, expectedS string) {
	prize, found := suite.k.GetDenomRewardPrize(suite.ctx, stakingDenom, prizeDenom)
	suite.Require().True(found)
	suite.Require().True(
		math.LegacyMustNewDecFromStr(expectedS).Equal(prize.DistributedStake),
		"expected S=%s got S=%s", expectedS, prize.DistributedStake,
	)
}

func (suite *IntegrationTestSuite) TestDenomRewardsDistributionHook_FiresOnlyOnDayEpoch() {
	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId:   "000000000001",
		StakingDenom: "ubze",
		PrizeDenom:   "uprize",
		DailyAmount:  math.NewInt(100),
		Duration:     5,
	})

	hook := suite.k.GetDenomRewardsDistributionHook()

	// hour epoch: no enqueue
	suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "hour", 1))
	_, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)

	// day epoch: enqueues with a fresh cursor
	suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "day", 1))
	queue, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
	suite.Require().Equal("", queue.Cursor)
}

func (suite *IntegrationTestSuite) TestEnqueueDenomRewardsDistribution_NoSchedules_NoQueue() {
	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)

	_, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}

// A day tick arriving while a previous distribution is still draining must not reset the cursor.
func (suite *IntegrationTestSuite) TestEnqueueDenomRewardsDistribution_AlreadyPending_KeepsCursor() {
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId:   "000000000001",
		StakingDenom: "ubze",
		PrizeDenom:   "uprize",
		DailyAmount:  math.NewInt(100),
		Duration:     5,
	})

	midDrain := types.DenomRewardsDistributionQueue{Pending: true, Cursor: string(types.DenomRewardScheduleKey("ubze", "000000000001"))}
	suite.k.SetDenomRewardsDistributionQueue(suite.ctx, midDrain)

	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)

	queue, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Equal(midDrain, queue)
}

func (suite *IntegrationTestSuite) TestProcessDenomRewardsDistributionQueue_NoQueueOrNotPending_NoOp() {
	// no queue at all: nothing to do
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)

	// queue present but not pending: left untouched
	stale := types.DenomRewardsDistributionQueue{Pending: false, Cursor: "leftover"}
	suite.k.SetDenomRewardsDistributionQueue(suite.ctx, stale)

	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)

	queue, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Equal(stale, queue)
}

// >100 schedules span multiple blocks via the cursor and every schedule distributes exactly once
// per day; the queue clears when drained and re-enqueues fine the next day.
func (suite *IntegrationTestSuite) TestProcessDenomRewardsDistributionQueue_CursorSpansBlocks_ExactlyOncePerDay() {
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "day").
		Return(int64(7), nil).
		AnyTimes()

	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)

	total := types.MaxDenomRewardDistributionsPerBlock + 50
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

	// block 1: a full batch is processed and the cursor points at the last processed schedule
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	queue, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
	expectedCursor := string(types.DenomRewardScheduleKey("ubze", fmt.Sprintf("%012d", types.MaxDenomRewardDistributionsPerBlock)))
	suite.Require().Equal(expectedCursor, queue.Cursor)

	// block 2: the remaining schedules drain and the queue clears
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	_, found = suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)

	// every schedule advanced exactly one payout
	count := 0
	suite.k.IterateDenomSchedules(suite.ctx, "ubze", func(_ sdk.Context, s types.DenomRewardSchedule) bool {
		count++
		suite.Require().Equal(uint32(1), s.Payouts, "schedule %s", s.ScheduleId)
		return false
	})
	suite.Require().Equal(total, count)

	// the shared accumulator advanced by total × (100/100)
	suite.requirePrizeS("ubze", "uprize", fmt.Sprintf("%d", total))
	prize, _ := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().Equal(int64(7), prize.LastDistributionEpoch)

	// next day: re-enqueue works and pays a second day to every schedule
	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
	queue, found = suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
	suite.Require().Equal("", queue.Cursor)

	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)

	suite.k.IterateDenomSchedules(suite.ctx, "ubze", func(_ sdk.Context, s types.DenomRewardSchedule) bool {
		suite.Require().Equal(uint32(2), s.Payouts, "schedule %s", s.ScheduleId)
		return false
	})
}

// A zero-staker day skips the schedule without counting a payout (rule 15): budget preserved, S
// untouched. Distribution resumes when stakers return, and the total distributed over the
// schedule's life equals daily × duration regardless of the gap.
func (suite *IntegrationTestSuite) TestProcessDenomRewardsDistributionQueue_ZeroStakerDay_SkipsWithoutPayout() {
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "day").
		Return(int64(3), nil).
		AnyTimes()

	suite.seedDenomRewardWithPrize("ubze", "uprize", 0)
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId:   "000000000001",
		StakingDenom: "ubze",
		PrizeDenom:   "uprize",
		DailyAmount:  math.NewInt(100),
		Duration:     3,
	})

	// day with zero stakers: no payout, no payouts increment, S untouched
	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)

	schedule, found := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(uint32(0), schedule.Payouts)
	suite.requirePrizeS("ubze", "uprize", "0")

	// stakers return: the schedule stretches and still pays its full daily × duration budget
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", StakedAmount: math.NewInt(50)})

	for day := 1; day <= 3; day++ {
		suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
		suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	}

	// 3 payouts of 100/50 each: S = 6, meaning S × T = 300 = daily × duration was distributed
	suite.requirePrizeS("ubze", "uprize", "6")
	_, found = suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000001")
	suite.Require().False(found, "schedule must be deleted after its final payout")
}

func (suite *IntegrationTestSuite) TestProcessDenomRewardsDistributionQueue_FinishingPayout_DeletesAndEmitsEvent() {
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "day").
		Return(int64(9), nil).
		AnyTimes()

	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId:   "000000000042",
		StakingDenom: "ubze",
		PrizeDenom:   "uprize",
		DailyAmount:  math.NewInt(100),
		Duration:     2,
		Payouts:      1, // one day left
	})

	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)

	// deleted exactly at payouts == duration, finish event emitted
	_, found := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000042")
	suite.Require().False(found)
	e, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardScheduleFinishEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "schedule_id", "000000000042")
	suite.requirePrizeS("ubze", "uprize", "1")

	// afterwards the accumulator (and with it claimability) is untouched: with no schedules left
	// the next day tick doesn't even enqueue
	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
	_, found = suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
	suite.requirePrizeS("ubze", "uprize", "1")
}

// Two schedules in the same prize denom advance ONE shared accumulator by the sum of their daily
// amounts, and T is read live each day — a stake change between days changes the per-unit bump.
func (suite *IntegrationTestSuite) TestProcessDenomRewardsDistributionQueue_SamePrizeDenom_SharedAccumulator_LiveT() {
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "day").
		Return(int64(5), nil).
		AnyTimes()

	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)
	for i, daily := range []int64{100, 50} {
		suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
			ScheduleId:   fmt.Sprintf("%012d", i+1),
			StakingDenom: "ubze",
			PrizeDenom:   "uprize",
			DailyAmount:  math.NewInt(daily),
			Duration:     5,
		})
	}

	// day 1: S += (100 + 50) / 100
	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	suite.requirePrizeS("ubze", "uprize", "1.5")

	// stake changes between days: day 2 uses the live T. The two schedules distribute separately, so
	// the accumulator gains 100/300 + 50/300, each truncated at 18dp (round down, never to nearest):
	// 0.333…333 + 0.166…666 = 0.499…999. The sub-unit shortfall stays as dust in the pool rather than
	// being rounded up into an over-credit (BZE-104).
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", StakedAmount: math.NewInt(300)})
	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	suite.requirePrizeS("ubze", "uprize", "1.999999999999999999") // 1.5 + trunc(100/300) + trunc(50/300)
}

// Defensive paths: a schedule whose denom reward or prize record is missing, or that is already
// finished, is skipped without payouts, deletions, or events. No epoch expectation is registered,
// so any accumulator bump would panic the test with an unexpected mock call.
func (suite *IntegrationTestSuite) TestDistributeDenomRewardSchedule_DefensiveSkips() {
	// schedule pointing at a denom reward that does not exist
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId:   "000000000001",
		StakingDenom: "nodr",
		PrizeDenom:   "uprize",
		DailyAmount:  math.NewInt(100),
		Duration:     5,
	})

	// schedule whose prize accumulator is missing
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", StakedAmount: math.NewInt(100)})
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId:   "000000000002",
		StakingDenom: "ubze",
		PrizeDenom:   "noprize",
		DailyAmount:  math.NewInt(100),
		Duration:     5,
	})

	// already-finished schedule (cannot normally exist — finish deletes it)
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("0")})
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId:   "000000000003",
		StakingDenom: "ubze",
		PrizeDenom:   "uprize",
		DailyAmount:  math.NewInt(100),
		Duration:     3,
		Payouts:      3,
	})

	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)

	// all three untouched and still present; no finish event; S untouched
	for _, key := range []struct{ denom, id string }{
		{"nodr", "000000000001"}, {"ubze", "000000000002"}, {"ubze", "000000000003"},
	} {
		schedule, found := suite.k.GetDenomRewardSchedule(suite.ctx, key.denom, key.id)
		suite.Require().True(found, "schedule %s/%s must survive", key.denom, key.id)
		suite.Require().True(schedule.Payouts == 0 || schedule.Payouts == 3)
	}
	_, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardScheduleFinishEvent{}))
	suite.Require().False(ok)
	suite.requirePrizeS("ubze", "uprize", "0")

	// queue drained regardless
	_, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}
