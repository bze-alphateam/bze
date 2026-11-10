package keeper_test

import (
	"fmt"

	"github.com/bze-alphateam/bze/x/rewards/types"
)

const (
	purgeTestPayoutEpoch = int64(100)
	purgeTestExpiryEpoch = purgeTestPayoutEpoch + 168
)

// seedPaidTradingReward stores an active reward whose payout already happened, so the expiry
// epoch takes the removal branch of the distribution hook, with its leaderboard and candidates.
func (suite *IntegrationTestSuite) seedPaidTradingReward(rewardId string, candidates int) {
	suite.k.SetActiveTradingReward(suite.ctx, types.TradingReward{
		RewardId:    rewardId,
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    1,
		MarketId:    "market-" + rewardId,
		Slots:       1,
		ExpireAt:    uint32(purgeTestPayoutEpoch),
	})
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{
		RewardId: rewardId,
		ExpireAt: uint32(purgeTestExpiryEpoch),
	})
	suite.k.SetTradingRewardLeaderboard(suite.ctx, types.TradingRewardLeaderboard{
		RewardId: rewardId,
		List:     []types.TradingRewardLeaderboardEntry{{Address: "bze1winner", Amount: "10"}},
	})
	suite.seedTradingRewardCandidates(rewardId, candidates)
}

func (suite *IntegrationTestSuite) seedTradingRewardCandidates(rewardId string, candidates int) {
	for i := 0; i < candidates; i++ {
		suite.k.SetTradingRewardCandidate(suite.ctx, types.TradingRewardCandidate{
			RewardId: rewardId,
			Amount:   "1",
			Address:  fmt.Sprintf("bze1trader%07d", i),
		})
	}
}

func (suite *IntegrationTestSuite) runTradingRewardExpiry() {
	hook := suite.k.GetTradingRewardsDistributionHook()
	suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "hour", purgeTestExpiryEpoch))
}

func (suite *IntegrationTestSuite) candidateCount(rewardId string) int {
	return len(suite.k.GetTradingRewardCandidateByRewardId(suite.ctx, rewardId))
}

func (suite *IntegrationTestSuite) TestCandidatePurge_RemovesBoundedBatchPerBlock() {
	const rewardId = "000000000001"
	limit := types.MaxTradingRewardCandidateRemovalsPerBlock
	suite.seedPaidTradingReward(rewardId, limit*2+5)

	suite.runTradingRewardExpiry()

	// the O(1) records go right away, the candidates wait for the per-block step
	_, found := suite.k.GetActiveTradingReward(suite.ctx, rewardId)
	suite.Require().False(found, "active reward removed by the hook")
	_, found = suite.k.GetTradingRewardLeaderboard(suite.ctx, rewardId)
	suite.Require().False(found, "leaderboard removed by the hook")
	suite.Require().Empty(suite.k.GetAllActiveTradingRewardExpirationByExpireAt(suite.ctx, uint32(purgeTestExpiryEpoch)))
	suite.Require().Equal(limit*2+5, suite.candidateCount(rewardId), "the hook itself removes no candidate")
	suite.Require().Equal([]string{rewardId}, suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))

	suite.k.ProcessTradingRewardCandidatePurgeQueue(suite.ctx)
	suite.Require().Equal(limit+5, suite.candidateCount(rewardId), "first block removes exactly the cap")
	suite.Require().Equal([]string{rewardId}, suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))

	suite.k.ProcessTradingRewardCandidatePurgeQueue(suite.ctx)
	suite.Require().Equal(5, suite.candidateCount(rewardId), "second block removes exactly the cap")
	suite.Require().Equal([]string{rewardId}, suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))

	suite.k.ProcessTradingRewardCandidatePurgeQueue(suite.ctx)
	suite.Require().Zero(suite.candidateCount(rewardId), "third block finishes the reward")
	suite.Require().Empty(suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))
}

func (suite *IntegrationTestSuite) TestCandidatePurge_FewerThanCapFinishesInOneStep() {
	const rewardId = "000000000001"
	suite.seedPaidTradingReward(rewardId, 7)

	suite.runTradingRewardExpiry()
	suite.Require().Equal(7, suite.candidateCount(rewardId), "the hook itself removes no candidate")

	suite.k.ProcessTradingRewardCandidatePurgeQueue(suite.ctx)

	suite.Require().Zero(suite.candidateCount(rewardId))
	suite.Require().Empty(suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))
}

