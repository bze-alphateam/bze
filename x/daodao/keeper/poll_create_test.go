package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// pollOpts bundles the optional MsgCreatePoll fields tests vary across.
type pollOpts struct {
	choices       []string
	maxSelections uint32
	quorumBps     uint32
	includeNota   bool
	deposit       sdk.Coin // default sdk.NewInt64Coin("ubze", 0)
}

func defaultPollOpts() pollOpts {
	return pollOpts{
		choices:       []string{"alpha", "beta", "gamma"},
		maxSelections: 1,
		quorumBps:     0,
		includeNota:   false,
		deposit:       sdk.NewInt64Coin("ubze", 0),
	}
}

// createPollMember runs MsgCreatePoll from the supplied member with
// pollOpts. If opts.deposit > 0 the helper also installs the
// proposer-→-escrow bank send mock.
func (suite *IntegrationTestSuite) createPollMember(daoID uint64, member string, opts pollOpts) uint64 {
	if !opts.deposit.IsZero() {
		suite.expectInitialDepositSend(daoID, member, opts.deposit)
	}
	resp, err := suite.msgServer.CreatePoll(suite.ctx, &types.MsgCreatePoll{
		Proposer:       member,
		DaoId:          daoID,
		Title:          "test poll",
		Choices:        opts.choices,
		MaxSelections:  opts.maxSelections,
		QuorumBps:      opts.quorumBps,
		IncludeNota:    opts.includeNota,
		InitialDeposit: opts.deposit,
	})
	suite.Require().NoError(err)
	return resp.PollId
}

// TestPollCreate_MemberStartsInDepositPeriod: a member with zero
// initial deposit lands the poll in DEPOSIT_PERIOD.
func (suite *IntegrationTestSuite) TestPollCreate_MemberStartsInDepositPeriod() {
	daoID, member := suite.createSampleDao("poll-deposit-period")
	pid := suite.createPollMember(daoID, member, defaultPollOpts())

	p, ok := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().True(ok)
	suite.Require().Equal(types.PollStatus_POLL_STATUS_DEPOSIT_PERIOD, p.Status)
	suite.Require().Equal("0ubze", p.DepositCollected.String())
}

// TestPollCreate_MemberFullDepositStartsInVoting: a member with full
// initial deposit skips DEPOSIT_PERIOD.
func (suite *IntegrationTestSuite) TestPollCreate_MemberFullDepositStartsInVoting() {
	daoID, member := suite.createSampleDao("poll-voting")
	opts := defaultPollOpts()
	opts.deposit = sdk.NewInt64Coin("ubze", 1) // matches validDeposit().MinDeposit
	pid := suite.createPollMember(daoID, member, opts)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(types.PollStatus_POLL_STATUS_VOTING, p.Status)
}

// TestPollCreate_NotaAppended: include_nota=true causes the keeper to
// append "None of the above" as the final choice.
func (suite *IntegrationTestSuite) TestPollCreate_NotaAppended() {
	daoID, member := suite.createSampleDao("poll-nota")
	opts := defaultPollOpts()
	opts.includeNota = true
	pid := suite.createPollMember(daoID, member, opts)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(4, len(p.Choices), "user 3 + NOTA = 4")
	suite.Require().Equal(types.NotaLabel, p.Choices[len(p.Choices)-1])
	suite.Require().True(p.IncludeNota)
}

// TestPollCreate_NotaNotAppended: include_nota=false leaves the choice
// list untouched.
func (suite *IntegrationTestSuite) TestPollCreate_NotaNotAppended() {
	daoID, member := suite.createSampleDao("poll-no-nota")
	opts := defaultPollOpts()
	opts.includeNota = false
	pid := suite.createPollMember(daoID, member, opts)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Equal(3, len(p.Choices))
	suite.Require().False(p.IncludeNota)
}

