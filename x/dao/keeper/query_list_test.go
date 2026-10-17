package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func daoIds(daos []types.Dao) []uint64 {
	ids := []uint64{}
	for _, dao := range daos {
		ids = append(ids, dao.Id)
	}

	return ids
}

func proposalIds(proposals []types.Proposal) []uint64 {
	ids := []uint64{}
	for _, proposal := range proposals {
		ids = append(ids, proposal.Id)
	}

	return ids
}

func ballotVoters(ballots []types.Ballot) []string {
	voters := []string{}
	for _, ballot := range ballots {
		voters = append(voters, ballot.Voter)
	}

	return voters
}

func (suite *IntegrationTestSuite) daos(page *query.PageRequest) *types.QueryDaosResponse {
	res, err := suite.k.Daos(suite.ctx, &types.QueryDaosRequest{Pagination: page})
	suite.Require().NoError(err)

	return res
}

func (suite *IntegrationTestSuite) daosByAdmin(admin sdk.AccAddress, page *query.PageRequest) *types.QueryDaosByAdminResponse {
	res, err := suite.k.DaosByAdmin(suite.ctx, &types.QueryDaosByAdminRequest{Admin: admin.String(), Pagination: page})
	suite.Require().NoError(err)

	return res
}

func (suite *IntegrationTestSuite) proposals(daoId uint64, statuses []types.ProposalStatus, page *query.PageRequest) *types.QueryProposalsResponse {
	res, err := suite.k.Proposals(suite.ctx, &types.QueryProposalsRequest{DaoId: daoId, Statuses: statuses, Pagination: page})
	suite.Require().NoError(err)

	return res
}

func (suite *IntegrationTestSuite) ballots(proposalId uint64, page *query.PageRequest) *types.QueryBallotsResponse {
	res, err := suite.k.Ballots(suite.ctx, &types.QueryBallotsRequest{ProposalId: proposalId, Pagination: page})
	suite.Require().NoError(err)

	return res
}

// DAOs are listed by id: in one page, in pages joined by next_key with the
// total counted, and in reverse.
func (suite *IntegrationTestSuite) TestQueryDaos() {
	res := suite.daos(nil)
	suite.Require().Empty(res.Daos)
	suite.Require().Nil(res.Pagination.NextKey)

	for id := uint64(1); id <= 3; id++ {
		suite.createDao(id, func(*types.MsgCreateDao) {})
	}

	res = suite.daos(nil)
	suite.Require().Equal([]uint64{1, 2, 3}, daoIds(res.Daos))
	stored, _ := suite.k.GetDao(suite.ctx, 2)
	suite.Require().Equal(stored, res.Daos[1])

	res = suite.daos(&query.PageRequest{Limit: 2, CountTotal: true})
	suite.Require().Equal([]uint64{1, 2}, daoIds(res.Daos))
	suite.Require().Equal(uint64(3), res.Pagination.Total)
	suite.Require().NotNil(res.Pagination.NextKey)
	res = suite.daos(&query.PageRequest{Key: res.Pagination.NextKey, Limit: 2})
	suite.Require().Equal([]uint64{3}, daoIds(res.Daos))
	suite.Require().Nil(res.Pagination.NextKey)

	res = suite.daos(&query.PageRequest{Reverse: true})
	suite.Require().Equal([]uint64{3, 2, 1}, daoIds(res.Daos))

	_, err := suite.k.Daos(suite.ctx, nil)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}

// An admin lists only the DAOs it administers, by id; after a handover the
// old admin no longer lists the DAO and the new one does.
func (suite *IntegrationTestSuite) TestQueryDaosByAdmin() {
	suite.Require().Empty(suite.daosByAdmin(adminAddr, nil).Daos)

	withAdmin := func(msg *types.MsgCreateDao) { msg.Admin = adminAddr.String() }
	suite.createDao(1, withAdmin)
	suite.createDao(2, func(*types.MsgCreateDao) {})
	suite.createDao(3, withAdmin)

	res := suite.daosByAdmin(adminAddr, nil)
	suite.Require().Equal([]uint64{1, 3}, daoIds(res.Daos))
	stored, _ := suite.k.GetDao(suite.ctx, 3)
	suite.Require().Equal(stored, res.Daos[1])
	// a self-owned DAO is its own admin
	suite.Require().Equal([]uint64{2}, daoIds(suite.daosByAdmin(daoAddress(2), nil).Daos))

	res = suite.daosByAdmin(adminAddr, &query.PageRequest{Limit: 1, CountTotal: true})
	suite.Require().Equal([]uint64{1}, daoIds(res.Daos))
	suite.Require().Equal(uint64(2), res.Pagination.Total)
	res = suite.daosByAdmin(adminAddr, &query.PageRequest{Key: res.Pagination.NextKey, Limit: 1})
	suite.Require().Equal([]uint64{3}, daoIds(res.Daos))
	suite.Require().Nil(res.Pagination.NextKey)
	suite.Require().Equal([]uint64{3, 1}, daoIds(suite.daosByAdmin(adminAddr, &query.PageRequest{Reverse: true}).Daos))

	// a pending nomination changes nothing; the acceptance moves the DAO
	suite.Require().NoError(suite.nominate(adminAddr, 1, successorAddr.String()))
	suite.Require().Empty(suite.daosByAdmin(successorAddr, nil).Daos)
	suite.Require().NoError(suite.accept(successorAddr, 1))
	suite.Require().Equal([]uint64{3}, daoIds(suite.daosByAdmin(adminAddr, nil).Daos))
	suite.Require().Equal([]uint64{1}, daoIds(suite.daosByAdmin(successorAddr, nil).Daos))

	for _, req := range []*types.QueryDaosByAdminRequest{{Admin: "bad"}, nil} {
		_, err := suite.k.DaosByAdmin(suite.ctx, req)
		suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	}
}

