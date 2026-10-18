package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/dao/types"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

// stakes is the x/rewards participation seen by the eligibility tests:
// staking reward id or Denom Reward staking denom -> address -> amount.
type stakes struct {
	sr map[string]map[string]int64
	dr map[string]map[string]int64
}

func (s stakes) joinSr(rewardId string, addr sdk.AccAddress, amount int64) {
	if s.sr[rewardId] == nil {
		s.sr[rewardId] = map[string]int64{}
	}
	s.sr[rewardId][addr.String()] = amount
}

func (s stakes) joinDr(denom string, addr sdk.AccAddress, amount int64) {
	if s.dr[denom] == nil {
		s.dr[denom] = map[string]int64{}
	}
	s.dr[denom][addr.String()] = amount
}

// stubStakes serves the participants of s (read at call time) to the dao keeper.
func (suite *IntegrationTestSuite) stubStakes() stakes {
	s := stakes{sr: map[string]map[string]int64{}, dr: map[string]map[string]int64{}}
	suite.rewards.EXPECT().GetStakingRewardParticipant(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ sdk.Context, address, rewardId string) (rewardstypes.StakingRewardParticipant, bool) {
			amount, ok := s.sr[rewardId][address]
			return rewardstypes.StakingRewardParticipant{Address: address, RewardId: rewardId, Amount: math.NewInt(amount).String()}, ok
		}).AnyTimes()
	suite.rewards.EXPECT().GetDenomRewardParticipant(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ sdk.Context, denom, address string) (rewardstypes.DenomRewardParticipant, bool) {
			amount, ok := s.dr[denom][address]
			return rewardstypes.DenomRewardParticipant{Address: address, StakingDenom: denom, Amount: math.NewInt(amount)}, ok
		}).AnyTimes()

	return s
}

// createStakeDao creates the self-owned stake DAO 1 voted by staking reward 1
// (100 staked) and the ubze Denom Reward (50 staked).
func (suite *IntegrationTestSuite) createStakeDao() stakes {
	suite.stubRewards(defaultRewardsState())
	s := suite.stubStakes()
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, stakeDenom))
	suite.Require().NoError(err)

	return s
}

var (
	srSource = types.StakingRewardSource("1")
	drSource = types.DenomRewardSource(stakeDenom)
)

// A staker proposes; the proposal pins the electorate and its total stake as
// the denominator. I4: every open proposal adds a pin to each of its sources,
// on top of the DAO's own.
func (suite *IntegrationTestSuite) TestSubmitProposal_StakerPinsElectorate() {
	s := suite.createStakeDao()
	s.joinSr("1", memberA, 30)
	s.joinDr(stakeDenom, memberB, 20)
	suite.atBlock(10)
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, srSource))
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, drSource))

	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.Require().Equal(&types.StakeElectorate{SrIds: []string{"1"}, DrDenom: stakeDenom}, proposal.PinnedElectorate)
	suite.Require().Equal(math.NewInt(150), proposal.PinnedDenominator)
	suite.Require().Equal(uint64(2), suite.k.GetSourcePins(suite.ctx, srSource))
	suite.Require().Equal(uint64(2), suite.k.GetSourcePins(suite.ctx, drSource))

	// a Denom Reward staker proposes too
	_, err = suite.submit(memberB, 1, false)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(3), suite.k.GetSourcePins(suite.ctx, srSource))
	suite.Require().Equal(uint64(3), suite.k.GetSourcePins(suite.ctx, drSource))
	// sources of no proposal stay unpinned
	suite.Require().Equal(uint64(0), suite.k.GetSourcePins(suite.ctx, types.StakingRewardSource("5")))
}

// An address with no stake in the electorate cannot propose; stake in a
// source outside the electorate does not count.
func (suite *IntegrationTestSuite) TestSubmitProposal_NonStakerRejected() {
	s := suite.createStakeDao()
	s.joinSr("5", attackerAddr, 1_000)

	_, err := suite.submit(attackerAddr, 1, false)

	suite.Require().ErrorIs(err, types.ErrNotProposer)
	suite.requireNothingStored()
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, srSource))
}

