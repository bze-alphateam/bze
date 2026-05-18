package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestUpdateVotingBackend_StaticToStaticRejected: STATIC → STATIC is
// rejected with a "use MsgUpdateMembers" pointer. Member-set churn has
// its own well-named operation.
func (suite *IntegrationTestSuite) TestUpdateVotingBackend_StaticToStaticRejected() {
	daoID, admin := suite.createSampleDao("static-to-static")

	_, err := suite.msgServer.UpdateVotingBackend(suite.ctx, &types.MsgUpdateVotingBackend{
		Authority: admin,
		DaoId:     daoID,
		VotingConfig: &types.MsgUpdateVotingBackend_Static{
			Static: &types.StaticVotingConfig{
				Members: []types.StaticMember{{Address: admin, Weight: 1}},
			},
		},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "MsgUpdateMembers")
}

// TestUpdateVotingBackend_CrossTypeRejected: STATIC → REWARD_STAKED is
// rejected as a backend-type change.
func (suite *IntegrationTestSuite) TestUpdateVotingBackend_CrossTypeRejected() {
	daoID, admin := suite.createSampleDao("static-to-rs")

	_, err := suite.msgServer.UpdateVotingBackend(suite.ctx, &types.MsgUpdateVotingBackend{
		Authority: admin,
		DaoId:     daoID,
		VotingConfig: &types.MsgUpdateVotingBackend_RewardStaked{
			RewardStaked: &types.RewardStakedVotingConfig{RewardId: "some-reward"},
		},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "cross-type")
}

// TestUpdateVotingBackend_RewardStakedHappyPath: REWARD_STAKED →
// REWARD_STAKED with sufficient lock is accepted.
func (suite *IntegrationTestSuite) TestUpdateVotingBackend_RewardStakedHappyPath() {
	oldRewardID := "old-reward"
	dao := suite.setupRewardStakedDao(oldRewardID)
	newRewardID := "new-reward"

	// New reward: lock 7 days, default governance.voting_period is 24h.
	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), newRewardID).
		Return(rewardstypes.StakingReward{
			RewardId:     newRewardID,
			Lock:         7,
			StakedAmount: "1000",
		}, true).
		Times(1)

	_, err := suite.msgServer.UpdateVotingBackend(suite.ctx, &types.MsgUpdateVotingBackend{
		Authority: dao.Admin,
		DaoId:     dao.Id,
		VotingConfig: &types.MsgUpdateVotingBackend_RewardStaked{
			RewardStaked: &types.RewardStakedVotingConfig{RewardId: newRewardID},
		},
	})
	suite.Require().NoError(err)

	got, _ := suite.k.GetDao(suite.ctx, dao.Id)
	suite.Require().Equal(newRewardID, got.RewardId)
}

// TestUpdateVotingBackend_LockTooShort: a new reward whose lock is below
// the DAO's voting_period is rejected by the flash-vote rule.
func (suite *IntegrationTestSuite) TestUpdateVotingBackend_LockTooShort() {
	dao := suite.setupRewardStakedDao("old-reward")
	newRewardID := "short-lock"

	// validGovernance().VotingPeriod = 24h. Lock 0 days = 0 → < 24h, rejected.
	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), newRewardID).
		Return(rewardstypes.StakingReward{
			RewardId: newRewardID,
			Lock:     0,
		}, true).
		Times(1)

	_, err := suite.msgServer.UpdateVotingBackend(suite.ctx, &types.MsgUpdateVotingBackend{
		Authority: dao.Admin,
		DaoId:     dao.Id,
		VotingConfig: &types.MsgUpdateVotingBackend_RewardStaked{
			RewardStaked: &types.RewardStakedVotingConfig{RewardId: newRewardID},
		},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "lock")
}

// TestUpdateVotingBackend_MissingReward: an unknown reward_id is rejected.
func (suite *IntegrationTestSuite) TestUpdateVotingBackend_MissingReward() {
	dao := suite.setupRewardStakedDao("existing-reward")

	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), "ghost-reward").
		Return(rewardstypes.StakingReward{}, false).
		Times(1)

	_, err := suite.msgServer.UpdateVotingBackend(suite.ctx, &types.MsgUpdateVotingBackend{
		Authority: dao.Admin,
		DaoId:     dao.Id,
		VotingConfig: &types.MsgUpdateVotingBackend_RewardStaked{
			RewardStaked: &types.RewardStakedVotingConfig{RewardId: "ghost-reward"},
		},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "not found")
	_ = time.Hour // keep unused import quiet if any path drops time refs
	_ = sdk.AccAddress(nil)
}

// TestUpdateVotingBackend_AdminGated: a non-admin can't update.
func (suite *IntegrationTestSuite) TestUpdateVotingBackend_AdminGated() {
	dao := suite.setupRewardStakedDao("existing")
	intruder := freshAddr()

	_, err := suite.msgServer.UpdateVotingBackend(suite.ctx, &types.MsgUpdateVotingBackend{
		Authority: intruder,
		DaoId:     dao.Id,
		VotingConfig: &types.MsgUpdateVotingBackend_RewardStaked{
			RewardStaked: &types.RewardStakedVotingConfig{RewardId: "something"},
		},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "unauthorized")
}