// A DAO's proposals are listed by id with their stored status, filtered by
// one or several statuses, paginated over the matches; a cleaned-up proposal
// is no longer listed and other DAOs' proposals never are.
func (suite *IntegrationTestSuite) TestQueryProposals() {
	suite.Require().Empty(suite.proposals(1, nil, nil).Proposals)

	suite.createStaticDao(noConfigChange)
	suite.createDao(2, func(*types.MsgCreateDao) {})
	suite.Require().Empty(suite.proposals(1, nil, nil).Proposals)

	cancelled, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	rejected, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	ended, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.cancel(memberA, cancelled))
	suite.endVoting(rejected, 0)
	settled, err := suite.settle(rejected)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_REJECTED, settled)
	voting, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	otherDao, err := suite.submit(memberA, 2, false)
	suite.Require().NoError(err)

	// every status, the stored one: ended is past its voting end, not settled
	res := suite.proposals(1, nil, nil)
	suite.Require().Equal([]uint64{cancelled, rejected, ended, voting}, proposalIds(res.Proposals))
	stored, _ := suite.k.GetProposal(suite.ctx, ended)
	suite.Require().Equal(stored, res.Proposals[2])
	suite.Require().Equal(types.ProposalStatus_VOTING, res.Proposals[2].Status)
	suite.Require().Equal([]uint64{otherDao}, proposalIds(suite.proposals(2, nil, nil).Proposals))

	votingOnly := []types.ProposalStatus{types.ProposalStatus_VOTING}
	suite.Require().Equal([]uint64{ended, voting}, proposalIds(suite.proposals(1, votingOnly, nil).Proposals))
	finished := []types.ProposalStatus{types.ProposalStatus_REJECTED, types.ProposalStatus_CANCELLED}
	suite.Require().Equal([]uint64{cancelled, rejected}, proposalIds(suite.proposals(1, finished, nil).Proposals))
	suite.Require().Empty(suite.proposals(1, []types.ProposalStatus{types.ProposalStatus_EXECUTED}, nil).Proposals)

	// pages and the total count only the matches
	res = suite.proposals(1, votingOnly, &query.PageRequest{Limit: 1, CountTotal: true})
	suite.Require().Equal([]uint64{ended}, proposalIds(res.Proposals))
	suite.Require().Equal(uint64(2), res.Pagination.Total)
	suite.Require().NotNil(res.Pagination.NextKey)
	res = suite.proposals(1, votingOnly, &query.PageRequest{Key: res.Pagination.NextKey, Limit: 1})
	suite.Require().Equal([]uint64{voting}, proposalIds(res.Proposals))
	suite.Require().Nil(res.Pagination.NextKey)

	res = suite.proposals(1, nil, &query.PageRequest{Limit: 3, CountTotal: true})
	suite.Require().Equal([]uint64{cancelled, rejected, ended}, proposalIds(res.Proposals))
	suite.Require().Equal(uint64(4), res.Pagination.Total)
	suite.Require().Equal([]uint64{voting, ended, rejected, cancelled}, proposalIds(suite.proposals(1, nil, &query.PageRequest{Reverse: true}).Proposals))

	cleaned, err := suite.cleanup(cancelled, 0)
	suite.Require().NoError(err)
	suite.Require().True(cleaned.Completed)
	suite.Require().Equal([]uint64{rejected, ended, voting}, proposalIds(suite.proposals(1, nil, nil).Proposals))
	suite.Require().Equal([]uint64{rejected}, proposalIds(suite.proposals(1, finished, nil).Proposals))

	_, err = suite.k.Proposals(suite.ctx, nil)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}

// A proposal's ballots are listed by voter key, without the ballots of other
// proposals; a cleanup empties the list.
func (suite *IntegrationTestSuite) TestQueryBallots() {
	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	other, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().Empty(suite.ballots(id, nil).Ballots)

	suite.Require().NoError(suite.vote(memberB, id, types.VoteOption_NO))
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberA, other, types.VoteOption_ABSTAIN))

	res := suite.ballots(id, nil)
	suite.Require().Equal([]string{memberA.String(), memberB.String()}, ballotVoters(res.Ballots))
	stored, _ := suite.k.GetBallot(suite.ctx, id, memberB)
	suite.Require().Equal(stored, res.Ballots[1])
	suite.Require().Equal([]string{memberA.String()}, ballotVoters(suite.ballots(other, nil).Ballots))

	res = suite.ballots(id, &query.PageRequest{Limit: 1, CountTotal: true})
	suite.Require().Equal([]string{memberA.String()}, ballotVoters(res.Ballots))
	suite.Require().Equal(uint64(2), res.Pagination.Total)
	res = suite.ballots(id, &query.PageRequest{Key: res.Pagination.NextKey, Limit: 1})
	suite.Require().Equal([]string{memberB.String()}, ballotVoters(res.Ballots))
	suite.Require().Nil(res.Pagination.NextKey)
	suite.Require().Equal([]string{memberB.String(), memberA.String()}, ballotVoters(suite.ballots(id, &query.PageRequest{Reverse: true}).Ballots))

	suite.Require().NoError(suite.cancel(memberA, id))
	cleaned, err := suite.cleanup(id, 0)
	suite.Require().NoError(err)
	suite.Require().True(cleaned.Completed)
	suite.Require().Empty(suite.ballots(id, nil).Ballots)
	suite.Require().Len(suite.ballots(other, nil).Ballots, 1)

	_, err = suite.k.Ballots(suite.ctx, nil)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}
