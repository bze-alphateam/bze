package keeper_test

import (
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/dao/types"
)

const submitTime = int64(1_700_000_000)

// atBlock moves the suite context to the given height, at submitTime plus one
// second per block.
func (suite *IntegrationTestSuite) atBlock(height int64) {
	suite.ctx = suite.ctx.WithBlockHeight(height).WithBlockTime(time.Unix(submitTime+height, 0))
}

// createDao creates the DAO with the given id from validCreateDaoMsg (a
// self-owned static list of members A and B) changed by mutate.
func (suite *IntegrationTestSuite) createDao(id uint64, mutate func(msg *types.MsgCreateDao)) types.Dao {
	suite.expectCreation(id)
	msg := validCreateDaoMsg()
	mutate(msg)
	res, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().Equal(id, res.Id)

	dao, found := suite.k.GetDao(suite.ctx, id)
	suite.Require().True(found)

	return dao
}

// createStaticDao creates the self-owned static-list DAO 1 (members A and B)
// with the given config changes.
func (suite *IntegrationTestSuite) createStaticDao(mutate func(c *types.DaoConfig)) types.Dao {
	return suite.createDao(1, func(msg *types.MsgCreateDao) { mutate(&msg.Config) })
}

func noConfigChange(*types.DaoConfig) {}

func (suite *IntegrationTestSuite) submit(proposer sdk.AccAddress, daoId uint64, expedited bool, msgs ...sdk.Msg) (uint64, error) {
	msg, err := types.NewMsgSubmitProposal(proposer.String(), daoId, "Fund tooling", "pays the tooling team", msgs, expedited)
	suite.Require().NoError(err)

	return suite.submitMsg(msg)
}

func (suite *IntegrationTestSuite) submitMsg(msg *types.MsgSubmitProposal) (uint64, error) {
	res, err := suite.msgServer.SubmitProposal(suite.ctx, msg)
	if err != nil {
		return 0, err
	}

	return res.ProposalId, nil
}

// requireNothingStored checks that no proposal was written.
func (suite *IntegrationTestSuite) requireNothingStored() {
	suite.Require().Equal(uint64(0), suite.k.GetProposalCounter(suite.ctx))
	_, found := suite.k.GetProposal(suite.ctx, 1)
	suite.Require().False(found)
	suite.Require().False(suite.k.IsOpenProposal(suite.ctx, 1, 1))
	suite.Require().False(suite.k.HasDaoProposalIndex(suite.ctx, 1, 1))
}

func coinPtr(amount int64) *sdk.Coin {
	coin := sdk.NewInt64Coin("ubze", amount)
	return &coin
}

// A member's proposal is stored in voting with the DAO's config pinned, its
// denominator (the member count), zero tallies, its open marker and its DAO
// index entry, and the event carries the pinned values.
func (suite *IntegrationTestSuite) TestSubmitProposal_StoresPinnedSnapshot() {
	dao := suite.createStaticDao(noConfigChange)
	suite.atBlock(10)

	id, err := suite.submit(memberA, 1, false, treasurySend(memberB, 5))
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(1), id)

	proposal, found := suite.k.GetProposal(suite.ctx, 1)
	suite.Require().True(found)
	suite.Require().Equal(uint64(1), proposal.DaoId)
	suite.Require().Equal(memberA.String(), proposal.Proposer)
	suite.Require().Equal("Fund tooling", proposal.Title)
	suite.Require().Equal("pays the tooling team", proposal.Summary)
	suite.Require().Len(proposal.Msgs, 1)
	suite.Require().Equal(treasurySend(memberB, 5), proposal.Msgs[0].GetCachedValue())
	suite.Require().False(proposal.Expedited)
	suite.Require().Equal(types.ProposalStatus_VOTING, proposal.Status)
	suite.Require().Equal(int64(10), proposal.SubmitHeight)
	suite.Require().Equal(submitTime+10, proposal.SubmitTime)
	suite.Require().Equal(submitTime+10+int64((72*time.Hour).Seconds()), proposal.VotingEnd)
	suite.Require().Equal(dao.Config, proposal.PinnedConfig)
	suite.Require().Nil(proposal.PinnedElectorate)
	suite.Require().Equal(math.NewInt(2), proposal.PinnedDenominator)
	suite.Require().Nil(proposal.PinnedDeposit)
	suite.Require().True(proposal.Yes.IsZero())
	suite.Require().True(proposal.No.IsZero())
	suite.Require().True(proposal.Abstain.IsZero())

	suite.Require().Equal(uint64(1), suite.k.GetProposalCounter(suite.ctx))
	suite.Require().True(suite.k.IsOpenProposal(suite.ctx, 1, 1))
	suite.Require().True(suite.k.HasDaoProposalIndex(suite.ctx, 1, 1))

	suite.Require().True(suite.hasEvent(proto.MessageName(&types.EventProposalSubmitted{})))
	for _, event := range suite.ctx.EventManager().Events() {
		if event.Type != proto.MessageName(&types.EventProposalSubmitted{}) {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range event.Attributes {
			attrs[attr.Key] = attr.Value
		}
		suite.Require().Equal(`"1"`, attrs["id"])
		suite.Require().Equal(`"1"`, attrs["dao_id"])
		suite.Require().Equal(`"`+memberA.String()+`"`, attrs["proposer"])
		suite.Require().Equal(`"2"`, attrs["denominator"])
	}
}

