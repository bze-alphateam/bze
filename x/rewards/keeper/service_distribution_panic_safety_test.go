package keeper_test

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/rewards/types"
)

// countTypedEvents counts the events of the given proto type emitted on the suite context.
func (suite *IntegrationTestSuite) countTypedEvents(evType string) int {
	n := 0
	for _, ev := range suite.ctx.EventManager().Events() {
		if ev.Type == evType {
			n++
		}
	}

	return n
}

// A panic while paying one schedule must not take the block down: the pass runs in a recovering
// cache context, so the offending schedule is skipped with every write it made discarded, the
// other schedules in the batch are paid, and the queue drains normally.
//
// The panic is injected through the epoch keeper mock, which stands in for any panic inside the
// per-schedule pass (a corrupt record failing MustUnmarshal, an accumulator overflow). The mock
// also writes a marker record through the context it receives before panicking, which proves the
// rollback: the marker must not exist afterwards.
func (suite *IntegrationTestSuite) TestProcessDenomRewardsDistributionQueue_PanicInOneSchedule_SkipsItAndPaysTheRest() {
	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)
	for i := 1; i <= 3; i++ {
		suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
			ScheduleId:   fmt.Sprintf("%012d", i),
			StakingDenom: "ubze",
			PrizeDenom:   "uprize",
			DailyAmount:  math.NewInt(100),
			Duration:     5,
		})
	}

	marker := types.DenomRewardSchedule{
		ScheduleId:   "000000000099",
		StakingDenom: "umarker",
		PrizeDenom:   "uprize",
		DailyAmount:  math.NewInt(1),
		Duration:     1,
	}

	// schedules are processed in key order: the first one's epoch read panics, the next two succeed
	gomock.InOrder(
		suite.epoch.EXPECT().
			SafeGetEpochCountByIdentifier(gomock.Any(), "day").
			DoAndReturn(func(c sdk.Context, _ string) (int64, error) {
				suite.k.SetDenomRewardSchedule(c, marker)
				panic("injected panic inside the distribution pass")
			}).
			Times(1),
		suite.epoch.EXPECT().
			SafeGetEpochCountByIdentifier(gomock.Any(), "day").
			Return(int64(7), nil).
			Times(2),
	)

	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
	suite.Require().NotPanics(func() { suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx) })

	// the panicking schedule is untouched, the other two were paid exactly once
	for i, expectedPayouts := range map[int]uint32{1: 0, 2: 1, 3: 1} {
		schedule, found := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", fmt.Sprintf("%012d", i))
		suite.Require().True(found, "schedule %d", i)
		suite.Require().Equal(expectedPayouts, schedule.Payouts, "schedule %d", i)
	}
	suite.requirePrizeS("ubze", "uprize", "2")
	prize, _ := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().Equal(int64(7), prize.LastDistributionEpoch)

	// nothing written inside the panicking pass survives
	_, found := suite.k.GetDenomRewardSchedule(suite.ctx, marker.StakingDenom, marker.ScheduleId)
	suite.Require().False(found, "writes made before the panic must be rolled back")

	// the queue drained (3 < batch limit) and the batch did not stall on the bad schedule
	_, found = suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)

	// the next day pays the previously skipped schedule like any other
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(gomock.Any(), "day").
		Return(int64(8), nil).
		Times(3)
	suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
	suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	for i, expectedPayouts := range map[int]uint32{1: 1, 2: 2, 3: 2} {
		schedule, _ := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", fmt.Sprintf("%012d", i))
		suite.Require().Equal(expectedPayouts, schedule.Payouts, "schedule %d", i)
	}
	suite.requirePrizeS("ubze", "uprize", "5")
}

// Same guarantee for the staking-reward queue. The panic is a genuine one from the accumulator
// math: an S already at the LegacyDec bit limit makes the S += r/T bump overflow ("Int overflow"
// panic inside cosmossdk.io/math). Such a record cannot come from the chain (Int values are
// 256-bit), it is written directly to the store to stand in for any corrupt record.
func (suite *IntegrationTestSuite) TestProcessStakingRewardsDistributionQueue_PanicInOneReward_SkipsItAndPaysTheRest() {
	atBitLimit := math.LegacyNewDecFromBigIntWithPrec(new(big.Int).Lsh(big.NewInt(1), 315), 18).String()

	newReward := func(id, prizeAmount, staked, distributed string) types.StakingReward {
		return types.StakingReward{
			RewardId:         id,
			PrizeAmount:      prizeAmount,
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         5,
			Payouts:          0,
			MinStake:         100,
			Lock:             7,
			StakedAmount:     staked,
			DistributedStake: distributed,
		}
	}
	// rewards are processed in id order: the first one overflows, the next two are ordinary
	suite.k.SetStakingReward(suite.ctx, newReward("panic-reward-001", atBitLimit, "1", atBitLimit))
	suite.k.SetStakingReward(suite.ctx, newReward("panic-reward-002", "1000", "5000", "0"))
	suite.k.SetStakingReward(suite.ctx, newReward("panic-reward-003", "1000", "5000", "0"))

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.Require().NotPanics(func() { suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx) })

	bad, found := suite.k.GetStakingReward(suite.ctx, "panic-reward-001")
	suite.Require().True(found)
	suite.Require().Equal(uint32(0), bad.Payouts, "the panicking reward must be left untouched")
	suite.Require().Equal(atBitLimit, bad.DistributedStake)

	for _, id := range []string{"panic-reward-002", "panic-reward-003"} {
		sr, found := suite.k.GetStakingReward(suite.ctx, id)
		suite.Require().True(found, id)
		suite.Require().Equal(uint32(1), sr.Payouts, id)
		suite.Require().Equal("0.200000000000000000", sr.DistributedStake, id)
	}

	// only the two successful passes emitted their distribution event
	suite.Require().Equal(2, suite.countTypedEvents(proto.MessageName(&types.StakingRewardDistributionEvent{})))

	_, found = suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found, "the queue must drain past the bad record")
}