func (suite *IntegrationTestSuite) TestCandidatePurge_ExactlyCapDropsTheEntryInTheSameStep() {
	const rewardId = "000000000001"
	suite.seedPaidTradingReward(rewardId, types.MaxTradingRewardCandidateRemovalsPerBlock)

	suite.runTradingRewardExpiry()
	suite.Require().Equal(types.MaxTradingRewardCandidateRemovalsPerBlock, suite.candidateCount(rewardId))
	suite.k.ProcessTradingRewardCandidatePurgeQueue(suite.ctx)

	suite.Require().Zero(suite.candidateCount(rewardId))
	suite.Require().Empty(suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))
}

func (suite *IntegrationTestSuite) TestCandidatePurge_RewardWithoutCandidatesLeavesNoEntry() {
	const rewardId = "000000000001"
	suite.seedPaidTradingReward(rewardId, 0)

	suite.runTradingRewardExpiry()
	suite.k.ProcessTradingRewardCandidatePurgeQueue(suite.ctx)

	suite.Require().Empty(suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))
}

func (suite *IntegrationTestSuite) TestCandidatePurge_TwoRewardsShareTheBudget() {
	const firstId, secondId = "000000000001", "000000000002"
	limit := types.MaxTradingRewardCandidateRemovalsPerBlock
	suite.seedPaidTradingReward(firstId, limit-10)
	suite.seedPaidTradingReward(secondId, 30)

	suite.runTradingRewardExpiry()
	suite.Require().Equal([]string{firstId, secondId}, suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))

	suite.k.ProcessTradingRewardCandidatePurgeQueue(suite.ctx)
	suite.Require().Zero(suite.candidateCount(firstId), "the first reward fits in the budget")
	suite.Require().Equal(20, suite.candidateCount(secondId), "the second reward takes what is left of it")
	suite.Require().Equal([]string{secondId}, suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))

	suite.k.ProcessTradingRewardCandidatePurgeQueue(suite.ctx)
	suite.Require().Zero(suite.candidateCount(secondId))
	suite.Require().Empty(suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))
}

func (suite *IntegrationTestSuite) TestCandidatePurge_BudgetSpentLeavesTheNextRewardQueued() {
	const firstId, secondId = "000000000001", "000000000002"
	limit := types.MaxTradingRewardCandidateRemovalsPerBlock
	suite.seedPaidTradingReward(firstId, limit)
	suite.seedPaidTradingReward(secondId, 3)

	suite.runTradingRewardExpiry()
	suite.k.ProcessTradingRewardCandidatePurgeQueue(suite.ctx)

	suite.Require().Zero(suite.candidateCount(firstId))
	suite.Require().Equal(3, suite.candidateCount(secondId), "no budget left for the second reward")
	suite.Require().Equal([]string{secondId}, suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))
}

func (suite *IntegrationTestSuite) TestCandidatePurge_LeavesOtherRewardsUntouched() {
	// reward ids that share a leading part: the key separator keeps them apart
	const expiredId, activeId = "1", "10"
	suite.seedPaidTradingReward(expiredId, 4)
	suite.seedTradingRewardCandidates(activeId, 6)

	suite.runTradingRewardExpiry()
	suite.Require().Equal(4, suite.candidateCount(expiredId))
	suite.k.ProcessTradingRewardCandidatePurgeQueue(suite.ctx)

	suite.Require().Zero(suite.candidateCount(expiredId))
	suite.Require().Equal(6, suite.candidateCount(activeId))
	suite.Require().Empty(suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))
}

func (suite *IntegrationTestSuite) TestCandidatePurge_FillAfterExpiryAddsNoRow() {
	const rewardId = "000000000001"
	marketId := "market-" + rewardId
	suite.seedPaidTradingReward(rewardId, 3)
	// the worst case: the market still points to the reward
	suite.k.SetMarketIdRewardId(suite.ctx, types.MarketIdTradingRewardId{RewardId: rewardId, MarketId: marketId})

	suite.runTradingRewardExpiry()
	suite.k.GetOnOrderFillHook()(suite.ctx, marketId, "10", "bze1latetrader")

	suite.Require().Equal(3, suite.candidateCount(rewardId))
	suite.Require().Equal([]string{rewardId}, suite.k.GetAllTradingRewardCandidatePurge(suite.ctx))
}

func (suite *IntegrationTestSuite) TestCandidatePurge_EmptyQueueIsNoOp() {
	suite.seedTradingRewardCandidates("000000000001", 2)

	suite.k.ProcessTradingRewardCandidatePurgeQueue(suite.ctx)

	suite.Require().Equal(2, suite.candidateCount("000000000001"))
}
