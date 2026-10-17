package keeper_test

import (
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// A staking reward used by a DAO electorate cannot be removed; one nobody
// uses can. A Denom Reward pin never blocks a staking reward.
func (suite *IntegrationTestSuite) TestStakingRewardHooks_RemovalVeto() {
	suite.stubRewards(defaultRewardsState())
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, stakeDenom))
	suite.Require().NoError(err)

	hooks := suite.k.StakingRewardHooks()

	err = hooks.BeforeStakingRewardRemoval(suite.ctx, "1")
	suite.Require().ErrorIs(err, types.ErrSourcePinned)
	suite.Require().ErrorContains(err, "staking reward 1")

	suite.Require().NoError(hooks.BeforeStakingRewardRemoval(suite.ctx, "2"))
	suite.Require().NoError(hooks.BeforeStakingRewardRemoval(suite.ctx, stakeDenom))
}

// Join records for a staking reward: a join while a DAO uses the source is
// recorded at the block height, a top-up keeps it, an exit deletes it. A join
// to a source nobody uses is not recorded.
func (suite *IntegrationTestSuite) TestStakingRewardHooks_JoinRecordLifecycle() {
	suite.stubRewards(defaultRewardsState())
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, stakeDenom))
	suite.Require().NoError(err)

	hooks := suite.k.StakingRewardHooks()
	source := types.StakingRewardSource("1")
	suite.ctx = suite.ctx.WithBlockHeight(7)

	suite.Require().NoError(hooks.AfterStakingRewardJoin(suite.ctx, "1", memberA.String(), math.NewInt(10), stakeDenom))
	record, found := suite.k.GetJoinRecord(suite.ctx, source, memberA)
	suite.Require().True(found)
	suite.Require().Equal(types.JoinRecord{Source: "sr/1", Address: memberA.String(), Height: 7}, record)

	suite.ctx = suite.ctx.WithBlockHeight(9)
	suite.Require().NoError(hooks.AfterStakingRewardIncrease(suite.ctx, "1", memberA.String(), math.NewInt(5), math.NewInt(15), stakeDenom))
	record, _ = suite.k.GetJoinRecord(suite.ctx, source, memberA)
	suite.Require().Equal(int64(7), record.Height)

	suite.Require().NoError(hooks.AfterStakingRewardExit(suite.ctx, "1", memberA.String(), math.NewInt(15), stakeDenom))
	_, found = suite.k.GetJoinRecord(suite.ctx, source, memberA)
	suite.Require().False(found)

	// staking reward 5 is in no electorate
	suite.Require().NoError(hooks.AfterStakingRewardJoin(suite.ctx, "5", memberA.String(), math.NewInt(10), stakeDenom))
	_, found = suite.k.GetJoinRecord(suite.ctx, types.StakingRewardSource("5"), memberA)
	suite.Require().False(found)

	// an exit without a record is fine
	suite.Require().NoError(hooks.AfterStakingRewardExit(suite.ctx, "5", memberA.String(), math.NewInt(10), stakeDenom))
}