// D3: stake that joined a pinned source in the submit block has no weight in
// it; one block later it has.
func (suite *IntegrationTestSuite) TestSubmitProposal_SameBlockJoinHasNoWeight() {
	s := suite.createStakeDao()
	suite.atBlock(10)
	s.joinSr("1", memberC, 10)
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardJoin(suite.ctx, "1", memberC.String(), math.NewInt(10), stakeDenom))

	_, err := suite.submit(memberC, 1, false)
	suite.Require().ErrorIs(err, types.ErrNotProposer)

	suite.atBlock(11)
	_, err = suite.submit(memberC, 1, false)
	suite.Require().NoError(err)
}

// D4: a join recorded after the reference height has no weight either. The
// chain only moves forward, so the context is rewound to put the join after
// the submit block.
func (suite *IntegrationTestSuite) TestSubmitProposal_LaterJoinHasNoWeight() {
	s := suite.createStakeDao()
	suite.atBlock(12)
	s.joinDr(stakeDenom, memberC, 10)
	suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardJoin(suite.ctx, stakeDenom, memberC.String(), math.NewInt(10)))

	suite.atBlock(11)
	_, err := suite.submit(memberC, 1, false)
	suite.Require().ErrorIs(err, types.ErrNotProposer)
}

// The join rule is per source: older stake in another source still lets the
// address propose.
func (suite *IntegrationTestSuite) TestSubmitProposal_OlderStakeInAnotherSourceCounts() {
	s := suite.createStakeDao()
	s.joinSr("1", memberC, 10) // joined before the DAO pinned the source: no record
	suite.atBlock(10)
	s.joinDr(stakeDenom, memberC, 10)
	suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardJoin(suite.ctx, stakeDenom, memberC.String(), math.NewInt(10)))

	_, err := suite.submit(memberC, 1, false)
	suite.Require().NoError(err)
}

// Stake with no join record counts: it joined while no DAO or proposal used
// the source, so before any proposal of it.
func (suite *IntegrationTestSuite) TestSubmitProposal_StakeWithoutJoinRecordCounts() {
	s := suite.createStakeDao()
	suite.atBlock(10)
	s.joinSr("1", memberC, 10)

	_, found := suite.k.GetJoinRecord(suite.ctx, srSource, memberC)
	suite.Require().False(found)
	_, err := suite.submit(memberC, 1, false)
	suite.Require().NoError(err)
}

// An open stake DAO takes proposals from anyone; the denominator is the
// electorate's stake whoever proposes.
func (suite *IntegrationTestSuite) TestSubmitProposal_OpenStakeDao() {
	suite.stubRewards(defaultRewardsState())
	suite.stubStakes()
	suite.expectCreation(1)
	msg := validStakeDaoMsg([]string{"1", "5"}, "")
	msg.Config.OpenSubmission = true
	_, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)

	id, err := suite.submit(attackerAddr, 1, false)
	suite.Require().NoError(err)

	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.Require().Equal(math.NewInt(110), proposal.PinnedDenominator)
	suite.Require().Equal(&types.StakeElectorate{SrIds: []string{"1", "5"}}, proposal.PinnedElectorate)
}

// stakeSource joins (or leaves, at amount 0) one source of the stake DAO
// through x/rewards: the participant changes and the matching hook fires.
type stakeSource struct {
	name   string
	source []byte
	join   func(suite *IntegrationTestSuite, s stakes, addr sdk.AccAddress, amount int64)
	exit   func(suite *IntegrationTestSuite, s stakes, addr sdk.AccAddress)
}

