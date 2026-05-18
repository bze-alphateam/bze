package keeper_test

import (
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestUpdateDaoMetadata_OnlyAdmin: admin can update, anyone else cannot.
func (suite *IntegrationTestSuite) TestUpdateDaoMetadata_OnlyAdmin() {
	daoID, admin := suite.createSampleDao("alpha")
	newMeta := sampleMetadata("alpha-renamed")

	// Random caller is rejected.
	_, err := suite.msgServer.UpdateDaoMetadata(suite.ctx, &types.MsgUpdateDaoMetadata{
		Authority: freshAddr(),
		DaoId:     daoID,
		Metadata:  newMeta,
	})
	suite.Require().ErrorContains(err, "unauthorized")

	// Admin call succeeds.
	_, err = suite.msgServer.UpdateDaoMetadata(suite.ctx, &types.MsgUpdateDaoMetadata{
		Authority: admin,
		DaoId:     daoID,
		Metadata:  newMeta,
	})
	suite.Require().NoError(err)

	dao, _ := suite.k.GetDao(suite.ctx, daoID)
	suite.Require().Equal("alpha-renamed", dao.Metadata.Name)
}

// TestAdminHandoff_TwoStep: nominate → accept changes admin; intermediate
// states are correct.
func (suite *IntegrationTestSuite) TestAdminHandoff_TwoStep() {
	daoID, oldAdmin := suite.createSampleDao("alpha")
	newAdmin := freshAddr()

	// Nominate.
	_, err := suite.msgServer.UpdateDaoAdmin(suite.ctx, &types.MsgUpdateDaoAdmin{
		Authority: oldAdmin,
		DaoId:     daoID,
		NewAdmin:  newAdmin,
	})
	suite.Require().NoError(err)
	dao, _ := suite.k.GetDao(suite.ctx, daoID)
	suite.Require().Equal(oldAdmin, dao.Admin)
	suite.Require().Equal(newAdmin, dao.PendingAdmin)

	// Wrong acceptor rejected.
	_, err = suite.msgServer.AcceptDaoAdmin(suite.ctx, &types.MsgAcceptDaoAdmin{
		NewAdmin: freshAddr(),
		DaoId:    daoID,
	})
	suite.Require().ErrorContains(err, "pending")

	// Correct acceptor succeeds.
	_, err = suite.msgServer.AcceptDaoAdmin(suite.ctx, &types.MsgAcceptDaoAdmin{
		NewAdmin: newAdmin,
		DaoId:    daoID,
	})
	suite.Require().NoError(err)
	dao, _ = suite.k.GetDao(suite.ctx, daoID)
	suite.Require().Equal(newAdmin, dao.Admin)
	suite.Require().Empty(dao.PendingAdmin)

	// Old admin can no longer act.
	_, err = suite.msgServer.UpdateDaoMetadata(suite.ctx, &types.MsgUpdateDaoMetadata{
		Authority: oldAdmin,
		DaoId:     daoID,
		Metadata:  sampleMetadata("blocked"),
	})
	suite.Require().ErrorContains(err, "unauthorized")
}

// TestAcceptDaoAdmin_NoPendingFails: AcceptDaoAdmin rejected when no
// nomination exists.
func (suite *IntegrationTestSuite) TestAcceptDaoAdmin_NoPendingFails() {
	daoID, _ := suite.createSampleDao("alpha")

	_, err := suite.msgServer.AcceptDaoAdmin(suite.ctx, &types.MsgAcceptDaoAdmin{
		NewAdmin: freshAddr(),
		DaoId:    daoID,
	})
	suite.Require().ErrorContains(err, "no pending")
}

// TestUpdateDaoAdmin_OverwriteNomination: re-nominating cancels the prior
// nominee.
func (suite *IntegrationTestSuite) TestUpdateDaoAdmin_OverwriteNomination() {
	daoID, admin := suite.createSampleDao("alpha")
	first := freshAddr()
	second := freshAddr()

	_, err := suite.msgServer.UpdateDaoAdmin(suite.ctx, &types.MsgUpdateDaoAdmin{Authority: admin, DaoId: daoID, NewAdmin: first})
	suite.Require().NoError(err)

	_, err = suite.msgServer.UpdateDaoAdmin(suite.ctx, &types.MsgUpdateDaoAdmin{Authority: admin, DaoId: daoID, NewAdmin: second})
	suite.Require().NoError(err)

	// First nominee can no longer accept.
	_, err = suite.msgServer.AcceptDaoAdmin(suite.ctx, &types.MsgAcceptDaoAdmin{NewAdmin: first, DaoId: daoID})
	suite.Require().Error(err)

	// Second nominee can.
	_, err = suite.msgServer.AcceptDaoAdmin(suite.ctx, &types.MsgAcceptDaoAdmin{NewAdmin: second, DaoId: daoID})
	suite.Require().NoError(err)
}

// TestUpdateDaoAdmin_RejectsSelfNomination: nominating the current admin
// again is rejected (caught by ValidateBasic).
func (suite *IntegrationTestSuite) TestUpdateDaoAdmin_RejectsSelfNomination() {
	daoID, admin := suite.createSampleDao("alpha")

	_, err := suite.msgServer.UpdateDaoAdmin(suite.ctx, &types.MsgUpdateDaoAdmin{
		Authority: admin,
		DaoId:     daoID,
		NewAdmin:  admin,
	})
	suite.Require().Error(err)
}
