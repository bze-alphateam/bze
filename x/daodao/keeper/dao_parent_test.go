package keeper_test

import (
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestCreateDao_WithParent verifies parent linkage + the SubDao index.
func (suite *IntegrationTestSuite) TestCreateDao_WithParent() {
	creator := freshAddr()

	suite.expectAccountCreated(1)
	parentResp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("parent"),
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)

	suite.expectAccountCreated(2)
	childResp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("child"),
		ParentDaoId:  parentResp.DaoId,
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)

	child, _ := suite.k.GetDao(suite.ctx, childResp.DaoId)
	suite.Require().Equal(parentResp.DaoId, child.ParentDaoId)

	subResp, err := suite.k.SubDaos(suite.ctx, &types.QuerySubDaosRequest{ParentDaoId: parentResp.DaoId})
	suite.Require().NoError(err)
	suite.Require().Len(subResp.Daos, 1)
	suite.Require().Equal(childResp.DaoId, subResp.Daos[0].Id)
}

// TestCreateDao_RejectsMissingParent rejects parent_dao_id pointing at a
// non-existent DAO. No account creation should happen.
func (suite *IntegrationTestSuite) TestCreateDao_RejectsMissingParent() {
	creator := freshAddr()
	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("orphan"),
		ParentDaoId:  999,
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().ErrorContains(err, "parent")
}

// TestParentChain_NoCycle confirms a multi-level chain with no cycle is OK.
func (suite *IntegrationTestSuite) TestParentChain_NoCycle() {
	creator := freshAddr()

	suite.expectAccountCreated(1)
	a, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator: creator, Metadata: sampleMetadata("A"), VotingConfig: staticConfig(creator), Governance: validGovernance(), Deposit: validDeposit(),
	})
	suite.Require().NoError(err)

	suite.expectAccountCreated(2)
	b, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator: creator, Metadata: sampleMetadata("B"), ParentDaoId: a.DaoId, VotingConfig: staticConfig(creator), Governance: validGovernance(), Deposit: validDeposit(),
	})
	suite.Require().NoError(err)

	suite.expectAccountCreated(3)
	c, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator: creator, Metadata: sampleMetadata("C"), ParentDaoId: b.DaoId, VotingConfig: staticConfig(creator), Governance: validGovernance(), Deposit: validDeposit(),
	})
	suite.Require().NoError(err)

	suite.expectAccountCreated(4)
	d, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator: creator, Metadata: sampleMetadata("D"), ParentDaoId: c.DaoId, VotingConfig: staticConfig(creator), Governance: validGovernance(), Deposit: validDeposit(),
	})
	suite.Require().NoError(err)

	dao, _ := suite.k.GetDao(suite.ctx, d.DaoId)
	suite.Require().Equal(c.DaoId, dao.ParentDaoId)
}

// TestParentChain_NonexistentParentRejected: parent_dao_id pointing at a
// DAO that doesn't exist yet is rejected.
func (suite *IntegrationTestSuite) TestParentChain_NonexistentParentRejected() {
	creator := freshAddr()
	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("S"),
		ParentDaoId:  1,
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().ErrorContains(err, "parent")
}
