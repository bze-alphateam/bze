package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestDepositExpiry_ForfeitToTreasury: a DEPOSIT_PERIOD proposal that
// expires without reaching min_deposit transitions to REJECTED_NO_DEPOSIT
// and forfeits any collected deposits to the DAO treasury (per the
// validDeposit() default).
func (suite *IntegrationTestSuite) TestDepositExpiry_ForfeitToTreasury() {
	daoID, admin := suite.createSampleDao("expire-treasury")

	// Member submits 0 → DEPOSIT_PERIOD.
	resp, err := suite.msgServer.CreateProposal(suite.ctx, &types.MsgCreateProposal{
		Proposer:       admin,
		DaoId:          daoID,
		Title:          "expire",
		InitialDeposit: sdk.NewInt64Coin("ubze", 0),
	})
	suite.Require().NoError(err)

	// No depositors. After 8 days the deposit_deadline (7d) has passed.
	suite.advanceTime(8 * 24 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ := suite.k.GetProposal(suite.ctx, daoID, resp.ProposalId)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_REJECTED_NO_DEPOSIT, p.Status)
}

// TestDepositExpiry_ForfeitToBurner: when forfeit_destination = BURNER,
// collected deposits flow into the burner module account via
// SendCoinsFromAccountToModule.
//
// We override the default refund-policy by creating a DAO with a custom
// deposit config (BURNER + NEVER). With min_deposit = 5, a single deposit
// of 3 leaves the proposal short of min and triggers forfeit at expiry.
func (suite *IntegrationTestSuite) TestDepositExpiry_ForfeitToBurner() {
	gov := validGovernance()
	dep := types.DepositConfig{
		MinDeposit:         sdk.NewInt64Coin("ubze", 5),
		DepositPeriod:      24 * time.Hour,
		ForfeitDestination: types.ForfeitDestination_FORFEIT_DESTINATION_BURNER,
		VotingRefundPolicy: types.RefundPolicy_REFUND_POLICY_NEVER,
	}
	creator := freshAddr()
	suite.expectAccountCreated(1)
	respDao, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("burn-on-expire"),
		VotingConfig: staticConfig(creator),
		Governance:   gov,
		Deposit:      dep,
	})
	suite.Require().NoError(err)

	suite.expectInitialDepositSend(respDao.DaoId, creator, sdk.NewInt64Coin("ubze", 3))
	resp, err := suite.msgServer.CreateProposal(suite.ctx, &types.MsgCreateProposal{
		Proposer:       creator,
		DaoId:          respDao.DaoId,
		Title:          "burn-expire",
		InitialDeposit: sdk.NewInt64Coin("ubze", 3), // < min(5) → DEPOSIT_PERIOD
	})
	suite.Require().NoError(err)

	// Expect a single send-to-burner-module call for the 3ubze forfeit.
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(
			gomock.Any(),
			types.DepositEscrowAddress(respDao.DaoId),
			burnertypes.ModuleName,
			sdk.NewCoins(sdk.NewInt64Coin("ubze", 3)),
		).
		Return(nil).
		Times(1)

	suite.advanceTime(48 * time.Hour)
	suite.Require().NoError(suite.k.EndBlock(suite.ctx))

	p, _ := suite.k.GetProposal(suite.ctx, respDao.DaoId, resp.ProposalId)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_REJECTED_NO_DEPOSIT, p.Status)

	// Records cleared after forfeit.
	creatorAddr := suite.mustAcc(creator)
	_, has := suite.k.GetDepositRecord(suite.ctx, respDao.DaoId, resp.ProposalId, creatorAddr)
	suite.Require().False(has, "deposit records should be cleared after forfeit")
}
