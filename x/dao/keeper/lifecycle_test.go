package keeper_test

import (
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
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

func (suite *IntegrationTestSuite) cancel(proposer sdk.AccAddress, proposalId uint64) error {
	_, err := suite.msgServer.CancelProposal(suite.ctx, types.NewMsgCancelProposal(proposer.String(), proposalId))
	return err
}

// lastEventAttrs returns the attributes of the last emitted event of the
// given type, or nil when there is none.
func (suite *IntegrationTestSuite) lastEventAttrs(event proto.Message) map[string]string {
	var attrs map[string]string
	for _, e := range suite.ctx.EventManager().Events() {
		if e.Type != proto.MessageName(event) {
			continue
		}
		attrs = map[string]string{}
		for _, attr := range e.Attributes {
			attrs[attr.Key] = attr.Value
		}
	}

	return attrs
}

// requireCancelled checks that the proposal left voting: CANCELLED, no open
// marker, still in its DAO's index.
func (suite *IntegrationTestSuite) requireCancelled(proposalId uint64) {
	proposal, found := suite.k.GetProposal(suite.ctx, proposalId)
	suite.Require().True(found)
	suite.Require().Equal(types.ProposalStatus_CANCELLED, proposal.Status)
	suite.Require().False(suite.k.IsOpenProposal(suite.ctx, proposal.DaoId, proposalId))
	suite.Require().True(suite.k.HasDaoProposalIndex(suite.ctx, proposal.DaoId, proposalId))
	suite.Require().Equal(`"`+strconv.FormatUint(proposalId, 10)+`"`, suite.lastEventAttrs(&types.EventProposalCancelled{})["id"])
}

// Cancel matrix: ALWAYS refunds the deposit to the proposer; ONLY_PASSED and
// NEVER keep it and send it to the pinned kept_deposit_destination, the burner
// or the DAO's treasury.
func (suite *IntegrationTestSuite) TestCancelProposal_DepositMatrix() {
	burner := authtypes.NewModuleAddress(burnerFeeCollector)
	for _, tc := range []struct {
		policy      types.DepositRefundPolicy
		destination types.FundsDestination
		refunded    bool
		to          sdk.AccAddress
	}{
		{types.DepositRefundPolicy_ALWAYS, types.FundsDestination_BURN, true, memberA},
		{types.DepositRefundPolicy_ALWAYS, types.FundsDestination_TREASURY, true, memberA},
		{types.DepositRefundPolicy_ONLY_PASSED, types.FundsDestination_BURN, false, burner},
		{types.DepositRefundPolicy_ONLY_PASSED, types.FundsDestination_TREASURY, false, daoAddress(1)},
		{types.DepositRefundPolicy_NEVER, types.FundsDestination_BURN, false, burner},
		{types.DepositRefundPolicy_NEVER, types.FundsDestination_TREASURY, false, daoAddress(1)},
	} {
		suite.Run(tc.policy.String()+"/"+tc.destination.String(), func() {
			suite.SetupTest()
			suite.createStaticDao(func(c *types.DaoConfig) {
				c.ProposalDeposit = coinPtr(40)
				c.DepositRefundPolicy = tc.policy
				c.KeptDepositDestination = tc.destination
			})
			suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, types.ModuleName, ubze(40)).Return(nil)
			id, err := suite.submit(memberA, 1, false)
			suite.Require().NoError(err)

			if tc.to.Equals(burner) {
				suite.bank.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnerFeeCollector, ubze(40)).Return(nil)
			} else {
				suite.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, tc.to, ubze(40)).Return(nil)
			}
			suite.Require().NoError(suite.cancel(memberA, id))

			suite.requireCancelled(id)
			attrs := suite.lastEventAttrs(&types.EventDepositReleased{})
			suite.Require().Equal(`"`+strconv.FormatUint(id, 10)+`"`, attrs["id"])
			suite.Require().Equal(strconv.FormatBool(tc.refunded), attrs["refunded"])
			suite.Require().Equal(`"`+tc.to.String()+`"`, attrs["destination"])
			suite.Require().Equal(`"40ubze"`, attrs["amount"])
		})
	}
}

// A proposal without a deposit is cancelled with no transfer and no release event.
func (suite *IntegrationTestSuite) TestCancelProposal_NoDeposit() {
	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	suite.Require().NoError(suite.cancel(memberA, id))
	suite.requireCancelled(id)
	suite.Require().Nil(suite.lastEventAttrs(&types.EventDepositReleased{}))
}