// A proposal with no message is a signalling proposal and is accepted.
func (suite *IntegrationTestSuite) TestSubmitProposal_NoMessages() {
	suite.createStaticDao(noConfigChange)

	id, err := suite.submit(memberB, 1, false)
	suite.Require().NoError(err)

	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.Require().Empty(proposal.Msgs)
}

// The expedited flag picks the expedited period, and is refused when the
// config does not allow it.
func (suite *IntegrationTestSuite) TestSubmitProposal_Expedited() {
	suite.createStaticDao(noConfigChange)
	suite.atBlock(10)

	id, err := suite.submit(memberA, 1, true)
	suite.Require().NoError(err)
	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.Require().True(proposal.Expedited)
	suite.Require().Equal(submitTime+10+int64((24*time.Hour).Seconds()), proposal.VotingEnd)
}

func (suite *IntegrationTestSuite) TestSubmitProposal_ExpeditedRefusedWhenDisallowed() {
	suite.createStaticDao(func(c *types.DaoConfig) { c.AllowExpedite = false })

	_, err := suite.submit(memberA, 1, true)
	suite.Require().ErrorIs(err, types.ErrExpediteNotAllowed)
	suite.requireNothingStored()

	_, err = suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
}

// D1 (submit part): a proposal cannot carry a send from someone else's account.
func (suite *IntegrationTestSuite) TestSubmitProposal_VictimSendRejected() {
	suite.createStaticDao(noConfigChange)

	_, err := suite.submit(memberA, 1, false, banktypes.NewMsgSend(victimAddr, memberA, ubze(100)))

	suite.Require().ErrorIs(err, types.ErrUnauthorizedSigner)
	suite.requireNothingStored()
}

// D24: every signer of every message must be the DAO; a second signer is refused.
func (suite *IntegrationTestSuite) TestSubmitProposal_SecondSignerRejected() {
	suite.createStaticDao(noConfigChange)

	multiSend := &banktypes.MsgMultiSend{
		Inputs: []banktypes.Input{
			banktypes.NewInput(daoAddress(1), ubze(10)),
			banktypes.NewInput(attackerAddr, ubze(10)),
		},
		Outputs: []banktypes.Output{banktypes.NewOutput(attackerAddr, ubze(20))},
	}
	_, err := suite.submit(memberA, 1, false, treasurySend(memberB, 1), multiSend)

	suite.Require().ErrorIs(err, types.ErrUnauthorizedSigner)
	suite.Require().ErrorContains(err, "message 1")
	suite.requireNothingStored()
}

// D22: a non-member is refused while submission is closed (and pays nothing:
// gomock fails on an unexpected bank call), and accepted and charged when it
// is open.
func (suite *IntegrationTestSuite) TestSubmitProposal_NonMemberNeedsOpenSubmission() {
	withCost := func(c *types.DaoConfig) { c.ProposalCost = coinPtr(7) }
	suite.createStaticDao(withCost)
	suite.createDao(2, func(msg *types.MsgCreateDao) {
		withCost(&msg.Config)
		msg.Config.OpenSubmission = true
	})

	_, err := suite.submit(attackerAddr, 1, false)
	suite.Require().ErrorIs(err, types.ErrNotProposer)
	suite.requireNothingStored()

	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), attackerAddr, burnerFeeCollector, ubze(7)).Return(nil)
	id, err := suite.submit(attackerAddr, 2, false)
	suite.Require().NoError(err)

	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.Require().Equal(uint64(2), proposal.DaoId)
	suite.Require().Equal(attackerAddr.String(), proposal.Proposer)
	suite.Require().True(proposal.PinnedConfig.OpenSubmission)
}

