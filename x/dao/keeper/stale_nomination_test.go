package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// A nomination a DAO made on its sub-DAO holds only while every DAO on the
// sub-DAO's admin chain keeps its admin: a former controller cannot accept it.

var heirAddr = sdk.AccAddress("dao_heir____________")

// createDaoAdministeredBy creates the next DAO, id, with admin as its admin.
func (suite *IntegrationTestSuite) createDaoAdministeredBy(id uint64, admin sdk.AccAddress) {
	suite.expectCreation(id)
	msg := validCreateDaoMsg()
	msg.Admin = admin.String()
	_, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)
}

// handOver hands daoId from its wallet admin to newAdmin in two steps.
func (suite *IntegrationTestSuite) handOver(admin sdk.AccAddress, daoId uint64, newAdmin sdk.AccAddress) {
	suite.Require().NoError(suite.nominate(admin, daoId, newAdmin.String()))
	suite.Require().NoError(suite.accept(newAdmin, daoId))
}

func (suite *IntegrationTestSuite) requireAdmin(daoId uint64, wantAdmin sdk.AccAddress, wantPending string) {
	dao, found := suite.k.GetDao(suite.ctx, daoId)
	suite.Require().True(found)
	suite.Require().Equal(wantAdmin.String(), dao.Admin, "dao %d admin", daoId)
	suite.Require().Equal(wantPending, dao.PendingAdmin, "dao %d pending admin", daoId)
}

// subDaoTree creates DAO 1 administered by adminAddr, DAO 2 administered by
// DAO 1 and DAO 3 administered by DAO 2.
func (suite *IntegrationTestSuite) subDaoTree() {
	suite.createAdminDao()
	suite.createDaoAdministeredBy(2, daoAddress(1))
	suite.createDaoAdministeredBy(3, daoAddress(2))
}

// nominateOnSub makes DAO 1, through its admin, nominate nominee on DAO 2.
func (suite *IntegrationTestSuite) nominateOnSub(admin sdk.AccAddress, nominee sdk.AccAddress) error {
	return suite.adminExecute(admin, 1, types.NewMsgNominateAdmin(daoAddress(1).String(), 2, nominee.String()))
}

// nominateOnSubSub makes DAO 2, through DAO 1 and its admin, nominate nominee on DAO 3.
func (suite *IntegrationTestSuite) nominateOnSubSub(admin sdk.AccAddress, nominee sdk.AccAddress) error {
	inner, err := types.NewMsgAdminExecute(daoAddress(1).String(), 2, []sdk.Msg{
		types.NewMsgNominateAdmin(daoAddress(2).String(), 3, nominee.String()),
	})
	suite.Require().NoError(err)

	return suite.adminExecute(admin, 1, inner)
}

// The admin makes DAO 1 nominate itself on DAO 2, then hands DAO 1 over: the
// nomination it left behind cannot be accepted.
func (suite *IntegrationTestSuite) TestStaleNomination_DirectSubDaoAfterHandover() {
	suite.subDaoTree()
	suite.Require().NoError(suite.nominateOnSub(adminAddr, adminAddr))
	suite.handOver(adminAddr, 1, successorAddr)

	err := suite.accept(adminAddr, 2)

	suite.Require().ErrorIs(err, types.ErrStaleNomination)
	suite.requireAdmin(2, daoAddress(1), adminAddr.String())
	suite.Require().ErrorIs(suite.adminExecute(adminAddr, 2, treasurySend(adminAddr, 1)), types.ErrNotDaoAdmin)
}

// One level deeper: the admin makes DAO 2 (through DAO 1) nominate itself on
// DAO 3. DAO 2's admin never changes, but DAO 1's does.
func (suite *IntegrationTestSuite) TestStaleNomination_NestedSubDaoAfterHandover() {
	suite.subDaoTree()
	suite.Require().NoError(suite.nominateOnSubSub(adminAddr, adminAddr))
	suite.handOver(adminAddr, 1, successorAddr)

	err := suite.accept(adminAddr, 3)

	suite.Require().ErrorIs(err, types.ErrStaleNomination)
	suite.requireAdmin(3, daoAddress(2), adminAddr.String())
	suite.requireAdmin(2, daoAddress(1), "")
}

// The change one level up the chain voids a nomination too: DAO 2 itself is
// handed to a new admin after it nominated on DAO 3.
func (suite *IntegrationTestSuite) TestStaleNomination_NestedIntermediateHandover() {
	suite.subDaoTree()
	suite.Require().NoError(suite.nominateOnSubSub(adminAddr, adminAddr))
	suite.Require().NoError(suite.nominateOnSub(adminAddr, successorAddr))
	suite.Require().NoError(suite.accept(successorAddr, 2))

	err := suite.accept(adminAddr, 3)

	suite.Require().ErrorIs(err, types.ErrStaleNomination)
	suite.requireAdmin(3, daoAddress(2), adminAddr.String())
}