var stakeSources = []stakeSource{
	{
		name:   "staking reward",
		source: srSource,
		join: func(suite *IntegrationTestSuite, s stakes, addr sdk.AccAddress, amount int64) {
			s.joinSr("1", addr, amount)
			suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardJoin(suite.ctx, "1", addr.String(), math.NewInt(amount), stakeDenom))
		},
		exit: func(suite *IntegrationTestSuite, s stakes, addr sdk.AccAddress) {
			amount := s.sr["1"][addr.String()]
			delete(s.sr["1"], addr.String())
			suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardExit(suite.ctx, "1", addr.String(), math.NewInt(amount), stakeDenom))
		},
	},
	{
		name:   "denom reward",
		source: drSource,
		join: func(suite *IntegrationTestSuite, s stakes, addr sdk.AccAddress, amount int64) {
			s.joinDr(stakeDenom, addr, amount)
			suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardJoin(suite.ctx, stakeDenom, addr.String(), math.NewInt(amount)))
		},
		exit: func(suite *IntegrationTestSuite, s stakes, addr sdk.AccAddress) {
			amount := s.dr[stakeDenom][addr.String()]
			delete(s.dr[stakeDenom], addr.String())
			suite.Require().NoError(suite.k.DenomRewardHooks().AfterDenomRewardExit(suite.ctx, stakeDenom, addr.String(), math.NewInt(amount)))
		},
	},
}

// counted is everything the proposal's tally holds.
func (suite *IntegrationTestSuite) counted(proposalId uint64) math.Int {
	proposal, _ := suite.k.GetProposal(suite.ctx, proposalId)
	return proposal.Yes.Add(proposal.No).Add(proposal.Abstain)
}

// D2: coins that voted, left and moved to another address cannot vote again
// on the same proposal, from either address: the counted weight never
// exceeds the coins.
func (suite *IntegrationTestSuite) TestVote_D2_ExitMoveRejoinCannotDoubleVote() {
	for _, src := range stakeSources {
		suite.Run(src.name, func() {
			suite.SetupTest()
			s := suite.createStakeDao()
			suite.atBlock(5)
			src.join(suite, s, memberA, 30)
			suite.atBlock(10)
			// memberB proposes with stake that has no join record
			s.joinDr(stakeDenom, memberB, 1)
			id, err := suite.submit(memberB, 1, false)
			suite.Require().NoError(err)

			suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
			suite.requireTally(id, 30, 0, 0)

			suite.atBlock(11)
			src.exit(suite, s, memberA)
			ballot, _ := suite.k.GetBallot(suite.ctx, id, memberA)
			suite.Require().True(ballot.Weight.IsZero())
			suite.Require().Empty(ballot.Shares)
			suite.Require().False(suite.k.HasVoteIndex(suite.ctx, src.source, memberA, id))
			suite.requireInvariants(id, memberA, memberC)

			// the coins move to memberC, who joins and votes
			src.join(suite, s, memberC, 30)
			suite.Require().ErrorIs(suite.vote(memberC, id, types.VoteOption_YES), types.ErrNoVotingPower)

			// memberA gets coins back and rejoins
			suite.atBlock(12)
			src.join(suite, s, memberA, 30)
			suite.Require().ErrorIs(suite.vote(memberA, id, types.VoteOption_NO), types.ErrNoVotingPower)

			suite.Require().True(suite.counted(id).LTE(math.NewInt(30)))
			suite.requireTally(id, 0, 0, 0)
			suite.requireInvariants(id, memberA, memberC)
		})
	}
}