// The DAO may have at most max_open_proposals proposals in voting; other
// DAOs' proposals do not count.
func (suite *IntegrationTestSuite) TestSubmitProposal_MaxOpenProposals() {
	suite.createStaticDao(noConfigChange)
	params := suite.k.GetParams(suite.ctx)
	params.MaxOpenProposals = 2
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	for range 2 {
		_, err := suite.submit(memberA, 1, false)
		suite.Require().NoError(err)
	}

	_, err := suite.submit(memberB, 1, false)
	suite.Require().ErrorIs(err, types.ErrTooManyProposals)
	suite.Require().Equal(uint64(2), suite.k.GetProposalCounter(suite.ctx))

	// the cap is per DAO: another DAO still submits
	suite.createDao(2, func(*types.MsgCreateDao) {})
	_, err = suite.submit(memberA, 2, false)
	suite.Require().NoError(err)
}

// The default cap is 10.
func (suite *IntegrationTestSuite) TestSubmitProposal_DefaultOpenProposalCap() {
	suite.createStaticDao(noConfigChange)

	for range types.DefaultMaxOpenProposals {
		_, err := suite.submit(memberA, 1, false)
		suite.Require().NoError(err)
	}

	_, err := suite.submit(memberA, 1, false)
	suite.Require().ErrorIs(err, types.ErrTooManyProposals)
}

// The title, summary, message count and message bytes caps, on both sides.
func (suite *IntegrationTestSuite) TestSubmitProposal_ContentCaps() {
	suite.createStaticDao(noConfigChange)
	params := suite.k.GetParams(suite.ctx)
	params.MaxTitleLen = 10
	params.MaxSummaryLen = 20
	params.MaxMsgsPerProposal = 2
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	send := treasurySend(memberB, 1)
	sendBytes := 0
	{
		msg, err := types.NewMsgSubmitProposal(memberA.String(), 1, "t", "", []sdk.Msg{send}, false)
		suite.Require().NoError(err)
		sendBytes = msg.Msgs[0].Size()
	}

	for _, tc := range []struct {
		desc     string
		title    string
		summary  string
		msgs     []sdk.Msg
		maxBytes uint64
		wantErr  error
	}{
		{desc: "title at the cap", title: strings.Repeat("t", 10)},
		{desc: "title over the cap", title: strings.Repeat("t", 11), wantErr: types.ErrInvalidProposal},
		{desc: "empty title", title: "", wantErr: types.ErrInvalidProposal},
		{desc: "summary at the cap", title: "t", summary: strings.Repeat("s", 20)},
		{desc: "summary over the cap", title: "t", summary: strings.Repeat("s", 21), wantErr: types.ErrInvalidProposal},
		{desc: "msgs at the cap", title: "t", msgs: []sdk.Msg{send, send}},
		{desc: "msgs over the cap", title: "t", msgs: []sdk.Msg{send, send, send}, wantErr: types.ErrInvalidMsgs},
		{desc: "bytes at the cap", title: "t", msgs: []sdk.Msg{send, send}, maxBytes: uint64(2 * sendBytes)},
		{desc: "bytes over the cap", title: "t", msgs: []sdk.Msg{send, send}, maxBytes: uint64(2*sendBytes - 1), wantErr: types.ErrInvalidMsgs},
	} {
		suite.Run(tc.desc, func() {
			p := params
			p.MaxProposalBytes = types.DefaultMaxProposalBytes
			if tc.maxBytes > 0 {
				p.MaxProposalBytes = tc.maxBytes
			}
			p.MaxOpenProposals = 100
			suite.Require().NoError(suite.k.SetParams(suite.ctx, p))
			before := suite.k.GetProposalCounter(suite.ctx)

			msg, err := types.NewMsgSubmitProposal(memberA.String(), 1, tc.title, tc.summary, tc.msgs, false)
			suite.Require().NoError(err)
			_, err = suite.submitMsg(msg)

			if tc.wantErr == nil {
				suite.Require().NoError(err)
				suite.Require().Equal(before+1, suite.k.GetProposalCounter(suite.ctx))
				return
			}
			suite.Require().ErrorIs(err, tc.wantErr)
			suite.Require().Equal(before, suite.k.GetProposalCounter(suite.ctx))
		})
	}
}

