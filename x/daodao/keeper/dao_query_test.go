package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestQueryDao_NotFoundAndOK exercises the single-DAO query.
func (suite *IntegrationTestSuite) TestQueryDao_NotFoundAndOK() {
	_, err := suite.k.Dao(suite.ctx, &types.QueryDaoRequest{DaoId: 0})
	suite.Require().Error(err)

	_, err = suite.k.Dao(suite.ctx, &types.QueryDaoRequest{DaoId: 99})
	suite.Require().Error(err)

	creator := freshAddr()
	suite.expectAccountCreated(1)
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("alpha"),
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)

	q, err := suite.k.Dao(suite.ctx, &types.QueryDaoRequest{DaoId: resp.DaoId})
	suite.Require().NoError(err)
	suite.Require().Equal(resp.DaoId, q.Dao.Id)
}

// TestQueryDaos_Pagination creates 5 DAOs and walks them via Daos.
func (suite *IntegrationTestSuite) TestQueryDaos_Pagination() {
	creator := freshAddr()
	for i := uint64(1); i <= 5; i++ {
		suite.expectAccountCreated(i)
		_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
			Creator: creator, Metadata: sampleMetadata("d"), VotingConfig: staticConfig(creator), Governance: validGovernance(), Deposit: validDeposit(),
		})
		suite.Require().NoError(err)
	}

	resp, err := suite.k.Daos(suite.ctx, &types.QueryDaosRequest{})
	suite.Require().NoError(err)
	suite.Require().Len(resp.Daos, 5)
	for i := range resp.Daos {
		suite.Require().Equal(uint64(i+1), resp.Daos[i].Id)
	}
}

func (suite *IntegrationTestSuite) TestQueryDaosByCreator_FiltersByAddress() {
	a := freshAddr()
	b := freshAddr()

	suite.expectAccountCreated(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator: a, Metadata: sampleMetadata("a1"), VotingConfig: staticConfig(a), Governance: validGovernance(), Deposit: validDeposit(),
	})
	suite.Require().NoError(err)

	suite.expectAccountCreated(2)
	_, err = suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator: b, Metadata: sampleMetadata("b1"), VotingConfig: staticConfig(b), Governance: validGovernance(), Deposit: validDeposit(),
	})
	suite.Require().NoError(err)

	suite.expectAccountCreated(3)
	_, err = suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator: a, Metadata: sampleMetadata("a2"), VotingConfig: staticConfig(a), Governance: validGovernance(), Deposit: validDeposit(),
	})
	suite.Require().NoError(err)

	respA, err := suite.k.DaosByCreator(suite.ctx, &types.QueryDaosByCreatorRequest{Creator: a})
	suite.Require().NoError(err)
	suite.Require().Len(respA.Daos, 2)
	for _, d := range respA.Daos {
		suite.Require().Equal(a, d.Creator)
	}

	respB, err := suite.k.DaosByCreator(suite.ctx, &types.QueryDaosByCreatorRequest{Creator: b})
	suite.Require().NoError(err)
	suite.Require().Len(respB.Daos, 1)
	suite.Require().Equal(b, respB.Daos[0].Creator)
}

