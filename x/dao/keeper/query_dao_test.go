package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (suite *IntegrationTestSuite) TestQueryDao_Found() {
	dao := suite.createStaticDao(noConfigChange)

	res, err := suite.k.Dao(suite.ctx, &types.QueryDaoRequest{Id: 1})

	suite.Require().NoError(err)
	suite.Require().Equal(dao, res.Dao)
}

func (suite *IntegrationTestSuite) TestQueryDao_NotFound() {
	suite.createStaticDao(noConfigChange)

	for _, id := range []uint64{0, 2} {
		res, err := suite.k.Dao(suite.ctx, &types.QueryDaoRequest{Id: id})

		suite.Require().Nil(res)
		suite.Require().Equal(codes.NotFound, status.Code(err), "id %d", id)
	}
}

func (suite *IntegrationTestSuite) TestQueryDaoByAddress_Found() {
	suite.createStaticDao(noConfigChange)
	dao := suite.createDao(2, func(msg *types.MsgCreateDao) { msg.Admin = adminAddr.String() })

	res, err := suite.k.DaoByAddress(suite.ctx, &types.QueryDaoByAddressRequest{Address: daoAddress(2).String()})

	suite.Require().NoError(err)
	suite.Require().Equal(dao, res.Dao)
}

func (suite *IntegrationTestSuite) TestQueryDaoByAddress_NotFound() {
	suite.createStaticDao(noConfigChange)

	// a DAO's admin and members are not DAO accounts, nor is an unused derived address
	for _, addr := range []sdk.AccAddress{memberA, creatorAddr, daoAddress(2)} {
		res, err := suite.k.DaoByAddress(suite.ctx, &types.QueryDaoByAddressRequest{Address: addr.String()})

		suite.Require().Nil(res)
		suite.Require().Equal(codes.NotFound, status.Code(err), "address %s", addr)
	}
}

func (suite *IntegrationTestSuite) TestQueryDaoByAddress_InvalidAddress() {
	suite.createStaticDao(noConfigChange)

	for _, addr := range []string{"", "not-an-address", "cosmos1qqqqqqqq"} {
		res, err := suite.k.DaoByAddress(suite.ctx, &types.QueryDaoByAddressRequest{Address: addr})

		suite.Require().Nil(res)
		suite.Require().Equal(codes.InvalidArgument, status.Code(err), "address %q", addr)
	}
}

func (suite *IntegrationTestSuite) TestQueryDao_NilRequests() {
	res, err := suite.k.Dao(suite.ctx, nil)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))

	byAddr, err := suite.k.DaoByAddress(suite.ctx, nil)
	suite.Require().Nil(byAddr)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}