func (suite *IntegrationTestSuite) TestSubmitProposal_InvalidMsgRejected() {
	suite.createStaticDao(noConfigChange)

	_, err := suite.submitMsg(&types.MsgSubmitProposal{Proposer: "bze1nope", DaoId: 1, Title: "t"})
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidAddress)
	suite.requireNothingStored()
}

func (suite *IntegrationTestSuite) TestSubmitProposal_UnknownDao() {
	_, err := suite.submit(memberA, 7, false)

	suite.Require().ErrorIs(err, types.ErrDaoNotFound)
	suite.requireNothingStored()
}

// The proposal cost goes to the burner with cost_destination BURN.
func (suite *IntegrationTestSuite) TestSubmitProposal_CostToBurner() {
	suite.createStaticDao(func(c *types.DaoConfig) {
		c.ProposalCost = coinPtr(25)
		c.CostDestination = types.FundsDestination_BURN
	})

	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, burnerFeeCollector, ubze(25)).Return(nil)
	_, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
}

// The proposal cost goes to the DAO's own account with cost_destination TREASURY.
func (suite *IntegrationTestSuite) TestSubmitProposal_CostToTreasury() {
	suite.createStaticDao(func(c *types.DaoConfig) {
		c.ProposalCost = coinPtr(25)
		c.CostDestination = types.FundsDestination_TREASURY
	})

	suite.bank.EXPECT().SendCoins(gomock.Any(), memberA, daoAddress(1), ubze(25)).Return(nil)
	_, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
}

// A zero cost and a zero deposit are not charged.
func (suite *IntegrationTestSuite) TestSubmitProposal_ZeroCostAndDepositSkipped() {
	suite.createStaticDao(func(c *types.DaoConfig) {
		c.ProposalCost = coinPtr(0)
		c.ProposalDeposit = coinPtr(0)
	})

	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.Require().Nil(proposal.PinnedDeposit)
}

// The deposit is escrowed in the dao module account and pinned on the proposal.
func (suite *IntegrationTestSuite) TestSubmitProposal_DepositEscrowed() {
	suite.createStaticDao(func(c *types.DaoConfig) {
		c.ProposalCost = coinPtr(25)
		c.ProposalDeposit = coinPtr(40)
	})

	gomock.InOrder(
		suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, burnerFeeCollector, ubze(25)).Return(nil),
		suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, types.ModuleName, ubze(40)).Return(nil),
	)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	proposal, _ := suite.k.GetProposal(suite.ctx, id)
	suite.Require().Equal(coinPtr(40), proposal.PinnedDeposit)
}

// A proposer who can't pay the cost, or the deposit, leaves nothing behind.
func (suite *IntegrationTestSuite) TestSubmitProposal_InsufficientFundsStoresNothing() {
	suite.createStaticDao(func(c *types.DaoConfig) {
		c.ProposalCost = coinPtr(25)
		c.ProposalDeposit = coinPtr(40)
	})

	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, burnerFeeCollector, ubze(25)).Return(sdkerrors.ErrInsufficientFunds)
	_, err := suite.submit(memberA, 1, false)
	suite.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds)
	suite.Require().ErrorContains(err, "proposal cost")
	suite.requireNothingStored()

	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, burnerFeeCollector, ubze(25)).Return(nil)
	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, types.ModuleName, ubze(40)).Return(sdkerrors.ErrInsufficientFunds)
	_, err = suite.submit(memberA, 1, false)
	suite.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds)
	suite.Require().ErrorContains(err, "proposal deposit")
	suite.requireNothingStored()
}