// Handing DAO 1 to its own voters ("nominate nobody") is an admin change too.
func (suite *IntegrationTestSuite) TestStaleNomination_AfterParentMadeSelfOwned() {
	suite.subDaoTree()
	suite.Require().NoError(suite.nominateOnSub(adminAddr, adminAddr))
	suite.Require().NoError(suite.nominateOnSubSub(adminAddr, adminAddr))
	suite.Require().NoError(suite.nominate(adminAddr, 1, ""))

	suite.Require().ErrorIs(suite.accept(adminAddr, 2), types.ErrStaleNomination)
	suite.Require().ErrorIs(suite.accept(adminAddr, 3), types.ErrStaleNomination)
	suite.requireAdmin(2, daoAddress(1), adminAddr.String())
	suite.requireAdmin(3, daoAddress(2), adminAddr.String())
}

// A self-owned DAO 1 (its proposal, called directly here) nominates on DAO 2,
// then is handed to an external admin: its voters' nomination is void.
func (suite *IntegrationTestSuite) TestStaleNomination_SelfOwnedParentHandedOver() {
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)
	suite.createDaoAdministeredBy(2, daoAddress(1))
	suite.Require().NoError(suite.nominate(daoAddress(1), 2, memberA.String()))
	suite.handOver(daoAddress(1), 1, successorAddr)

	err = suite.accept(memberA, 2)

	suite.Require().ErrorIs(err, types.ErrStaleNomination)
	suite.requireAdmin(2, daoAddress(1), memberA.String())
}

// The new admin clears the stale nomination through DAO 1 and nominates anew;
// that nomination is accepted.
func (suite *IntegrationTestSuite) TestStaleNomination_NewControllerWithdrawsAndRenominates() {
	suite.subDaoTree()
	suite.Require().NoError(suite.nominateOnSub(adminAddr, adminAddr))
	suite.handOver(adminAddr, 1, successorAddr)

	// the stale nomination still blocks a new one until it is withdrawn
	suite.Require().ErrorIs(suite.nominateOnSub(successorAddr, heirAddr), types.ErrNominationPending)
	suite.Require().NoError(suite.adminExecute(successorAddr, 1, types.NewMsgWithdrawAdminNomination(daoAddress(1).String(), 2)))
	suite.Require().NoError(suite.nominateOnSub(successorAddr, heirAddr))

	suite.Require().NoError(suite.accept(heirAddr, 2))
	suite.requireAdmin(2, heirAddr, "")
}

// A nomination made by the current controller, after the handover, is accepted.
func (suite *IntegrationTestSuite) TestStaleNomination_MadeAfterHandoverAccepted() {
	suite.subDaoTree()
	suite.handOver(adminAddr, 1, successorAddr)
	suite.Require().NoError(suite.nominateOnSubSub(successorAddr, heirAddr))

	suite.Require().NoError(suite.accept(heirAddr, 3))
	suite.requireAdmin(3, heirAddr, "")
}

// Changes that move no admin on the chain leave a nomination valid: an admin
// change of an unrelated DAO, a metadata update and a members update of the
// nominating DAO, and a handover of the nominee DAO's own sub-DAO.
func (suite *IntegrationTestSuite) TestStaleNomination_UnrelatedChangesKeepItValid() {
	suite.subDaoTree()
	suite.createDaoAdministeredBy(4, adminAddr)
	suite.Require().NoError(suite.nominateOnSub(adminAddr, heirAddr))
	suite.Require().NoError(suite.nominate(adminAddr, 1, memberA.String()))

	// another DAO changes its admin
	suite.handOver(adminAddr, 4, successorAddr)
	// DAO 3, below DAO 2, changes its admin: it is not on DAO 2's admin chain
	inner, err := types.NewMsgAdminExecute(daoAddress(1).String(), 2, []sdk.Msg{
		types.NewMsgNominateAdmin(daoAddress(2).String(), 3, successorAddr.String()),
	})
	suite.Require().NoError(err)
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, inner))
	suite.Require().NoError(suite.accept(successorAddr, 3))
	// DAO 1's metadata and members change
	suite.Require().NoError(suite.adminExecute(adminAddr, 1,
		types.NewMsgUpdateMetadata(daoAddress(1).String(), 1, "renamed", "still the same admin"),
		types.NewMsgUpdateMembers(daoAddress(1).String(), 1, []string{memberC.String()}, nil),
	))

	suite.Require().NoError(suite.accept(heirAddr, 2))
	suite.requireAdmin(2, heirAddr, "")
	// a wallet-administered DAO's own nomination is unaffected too
	suite.Require().NoError(suite.accept(memberA, 1))
	suite.requireAdmin(1, memberA, "")
}