// A failed release fails the cancel: the proposal stays in voting.
func (suite *IntegrationTestSuite) TestCancelProposal_ReleaseFailureKeepsVoting() {
	suite.createStaticDao(func(c *types.DaoConfig) { c.ProposalDeposit = coinPtr(40) })
	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, types.ModuleName, ubze(40)).Return(nil)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	suite.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, memberA, ubze(40)).Return(sdkerrors.ErrInsufficientFunds)
	err = suite.cancel(memberA, id)
	suite.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds)
	suite.Require().ErrorContains(err, "release the deposit")
}

// Only the proposer can cancel, only while the proposal is in voting, and only once.
func (suite *IntegrationTestSuite) TestCancelProposal_Refusals() {
	suite.createStaticDao(noConfigChange)

	err := suite.cancel(memberA, 1)
	suite.Require().ErrorIs(err, types.ErrProposalNotFound)

	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	for _, signer := range []sdk.AccAddress{memberB, attackerAddr, daoAddress(1)} {
		err = suite.cancel(signer, id)
		suite.Require().ErrorIs(err, types.ErrNotProposalOwner, signer.String())
	}
	_, err = suite.msgServer.CancelProposal(suite.ctx, &types.MsgCancelProposal{Proposer: "bze1nope", ProposalId: id})
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidAddress)
	suite.Require().True(suite.k.IsOpenProposal(suite.ctx, 1, id))

	suite.Require().NoError(suite.cancel(memberA, id))
	err = suite.cancel(memberA, id)
	suite.Require().ErrorIs(err, types.ErrVotingClosed)
	suite.requireCancelled(id)
}

// The proposer can still cancel after voting_end, until the proposal is settled.
func (suite *IntegrationTestSuite) TestCancelProposal_AfterVotingEnd() {
	suite.createStaticDao(noConfigChange)
	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	proposal, _ := suite.k.GetProposal(suite.ctx, id)

	suite.ctx = suite.ctx.WithBlockTime(time.Unix(proposal.VotingEnd+3600, 0))
	suite.Require().NoError(suite.cancel(memberA, id))
	suite.requireCancelled(id)
}

// A cancelled proposal takes no votes, keeps its ballots and tally, and frees
// its slot under max_open_proposals.
func (suite *IntegrationTestSuite) TestCancelProposal_ClosesVoting() {
	suite.createStaticDao(noConfigChange)
	params := suite.k.GetParams(suite.ctx)
	params.MaxOpenProposals = 1
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	id, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberB, id, types.VoteOption_YES))
	_, err = suite.submit(memberA, 1, false)
	suite.Require().ErrorIs(err, types.ErrTooManyProposals)

	suite.Require().NoError(suite.cancel(memberA, id))

	err = suite.vote(memberA, id, types.VoteOption_NO)
	suite.Require().ErrorIs(err, types.ErrVotingClosed)
	err = suite.vote(memberB, id, types.VoteOption_NO)
	suite.Require().ErrorIs(err, types.ErrVotingClosed)
	suite.requireTally(id, 1, 0, 0)
	_, found := suite.k.GetBallot(suite.ctx, id, memberB)
	suite.Require().True(found)

	_, err = suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
}

// I1: after cancels, the module account holds exactly the deposits of the
// proposals still in voting.
func (suite *IntegrationTestSuite) TestCancelProposal_EscrowMatchesOpenDeposits() {
	suite.createStaticDao(func(c *types.DaoConfig) { c.ProposalDeposit = coinPtr(40) })
	suite.createDao(2, func(msg *types.MsgCreateDao) {
		msg.Config.ProposalDeposit = coinPtr(15)
		msg.Config.DepositRefundPolicy = types.DepositRefundPolicy_NEVER
	})

	escrow := sdk.NewCoins()
	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), types.ModuleName, gomock.Any()).
		DoAndReturn(func(_ any, _ sdk.AccAddress, _ string, amt sdk.Coins) error {
			escrow = escrow.Add(amt...)
			return nil
		}).Times(4)
	release := func(_ any, _ string, _ any, amt sdk.Coins) error {
		escrow = escrow.Sub(amt...)
		return nil
	}
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).DoAndReturn(release)
	suite.bank.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).DoAndReturn(release)

	var ids []uint64
	for _, s := range []struct {
		proposer sdk.AccAddress
		daoId    uint64
	}{{memberA, 1}, {memberB, 2}, {memberB, 1}, {memberA, 2}} {
		id, err := suite.submit(s.proposer, s.daoId, false)
		suite.Require().NoError(err)
		ids = append(ids, id)
	}
	suite.Require().NoError(suite.cancel(memberA, ids[0]))
	suite.Require().NoError(suite.cancel(memberB, ids[1]))

	open := sdk.NewCoins()
	for _, id := range ids {
		proposal, _ := suite.k.GetProposal(suite.ctx, id)
		if proposal.Status == types.ProposalStatus_VOTING {
			open = open.Add(*proposal.PinnedDeposit)
		}
	}
	suite.Require().Equal(ubze(40+15), open)
	suite.Require().Equal(open, escrow)
}

