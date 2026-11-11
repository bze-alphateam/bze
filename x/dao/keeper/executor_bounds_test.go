package keeper_test

import (
	"cosmossdk.io/x/feegrant"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/dao/types"
)

const (
	reasonNotAllowed = "message not allowed"
	reasonTooDeep    = "execution nested too deep"
)

// sendAuthorization is a generic authz grant of MsgSend from granter to grantee.
func sendAuthorization(granter, grantee sdk.AccAddress) *authz.MsgGrant {
	msg, err := authz.NewMsgGrant(granter, grantee, authz.NewGenericAuthorization(sdk.MsgTypeURL(&banktypes.MsgSend{})), nil)
	if err != nil {
		panic(err)
	}

	return msg
}

// feeAllowance is a basic fee allowance from granter to grantee.
func feeAllowance(granter, grantee sdk.AccAddress) *feegrant.MsgGrantAllowance {
	msg, err := feegrant.NewMsgGrantAllowance(&feegrant.BasicAllowance{SpendLimit: ubze(10)}, granter, grantee)
	if err != nil {
		panic(err)
	}

	return msg
}

func execAs(grantee sdk.AccAddress, msgs ...sdk.Msg) *authz.MsgExec {
	msg := authz.NewMsgExec(grantee, msgs)

	return &msg
}

// The admin cannot leave grants signed by the DAO behind: authz grants, fee
// allowances and authz execs are refused on the admin path, and nothing reaches
// the authz or feegrant services.
func (suite *IntegrationTestSuite) TestAdminExecute_GrantsRefused() {
	suite.createAdminDao()
	dao := daoAddress(1)

	for _, tc := range []struct {
		desc string
		msg  sdk.Msg
	}{
		{desc: "authz grant", msg: sendAuthorization(dao, adminAddr)},
		{desc: "fee allowance", msg: feeAllowance(dao, adminAddr)},
		{desc: "authz exec", msg: execAs(dao, sendAuthorization(dao, adminAddr))},
		{desc: "grant after a valid send", msg: sendAuthorization(dao, adminAddr)},
	} {
		msgs := []sdk.Msg{tc.msg}
		if tc.desc == "grant after a valid send" {
			msgs = []sdk.Msg{treasurySend(memberA, 1), tc.msg}
		}

		err := suite.adminExecute(adminAddr, 1, msgs...)
		suite.Require().ErrorIs(err, types.ErrMsgNotAllowed, tc.desc)
	}

	suite.Require().Empty(suite.authz.grants)
	suite.Require().Empty(suite.authz.execs)
	suite.Require().Empty(suite.feegrant.allowances)
	suite.Require().Equal(ubze(100), suite.ledger.balance(dao))
}

// Members voting a proposal may still grant as the DAO.
func (suite *IntegrationTestSuite) TestSettleProposal_ProposalStillGrants() {
	suite.createStaticDao(noConfigChange)
	dao := daoAddress(1)
	grant := sendAuthorization(dao, memberA)
	allowance := feeAllowance(dao, memberA)
	exec := execAs(dao, treasurySend(memberB, 1))
	id := suite.passedProposal(grant, allowance, exec)

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, status)
	suite.requireExecuted("")

	suite.Require().Len(suite.authz.grants, 1)
	suite.Require().Equal(dao.String(), suite.authz.grants[0].Granter)
	suite.Require().Equal(memberA.String(), suite.authz.grants[0].Grantee)
	suite.Require().Len(suite.feegrant.allowances, 1)
	suite.Require().Equal(dao.String(), suite.feegrant.allowances[0].Granter)
	suite.Require().Len(suite.authz.execs, 1)
}