// The same lifecycle for a Denom Reward.
func (suite *IntegrationTestSuite) TestDenomRewardHooks_JoinRecordLifecycle() {
	suite.stubRewards(defaultRewardsState())
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, stakeDenom))
	suite.Require().NoError(err)

	hooks := suite.k.DenomRewardHooks()
	source := types.DenomRewardSource(stakeDenom)
	suite.ctx = suite.ctx.WithBlockHeight(7)

	suite.Require().NoError(hooks.AfterDenomRewardJoin(suite.ctx, stakeDenom, memberB.String(), math.NewInt(10)))
	record, found := suite.k.GetJoinRecord(suite.ctx, source, memberB)
	suite.Require().True(found)
	suite.Require().Equal(types.JoinRecord{Source: "dr/ubze", Address: memberB.String(), Height: 7}, record)

	suite.ctx = suite.ctx.WithBlockHeight(9)
	suite.Require().NoError(hooks.AfterDenomRewardIncrease(suite.ctx, stakeDenom, memberB.String(), math.NewInt(5), math.NewInt(15)))
	record, _ = suite.k.GetJoinRecord(suite.ctx, source, memberB)
	suite.Require().Equal(int64(7), record.Height)

	suite.Require().NoError(hooks.AfterDenomRewardExit(suite.ctx, stakeDenom, memberB.String(), math.NewInt(15)))
	_, found = suite.k.GetJoinRecord(suite.ctx, source, memberB)
	suite.Require().False(found)

	// the uother Denom Reward is in no electorate
	suite.Require().NoError(hooks.AfterDenomRewardJoin(suite.ctx, otherDenom, memberB.String(), math.NewInt(10)))
	_, found = suite.k.GetJoinRecord(suite.ctx, types.DenomRewardSource(otherDenom), memberB)
	suite.Require().False(found)
}

// The After* hooks would abort the x/rewards transaction on error, so an
// address this module can't parse is skipped instead.
func (suite *IntegrationTestSuite) TestRewardHooks_InvalidAddressNeverFails() {
	suite.stubRewards(defaultRewardsState())
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, stakeDenom))
	suite.Require().NoError(err)

	sr := suite.k.StakingRewardHooks()
	dr := suite.k.DenomRewardHooks()
	suite.Require().NoError(sr.AfterStakingRewardJoin(suite.ctx, "1", "bze1nope", math.NewInt(1), stakeDenom))
	suite.Require().NoError(sr.AfterStakingRewardExit(suite.ctx, "1", "bze1nope", math.NewInt(1), stakeDenom))
	suite.Require().NoError(dr.AfterDenomRewardJoin(suite.ctx, stakeDenom, "bze1nope", math.NewInt(1)))
	suite.Require().NoError(dr.AfterDenomRewardExit(suite.ctx, stakeDenom, "bze1nope", math.NewInt(1)))
}

// exitGas returns the gas the staking reward Exit hook of addr consumes.
func (suite *IntegrationTestSuite) exitGas(addr sdk.AccAddress) uint64 {
	ctx := suite.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardExit(ctx, "1", addr.String(), math.NewInt(30), stakeDenom))

	return ctx.GasMeter().GasConsumed()
}

// ballotReducedEvents counts the EventBallotReduced events emitted so far.
func (suite *IntegrationTestSuite) ballotReducedEvents() int {
	count := 0
	for _, event := range suite.ctx.EventManager().Events() {
		if event.Type == proto.MessageName(&types.EventBallotReduced{}) {
			count++
		}
	}

	return count
}

// D19: the Exit hook's cost depends on the leaving voter's own open ballots
// only. Other voters' ballots on other proposals leave it unchanged; more
// ballots of the voter make it grow, one reduction each.
func (suite *IntegrationTestSuite) TestExitHook_D19_CostBoundedByOwnBallots() {
	// setup creates three proposals; memberA votes on the first ownVotes of
	// them, and crowd other stakers vote on the two others
	setup := func(ownVotes, crowd int) {
		suite.SetupTest()
		s := suite.createStakeDao()
		s.joinSr("1", memberA, 30)
		for i := 0; i < crowd; i++ {
			s.joinSr("1", memberAddress(i), 5)
		}
		suite.atBlock(10)
		for p := 0; p < 3; p++ {
			_, err := suite.submit(memberA, 1, false)
			suite.Require().NoError(err)
		}
		for p := uint64(1); p <= uint64(ownVotes); p++ {
			suite.Require().NoError(suite.vote(memberA, p, types.VoteOption_YES))
		}
		for i := 0; i < crowd; i++ {
			suite.Require().NoError(suite.vote(memberAddress(i), 2, types.VoteOption_NO))
			suite.Require().NoError(suite.vote(memberAddress(i), 3, types.VoteOption_YES))
		}
	}

	setup(1, 0)
	quiet := suite.exitGas(memberA)
	suite.Require().Equal(1, suite.ballotReducedEvents())

	setup(1, 40)
	crowded := suite.exitGas(memberA)
	suite.Require().Equal(1, suite.ballotReducedEvents())
	suite.Require().Equal(quiet, crowded)
	suite.requireTally(2, 0, 200, 0)
	suite.requireTally(3, 200, 0, 0)

	setup(3, 0)
	busy := suite.exitGas(memberA)
	suite.Require().Equal(3, suite.ballotReducedEvents())
	suite.Require().Greater(busy, quiet)
	for p := uint64(1); p <= 3; p++ {
		suite.requireTally(p, 0, 0, 0)
		suite.requireInvariants(p, memberA)
	}
}