var cleanerAddr = sdk.AccAddress("dao_cleaner_________")

func (suite *IntegrationTestSuite) cleanup(proposalId, limit uint64) (*types.MsgCleanupProposalResponse, error) {
	return suite.msgServer.CleanupProposal(suite.ctx, types.NewMsgCleanupProposal(cleanerAddr.String(), proposalId, limit))
}

// votedStakeProposal creates stake DAO 1 over SR 1 (and the ubze DR when
// withDr), has voters addresses stake 1 in each source and vote yes on one
// proposal, and settles it EXECUTED.
func (suite *IntegrationTestSuite) votedStakeProposal(voters int, withDr bool) uint64 {
	state := defaultRewardsState()
	state.srs["1"] = stakingReward("1", stakeDenom, 5, strconv.Itoa(voters))
	state.drs[stakeDenom] = denomReward(stakeDenom, 5, int64(voters))
	suite.stubRewards(state)
	s := suite.stubStakes()
	suite.expectCreation(1)
	drDenom := ""
	if withDr {
		drDenom = stakeDenom
	}
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, drDenom))
	suite.Require().NoError(err)
	for i := 1; i <= voters; i++ {
		s.joinSr("1", memberAddress(i), 1)
		if withDr {
			s.joinDr(stakeDenom, memberAddress(i), 1)
		}
	}

	suite.atBlock(10)
	id, err := suite.submit(memberAddress(1), 1, false)
	suite.Require().NoError(err)
	for i := 1; i <= voters; i++ {
		suite.Require().NoError(suite.vote(memberAddress(i), id, types.VoteOption_YES))
	}

	suite.endVoting(id, 0)
	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, status)

	return id
}

// requireCleanupBatch runs one cleanup and checks its response and event.
func (suite *IntegrationTestSuite) requireCleanupBatch(proposalId, limit, processed uint64, completed bool) {
	suite.ctx = suite.ctx.WithEventManager(sdk.NewEventManager())
	res, err := suite.cleanup(proposalId, limit)
	suite.Require().NoError(err)
	suite.Require().Equal(processed, res.Processed)
	suite.Require().Equal(completed, res.Completed)

	attrs := suite.lastEventAttrs(&types.EventProposalCleaned{})
	suite.Require().Equal(`"`+strconv.FormatUint(proposalId, 10)+`"`, attrs["id"])
	suite.Require().Equal(`"`+strconv.FormatUint(processed, 10)+`"`, attrs["processed"])
	suite.Require().Equal(strconv.FormatBool(completed), attrs["completed"])
}

// requireBallots checks which of voters 1..total still have a ballot: exactly
// those above deleted.
func (suite *IntegrationTestSuite) requireBallots(proposalId uint64, deleted, total int) {
	for i := 1; i <= total; i++ {
		_, found := suite.k.GetBallot(suite.ctx, proposalId, memberAddress(i))
		suite.Require().Equal(i > deleted, found, "voter %d", i)
	}
}

// requireCleanedUp checks that the proposal, its DAO index entry and every
// ballot of voters 1..total are gone.
func (suite *IntegrationTestSuite) requireCleanedUp(daoId, proposalId uint64, total int) {
	_, found := suite.k.GetProposal(suite.ctx, proposalId)
	suite.Require().False(found)
	suite.Require().False(suite.k.HasDaoProposalIndex(suite.ctx, daoId, proposalId))
	suite.requireBallots(proposalId, total, total)
}

