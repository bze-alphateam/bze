package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// The grant refusal of the admin path holds at every level: an admin that
// reaches a sub-DAO through a nested MsgAdminExecute cannot make the sub-DAO
// grant either.
func (suite *IntegrationTestSuite) TestAdminExecute_NestedSubDaoGrantsRefused() {
	sub := daoAddress(2)
	for _, tc := range []struct {
		desc string
		msg  sdk.Msg
	}{
		{desc: "authz grant", msg: sendAuthorization(sub, adminAddr)},
		{desc: "fee allowance", msg: feeAllowance(sub, adminAddr)},
		{desc: "authz exec", msg: execAs(sub, sendAuthorization(sub, adminAddr))},
	} {
		suite.SetupTest()
		suite.createAdminDao()
		suite.createDaoAdministeredBy(2, daoAddress(1))

		inner, err := types.NewMsgAdminExecute(daoAddress(1).String(), 2, []sdk.Msg{tc.msg})
		suite.Require().NoError(err, tc.desc)
		err = suite.adminExecute(adminAddr, 1, inner)

		suite.Require().ErrorIs(err, types.ErrMsgNotAllowed, tc.desc)
		suite.Require().Empty(suite.authz.grants, tc.desc)
		suite.Require().Empty(suite.authz.execs, tc.desc)
		suite.Require().Empty(suite.feegrant.allowances, tc.desc)
	}
}

// The admin-path refusal checks every message, not only the first: an authz
// exec after a valid send is refused and the send is not kept.
func (suite *IntegrationTestSuite) TestAdminExecute_ExecAfterFirstMsgRefused() {
	suite.createAdminDao()
	dao := daoAddress(1)

	for _, msgs := range [][]sdk.Msg{
		{treasurySend(memberA, 1), execAs(dao, treasurySend(memberB, 1))},
		{treasurySend(memberA, 1), treasurySend(memberA, 1), execAs(dao, treasurySend(memberB, 1))},
	} {
		err := suite.adminExecute(adminAddr, 1, msgs...)
		suite.Require().ErrorIs(err, types.ErrMsgNotAllowed, "%d msgs", len(msgs))
	}

	suite.Require().Empty(suite.authz.execs)
	suite.Require().Empty(suite.ledger.balance(memberA))
	suite.Require().Equal(ubze(100), suite.ledger.balance(dao))
}

// A retry of an EXECUTION_FAILED proposal runs its messages again through the
// same checks: once the cause of the first failure is fixed, the settle it
// carries is still refused and its target stays in voting.
func (suite *IntegrationTestSuite) TestSettleProposal_RetryStillRefusesNestedSettle() {
	suite.createStaticDao(noConfigChange)
	dao := daoAddress(1)
	target := suite.passedProposal()
	outer := suite.passedProposal(treasurySend(memberB, 30), types.NewMsgSettleProposal(dao.String(), target))

	// the unfunded treasury fails the first message before the settle is reached
	suite.endVoting(outer, 0)
	status, err := suite.settle(outer)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)
	suite.requireExecuted("message failed")

	suite.ledger.fund(dao, ubze(30))
	suite.endVoting(outer, 60)
	status, err = suite.settle(outer)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)
	suite.requireStatus(outer, types.ProposalStatus_EXECUTION_FAILED)
	suite.requireExecuted(reasonNotAllowed)

	proposal, found := suite.k.GetProposal(suite.ctx, target)
	suite.Require().True(found)
	suite.Require().Equal(types.ProposalStatus_VOTING, proposal.Status)
}