// An Exit of a source the voter's ballot does not count, or of a voter with
// no ballot, leaves every ballot and tally as it is.
func (suite *IntegrationTestSuite) TestExitHook_OtherSourceLeavesBallot() {
	s := suite.createStakeDao()
	s.joinSr("1", memberA, 30)
	suite.atBlock(10)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))

	suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardExit(suite.ctx, stakeDenom, memberA.String(), math.NewInt(5)))
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardExit(suite.ctx, "1", memberB.String(), math.NewInt(5), stakeDenom))

	suite.requireTally(id, 30, 0, 0)
	suite.Require().True(suite.k.HasVoteIndex(suite.ctx, srSource, memberA, id))
	suite.Require().Equal(0, suite.ballotReducedEvents())
}

// I6: an Exit after the proposal left voting only prunes the voter's vote
// index entry; the cancelled proposal's ballot and tally stay as they were.
func (suite *IntegrationTestSuite) TestExitHook_I6_PrunesIndexOfFinishedProposal() {
	s := suite.createStakeDao()
	s.joinSr("1", memberA, 30)
	s.joinDr(stakeDenom, memberA, 20)
	suite.atBlock(10)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
	suite.Require().True(suite.k.HasVoteIndex(suite.ctx, srSource, memberA, id))
	suite.Require().True(suite.k.HasVoteIndex(suite.ctx, drSource, memberA, id))
	ballot, _ := suite.k.GetBallot(suite.ctx, id, memberA)

	suite.Require().NoError(suite.cancel(memberA, id))
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardExit(suite.ctx, "1", memberA.String(), math.NewInt(30), stakeDenom))
	suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardExit(suite.ctx, stakeDenom, memberA.String(), math.NewInt(20)))

	suite.Require().False(suite.k.HasVoteIndex(suite.ctx, srSource, memberA, id))
	suite.Require().False(suite.k.HasVoteIndex(suite.ctx, drSource, memberA, id))
	after, found := suite.k.GetBallot(suite.ctx, id, memberA)
	suite.Require().True(found)
	suite.Require().Equal(ballot, after)
	suite.requireTally(id, 50, 0, 0)
	suite.Require().Equal(0, suite.ballotReducedEvents())
}

