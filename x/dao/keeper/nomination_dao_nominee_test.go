package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// acceptThroughAdmin makes DAO nomineeId accept the nomination on DAO daoId,
// signed by the DAO through MsgAdminExecute from admin.
func (suite *IntegrationTestSuite) acceptThroughAdmin(admin sdk.AccAddress, nomineeId, daoId uint64) error {
	return suite.adminExecute(admin, nomineeId, types.NewMsgAcceptAdminNomination(daoAddress(nomineeId).String(), daoId))
}

// A DAO nominee with an external admin accepts through admin-execute: it
// becomes the admin and its own admin then runs messages as the DAO it took
// over; the former admin cannot.
func (suite *IntegrationTestSuite) TestAcceptAdminNomination_DaoNomineeThroughAdminExecute() {
	suite.createAdminDao()
	suite.createDaoAdministeredBy(2, heirAddr)
	suite.Require().NoError(suite.nominate(adminAddr, 1, daoAddress(2).String()))

	suite.Require().NoError(suite.acceptThroughAdmin(heirAddr, 2, 1))

	suite.requireAdmin(1, daoAddress(2), "")
	suite.requireAdmin(2, heirAddr, "")
	suite.Require().ErrorIs(suite.adminExecute(adminAddr, 1, treasurySend(adminAddr, 1)), types.ErrNotDaoAdmin)
	inner, err := types.NewMsgAdminExecute(daoAddress(2).String(), 1, []sdk.Msg{treasurySend(memberA, 40)})
	suite.Require().NoError(err)
	suite.Require().NoError(suite.adminExecute(heirAddr, 2, inner))
	suite.Require().Equal(ubze(60), suite.ledger.balance(daoAddress(1)))
}

// adminCycle makes DAO 1 (administered by adminAddr) and DAO 2 (administered
// by DAO 1) administer each other: DAO 1 nominates DAO 2, and DAO 2 accepts
// through DAO 1's admin, nested.
func (suite *IntegrationTestSuite) adminCycle() {
	suite.createAdminDao()
	suite.createDaoAdministeredBy(2, daoAddress(1))
	suite.Require().NoError(suite.nominate(adminAddr, 1, daoAddress(2).String()))

	inner, err := types.NewMsgAdminExecute(daoAddress(1).String(), 2, []sdk.Msg{
		types.NewMsgAcceptAdminNomination(daoAddress(2).String(), 1),
	})
	suite.Require().NoError(err)
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, inner))

	suite.requireAdmin(1, daoAddress(2), "")
	suite.requireAdmin(2, daoAddress(1), "")
}

// Two DAOs administering each other are a cycle: the controller query reports
// it, the former wallet admin controls neither, and the members' proposals
// (called directly here) still hand a DAO out of the cycle: the accept walks
// the cycle and ends.
func (suite *IntegrationTestSuite) TestAdminCycle_FormedAndLeftThroughNominations() {
	suite.adminCycle()

	suite.requireController(1, daoAddress(1).String(), types.ControllerEndReason_CYCLE, daoPath(1, 2))
	suite.requireController(2, daoAddress(2).String(), types.ControllerEndReason_CYCLE, daoPath(2, 1))
	for _, id := range []uint64{1, 2} {
		err := suite.adminExecute(adminAddr, id, treasurySend(adminAddr, 1))
		suite.Require().ErrorIs(err, types.ErrNotDaoAdmin, "dao %d", id)
	}

	// DAO 2, the admin of DAO 1, nominates a wallet on it
	suite.Require().NoError(suite.nominate(daoAddress(2), 1, heirAddr.String()))
	suite.Require().NoError(suite.accept(heirAddr, 1))

	suite.requireAdmin(1, heirAddr, "")
	suite.requireController(1, heirAddr.String(), types.ControllerEndReason_ACCOUNT, daoPath(1))
	suite.requireController(2, heirAddr.String(), types.ControllerEndReason_ACCOUNT, daoPath(2, 1))
}

// A nomination on a DAO inside a cycle is void once the other DAO of the cycle
// changed its admin after it was made, even when the cycle is formed again.
func (suite *IntegrationTestSuite) TestAdminCycle_NominationVoidAfterCycleChanges() {
	suite.adminCycle()
	suite.Require().NoError(suite.nominate(daoAddress(2), 1, heirAddr.String()))

	// DAO 1 hands DAO 2 to a wallet, which hands it back: the cycle is formed again
	suite.Require().NoError(suite.nominate(daoAddress(1), 2, successorAddr.String()))
	suite.Require().NoError(suite.accept(successorAddr, 2))
	suite.Require().NoError(suite.nominate(successorAddr, 2, daoAddress(1).String()))
	suite.Require().NoError(suite.accept(daoAddress(1), 2))
	suite.requireController(1, daoAddress(1).String(), types.ControllerEndReason_CYCLE, daoPath(1, 2))

	err := suite.accept(heirAddr, 1)

	suite.Require().ErrorIs(err, types.ErrStaleNomination)
	suite.requireAdmin(1, daoAddress(2), heirAddr.String())
}