// D19 (cleanup part): 250 ballots with a batch of 100 take three calls, 100,
// 100 and 50, the last one completing; a user limit above the param is capped.
// The cursor only moves forward: each call deletes the next range in key order
// and leaves the rest.
func (suite *IntegrationTestSuite) TestCleanupProposal_D19_BatchesOf250() {
	id := suite.votedStakeProposal(250, false)
	suite.Require().Equal(types.DefaultCleanupBatchSize, suite.k.GetParams(suite.ctx).CleanupBatchSize)

	suite.requireCleanupBatch(id, 1_000, 100, false)
	suite.requireBallots(id, 100, 250)
	proposal, found := suite.k.GetProposal(suite.ctx, id)
	suite.Require().True(found)
	suite.Require().Equal(memberAddress(100).String(), proposal.CleanupCursor)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, proposal.Status)
	suite.Require().True(suite.k.HasDaoProposalIndex(suite.ctx, 1, id))

	suite.requireCleanupBatch(id, 0, 100, false)
	suite.requireBallots(id, 200, 250)
	proposal, _ = suite.k.GetProposal(suite.ctx, id)
	suite.Require().Equal(memberAddress(200).String(), proposal.CleanupCursor)

	suite.requireCleanupBatch(id, 0, 50, true)
	suite.requireCleanedUp(1, id, 250)
}

// A limit below the param is used as is; a batch that ends exactly on the
// last ballot completes in the same call.
func (suite *IntegrationTestSuite) TestCleanupProposal_UserLimitAndExactEnd() {
	id := suite.votedStakeProposal(10, false)

	suite.requireCleanupBatch(id, 3, 3, false)
	suite.requireBallots(id, 3, 10)
	suite.requireCleanupBatch(id, 7, 7, true)
	suite.requireCleanedUp(1, id, 10)
}

// The batch is the param at the time of the call.
func (suite *IntegrationTestSuite) TestCleanupProposal_ReadsBatchParam() {
	id := suite.votedStakeProposal(10, false)
	params := suite.k.GetParams(suite.ctx)
	params.CleanupBatchSize = 4
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	suite.requireCleanupBatch(id, 0, 4, false)
	suite.requireCleanupBatch(id, 100, 4, false)
	suite.requireCleanupBatch(id, 0, 2, true)
	suite.requireCleanedUp(1, id, 10)
}

// Every terminal status can be cleaned up, also with no ballots at all; the
// DAO's other proposals keep their records.
func (suite *IntegrationTestSuite) TestCleanupProposal_TerminalStatuses() {
	for _, tc := range []struct {
		status types.ProposalStatus
		finish func(id uint64)
	}{
		{types.ProposalStatus_EXECUTED, func(id uint64) {
			suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
			suite.endVoting(id, 0)
			_, err := suite.settle(id)
			suite.Require().NoError(err)
		}},
		// nobody voted: rejected with no ballot to delete
		{types.ProposalStatus_REJECTED, func(id uint64) {
			suite.endVoting(id, 0)
			_, err := suite.settle(id)
			suite.Require().NoError(err)
		}},
		{types.ProposalStatus_EXPIRED, func(id uint64) {
			suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
			suite.Require().NoError(suite.vote(memberB, id, types.VoteOption_YES))
			params := suite.k.GetParams(suite.ctx)
			params.ExecutionWindow = 0
			suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
			suite.endVoting(id, 0)
			_, err := suite.settle(id)
			suite.Require().NoError(err)
			_, err = suite.settle(id)
			suite.Require().NoError(err)
		}},
		{types.ProposalStatus_CANCELLED, func(id uint64) {
			suite.Require().NoError(suite.vote(memberB, id, types.VoteOption_YES))
			suite.Require().NoError(suite.cancel(memberA, id))
		}},
	} {
		suite.Run(tc.status.String(), func() {
			suite.SetupTest()
			suite.createStaticDao(noConfigChange)
			other, err := suite.submit(memberA, 1, false)
			suite.Require().NoError(err)
			var msgs []sdk.Msg
			if tc.status == types.ProposalStatus_EXPIRED {
				msgs = append(msgs, treasurySend(memberB, 30))
			}
			id, err := suite.submit(memberA, 1, false, msgs...)
			suite.Require().NoError(err)
			tc.finish(id)
			proposal, _ := suite.k.GetProposal(suite.ctx, id)
			suite.Require().Equal(tc.status, proposal.Status)

			res, err := suite.cleanup(id, 0)
			suite.Require().NoError(err)
			suite.Require().True(res.Completed)
			_, found := suite.k.GetProposal(suite.ctx, id)
			suite.Require().False(found)
			suite.Require().False(suite.k.HasDaoProposalIndex(suite.ctx, 1, id))
			for _, voter := range []sdk.AccAddress{memberA, memberB} {
				_, found := suite.k.GetBallot(suite.ctx, id, voter)
				suite.Require().False(found)
			}

			_, found = suite.k.GetProposal(suite.ctx, other)
			suite.Require().True(found)
			suite.Require().True(suite.k.HasDaoProposalIndex(suite.ctx, 1, other))
			suite.Require().True(suite.k.IsOpenProposal(suite.ctx, 1, other))
		})
	}
}