// I6 after cleanup: the cleanup deletes the vote-index entry of every share of
// every ballot, so no vidx/ entry points at the proposal and a later Exit
// finds nothing to prune. Entries of the voters' other proposals stay.
func (suite *IntegrationTestSuite) TestCleanupProposal_I6_LeavesNoVoteIndex() {
	id := suite.votedStakeProposal(3, true)
	other, err := suite.submit(memberAddress(1), 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberAddress(1), other, types.VoteOption_YES))
	for i := 1; i <= 3; i++ {
		suite.Require().True(suite.k.HasVoteIndex(suite.ctx, srSource, memberAddress(i), id))
		suite.Require().True(suite.k.HasVoteIndex(suite.ctx, drSource, memberAddress(i), id))
	}

	res, err := suite.cleanup(id, 2)
	suite.Require().NoError(err)
	suite.Require().False(res.Completed)
	res, err = suite.cleanup(id, 2)
	suite.Require().NoError(err)
	suite.Require().True(res.Completed)

	for i := 1; i <= 3; i++ {
		suite.Require().False(suite.k.HasVoteIndex(suite.ctx, srSource, memberAddress(i), id))
		suite.Require().False(suite.k.HasVoteIndex(suite.ctx, drSource, memberAddress(i), id))
	}
	suite.Require().True(suite.k.HasVoteIndex(suite.ctx, srSource, memberAddress(1), other))
	suite.Require().True(suite.k.HasVoteIndex(suite.ctx, drSource, memberAddress(1), other))

	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardExit(suite.ctx, "1", memberAddress(2).String(), math.NewInt(1), stakeDenom))
	suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardExit(suite.ctx, stakeDenom, memberAddress(2).String(), math.NewInt(1)))
	suite.Require().Equal(0, suite.ballotReducedEvents())
	_, found := suite.k.GetProposal(suite.ctx, id)
	suite.Require().False(found)
}

// D25 end to end: an open proposal keeps its sources pinned after the DAO
// drops them. Staking reward 5 leaves DAO 1 while P is open; its pin stays at
// 1 (P's), so a whale joining it gets a join record and no weight on P, and
// its removal is still vetoed. Once P is cancelled the pin is 0, the veto lifts
// and joins are no longer recorded.
func (suite *IntegrationTestSuite) TestRewardHooks_D25_RemovedSourceStaysPinnedForOpenProposals() {
	suite.stubRewards(defaultRewardsState())
	s := suite.stubStakes()
	suite.createAdminStakeDao([]string{"1", "5"}, stakeDenom)
	s.joinSr("1", memberA, 100)
	sr5 := types.StakingRewardSource("5")
	hooks := suite.k.StakingRewardHooks()

	suite.atBlock(10)
	p, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(2), suite.k.GetSourcePins(suite.ctx, sr5))

	suite.Require().NoError(suite.adminExecute(adminAddr, 1, types.NewMsgUpdateElectorate(daoAddress(1).String(), 1, &types.StakeElectorate{SrIds: []string{"1"}, DrDenom: stakeDenom})))
	suite.requireElectorate([]string{"1"}, stakeDenom)
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, sr5))
	suite.requirePinInvariant([]uint64{1}, []uint64{p})

	suite.atBlock(11)
	s.joinSr("5", memberC, 1_000_000)
	suite.Require().NoError(hooks.AfterStakingRewardJoin(suite.ctx, "5", memberC.String(), math.NewInt(1_000_000), stakeDenom))
	record, found := suite.k.GetJoinRecord(suite.ctx, sr5, memberC)
	suite.Require().True(found)
	suite.Require().Equal(int64(11), record.Height)

	suite.Require().ErrorIs(suite.vote(memberC, p, types.VoteOption_YES), types.ErrNoVotingPower)
	_, found = suite.k.GetBallot(suite.ctx, p, memberC)
	suite.Require().False(found)
	suite.requireTally(p, 0, 0, 0)
	suite.Require().ErrorIs(hooks.BeforeStakingRewardRemoval(suite.ctx, "5"), types.ErrSourcePinned)

	suite.Require().NoError(suite.cancel(memberA, p))
	suite.Require().Zero(suite.k.GetSourcePins(suite.ctx, sr5))
	suite.Require().NoError(hooks.BeforeStakingRewardRemoval(suite.ctx, "5"))
	suite.requirePinInvariant([]uint64{1}, []uint64{p})

	suite.Require().NoError(hooks.AfterStakingRewardJoin(suite.ctx, "5", memberB.String(), math.NewInt(10), stakeDenom))
	_, found = suite.k.GetJoinRecord(suite.ctx, sr5, memberB)
	suite.Require().False(found)
}
