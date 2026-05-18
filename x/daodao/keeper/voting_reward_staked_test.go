package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// setupRewardStakedDao writes a REWARD_STAKED DAO directly via the keeper,
// bypassing MsgCreateDao (which rejects the variant). Returns the DAO and
// its derived account address.
//
// Epic 2 cannot create REWARD_STAKED DAOs via the public message path —
// that arrives in Epic 5's MsgUpdateVotingBackend. We still want to
// exercise the backend impl, so we plant a DAO row directly.
func (suite *IntegrationTestSuite) setupRewardStakedDao(rewardID string) types.Dao {
	id := uint64(1)
	addr := types.DaoAccountAddress(id)

	dao := types.Dao{
		Id:             id,
		Metadata:       types.DaoMetadata{Name: "rs"},
		Creator:        freshAddr(),
		AccountAddress: addr.String(),
		Admin:          freshAddr(),
		CreatedAtBlock: 1,
		VotingBackend:  types.VotingBackendType_VOTING_BACKEND_REWARD_STAKED,
		RewardId:       rewardID,
		// Governance is now a required field on every DAO record (Epic 3).
		// We pick a permissive config; these tests exercise the voting-power
		// backend itself, not governance logic.
		Governance: validGovernance(),
		Deposit:    validDeposit(),
	}
	suite.k.SetDao(suite.ctx, dao)
	suite.Require().NoError(suite.k.SetDaoIndices(suite.ctx, dao))
	// Bump the counter so subsequent operations don't reuse id 1.
	suite.k.SetDaoIDCounter(suite.ctx, 2)
	return dao
}

func (suite *IntegrationTestSuite) TestRewardStaked_VotingPower() {
	rewardID := "00000000-0000-0000-0000-000000000001"
	dao := suite.setupRewardStakedDao(rewardID)
	voter := suite.mustAcc(freshAddr())

	suite.rewards.EXPECT().
		GetStakingRewardParticipant(gomock.Any(), voter.String(), rewardID).
		Return(rewardstypes.StakingRewardParticipant{Address: voter.String(), Amount: "1234"}, true).
		Times(1)
	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), rewardID).
		Return(rewardstypes.StakingReward{RewardId: rewardID, StakedAmount: "10000"}, true).
		Times(1)

	resp, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{
		DaoId:   dao.Id,
		Address: voter.String(),
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(1234), resp.Power)
	suite.Require().Equal(uint64(10000), resp.Total)
}

func (suite *IntegrationTestSuite) TestRewardStaked_VotingPower_NonParticipant() {
	rewardID := "00000000-0000-0000-0000-000000000001"
	dao := suite.setupRewardStakedDao(rewardID)
	nonVoter := suite.mustAcc(freshAddr())

	// rewards returns "not found" → daodao reports 0 power.
	suite.rewards.EXPECT().
		GetStakingRewardParticipant(gomock.Any(), nonVoter.String(), rewardID).
		Return(rewardstypes.StakingRewardParticipant{}, false).
		Times(1)
	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), rewardID).
		Return(rewardstypes.StakingReward{RewardId: rewardID, StakedAmount: "5000"}, true).
		Times(1)

	resp, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{
		DaoId:   dao.Id,
		Address: nonVoter.String(),
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(0), resp.Power)
	suite.Require().Equal(uint64(5000), resp.Total)
}

func (suite *IntegrationTestSuite) TestRewardStaked_TotalVotingPower() {
	rewardID := "00000000-0000-0000-0000-000000000001"
	dao := suite.setupRewardStakedDao(rewardID)

	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), rewardID).
		Return(rewardstypes.StakingReward{RewardId: rewardID, StakedAmount: "999999"}, true).
		Times(1)

	resp, err := suite.k.TotalVotingPower(suite.ctx, &types.QueryTotalVotingPowerRequest{DaoId: dao.Id})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(999999), resp.Total)
}

func (suite *IntegrationTestSuite) TestRewardStaked_MissingReward() {
	rewardID := "00000000-0000-0000-0000-000000000999"
	dao := suite.setupRewardStakedDao(rewardID)

	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), rewardID).
		Return(rewardstypes.StakingReward{}, false).
		Times(1)

	_, err := suite.k.TotalVotingPower(suite.ctx, &types.QueryTotalVotingPowerRequest{DaoId: dao.Id})
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestRewardStaked_MembersQueryRejected() {
	dao := suite.setupRewardStakedDao("00000000-0000-0000-0000-000000000001")

	_, err := suite.k.Members(suite.ctx, &types.QueryMembersRequest{DaoId: dao.Id})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "STATIC-only")
}

func (suite *IntegrationTestSuite) TestRewardStaked_UpdateMembersRejected() {
	dao := suite.setupRewardStakedDao("00000000-0000-0000-0000-000000000001")
	addr := freshAddr()

	_, err := suite.msgServer.UpdateMembers(suite.ctx, &types.MsgUpdateMembers{
		Authority: dao.Admin,
		DaoId:     dao.Id,
		Add:       []types.StaticMember{{Address: addr, Weight: 1}},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "STATIC-only")
}

// unusedSdkAccAddr is here just to keep the sdk import meaningful in tests
// that don't currently use it directly; remove if/when no longer needed.
var _ = sdk.AccAddress(nil)
