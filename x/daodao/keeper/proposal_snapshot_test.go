package keeper_test

import (
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestSnapshot_StableUnderMembershipEdits: votes use the proposal's
// snapshot, not the current member set. Editing the DAO's members after
// proposal creation must NOT change the outcome.
func (suite *IntegrationTestSuite) TestSnapshot_StableUnderMembershipEdits() {
	a := freshAddr()
	b := freshAddr()
	c := freshAddr()
	gov := validGovernance()
	gov.AllowRevote = true
	daoID, _ := suite.createDaoWithMembers("snapshot",
		[]types.StaticMember{
			{Address: a, Weight: 5},
			{Address: b, Weight: 3},
		}, gov)

	pid := suite.createTestProposal(daoID, a)

	// Now mutate the member set: remove b, add c with weight 100. None of
	// this should affect votes on `pid` because the snapshot is frozen.
	_, err := suite.msgServer.UpdateMembers(suite.ctx, &types.MsgUpdateMembers{
		Authority: a,
		DaoId:     daoID,
		Add:       []types.StaticMember{{Address: c, Weight: 100}},
		Remove:    []string{b},
	})
	suite.Require().NoError(err)

	// b can STILL vote on pid (snapshot has them at weight 3) even though
	// they're no longer a current member.
	_, err = suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter: b, DaoId: daoID, ProposalId: pid, Option: types.VoteOption_VOTE_OPTION_YES,
	})
	suite.Require().NoError(err)

	// c CANNOT vote on pid (snapshot has them at weight 0) even though
	// they're now a current member.
	_, err = suite.msgServer.Vote(suite.ctx, &types.MsgVote{
		Voter: c, DaoId: daoID, ProposalId: pid, Option: types.VoteOption_VOTE_OPTION_YES,
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "no snapshot power")

	// Tally reflects snapshot weights, not current weights.
	p, _ := suite.k.GetProposal(suite.ctx, daoID, pid)
	suite.Require().Equal(uint64(3), p.Tally.YesPower)
	suite.Require().Equal(uint64(8), p.Tally.TotalPower) // a:5 + b:3 at snapshot time
}

// TestSnapshot_TotalFrozen: a NEW proposal created after the membership
// edit reflects the new totals; the OLD proposal stays frozen.
func (suite *IntegrationTestSuite) TestSnapshot_TotalFrozen() {
	a := freshAddr()
	b := freshAddr()
	gov := validGovernance()
	gov.AllowRevote = true
	daoID, _ := suite.createDaoWithMembers("snapshot-frozen",
		[]types.StaticMember{
			{Address: a, Weight: 1},
			{Address: b, Weight: 1},
		}, gov)

	firstPid := suite.createTestProposal(daoID, a)

	// Bump b's weight.
	_, err := suite.msgServer.UpdateMembers(suite.ctx, &types.MsgUpdateMembers{
		Authority: a,
		DaoId:     daoID,
		Add:       []types.StaticMember{{Address: b, Weight: 99}},
	})
	suite.Require().NoError(err)

	secondPid := suite.createTestProposal(daoID, a)

	first, _ := suite.k.GetProposal(suite.ctx, daoID, firstPid)
	second, _ := suite.k.GetProposal(suite.ctx, daoID, secondPid)

	suite.Require().Equal(uint64(2), first.Tally.TotalPower, "first proposal: snapshot at create-time")
	suite.Require().Equal(uint64(100), second.Tally.TotalPower, "second proposal: snapshot post-update")
}
