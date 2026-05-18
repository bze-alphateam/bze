package keeper_test

import (
	"time"

	"go.uber.org/mock/gomock"

	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestFlashVote_UpdateGovernance_LockTooShort: updating a REWARD_STAKED
// DAO's voting_period past the reward program's lock is rejected.
//
// REWARD_STAKED DAOs cannot be created via MsgCreateDao (chicken-and-egg),
// so this exercises the check via setupRewardStakedDao + MsgUpdateGovernanceConfig.
// Epic 5 will land the equivalent guard inside MsgUpdateVotingBackend.
func (suite *IntegrationTestSuite) TestFlashVote_UpdateGovernance_LockTooShort() {
	rewardID := "00000000-0000-0000-0000-000000000aa1"
	dao := suite.setupRewardStakedDao(rewardID)

	// Program lock = 7 days. Try to set voting_period = 8 days — must reject.
	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), rewardID).
		Return(rewardstypes.StakingReward{
			RewardId:     rewardID,
			Lock:         7,
			StakedAmount: "10000",
		}, true).
		Times(1)

	_, err := suite.msgServer.UpdateGovernanceConfig(suite.ctx, &types.MsgUpdateGovernanceConfig{
		Authority: dao.Admin,
		DaoId:     dao.Id,
		Governance: types.GovernanceConfig{
			ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
			ThresholdBps: 5_000,
			VotingPeriod: 8 * 24 * time.Hour,
		},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "lock")
}

// TestFlashVote_UpdateGovernance_LockSufficient: voting_period equal to
// the program lock is accepted.
func (suite *IntegrationTestSuite) TestFlashVote_UpdateGovernance_LockSufficient() {
	rewardID := "00000000-0000-0000-0000-000000000aa2"
	dao := suite.setupRewardStakedDao(rewardID)

	// Program lock = 7 days. voting_period = 7 days exactly → accepted.
	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), rewardID).
		Return(rewardstypes.StakingReward{
			RewardId:     rewardID,
			Lock:         7,
			StakedAmount: "10000",
		}, true).
		Times(1)

	_, err := suite.msgServer.UpdateGovernanceConfig(suite.ctx, &types.MsgUpdateGovernanceConfig{
		Authority: dao.Admin,
		DaoId:     dao.Id,
		Governance: types.GovernanceConfig{
			ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
			ThresholdBps: 5_000,
			VotingPeriod: 7 * 24 * time.Hour,
		},
	})
	suite.Require().NoError(err)
}