func (suite *IntegrationTestSuite) TestQueryEndpoints_NilRequest() {
	_, err := suite.k.Dao(suite.ctx, nil)
	suite.Require().Error(err)

	_, err = suite.k.DaoByAddress(suite.ctx, nil)
	suite.Require().Error(err)

	_, err = suite.k.Daos(suite.ctx, nil)
	suite.Require().Error(err)

	_, err = suite.k.DaosByCreator(suite.ctx, nil)
	suite.Require().Error(err)

	_, err = suite.k.SubDaos(suite.ctx, nil)
	suite.Require().Error(err)

	_, err = suite.k.VotingPower(suite.ctx, nil)
	suite.Require().Error(err)

	_, err = suite.k.TotalVotingPower(suite.ctx, nil)
	suite.Require().Error(err)

	_, err = suite.k.Members(suite.ctx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryDaoByAddress_BadBech32() {
	_, err := suite.k.DaoByAddress(suite.ctx, &types.QueryDaoByAddressRequest{
		Address: "not-a-bech32",
	})
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryDaosByCreator_BadBech32() {
	_, err := suite.k.DaosByCreator(suite.ctx, &types.QueryDaosByCreatorRequest{
		Creator: "not-a-bech32",
	})
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQuerySubDaos_ZeroParentRejected() {
	_, err := suite.k.SubDaos(suite.ctx, &types.QuerySubDaosRequest{ParentDaoId: 0})
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQuery_EmptyResults() {
	resp, err := suite.k.Daos(suite.ctx, &types.QueryDaosRequest{})
	suite.Require().NoError(err)
	suite.Require().Empty(resp.Daos)

	respCreator, err := suite.k.DaosByCreator(suite.ctx, &types.QueryDaosByCreatorRequest{
		Creator: freshAddr(),
	})
	suite.Require().NoError(err)
	suite.Require().Empty(respCreator.Daos)
}

func (suite *IntegrationTestSuite) TestQuerySubDaos_EmptyResults() {
	creator := freshAddr()
	suite.expectAccountCreated(1)
	parent, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("lone"),
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)

	resp, err := suite.k.SubDaos(suite.ctx, &types.QuerySubDaosRequest{
		ParentDaoId: parent.DaoId,
	})
	suite.Require().NoError(err)
	suite.Require().Empty(resp.Daos)
}

func (suite *IntegrationTestSuite) TestQueryDaos_PaginationLimitAndOffset() {
	creator := freshAddr()
	for i := uint64(1); i <= 5; i++ {
		suite.expectAccountCreated(i)
		_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
			Creator: creator, Metadata: sampleMetadata("d"), VotingConfig: staticConfig(creator), Governance: validGovernance(), Deposit: validDeposit(),
		})
		suite.Require().NoError(err)
	}

	resp, err := suite.k.Daos(suite.ctx, &types.QueryDaosRequest{
		Pagination: &query.PageRequest{Limit: 2, Offset: 0},
	})
	suite.Require().NoError(err)
	suite.Require().Len(resp.Daos, 2)
	suite.Require().Equal(uint64(1), resp.Daos[0].Id)
	suite.Require().Equal(uint64(2), resp.Daos[1].Id)

	resp, err = suite.k.Daos(suite.ctx, &types.QueryDaosRequest{
		Pagination: &query.PageRequest{Limit: 10, Offset: 3},
	})
	suite.Require().NoError(err)
	suite.Require().Len(resp.Daos, 2)
	suite.Require().Equal(uint64(4), resp.Daos[0].Id)
	suite.Require().Equal(uint64(5), resp.Daos[1].Id)
}

func (suite *IntegrationTestSuite) TestQueryDaoByAddress_RoundTrip() {
	creator := freshAddr()
	suite.expectAccountCreated(1)
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("zz"),
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)

	q, err := suite.k.DaoByAddress(suite.ctx, &types.QueryDaoByAddressRequest{Address: resp.AccountAddress})
	suite.Require().NoError(err)
	suite.Require().Equal(resp.DaoId, q.Dao.Id)

	_, err = suite.k.DaoByAddress(suite.ctx, &types.QueryDaoByAddressRequest{
		Address: freshAddr(),
	})
	suite.Require().Error(err)
}

// -------- Voting-power & members queries --------

func (suite *IntegrationTestSuite) TestQueryMembers_Static() {
	creator := freshAddr()
	other := freshAddr()
	suite.expectAccountCreated(1)
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:  creator,
		Metadata: sampleMetadata("mems"),
		VotingConfig: staticConfigWithMembers([]types.StaticMember{
			{Address: creator, Weight: 1},
			{Address: other, Weight: 2},
		}),
		Governance: validGovernance(),
		Deposit:    validDeposit(),
	})
	suite.Require().NoError(err)

	q, err := suite.k.Members(suite.ctx, &types.QueryMembersRequest{DaoId: resp.DaoId})
	suite.Require().NoError(err)
	suite.Require().Len(q.Members, 2)

	// Weights are correct regardless of iteration order.
	got := map[string]uint64{}
	for _, m := range q.Members {
		got[m.Address] = m.Weight
	}
	suite.Require().Equal(uint64(1), got[creator])
	suite.Require().Equal(uint64(2), got[other])
}

func (suite *IntegrationTestSuite) TestQueryVotingPower_NonMember() {
	creator := freshAddr()
	suite.expectAccountCreated(1)
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("solo"),
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)

	q, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{
		DaoId:   resp.DaoId,
		Address: freshAddr(), // not a member
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(0), q.Power)
	suite.Require().Equal(uint64(1), q.Total)
}
