package keeper_test

import (
	"time"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestUpdateGovernanceConfig_AdminGated: only the DAO admin can update.
func (suite *IntegrationTestSuite) TestUpdateGovernanceConfig_AdminGated() {
	daoID, _ := suite.createSampleDao("alpha")
	intruder := freshAddr()

	_, err := suite.msgServer.UpdateGovernanceConfig(suite.ctx, &types.MsgUpdateGovernanceConfig{
		Authority:  intruder,
		DaoId:      daoID,
		Governance: validGovernance(),
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "unauthorized")
}

// TestUpdateGovernanceConfig_HappyPath: admin can replace the config; the
// new value is persisted and reflected on subsequent proposals — but
// existing proposals keep their frozen snapshot.
func (suite *IntegrationTestSuite) TestUpdateGovernanceConfig_HappyPath() {
	daoID, admin := suite.createSampleDao("alpha")
	existingPid := suite.createTestProposal(daoID, admin)

	newGov := types.GovernanceConfig{
		ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITH_QUORUM,
		ThresholdBps: 6_000,
		QuorumBps:    3_000,
		VotingPeriod: 2 * time.Hour,
		AllowRevote:  false,
	}
	_, err := suite.msgServer.UpdateGovernanceConfig(suite.ctx, &types.MsgUpdateGovernanceConfig{
		Authority:  admin,
		DaoId:      daoID,
		Governance: newGov,
	})
	suite.Require().NoError(err)

	// Existing proposal still has its frozen snapshot.
	existing, _ := suite.k.GetProposal(suite.ctx, daoID, existingPid)
	suite.Require().Equal(uint32(5_000), existing.GovernanceSnapshot.ThresholdBps,
		"in-flight proposal must keep its frozen governance_snapshot")

	// New proposal adopts the new config.
	newPid := suite.createTestProposal(daoID, admin)
	newP, _ := suite.k.GetProposal(suite.ctx, daoID, newPid)
	suite.Require().Equal(uint32(6_000), newP.GovernanceSnapshot.ThresholdBps)
	suite.Require().Equal(types.ApprovalRule_APPROVAL_RULE_WITH_QUORUM, newP.GovernanceSnapshot.ApprovalRule)
}

// TestUpdateGovernanceConfig_BrickCaps: threshold/quorum/voting_period
// values outside their respective bounds are rejected.
func (suite *IntegrationTestSuite) TestUpdateGovernanceConfig_BrickCaps() {
	daoID, admin := suite.createSampleDao("alpha")

	cases := []struct {
		name string
		gov  types.GovernanceConfig
	}{
		{
			name: "threshold above cap",
			gov: types.GovernanceConfig{
				ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
				ThresholdBps: 9_999,
				QuorumBps:    0,
				VotingPeriod: time.Hour,
			},
		},
		{
			name: "threshold zero",
			gov: types.GovernanceConfig{
				ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
				ThresholdBps: 0,
				QuorumBps:    0,
				VotingPeriod: time.Hour,
			},
		},
		{
			name: "quorum above cap on WITH_QUORUM",
			gov: types.GovernanceConfig{
				ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITH_QUORUM,
				ThresholdBps: 5_000,
				QuorumBps:    9_999,
				VotingPeriod: time.Hour,
			},
		},
		{
			name: "quorum non-zero on WITHOUT_QUORUM",
			gov: types.GovernanceConfig{
				ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
				ThresholdBps: 5_000,
				QuorumBps:    1,
				VotingPeriod: time.Hour,
			},
		},
		{
			name: "voting_period below floor",
			gov: types.GovernanceConfig{
				ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
				ThresholdBps: 5_000,
				VotingPeriod: 30 * time.Minute,
			},
		},
		{
			name: "voting_period above Param cap",
			gov: types.GovernanceConfig{
				ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
				ThresholdBps: 5_000,
				// DefaultMaxVotingPeriod is 30 days; anything bigger should be rejected.
				VotingPeriod: 365 * 24 * time.Hour,
			},
		},
		{
			name: "approval_rule UNSPECIFIED",
			gov: types.GovernanceConfig{
				ThresholdBps: 5_000,
				VotingPeriod: time.Hour,
			},
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			_, err := suite.msgServer.UpdateGovernanceConfig(suite.ctx, &types.MsgUpdateGovernanceConfig{
				Authority:  admin,
				DaoId:      daoID,
				Governance: tc.gov,
			})
			suite.Require().Error(err)
		})
	}
}

// TestGovernanceConfigQuery: read-back of a DAO's current config.
func (suite *IntegrationTestSuite) TestGovernanceConfigQuery() {
	daoID, _ := suite.createSampleDao("alpha")
	resp, err := suite.k.GovernanceConfig(suite.ctx, &types.QueryGovernanceConfigRequest{DaoId: daoID})
	suite.Require().NoError(err)
	suite.Require().Equal(validGovernance().ThresholdBps, resp.Governance.ThresholdBps)
	suite.Require().Equal(validGovernance().VotingPeriod, resp.Governance.VotingPeriod)
}