// D20: a proposal in voting, also after its voting end, and a retryable
// EXECUTION_FAILED one are refused and left untouched; a cleanup after
// completion finds nothing and changes nothing.
func (suite *IntegrationTestSuite) TestCleanupProposal_D20_Refusals() {
	suite.createStaticDao(noConfigChange)
	id := suite.passedProposal(treasurySend(memberB, 30))

	requireRefused := func(want error, status types.ProposalStatus) {
		suite.ctx = suite.ctx.WithEventManager(sdk.NewEventManager())
		_, err := suite.cleanup(id, 0)
		suite.Require().ErrorIs(err, want)
		proposal, found := suite.k.GetProposal(suite.ctx, id)
		suite.Require().True(found)
		suite.Require().Equal(status, proposal.Status)
		suite.Require().Empty(proposal.CleanupCursor)
		_, found = suite.k.GetBallot(suite.ctx, id, memberA)
		suite.Require().True(found)
		suite.Require().Nil(suite.lastEventAttrs(&types.EventProposalCleaned{}))
	}

	requireRefused(types.ErrProposalNotFinished, types.ProposalStatus_VOTING)
	suite.endVoting(id, 0)
	requireRefused(types.ErrProposalNotFinished, types.ProposalStatus_VOTING)

	_, err := suite.settle(id)
	suite.Require().NoError(err)
	requireRefused(types.ErrProposalNotFinished, types.ProposalStatus_EXECUTION_FAILED)

	suite.ledger.fund(daoAddress(1), ubze(30))
	_, err = suite.settle(id)
	suite.Require().NoError(err)
	res, err := suite.cleanup(id, 0)
	suite.Require().NoError(err)
	suite.Require().True(res.Completed)
	suite.Require().Equal(uint64(2), res.Processed)

	suite.ctx = suite.ctx.WithEventManager(sdk.NewEventManager())
	_, err = suite.cleanup(id, 0)
	suite.Require().ErrorIs(err, types.ErrProposalNotFound)
	suite.Require().Nil(suite.lastEventAttrs(&types.EventProposalCleaned{}))
	suite.Require().False(suite.k.HasDaoProposalIndex(suite.ctx, 1, id))

	_, err = suite.cleanup(99, 0)
	suite.Require().ErrorIs(err, types.ErrProposalNotFound)
	_, err = suite.msgServer.CleanupProposal(suite.ctx, types.NewMsgCleanupProposal("bze1nope", id, 0))
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidAddress)
}

// requireBallot checks the voter's stored ballot on the proposal.
func (suite *IntegrationTestSuite) requireBallot(proposalId uint64, voter sdk.AccAddress, option types.VoteOption, weight int64) {
	ballot, found := suite.k.GetBallot(suite.ctx, proposalId, voter)
	suite.Require().True(found, "ballot of %s", voter)
	suite.Require().Equal(option, ballot.Option)
	suite.Require().Equal(math.NewInt(weight), ballot.Weight)
}