// A proposal cannot settle another proposal, directly or through an authz
// exec; the settled-by-anyone path stays open for the target.
func (suite *IntegrationTestSuite) TestSettleProposal_NestedSettleRefused() {
	dao := daoAddress(1)
	for _, tc := range []struct {
		desc  string
		inner func(target uint64) sdk.Msg
	}{
		{desc: "signed by the dao", inner: func(target uint64) sdk.Msg {
			return types.NewMsgSettleProposal(dao.String(), target)
		}},
		{desc: "inside an authz exec", inner: func(target uint64) sdk.Msg {
			return execAs(dao, types.NewMsgSettleProposal(dao.String(), target))
		}},
	} {
		suite.SetupTest()
		suite.createStaticDao(noConfigChange)
		suite.ledger.fund(dao, ubze(100))
		target := suite.passedProposal(treasurySend(memberB, 10))
		outer := suite.passedProposal(tc.inner(target))

		suite.endVoting(outer, 0)
		status, err := suite.settle(outer)
		suite.Require().NoError(err, tc.desc)
		suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status, tc.desc)
		suite.requireExecuted(reasonNotAllowed)
		suite.Require().Empty(suite.authz.execs, tc.desc)

		// the target was not touched and settles directly, by anyone
		proposal, found := suite.k.GetProposal(suite.ctx, target)
		suite.Require().True(found)
		suite.Require().Equal(types.ProposalStatus_VOTING, proposal.Status, tc.desc)
		suite.Require().Equal(ubze(100), suite.ledger.balance(dao), tc.desc)

		res, err := suite.msgServer.SettleProposal(suite.ctx, types.NewMsgSettleProposal(attackerAddr.String(), target))
		suite.Require().NoError(err, tc.desc)
		suite.Require().Equal(types.ProposalStatus_EXECUTED, res.Status, tc.desc)
		suite.Require().Equal(ubze(90), suite.ledger.balance(dao), tc.desc)
	}
}

// The admin path refuses a settle too.
func (suite *IntegrationTestSuite) TestAdminExecute_SettleRefused() {
	suite.createAdminDao()

	err := suite.adminExecute(adminAddr, 1, types.NewMsgSettleProposal(daoAddress(1).String(), 1))
	suite.Require().ErrorIs(err, types.ErrMsgNotAllowed)
}

// createDaoChain creates DAOs first..last, each administered by the DAO before
// it; firstAdmin administers DAO first. The last DAO gets 100ubze.
func (suite *IntegrationTestSuite) createDaoChain(first, last uint64, firstAdmin sdk.AccAddress) {
	for id := first; id <= last; id++ {
		admin := firstAdmin
		if id > first {
			admin = daoAddress(id - 1)
		}
		suite.createDao(id, func(msg *types.MsgCreateDao) { msg.Admin = admin.String() })
	}
	suite.ledger.fund(daoAddress(last), ubze(100))
}

// nestedAdminExecute returns the MsgAdminExecute that DAO from runs on DAO
// from+1, which runs one on from+2, and so on down to DAO last, which sends
// 1ubze to memberC.
func (suite *IntegrationTestSuite) nestedAdminExecute(from, last uint64) sdk.Msg {
	var msg sdk.Msg = banktypes.NewMsgSend(daoAddress(last), memberC, ubze(1))
	for id := last; id > from; id-- {
		inner, err := types.NewMsgAdminExecute(daoAddress(id-1).String(), id, []sdk.Msg{msg})
		suite.Require().NoError(err)
		msg = inner
	}

	return msg
}