// D3: stake that left and came back after the submit has no weight on the
// proposal, nor has stake that joined in the submit block; a proposal
// submitted after the rejoin counts it.
func (suite *IntegrationTestSuite) TestVote_D3_RejoinAfterSubmitHasNoWeight() {
	for _, src := range stakeSources {
		suite.Run(src.name, func() {
			suite.SetupTest()
			s := suite.createStakeDao()
			s.joinSr("1", memberB, 1) // no record: proposes and stays eligible
			if src.name == "staking reward" {
				s.joinSr("1", memberA, 30)
			} else {
				s.joinDr(stakeDenom, memberA, 30)
			}

			suite.atBlock(5)
			src.exit(suite, s, memberA)

			suite.atBlock(10)
			src.join(suite, s, memberC, 20) // same block as the submit
			first, err := suite.submit(memberB, 1, false)
			suite.Require().NoError(err)

			suite.atBlock(11)
			src.join(suite, s, memberA, 30)
			suite.Require().ErrorIs(suite.vote(memberA, first, types.VoteOption_YES), types.ErrNoVotingPower)
			suite.Require().ErrorIs(suite.vote(memberC, first, types.VoteOption_YES), types.ErrNoVotingPower)
			suite.requireTally(first, 0, 0, 0)

			suite.atBlock(12)
			second, err := suite.submit(memberB, 1, false)
			suite.Require().NoError(err)
			suite.Require().NoError(suite.vote(memberA, second, types.VoteOption_YES))
			suite.Require().NoError(suite.vote(memberC, second, types.VoteOption_NO))
			suite.requireTally(second, 30, 20, 0)
			suite.requireInvariants(second, memberA, memberC)
		})
	}
}

// D4: stake that joined after the submit has no weight; stake with no join
// record (it joined before the source was pinned) counts.
func (suite *IntegrationTestSuite) TestVote_D4_JoinAfterSubmit() {
	for _, src := range stakeSources {
		suite.Run(src.name, func() {
			suite.SetupTest()
			s := suite.createStakeDao()
			s.joinSr("1", memberB, 40)
			s.joinDr(stakeDenom, memberB, 10)
			suite.atBlock(10)
			id, err := suite.submit(memberB, 1, false)
			suite.Require().NoError(err)

			suite.atBlock(11)
			src.join(suite, s, memberC, 25)
			suite.Require().ErrorIs(suite.vote(memberC, id, types.VoteOption_YES), types.ErrNoVotingPower)

			_, found := suite.k.GetJoinRecord(suite.ctx, src.source, memberB)
			suite.Require().False(found)
			suite.Require().NoError(suite.vote(memberB, id, types.VoteOption_YES))
			suite.requireTally(id, 50, 0, 0)
			suite.requireInvariants(id, memberB, memberC)
		})
	}
}

// D5: a top-up during the vote counts once the voter votes again (the
// accepted residual); the cast ballot is not amplified on its own, and an
// exit zeroes it.
func (suite *IntegrationTestSuite) TestVote_D5_IncreaseCountsOnRevote() {
	s := suite.createStakeDao()
	s.joinSr("1", memberA, 30)
	suite.atBlock(10)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))

	suite.atBlock(11)
	s.joinSr("1", memberA, 50)
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardIncrease(suite.ctx, "1", memberA.String(), math.NewInt(20), math.NewInt(50), stakeDenom))
	suite.requireTally(id, 30, 0, 0)

	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_NO))
	suite.requireTally(id, 0, 50, 0)
	suite.requireInvariants(id, memberA)

	stakeSources[0].exit(suite, s, memberA)
	suite.requireTally(id, 0, 0, 0)
	ballot, _ := suite.k.GetBallot(suite.ctx, id, memberA)
	suite.Require().True(ballot.Weight.IsZero())
	suite.requireInvariants(id, memberA)
}