// TestPollCreate_MaxSelectionsOverUserChoices: max_selections exceeding
// len(choices) is rejected by ValidateBasic.
func (suite *IntegrationTestSuite) TestPollCreate_MaxSelectionsOverUserChoices() {
	daoID, member := suite.createSampleDao("poll-max-bad")

	_, err := suite.msgServer.CreatePoll(suite.ctx, &types.MsgCreatePoll{
		Proposer:       member,
		DaoId:          daoID,
		Title:          "bad-max",
		Choices:        []string{"a", "b"},
		MaxSelections:  5,
		InitialDeposit: sdk.NewInt64Coin("ubze", 0),
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "max_selections")
}

// TestPollCreate_TooFewChoices: a single-choice poll is rejected.
func (suite *IntegrationTestSuite) TestPollCreate_TooFewChoices() {
	daoID, member := suite.createSampleDao("poll-too-few")

	_, err := suite.msgServer.CreatePoll(suite.ctx, &types.MsgCreatePoll{
		Proposer:       member,
		DaoId:          daoID,
		Title:          "too-few",
		Choices:        []string{"only-one"},
		MaxSelections:  1,
		InitialDeposit: sdk.NewInt64Coin("ubze", 0),
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "choices count")
}

// TestPollCreate_DuplicateChoiceLabel: duplicates rejected.
func (suite *IntegrationTestSuite) TestPollCreate_DuplicateChoiceLabel() {
	daoID, member := suite.createSampleDao("poll-dup")

	_, err := suite.msgServer.CreatePoll(suite.ctx, &types.MsgCreatePoll{
		Proposer:       member,
		DaoId:          daoID,
		Title:          "dup",
		Choices:        []string{"alpha", "alpha"},
		MaxSelections:  1,
		InitialDeposit: sdk.NewInt64Coin("ubze", 0),
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "duplicate")
}

// TestPollCreate_ReservedNotaLabel: a user label equal to NotaLabel is
// rejected so the keeper-appended NOTA stays unambiguous.
func (suite *IntegrationTestSuite) TestPollCreate_ReservedNotaLabel() {
	daoID, member := suite.createSampleDao("poll-nota-reserved")

	_, err := suite.msgServer.CreatePoll(suite.ctx, &types.MsgCreatePoll{
		Proposer:       member,
		DaoId:          daoID,
		Title:          "reserved",
		Choices:        []string{"alpha", types.NotaLabel},
		MaxSelections:  1,
		InitialDeposit: sdk.NewInt64Coin("ubze", 0),
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "reserved")
}

// TestPollCreate_NonMemberWithoutFullDepositRejected: same submission
// gating as proposals — non-member must attach >= min_deposit.
func (suite *IntegrationTestSuite) TestPollCreate_NonMemberWithoutFullDepositRejected() {
	daoID, _ := suite.createSampleDao("poll-nonmember-reject")
	outsider := freshAddr()

	_, err := suite.msgServer.CreatePoll(suite.ctx, &types.MsgCreatePoll{
		Proposer:       outsider,
		DaoId:          daoID,
		Title:          "outsider",
		Choices:        []string{"a", "b"},
		MaxSelections:  1,
		InitialDeposit: sdk.NewInt64Coin("ubze", 0),
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "non-member")
}

// TestPollCreate_TallyInitialized: tally.choice_power is zero-init at
// the right length; total_power matches snapshot.
func (suite *IntegrationTestSuite) TestPollCreate_TallyInitialized() {
	daoID, member := suite.createSampleDao("poll-tally-init")
	opts := defaultPollOpts()
	opts.includeNota = true
	opts.deposit = sdk.NewInt64Coin("ubze", 1) // straight to VOTING
	pid := suite.createPollMember(daoID, member, opts)

	p, _ := suite.k.GetPoll(suite.ctx, daoID, pid)
	suite.Require().Len(p.Tally.ChoicePower, 4, "3 user + NOTA")
	for i, v := range p.Tally.ChoicePower {
		suite.Require().Equal(uint64(0), v, "choice_power[%d] starts at 0", i)
	}
	// validGovernance.staticConfig is creator-only at weight 1.
	suite.Require().Equal(uint64(1), p.Tally.TotalPower)
}