// A DAO administering a sub-DAO administering another, and so on: the admin
// path runs up to MaxDaoExecutionDepth nested executions and refuses one more.
func (suite *IntegrationTestSuite) TestAdminExecute_NestingDepth() {
	suite.Require().Equal(5, types.MaxDaoExecutionDepth)

	for _, tc := range []struct {
		depth   uint64
		wantErr error
	}{
		{depth: 1},
		{depth: 2},
		{depth: types.MaxDaoExecutionDepth},
		{depth: types.MaxDaoExecutionDepth + 1, wantErr: types.ErrExecutionTooDeep},
	} {
		suite.SetupTest()
		suite.createDaoChain(1, tc.depth, adminAddr)

		var top sdk.Msg = banktypes.NewMsgSend(daoAddress(1), memberC, ubze(1))
		if tc.depth > 1 {
			top = suite.nestedAdminExecute(1, tc.depth)
		}
		err := suite.adminExecute(adminAddr, 1, top)

		if tc.wantErr != nil {
			suite.Require().ErrorIs(err, tc.wantErr, "depth %d", tc.depth)
			suite.Require().Empty(suite.ledger.balance(memberC), "depth %d", tc.depth)

			continue
		}
		suite.Require().NoError(err, "depth %d", tc.depth)
		suite.Require().Equal(ubze(1), suite.ledger.balance(memberC), "depth %d", tc.depth)
	}
}

// The proposal counts as the first execution: a self-owned DAO's proposal runs
// sub-DAO executions up to the cap, and one more ends EXECUTION_FAILED with a
// fixed reason and nothing written.
func (suite *IntegrationTestSuite) TestSettleProposal_NestingDepth() {
	for _, tc := range []struct {
		depth      uint64
		wantStatus types.ProposalStatus
		wantReason string
	}{
		{depth: types.MaxDaoExecutionDepth, wantStatus: types.ProposalStatus_EXECUTED},
		{depth: types.MaxDaoExecutionDepth + 1, wantStatus: types.ProposalStatus_EXECUTION_FAILED, wantReason: reasonTooDeep},
	} {
		suite.SetupTest()
		suite.createStaticDao(noConfigChange)
		suite.createDaoChain(2, tc.depth, daoAddress(1))
		id := suite.passedProposal(suite.nestedAdminExecute(1, tc.depth))

		suite.endVoting(id, 0)
		status, err := suite.settle(id)
		suite.Require().NoError(err, "depth %d", tc.depth)
		suite.Require().Equal(tc.wantStatus, status, "depth %d", tc.depth)
		suite.requireExecuted(tc.wantReason)

		want := ubze(1)
		if tc.wantReason != "" {
			want = nil
		}
		suite.Require().Equal(want, suite.ledger.balance(memberC), "depth %d", tc.depth)
	}
}

// expectDaoCreatesDao expects DAO payer to create DAO id, paying the fee.
func (suite *IntegrationTestSuite) expectDaoCreatesDao(payer sdk.AccAddress, id uint64) {
	suite.expectNewDaoAccount(id)
	suite.trade.EXPECT().
		SafeCaptureAndSwapUserFee(gomock.Any(), payer, sdk.NewCoins(types.DefaultDaoCreationFee), burnerModule).
		Return(sdk.NewCoins(types.DefaultDaoCreationFee), nil)
}

func daoCreatesDao(creator sdk.AccAddress) *types.MsgCreateDao {
	msg := validCreateDaoMsg()
	msg.Creator = creator.String()

	return msg
}

// A DAO still creates DAOs, through a proposal and through its admin; the
// creation fee is taken from the DAO (the mock expects the DAO as payer).
func (suite *IntegrationTestSuite) TestCreateDao_FromDao() {
	suite.createStaticDao(noConfigChange)
	suite.expectDaoCreatesDao(daoAddress(1), 2)
	id := suite.passedProposal(daoCreatesDao(daoAddress(1)))

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, status)
	suite.requireExecuted("")
	_, found := suite.k.GetDao(suite.ctx, 2)
	suite.Require().True(found)

	// DAO 3 administered by adminAddr creates DAO 4
	suite.createDao(3, func(msg *types.MsgCreateDao) { msg.Admin = adminAddr.String() })
	suite.expectDaoCreatesDao(daoAddress(3), 4)
	suite.Require().NoError(suite.adminExecute(adminAddr, 3, daoCreatesDao(daoAddress(3))))
	_, found = suite.k.GetDao(suite.ctx, 4)
	suite.Require().True(found)
}
