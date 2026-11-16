package keeper_test

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

var (
	memberD = sdk.AccAddress("dao_member_d________")
	memberE = sdk.AccAddress("dao_member_e________")
)

// createCouncil creates DAO 1, a static list of five members with expedite
// bars 0.5 (threshold) and 0.6 (quorum): three yes clear both.
func (suite *IntegrationTestSuite) createCouncil() {
	suite.createDao(1, func(msg *types.MsgCreateDao) {
		msg.Members = []string{memberA.String(), memberB.String(), memberC.String(), memberD.String(), memberE.String()}
		msg.Config.ExpediteThreshold = dec("0.5")
		msg.Config.ExpediteQuorum = dec("0.6")
	})
}

// A static-list DAO is exempt from min_expedite_delay: with the default 24h
// delay, an expedited proposal settles in its submit block once 3 of 5 voted
// yes, and its messages run as the DAO.
func (suite *IntegrationTestSuite) TestSettleProposal_StaticListSettlesInSubmitBlock() {
	suite.Require().Equal(24*time.Hour, suite.k.GetParams(suite.ctx).MinExpediteDelay)
	suite.createCouncil()
	suite.ledger.fund(daoAddress(1), ubze(100))

	id, err := suite.submit(memberA, 1, true, treasurySend(memberB, 30))
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberB, id, types.VoteOption_YES))
	suite.requireLiveStatus(id, false, 0, false)
	_, err = suite.settle(id)
	suite.Require().ErrorIs(err, types.ErrExpediteBarNotMet, "2 of 5 misses the bars, the delay is not what refuses it")
	suite.requireStillVoting(id)

	suite.Require().NoError(suite.vote(memberC, id, types.VoteOption_YES))
	suite.requireLiveStatus(id, false, 0, true)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, status)
	suite.requireSettled(id, types.ProposalStatus_EXECUTED, true)
	suite.requireExecuted("")
	suite.Require().Equal(ubze(70), suite.ledger.balance(daoAddress(1)))
	suite.Require().Equal(ubze(30), suite.ledger.balance(memberB))

	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.Require().Equal(suite.ctx.BlockTime().Unix(), proposal.SubmitTime, "settled in the submit block")
}

// The exemption only drops the delay: a regular proposal of a static-list DAO
// still waits for its voting end, even unanimous.
func (suite *IntegrationTestSuite) TestSettleProposal_StaticListRegularStillWaits() {
	suite.createCouncil()
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	for _, m := range []sdk.AccAddress{memberA, memberB, memberC, memberD, memberE} {
		suite.Require().NoError(suite.vote(m, id, types.VoteOption_YES))
	}

	suite.requireLiveStatus(id, false, 0, false)
	_, err = suite.settle(id)
	suite.Require().ErrorIs(err, types.ErrVotingNotEnded)
	suite.requireStillVoting(id)
}

// A stake DAO meeting the same bars keeps the delay: refused as too early in
// the submit block, settleable from submit_time + min_expedite_delay.
func (suite *IntegrationTestSuite) TestSettleProposal_StakeDaoKeepsExpediteDelay() {
	_, s := suite.expediteStakeDao(noConfigChange)
	s.joinSr("1", memberA, 100)
	suite.atBlock(10)
	id, err := suite.submit(memberA, 1, true)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))

	suite.requireLiveStatus(id, false, 0, false)
	_, err = suite.settle(id)
	suite.Require().ErrorIs(err, types.ErrExpediteTooEarly)
	suite.requireStillVoting(id)

	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.endVoting(id, -1)
	suite.Require().Less(suite.ctx.BlockTime().Unix(), proposal.SubmitTime+int64((24*time.Hour).Seconds()),
		"the 24h expedited period ends with the delay")
	suite.requireLiveStatus(id, false, 0, false)
	_, err = suite.settle(id)
	suite.Require().ErrorIs(err, types.ErrExpediteTooEarly)
}

// Fail closed: a proposal whose DAO record cannot be found keeps the delay,
// so the exemption only applies to a DAO known to be a static list.
func (suite *IntegrationTestSuite) TestSettleProposal_MissingDaoKeepsExpediteDelay() {
	suite.setExpediteDelay(time.Hour)
	now := suite.ctx.BlockTime().Unix()
	suite.k.SetAllProposals(suite.ctx, []types.Proposal{{
		Id:                1,
		DaoId:             7,
		Proposer:          memberA.String(),
		Expedited:         true,
		Status:            types.ProposalStatus_VOTING,
		SubmitTime:        now,
		VotingEnd:         now + int64((24 * time.Hour).Seconds()),
		PinnedConfig:      validConfig(),
		PinnedDenominator: math.NewInt(2),
		Yes:               math.NewInt(2),
		No:                math.ZeroInt(),
		Abstain:           math.ZeroInt(),
	}})
	_, found := suite.k.GetDao(suite.ctx, 7)
	suite.Require().False(found)

	suite.requireLiveStatus(1, false, 0, false)
	_, err := suite.settle(1)
	suite.Require().ErrorIs(err, types.ErrExpediteTooEarly)
	suite.requireStillVoting(1)

	// only the delay held it back
	suite.ctx = suite.ctx.WithBlockTime(time.Unix(now+3600, 0))
	suite.requireLiveStatus(1, false, 0, true)
}
