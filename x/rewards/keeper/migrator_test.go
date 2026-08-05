package keeper_test

import (
	"sort"

	"github.com/bze-alphateam/bze/testutil/sample"
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

// TestMigrator_Migrate4to5_BackfillsParticipantIndex covers C2: the migration
// over multi-reward, multi-address pre-index state writes exactly one reverse
// index entry per participant — completeness is consensus-critical, a missing
// entry means the cleanup sweep silently skips that participant.
func (suite *IntegrationTestSuite) TestMigrator_Migrate4to5_BackfillsParticipantIndex() {
	addrs := make([]string, 3)
	for i := range addrs {
		addrs[i] = sample.AccAddress()
	}
	sort.Strings(addrs)

	//pre-index state: participants written directly, no index entries exist
	for _, addr := range addrs {
		suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
			Address: addr, RewardId: "000000000001", Amount: "10", JoinedAt: "1",
		})
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: addrs[0], RewardId: "000000000002", Amount: "20", JoinedAt: "2",
	})
	suite.Require().Empty(suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", "", 100))

	migrator := keeper.NewMigrator(*suite.k, nil)
	suite.Require().NoError(migrator.Migrate4to5(suite.ctx))

	//exactly one entry per participant, per reward
	suite.Require().Equal(addrs, suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", "", 100))
	suite.Require().Equal(addrs[:1], suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000002", "", 100))

	//idempotent: a second run changes nothing
	suite.Require().NoError(migrator.Migrate4to5(suite.ctx))
	suite.Require().Equal(addrs, suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", "", 100))
	suite.Require().Equal(addrs[:1], suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000002", "", 100))
}