// I1: what the module account received equals the sum of the pinned deposits.
// I5: proposal ids grow by one across DAOs and never repeat.
func (suite *IntegrationTestSuite) TestSubmitProposal_DepositsMatchEscrowAndIdsGrow() {
	suite.createStaticDao(func(c *types.DaoConfig) { c.ProposalDeposit = coinPtr(40) })
	suite.createDao(2, func(msg *types.MsgCreateDao) { msg.Config.ProposalDeposit = coinPtr(15) })

	escrowed := sdk.NewCoins()
	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), types.ModuleName, gomock.Any()).
		DoAndReturn(func(_ any, _ sdk.AccAddress, _ string, amt sdk.Coins) error {
			escrowed = escrowed.Add(amt...)
			return nil
		}).Times(5)

	var ids []uint64
	for _, s := range []struct {
		proposer sdk.AccAddress
		daoId    uint64
	}{{memberA, 1}, {memberB, 2}, {memberB, 1}, {memberA, 2}, {memberA, 1}} {
		id, err := suite.submit(s.proposer, s.daoId, false)
		suite.Require().NoError(err)
		ids = append(ids, id)
	}
	suite.Require().Equal([]uint64{1, 2, 3, 4, 5}, ids)

	pinned := sdk.NewCoins()
	for _, id := range ids {
		proposal, found := suite.k.GetProposal(suite.ctx, id)
		suite.Require().True(found)
		pinned = pinned.Add(*proposal.PinnedDeposit)
		suite.Require().True(suite.k.HasDaoProposalIndex(suite.ctx, proposal.DaoId, id))
		suite.Require().False(suite.k.HasDaoProposalIndex(suite.ctx, 3-proposal.DaoId, id))
	}
	suite.Require().Equal(escrowed, pinned)
	suite.Require().Equal(ubze(3*40+2*15), pinned)
}

// The admin can run MsgSubmitProposal as the DAO: the DAO account is the
// proposer even though it is not a member, and pays from its treasury. A
// treasury cost would be a payment to itself, so it is skipped.
func (suite *IntegrationTestSuite) TestSubmitProposal_DaoAccountAsProposer() {
	suite.createDao(1, func(msg *types.MsgCreateDao) {
		msg.Admin = adminAddr.String()
		msg.Config.ProposalCost = coinPtr(25)
		msg.Config.CostDestination = types.FundsDestination_TREASURY
		msg.Config.ProposalDeposit = coinPtr(40)
	})

	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), daoAddress(1), types.ModuleName, ubze(40)).Return(nil)
	submit, err := types.NewMsgSubmitProposal(daoAddress(1).String(), 1, "self", "", nil, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, submit))

	proposal, found := suite.k.GetProposal(suite.ctx, 1)
	suite.Require().True(found)
	suite.Require().Equal(daoAddress(1).String(), proposal.Proposer)
}

// A stake DAO whose sources hold no stake has no voting power to pin.
func (suite *IntegrationTestSuite) TestSubmitProposal_NoVotingPowerRejected() {
	suite.stubRewards(defaultRewardsState())
	suite.expectCreation(1)
	msg := validStakeDaoMsg([]string{"2"}, "uzero")
	msg.Electorate.DrDenom = ""
	msg.Admin = adminAddr.String()
	msg.Config.OpenSubmission = true
	_, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)

	_, err = suite.submit(memberA, 1, false)
	suite.Require().ErrorIs(err, types.ErrNoVotingPower)
	suite.requireNothingStored()

	// an external admin may leave the electorate empty: no power either
	suite.expectCreation(2)
	empty := validStakeDaoMsg(nil, "")
	empty.Admin = adminAddr.String()
	empty.Config.OpenSubmission = true
	_, err = suite.msgServer.CreateDao(suite.ctx, empty)
	suite.Require().NoError(err)

	_, err = suite.submit(memberA, 2, false)
	suite.Require().ErrorIs(err, types.ErrNoVotingPower)
}

// lastVoteEvent returns the attributes of the last EventVote emitted.
func (suite *IntegrationTestSuite) lastVoteEvent() map[string]string {
	var attrs map[string]string
	for _, event := range suite.ctx.EventManager().Events() {
		if event.Type != proto.MessageName(&types.EventVote{}) {
			continue
		}
		attrs = map[string]string{}
		for _, attr := range event.Attributes {
			attrs[attr.Key] = attr.Value
		}
	}

	return attrs
}