// D15: a staker whose every share is ineligible, or who has no stake in the
// proposal's sources, has no vote.
func (suite *IntegrationTestSuite) TestVote_D15_NoEligibleShare() {
	s := suite.createStakeDao()
	s.joinSr("1", memberB, 1)
	s.joinSr("5", attackerAddr, 1_000) // outside the electorate
	suite.atBlock(10)
	id, err := suite.submit(memberB, 1, false)
	suite.Require().NoError(err)

	suite.atBlock(11)
	stakeSources[0].join(suite, s, memberA, 10)
	stakeSources[1].join(suite, s, memberA, 10)

	suite.Require().ErrorIs(suite.vote(memberA, id, types.VoteOption_YES), types.ErrNoVotingPower)
	suite.Require().ErrorIs(suite.vote(attackerAddr, id, types.VoteOption_YES), types.ErrNoVotingPower)
	_, found := suite.k.GetBallot(suite.ctx, id, memberA)
	suite.Require().False(found)
	suite.Require().False(suite.k.HasVoteIndex(suite.ctx, srSource, memberA, id))
	suite.requireTally(id, 0, 0, 0)
}

// A voter in two staking rewards and the Denom Reward gets one share per
// source; leaving one source zeroes only that share.
func (suite *IntegrationTestSuite) TestVote_SharePerSource() {
	suite.stubRewards(defaultRewardsState())
	s := suite.stubStakes()
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1", "5"}, stakeDenom))
	suite.Require().NoError(err)

	s.joinSr("1", memberA, 10)
	s.joinSr("5", memberA, 20)
	s.joinDr(stakeDenom, memberA, 30)
	suite.atBlock(10)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_ABSTAIN))
	ballot, _ := suite.k.GetBallot(suite.ctx, id, memberA)
	suite.Require().Equal(math.NewInt(60), ballot.Weight)
	suite.Require().Equal([]types.BallotShare{
		{Source: "sr/1", Amount: math.NewInt(10)},
		{Source: "sr/5", Amount: math.NewInt(20)},
		{Source: "dr/ubze", Amount: math.NewInt(30)},
	}, ballot.Shares)
	suite.requireTally(id, 0, 0, 60)
	suite.requireInvariants(id, memberA)
	// the event carries the ballot's weight, the sum of the shares
	attrs := suite.lastVoteEvent()
	suite.Require().Equal(`"ABSTAIN"`, attrs["option"])
	suite.Require().Equal(`"60"`, attrs["weight"])

	sr5 := types.StakingRewardSource("5")
	delete(s.sr["5"], memberA.String())
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardExit(suite.ctx, "5", memberA.String(), math.NewInt(20), stakeDenom))

	ballot, _ = suite.k.GetBallot(suite.ctx, id, memberA)
	suite.Require().Equal(math.NewInt(40), ballot.Weight)
	suite.Require().Equal([]types.BallotShare{
		{Source: "sr/1", Amount: math.NewInt(10)},
		{Source: "dr/ubze", Amount: math.NewInt(30)},
	}, ballot.Shares)
	suite.Require().False(suite.k.HasVoteIndex(suite.ctx, sr5, memberA, id))
	suite.Require().True(suite.k.HasVoteIndex(suite.ctx, srSource, memberA, id))
	suite.Require().True(suite.k.HasVoteIndex(suite.ctx, drSource, memberA, id))
	suite.requireTally(id, 0, 0, 40)
	suite.requireInvariants(id, memberA)
}

// I4: cancelling a proposal drops each of its sources' pins by exactly one,
// back to the DAO's own pin once no proposal is in voting.
func (suite *IntegrationTestSuite) TestCancelProposal_UnpinsSources() {
	s := suite.createStakeDao()
	s.joinSr("1", memberA, 30)
	suite.atBlock(10)
	first, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	second, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(3), suite.k.GetSourcePins(suite.ctx, srSource))
	suite.Require().Equal(uint64(3), suite.k.GetSourcePins(suite.ctx, drSource))

	suite.Require().NoError(suite.cancel(memberA, first))
	suite.Require().Equal(uint64(2), suite.k.GetSourcePins(suite.ctx, srSource))
	suite.Require().Equal(uint64(2), suite.k.GetSourcePins(suite.ctx, drSource))

	suite.Require().NoError(suite.cancel(memberA, second))
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, srSource))
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, drSource))
}
