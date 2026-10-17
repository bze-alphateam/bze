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
