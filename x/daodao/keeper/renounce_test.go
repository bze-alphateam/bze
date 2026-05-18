package keeper_test

import (
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestRenounceAdmin_HappyPath: the current admin renounces; the DAO's
// admin becomes its own account; pending nomination (if any) is cleared.
func (suite *IntegrationTestSuite) TestRenounceAdmin_HappyPath() {
	daoID, admin := suite.createSampleDao("renounce")
	dao, _ := suite.k.GetDao(suite.ctx, daoID)
	suite.Require().Equal(admin, dao.Admin)

	_, err := suite.msgServer.RenounceAdmin(suite.ctx, &types.MsgRenounceAdmin{
		Authority: admin,
		DaoId:     daoID,
	})
	suite.Require().NoError(err)

	dao, _ = suite.k.GetDao(suite.ctx, daoID)
	suite.Require().Equal(dao.AccountAddress, dao.Admin,
		"after renunciation, admin must equal the DAO's own account")
	suite.Require().Empty(dao.PendingAdmin, "pending nomination must be cleared on renounce")
}

// TestRenounceAdmin_NonAdminRejected: only the current admin may renounce.
func (suite *IntegrationTestSuite) TestRenounceAdmin_NonAdminRejected() {
	daoID, _ := suite.createSampleDao("renounce-auth")
	intruder := freshAddr()

	_, err := suite.msgServer.RenounceAdmin(suite.ctx, &types.MsgRenounceAdmin{
		Authority: intruder,
		DaoId:     daoID,
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "unauthorized")
}

// TestRenounceAdmin_AlreadySelfRejected: second renounce is rejected.
// Idempotency is rejected loudly rather than silently no-op'd so a
// confused operator doesn't think the call did something.
func (suite *IntegrationTestSuite) TestRenounceAdmin_AlreadySelfRejected() {
	daoID, admin := suite.createSampleDao("renounce-twice")

	_, err := suite.msgServer.RenounceAdmin(suite.ctx, &types.MsgRenounceAdmin{
		Authority: admin,
		DaoId:     daoID,
	})
	suite.Require().NoError(err)

	// Second call must fail. Note: the second call is signed by the OLD
	// admin, which is no longer admin → assertAdmin rejects with
	// "unauthorized" BEFORE we ever reach the self-admin guard. Either
	// rejection is correct; we just need it to fail.
	_, err = suite.msgServer.RenounceAdmin(suite.ctx, &types.MsgRenounceAdmin{
		Authority: admin,
		DaoId:     daoID,
	})
	suite.Require().Error(err)
}

// TestRenounceAdmin_PendingAdminCleared: a handoff-in-flight is canceled
// when renounce fires.
func (suite *IntegrationTestSuite) TestRenounceAdmin_PendingAdminCleared() {
	daoID, admin := suite.createSampleDao("renounce-pending")
	nominee := freshAddr()

	// Nominate a new admin first.
	_, err := suite.msgServer.UpdateDaoAdmin(suite.ctx, &types.MsgUpdateDaoAdmin{
		Authority: admin,
		DaoId:     daoID,
		NewAdmin:  nominee,
	})
	suite.Require().NoError(err)

	dao, _ := suite.k.GetDao(suite.ctx, daoID)
	suite.Require().Equal(nominee, dao.PendingAdmin)

	// Renounce — pending must be cleared.
	_, err = suite.msgServer.RenounceAdmin(suite.ctx, &types.MsgRenounceAdmin{
		Authority: admin,
		DaoId:     daoID,
	})
	suite.Require().NoError(err)

	dao, _ = suite.k.GetDao(suite.ctx, daoID)
	suite.Require().Empty(dao.PendingAdmin)
	suite.Require().Equal(dao.AccountAddress, dao.Admin)

	// Nominee trying to accept the canceled handoff fails.
	_, err = suite.msgServer.AcceptDaoAdmin(suite.ctx, &types.MsgAcceptDaoAdmin{
		NewAdmin: nominee,
		DaoId:    daoID,
	})
	suite.Require().Error(err)
}

// TestRenounceAdmin_PostRenounceAdminCallsFail: after renunciation, the
// previous human admin's direct admin-gated calls are rejected.
func (suite *IntegrationTestSuite) TestRenounceAdmin_PostRenounceAdminCallsFail() {
	daoID, admin := suite.createSampleDao("post-renounce")

	_, err := suite.msgServer.RenounceAdmin(suite.ctx, &types.MsgRenounceAdmin{
		Authority: admin,
		DaoId:     daoID,
	})
	suite.Require().NoError(err)

	// The previous admin can no longer update metadata.
	_, err = suite.msgServer.UpdateDaoMetadata(suite.ctx, &types.MsgUpdateDaoMetadata{
		Authority: admin,
		DaoId:     daoID,
		Metadata:  sampleMetadata("renamed"),
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "unauthorized")
}
