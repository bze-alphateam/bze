package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (suite *IntegrationTestSuite) vote(voter sdk.AccAddress, proposalId uint64, option types.VoteOption) error {
	_, err := suite.msgServer.Vote(suite.ctx, types.NewMsgVote(voter.String(), proposalId, option))
	return err
}

// requireTally checks the proposal's running yes/no/abstain sums.
func (suite *IntegrationTestSuite) requireTally(proposalId uint64, yes, no, abstain int64) {
	proposal, found := suite.k.GetProposal(suite.ctx, proposalId)
	suite.Require().True(found)
	suite.Require().Equal(math.NewInt(yes), proposal.Yes, "yes")
	suite.Require().Equal(math.NewInt(no), proposal.No, "no")
	suite.Require().Equal(math.NewInt(abstain), proposal.Abstain, "abstain")
}

// requireInvariants checks I2 (each running tally is the sum of the weights of
// the ballots cast for it) over the given voters, who must be every voter of
// the proposal, and I3 (a stake ballot's weight is the sum of its shares,
// each indexed under its source).
func (suite *IntegrationTestSuite) requireInvariants(proposalId uint64, voters ...sdk.AccAddress) {
	sums := map[types.VoteOption]math.Int{
		types.VoteOption_YES:     math.ZeroInt(),
		types.VoteOption_NO:      math.ZeroInt(),
		types.VoteOption_ABSTAIN: math.ZeroInt(),
	}

	for _, voter := range voters {
		ballot, found := suite.k.GetBallot(suite.ctx, proposalId, voter)
		if !found {
			continue
		}
		sums[ballot.Option] = sums[ballot.Option].Add(ballot.Weight)

		if len(ballot.Shares) == 0 {
			continue
		}
		shares := math.ZeroInt()
		for _, share := range ballot.Shares {
			suite.Require().True(share.Amount.IsPositive(), "I3: share %s", share.Source)
			suite.Require().True(suite.k.HasVoteIndex(suite.ctx, []byte(share.Source), voter, proposalId), "vote index of %s", share.Source)
			shares = shares.Add(share.Amount)
		}
		suite.Require().Equal(ballot.Weight, shares, "I3")
	}

	proposal, _ := suite.k.GetProposal(suite.ctx, proposalId)
	suite.Require().Equal(proposal.Yes, sums[types.VoteOption_YES], "I2 yes")
	suite.Require().Equal(proposal.No, sums[types.VoteOption_NO], "I2 no")
	suite.Require().Equal(proposal.Abstain, sums[types.VoteOption_ABSTAIN], "I2 abstain")
}

// D23: a revote moves the voter's weight to the new option; the same option
// twice is refused and changes nothing.
func (suite *IntegrationTestSuite) TestVote_RevoteFlipsTally() {
	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberB, id, types.VoteOption_NO))
	suite.requireTally(id, 1, 1, 0)
	suite.requireInvariants(id, memberA, memberB)

	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_NO))
	suite.requireTally(id, 0, 2, 0)
	suite.requireInvariants(id, memberA, memberB)

	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_ABSTAIN))
	suite.requireTally(id, 0, 1, 1)
	suite.requireInvariants(id, memberA, memberB)

	err = suite.vote(memberA, id, types.VoteOption_ABSTAIN)
	suite.Require().ErrorIs(err, types.ErrSameVote)
	suite.requireTally(id, 0, 1, 1)

	ballot, _ := suite.k.GetBallot(suite.ctx, id, memberA)
	suite.Require().Equal(types.VoteOption_ABSTAIN, ballot.Option)
}

// D23 in a stake DAO: the revote drops the old ballot's shares and index
// entries and stores the voter's current ones.
func (suite *IntegrationTestSuite) TestVote_StakeRevoteUsesCurrentShares() {
	s := suite.createStakeDao()
	s.joinSr("1", memberA, 30)
	s.joinDr(stakeDenom, memberA, 20)
	suite.atBlock(10)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
	suite.requireTally(id, 50, 0, 0)
	suite.requireInvariants(id, memberA)

	delete(s.dr[stakeDenom], memberA.String())
	suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardExit(suite.ctx, stakeDenom, memberA.String(), math.NewInt(20)))
	suite.requireTally(id, 30, 0, 0)

	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_NO))
	suite.requireTally(id, 0, 30, 0)
	suite.requireInvariants(id, memberA)

	ballot, _ := suite.k.GetBallot(suite.ctx, id, memberA)
	suite.Require().Equal([]types.BallotShare{{Source: "sr/1", Amount: math.NewInt(30)}}, ballot.Shares)
	suite.Require().True(suite.k.HasVoteIndex(suite.ctx, srSource, memberA, id))
	suite.Require().False(suite.k.HasVoteIndex(suite.ctx, drSource, memberA, id))
}

// Ballots on two proposals of one DAO are independent.
func (suite *IntegrationTestSuite) TestVote_TalliesPerProposal() {
	suite.createStaticDao(noConfigChange)
	first, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	second, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	suite.Require().NoError(suite.vote(memberA, first, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberA, second, types.VoteOption_NO))
	suite.Require().NoError(suite.vote(memberB, second, types.VoteOption_NO))

	suite.requireTally(first, 1, 0, 0)
	suite.requireTally(second, 0, 2, 0)
	suite.requireInvariants(first, memberA, memberB)
	suite.requireInvariants(second, memberA, memberB)
}
