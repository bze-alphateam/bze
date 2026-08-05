package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestMigrator_Migrate4to5 checks the v5 migration forces the boost params to
// their defaults, leaves every other param as stored, and touches nothing
// else in the store.
func (suite *IntegrationTestSuite) TestMigrator_Migrate4to5() {
	params := types.DefaultParams()
	params.CreateBoostFee = sdk.NewInt64Coin("ubze", 1)
	params.MaxBoostsPerReward = 3
	params.ExtraGasForExitStake = 5
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	sr, participant := suite.seedRewardAndParticipant("000000000001", "100")
	boost := suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", false)

	migrator := keeper.NewMigrator(*suite.k, nil)
	suite.Require().NoError(migrator.Migrate4to5(suite.ctx))

	got := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(types.DefaultCreateBoostFee, got.CreateBoostFee)
	suite.Require().Equal(types.DefaultMaxBoostsPerReward, got.MaxBoostsPerReward)
	suite.Require().Equal(uint64(5), got.ExtraGasForExitStake)

	gotSr, found := suite.k.GetStakingReward(suite.ctx, sr.RewardId)
	suite.Require().True(found)
	suite.Require().Equal(sr, gotSr)

	gotParticipant, found := suite.k.GetStakingRewardParticipant(suite.ctx, participant.Address, participant.RewardId)
	suite.Require().True(found)
	suite.Require().Equal(participant, gotParticipant)

	gotBoost, found := suite.k.GetBoost(suite.ctx, boost.RewardId, boost.Id)
	suite.Require().True(found)
	suite.Require().Equal(boost, gotBoost)
}

func (suite *IntegrationTestSuite) TestConsensusVersionBumped() {
	suite.Require().Equal(uint64(5), rewards.AppModule{}.ConsensusVersion())
}
