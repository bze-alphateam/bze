package keeper_test

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// daoPath returns the addresses of the given DAO ids, in order.
func daoPath(ids ...uint64) []string {
	path := make([]string, 0, len(ids))
	for _, id := range ids {
		path = append(path, daoAddress(id).String())
	}

	return path
}

// createAdminChain creates DAOs 1..n where DAO i is administered by DAO i+1
// and DAO n is self-owned.
func (suite *IntegrationTestSuite) createAdminChain(n uint64) {
	for id := uint64(1); id < n; id++ {
		suite.createDao(id, func(msg *types.MsgCreateDao) { msg.Admin = daoAddress(id + 1).String() })
	}
	suite.createDao(n, func(*types.MsgCreateDao) {})
}

func (suite *IntegrationTestSuite) requireController(daoId uint64, controller string, reason types.ControllerEndReason, path []string) {
	res, err := suite.k.DaoController(suite.ctx, &types.QueryDaoControllerRequest{DaoId: daoId})
	suite.Require().NoError(err)

	suite.Require().Equal(controller, res.Controller)
	suite.Require().Equal(reason, res.EndReason)
	suite.Require().Equal(path, res.Path)
}

// D10: a person as admin controls the DAO.
func (suite *IntegrationTestSuite) TestDaoController_ExternalAdmin() {
	suite.createAdminDao()

	suite.requireController(1, adminAddr.String(), types.ControllerEndReason_ACCOUNT, daoPath(1))
}

// D10: a self-owned DAO is controlled by itself (its voters).
func (suite *IntegrationTestSuite) TestDaoController_SelfOwned() {
	suite.createStaticDao(noConfigChange)

	suite.requireController(1, daoAddress(1).String(), types.ControllerEndReason_SELF_OWNED, daoPath(1))
}

// D10: A -> B -> C with C self-owned ends at C, through every DAO on the way.
func (suite *IntegrationTestSuite) TestDaoController_ParentChain() {
	suite.createAdminChain(3)

	suite.requireController(1, daoAddress(3).String(), types.ControllerEndReason_SELF_OWNED, daoPath(1, 2, 3))
	suite.requireController(2, daoAddress(3).String(), types.ControllerEndReason_SELF_OWNED, daoPath(2, 3))
}

// D10: a chain ending at a person names that person, not the last DAO.
func (suite *IntegrationTestSuite) TestDaoController_ChainEndsAtAccount() {
	suite.createDao(1, func(msg *types.MsgCreateDao) { msg.Admin = daoAddress(2).String() })
	suite.createDao(2, func(msg *types.MsgCreateDao) { msg.Admin = adminAddr.String() })

	suite.requireController(1, adminAddr.String(), types.ControllerEndReason_ACCOUNT, daoPath(1, 2))
}

// D10: X <-> Y is reported as a cycle on the DAO seen twice.
func (suite *IntegrationTestSuite) TestDaoController_Cycle() {
	suite.createDao(1, func(msg *types.MsgCreateDao) { msg.Admin = daoAddress(2).String() })
	suite.createDao(2, func(msg *types.MsgCreateDao) { msg.Admin = daoAddress(1).String() })

	suite.requireController(1, daoAddress(1).String(), types.ControllerEndReason_CYCLE, daoPath(1, 2))
	suite.requireController(2, daoAddress(2).String(), types.ControllerEndReason_CYCLE, daoPath(2, 1))
}

// D10: a chain of 20 DAOs stops after 16; a walk that fits in 16 still ends normally.
func (suite *IntegrationTestSuite) TestDaoController_DepthCap() {
	suite.createAdminChain(20)

	suite.requireController(1, daoAddress(16).String(), types.ControllerEndReason_DEPTH_CAP,
		daoPath(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16))
	suite.requireController(4, daoAddress(19).String(), types.ControllerEndReason_DEPTH_CAP,
		daoPath(4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19))
	suite.requireController(5, daoAddress(20).String(), types.ControllerEndReason_SELF_OWNED,
		daoPath(5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20))
}

// A pending nomination does not change who controls the DAO.
func (suite *IntegrationTestSuite) TestDaoController_PendingNominationIgnored() {
	suite.createAdminDao()
	suite.createDao(2, func(*types.MsgCreateDao) {})
	suite.Require().NoError(suite.nominate(adminAddr, 1, daoAddress(2).String()))

	suite.requireController(1, adminAddr.String(), types.ControllerEndReason_ACCOUNT, daoPath(1))
}

func (suite *IntegrationTestSuite) TestDaoController_NotFound() {
	suite.createStaticDao(noConfigChange)

	for _, id := range []uint64{0, 2} {
		res, err := suite.k.DaoController(suite.ctx, &types.QueryDaoControllerRequest{DaoId: id})

		suite.Require().Nil(res)
		suite.Require().Equal(codes.NotFound, status.Code(err), "id %d", id)
	}

	res, err := suite.k.DaoController(suite.ctx, nil)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}