// D21: removing a member zeroes their ballots on the DAO's proposals in
// voting, keeping the records; a closed proposal keeps its ballot. Re-added,
// the member's old ballots stay 0: they vote only on proposals submitted after
// the re-add, as stake that rejoins does. I2 holds throughout.
func (suite *IntegrationTestSuite) TestUpdateMembers_D21_RemovedMemberStopsCounting() {
	suite.createAdminDao()
	closed, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberB, closed, types.VoteOption_YES))
	suite.Require().NoError(suite.cancel(memberA, closed))

	first, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	second, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, first, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberB, first, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberB, second, types.VoteOption_NO))
	suite.requireTally(first, 2, 0, 0)
	suite.requireTally(second, 0, 1, 0)

	remove := types.NewMsgUpdateMembers(daoAddress(1).String(), 1, nil, []string{memberB.String()})
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, remove))

	suite.requireMembers(memberA)
	suite.requireTally(first, 1, 0, 0)
	suite.requireTally(second, 0, 0, 0)
	suite.requireBallot(first, memberB, types.VoteOption_YES, 0)
	suite.requireBallot(second, memberB, types.VoteOption_NO, 0)
	suite.requireBallot(closed, memberB, types.VoteOption_YES, 1)
	suite.requireInvariants(first, memberA, memberB)
	suite.requireInvariants(second, memberA, memberB)
	attrs := suite.lastEventAttrs(&types.EventBallotReduced{})
	suite.Require().Equal(`"`+strconv.FormatUint(second, 10)+`"`, attrs["proposal_id"])
	suite.Require().Equal(`"`+memberB.String()+`"`, attrs["voter"])
	suite.Require().Equal(`""`, attrs["source"])
	suite.Require().Equal(`"1"`, attrs["amount"])

	suite.Require().ErrorIs(suite.vote(memberB, first, types.VoteOption_NO), types.ErrNoVotingPower)

	add := types.NewMsgUpdateMembers(daoAddress(1).String(), 1, []string{memberB.String()}, nil)
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, add))

	suite.requireMembers(memberA, memberB)
	suite.requireBallot(first, memberB, types.VoteOption_YES, 0)
	suite.requireTally(first, 1, 0, 0)

	suite.Require().ErrorIs(suite.vote(memberB, first, types.VoteOption_YES), types.ErrNoVotingPower)
	suite.Require().ErrorIs(suite.vote(memberB, second, types.VoteOption_ABSTAIN), types.ErrNoVotingPower)
	suite.requireBallot(first, memberB, types.VoteOption_YES, 0)
	suite.requireTally(first, 1, 0, 0)
	suite.requireTally(second, 0, 0, 0)
	suite.requireInvariants(first, memberA, memberB)
	suite.requireInvariants(second, memberA, memberB)

	suite.atBlock(suite.ctx.BlockHeight() + 1)
	third, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberB, third, types.VoteOption_YES))
	suite.requireBallot(third, memberB, types.VoteOption_YES, 1)
}

// A removal zeroes the member's ballots only on its own DAO's proposals: the
// same address keeps its full ballot in another static DAO it belongs to.
func (suite *IntegrationTestSuite) TestUpdateMembers_OtherDaoBallotsUntouched() {
	suite.createAdminDao()
	suite.createDao(2, func(msg *types.MsgCreateDao) { msg.Admin = adminAddr.String() })

	first, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	other, err := suite.submit(memberA, 2, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, first, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberB, first, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberA, other, types.VoteOption_NO))
	suite.Require().NoError(suite.vote(memberB, other, types.VoteOption_YES))

	remove := types.NewMsgUpdateMembers(daoAddress(1).String(), 1, nil, []string{memberB.String()})
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, remove))

	suite.requireTally(first, 1, 0, 0)
	suite.requireBallot(first, memberB, types.VoteOption_YES, 0)
	suite.requireTally(other, 1, 1, 0)
	suite.requireBallot(other, memberB, types.VoteOption_YES, 1)
	suite.requireInvariants(first, memberA, memberB)
	suite.requireInvariants(other, memberA, memberB)
}