// A member's vote is stored with weight 1, counted in the tally and announced.
func (suite *IntegrationTestSuite) TestVote_StaticMemberVotes() {
	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	suite.Require().NoError(suite.vote(memberB, id, types.VoteOption_NO))

	ballot, found := suite.k.GetBallot(suite.ctx, id, memberB)
	suite.Require().True(found)
	suite.Require().Equal(types.Ballot{ProposalId: id, Voter: memberB.String(), Option: types.VoteOption_NO, Weight: math.OneInt()}, ballot)
	suite.requireTally(id, 0, 1, 0)
	suite.requireInvariants(id, memberA, memberB)

	attrs := suite.lastVoteEvent()
	suite.Require().Equal(`"1"`, attrs["proposal_id"])
	suite.Require().Equal(`"`+memberB.String()+`"`, attrs["voter"])
	suite.Require().Equal(`"NO"`, attrs["option"])
	suite.Require().Equal(`"1"`, attrs["weight"])
}

// D15: an address that is not on the DAO's list has no vote.
func (suite *IntegrationTestSuite) TestVote_NonMemberRejected() {
	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	for _, voter := range []sdk.AccAddress{memberC, attackerAddr, daoAddress(1)} {
		err = suite.vote(voter, id, types.VoteOption_YES)
		suite.Require().ErrorIs(err, types.ErrNoVotingPower, voter.String())
		_, found := suite.k.GetBallot(suite.ctx, id, voter)
		suite.Require().False(found)
	}
	suite.requireTally(id, 0, 0, 0)
}

// D18: voting closes at voting_end itself; the last second before it is open.
func (suite *IntegrationTestSuite) TestVote_ClosesAtVotingEnd() {
	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	proposal, _ := suite.k.GetProposal(suite.ctx, id)

	suite.ctx = suite.ctx.WithBlockTime(time.Unix(proposal.VotingEnd-1, 0))
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))

	suite.ctx = suite.ctx.WithBlockTime(time.Unix(proposal.VotingEnd, 0))
	err = suite.vote(memberB, id, types.VoteOption_YES)
	suite.Require().ErrorIs(err, types.ErrVotingClosed)

	suite.ctx = suite.ctx.WithBlockTime(time.Unix(proposal.VotingEnd+3600, 0))
	err = suite.vote(memberA, id, types.VoteOption_NO)
	suite.Require().ErrorIs(err, types.ErrVotingClosed)

	suite.requireTally(id, 1, 0, 0)
}

// An expedited proposal closes at the end of its shorter period.
func (suite *IntegrationTestSuite) TestVote_ExpeditedClosesEarlier() {
	suite.createStaticDao(noConfigChange)
	suite.atBlock(10)
	id, err := suite.submit(memberA, 1, true)
	suite.Require().NoError(err)

	suite.ctx = suite.ctx.WithBlockTime(time.Unix(submitTime+10+int64((24*time.Hour).Seconds()), 0))
	err = suite.vote(memberA, id, types.VoteOption_YES)
	suite.Require().ErrorIs(err, types.ErrVotingClosed)
}

// A vote on an unknown proposal, or an invalid message, changes nothing.
func (suite *IntegrationTestSuite) TestVote_InvalidRequests() {
	suite.createStaticDao(noConfigChange)

	err := suite.vote(memberA, 1, types.VoteOption_YES)
	suite.Require().ErrorIs(err, types.ErrProposalNotFound)

	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	err = suite.vote(memberA, id, types.VoteOption(7))
	suite.Require().ErrorIs(err, types.ErrInvalidVote)

	_, err = suite.msgServer.Vote(suite.ctx, &types.MsgVote{Voter: "bze1nope", ProposalId: id})
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidAddress)

	suite.requireTally(id, 0, 0, 0)
}

// A vote whose option is left out decodes to UNSPECIFIED and is rejected, so
// it can never count as YES; no ballot is stored.
func (suite *IntegrationTestSuite) TestVote_OmittedOptionRejected() {
	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	// the wire form of a vote without the option field
	bz, err := (&types.MsgVote{Voter: memberA.String(), ProposalId: id}).Marshal()
	suite.Require().NoError(err)
	var decoded types.MsgVote
	suite.Require().NoError(decoded.Unmarshal(bz))
	suite.Require().Equal(types.VoteOption_VOTE_OPTION_UNSPECIFIED, decoded.Option)

	_, err = suite.msgServer.Vote(suite.ctx, &decoded)
	suite.Require().ErrorIs(err, types.ErrInvalidVote)
	_, found := suite.k.GetBallot(suite.ctx, id, memberA)
	suite.Require().False(found)
	suite.requireTally(id, 0, 0, 0)
}
