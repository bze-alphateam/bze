package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestUpdateDepositConfig_AdminGated: only the DAO admin can update the
// deposit config.
func (suite *IntegrationTestSuite) TestUpdateDepositConfig_AdminGated() {
	daoID, _ := suite.createSampleDao("dep-admin")
	intruder := freshAddr()

	_, err := suite.msgServer.UpdateDepositConfig(suite.ctx, &types.MsgUpdateDepositConfig{
		Authority: intruder,
		DaoId:     daoID,
		Deposit:   validDeposit(),
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "unauthorized")
}

// TestUpdateDepositConfig_HappyPath: admin can replace; existing proposals
// keep their frozen deposit_snapshot; new proposals adopt the new config.
func (suite *IntegrationTestSuite) TestUpdateDepositConfig_HappyPath() {
	daoID, admin := suite.createSampleDao("dep-update")
	// Existing proposal under old (validDeposit) config — min = 1ubze.
	pid := suite.createTestProposal(daoID, admin)
	pre, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal("1ubze", pre.DepositSnapshot.MinDeposit.String())

	// Replace with a stricter config.
	newCfg := types.DepositConfig{
		MinDeposit:         sdk.NewInt64Coin("ubze", 100),
		DepositPeriod:      14 * 24 * time.Hour,
		ForfeitDestination: types.ForfeitDestination_FORFEIT_DESTINATION_BURNER,
		VotingRefundPolicy: types.RefundPolicy_REFUND_POLICY_ALWAYS,
	}
	_, err := suite.msgServer.UpdateDepositConfig(suite.ctx, &types.MsgUpdateDepositConfig{
		Authority: admin,
		DaoId:     daoID,
		Deposit:   newCfg,
	})
	suite.Require().NoError(err)

	// In-flight proposal retains the OLD snapshot.
	post, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal("1ubze", post.DepositSnapshot.MinDeposit.String())

	// Query reflects the new live config.
	q, err := suite.k.DepositConfig(suite.ctx, &types.QueryDepositConfigRequest{DaoId: daoID})
	suite.Require().NoError(err)
	suite.Require().Equal("100ubze", q.Deposit.MinDeposit.String())
}

// TestUpdateDepositConfig_BrickCaps verifies the stateless caps reject
// obvious misconfigurations.
func (suite *IntegrationTestSuite) TestUpdateDepositConfig_BrickCaps() {
	daoID, admin := suite.createSampleDao("dep-caps")

	cases := []struct {
		name string
		cfg  types.DepositConfig
	}{
		{
			name: "min_deposit amount zero",
			cfg: types.DepositConfig{
				MinDeposit:         sdk.NewInt64Coin("ubze", 0),
				DepositPeriod:      7 * 24 * time.Hour,
				ForfeitDestination: types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
				VotingRefundPolicy: types.RefundPolicy_REFUND_POLICY_ON_PASS,
			},
		},
		{
			name: "deposit_period below floor",
			cfg: types.DepositConfig{
				MinDeposit:         sdk.NewInt64Coin("ubze", 1),
				DepositPeriod:      time.Hour, // below 24h floor
				ForfeitDestination: types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
				VotingRefundPolicy: types.RefundPolicy_REFUND_POLICY_ON_PASS,
			},
		},
		{
			name: "deposit_period above Param cap",
			cfg: types.DepositConfig{
				MinDeposit:         sdk.NewInt64Coin("ubze", 1),
				DepositPeriod:      365 * 24 * time.Hour, // > default 30d
				ForfeitDestination: types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
				VotingRefundPolicy: types.RefundPolicy_REFUND_POLICY_ON_PASS,
			},
		},
		{
			name: "forfeit UNSPECIFIED",
			cfg: types.DepositConfig{
				MinDeposit:         sdk.NewInt64Coin("ubze", 1),
				DepositPeriod:      7 * 24 * time.Hour,
				VotingRefundPolicy: types.RefundPolicy_REFUND_POLICY_ON_PASS,
			},
		},
		{
			name: "refund_policy UNSPECIFIED",
			cfg: types.DepositConfig{
				MinDeposit:         sdk.NewInt64Coin("ubze", 1),
				DepositPeriod:      7 * 24 * time.Hour,
				ForfeitDestination: types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
			},
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			_, err := suite.msgServer.UpdateDepositConfig(suite.ctx, &types.MsgUpdateDepositConfig{
				Authority: admin,
				DaoId:     daoID,
				Deposit:   tc.cfg,
			})
			suite.Require().Error(err)
		})
	}
}