// An added member gets a member record at the block height and votes only on
// proposals submitted after that block, the rule stake follows through join
// records: not on a proposal opened before the add, nor on one submitted in
// the same block. The pinned denominators count the members at submission.
func (suite *IntegrationTestSuite) TestUpdateMembers_AddedMemberVotesOnLaterProposals() {
	suite.createAdminDao()
	suite.atBlock(10)
	before, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	suite.atBlock(11)
	add := types.NewMsgUpdateMembers(daoAddress(1).String(), 1, []string{memberC.String()}, nil)
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, add))
	record, found := suite.k.GetMemberRecord(suite.ctx, 1, memberC)
	suite.Require().True(found)
	suite.Require().Equal(int64(11), record.Height)
	sameBlock, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)

	suite.Require().ErrorIs(suite.vote(memberC, before, types.VoteOption_YES), types.ErrNoVotingPower)
	suite.Require().ErrorIs(suite.vote(memberC, sameBlock, types.VoteOption_YES), types.ErrNoVotingPower)
	suite.requireTally(before, 0, 0, 0)
	suite.requireTally(sameBlock, 0, 0, 0)

	suite.atBlock(12)
	after, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberC, after, types.VoteOption_YES))
	suite.requireBallot(after, memberC, types.VoteOption_YES, 1)
	suite.requireInvariants(after, memberA, memberB, memberC)

	for id, want := range map[uint64]int64{before: 2, sameBlock: 3, after: 3} {
		proposal, _ := suite.k.GetProposal(suite.ctx, id)
		suite.Require().Equal(math.NewInt(want), proposal.PinnedDenominator)
	}

	// the voting power query applies the same rule
	res, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{DaoId: 1, Address: memberC.String()})
	suite.Require().NoError(err)
	suite.Require().Equal(math.OneInt(), res.Current)
	for _, p := range res.Proposals {
		want := math.ZeroInt()
		if p.ProposalId == after {
			want = math.OneInt()
		}
		suite.Require().Equal(want, p.EligibleWeight, "proposal %d", p.ProposalId)
	}
}

// Members listed at creation have no member record; a removal deletes the
// record of a member added later.
func (suite *IntegrationTestSuite) TestUpdateMembers_MemberRecordLifecycle() {
	suite.createAdminDao()
	_, found := suite.k.GetMemberRecord(suite.ctx, 1, memberA)
	suite.Require().False(found)

	suite.atBlock(5)
	add := types.NewMsgUpdateMembers(daoAddress(1).String(), 1, []string{memberC.String()}, nil)
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, add))
	_, found = suite.k.GetMemberRecord(suite.ctx, 1, memberC)
	suite.Require().True(found)
	suite.Require().Len(suite.k.GetAllMemberRecords(suite.ctx), 1)

	remove := types.NewMsgUpdateMembers(daoAddress(1).String(), 1, nil, []string{memberC.String()})
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, remove))
	_, found = suite.k.GetMemberRecord(suite.ctx, 1, memberC)
	suite.Require().False(found)
	suite.Require().Empty(suite.k.GetAllMemberRecords(suite.ctx))
}

// A DAO cannot list its own address as a member: not at creation, nor
// through MsgUpdateMembers. Another DAO stays a valid member.
func (suite *IntegrationTestSuite) TestUpdateMembers_DaoCannotBeItsOwnMember() {
	suite.createAdminDao()
	add := types.NewMsgUpdateMembers(daoAddress(1).String(), 1, []string{daoAddress(1).String()}, nil)
	suite.Require().ErrorIs(suite.adminExecute(adminAddr, 1, add), types.ErrInvalidMembers)
	suite.requireMembers(memberA, memberB)

	add = types.NewMsgUpdateMembers(daoAddress(1).String(), 1, []string{daoAddress(2).String()}, nil)
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, add))
	suite.requireMembers(memberA, memberB, daoAddress(2))
}

// A self-owned DAO changes its members through a passed proposal. The
// proposal left voting before its messages ran, so its own ballot of the
// removed member stays.
func (suite *IntegrationTestSuite) TestUpdateMembers_ThroughProposal() {
	suite.createStaticDao(noConfigChange)
	id := suite.passedProposal(types.NewMsgUpdateMembers(daoAddress(1).String(), 1, []string{memberC.String()}, []string{memberB.String()}))
	suite.endVoting(id, 0)

	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, status)
	suite.requireExecuted("")
	suite.requireMembers(memberA, memberC)
	suite.requireBallot(id, memberB, types.VoteOption_YES, 1)
	suite.requireTally(id, 2, 0, 0)
}

// D11 through a proposal: removing every member ends EXECUTION_FAILED and
// changes nothing.
func (suite *IntegrationTestSuite) TestUpdateMembers_D11_ThroughProposal() {
	suite.createStaticDao(noConfigChange)
	id := suite.passedProposal(types.NewMsgUpdateMembers(daoAddress(1).String(), 1, nil, []string{memberA.String(), memberB.String()}))
	suite.endVoting(id, 0)

	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)
	suite.requireExecuted("message failed")
	suite.requireMembers(memberA, memberB)
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
