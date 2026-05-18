package keeper_test

import (
	"errors"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Test-only sentinel errors used as gomock return values when we want to
// simulate a transient dependency failure (bank or distribution rejecting
// a send/fund call).
var (
	errSimulatedBankFailure  = errors.New("simulated bank failure")
	errSimulatedDistrFailure = errors.New("simulated distr failure")
)

// ---------- Address / metadata helpers ----------

// sampleMetadata returns a metadata block that passes ValidateBasic.
func sampleMetadata(name string) types.DaoMetadata {
	return types.DaoMetadata{
		Name:        name,
		Description: name + " description",
		ImageUrl:    "ipfs://example",
		Twitter:     "@" + name,
		Discord:     "discord-" + name,
		Telegram:    "tg-" + name,
		Website:     "https://" + name + ".example",
		Other:       "linktree/" + name,
	}
}

// freshAddr returns a freshly-generated bech32 address string. Uses
// testutil/sample, the same helper every other module's tests use.
func freshAddr() string {
	return sample.AccAddress()
}

// staticConfig produces the MsgCreateDao voting_config oneof wrapper for a
// STATIC DAO with a single member at weight 1. The vast majority of tests
// just want "a creator who is also the sole voting member."
func staticConfig(addr string) *types.MsgCreateDao_Static {
	return staticConfigWithMembers([]types.StaticMember{{Address: addr, Weight: 1}})
}

// staticConfigWithMembers is the explicit variant: the caller controls the
// member list. Useful for multi-member or custom-weight tests.
func staticConfigWithMembers(members []types.StaticMember) *types.MsgCreateDao_Static {
	return &types.MsgCreateDao_Static{
		Static: &types.StaticVotingConfig{Members: members},
	}
}

// validGovernance returns a permissive GovernanceConfig that passes
// ValidateGovernanceConfigStateless and the Param-driven upper bound at
// default Params:
//   - WITHOUT_QUORUM, 50% threshold, 24h voting period, revote allowed.
//
// Used as the default for every keeper test that creates a DAO but does
// not specifically exercise governance config behavior.
func validGovernance() types.GovernanceConfig {
	return types.GovernanceConfig{
		ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
		ThresholdBps: 5_000,
		QuorumBps:    0,
		VotingPeriod: 24 * time.Hour,
		AllowRevote:  true,
	}
}

// validDeposit returns a permissive DepositConfig that passes
// ValidateDepositConfigStateless and the Param-driven upper bound at
// default Params:
//   - min_deposit = 1ubze, deposit_period = 7d, forfeit TREASURY,
//     refund ON_PASS.
//
// Default for every keeper test that creates a DAO but does not
// specifically exercise deposit config behavior. Use freeDeposit() for
// tests that want the deposit phase to be trivially satisfied (min=1 so
// any non-zero initial_deposit auto-promotes to VOTING).
func validDeposit() types.DepositConfig {
	return types.DepositConfig{
		MinDeposit:         sdk.NewInt64Coin("ubze", 1),
		DepositPeriod:      7 * 24 * time.Hour,
		ForfeitDestination: types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
		VotingRefundPolicy: types.RefundPolicy_REFUND_POLICY_ON_PASS,
	}
}

// zeroUbze returns a zero-amount ubze Coin. Member proposers attach this
// to MsgCreateProposal so the proposal starts in DEPOSIT_PERIOD; the
// denom must still match the DAO's min_deposit.denom even at amount 0.
func zeroUbze() sdk.Coin {
	return sdk.NewInt64Coin("ubze", 0)
}

// withQuorumGovernance returns a WITH_QUORUM config with the given bps
// values. Useful for tally-edge tests.
func withQuorumGovernance(thresholdBps, quorumBps uint32, votingPeriod time.Duration, allowRevote bool) types.GovernanceConfig {
	return types.GovernanceConfig{
		ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITH_QUORUM,
		ThresholdBps: thresholdBps,
		QuorumBps:    quorumBps,
		VotingPeriod: votingPeriod,
		AllowRevote:  allowRevote,
	}
}

// mustAcc parses bech32 and panics on failure. Test-only.
func (suite *IntegrationTestSuite) mustAcc(b32 string) sdk.AccAddress {
	a, err := sdk.AccAddressFromBech32(b32)
	suite.Require().NoError(err)
	return a
}

// ---------- Mock expectation helpers ----------

// expectAccountCreated sets up the standard account-keeper call sequence
// that MsgCreateDao performs to register a DAO's BaseAccount.
func (suite *IntegrationTestSuite) expectAccountCreated(daoID uint64) {
	daoAddr := types.DaoAccountAddress(daoID)
	suite.acc.EXPECT().HasAccount(gomock.Any(), daoAddr).Return(false).Times(1)
	suite.acc.EXPECT().NewAccountWithAddress(gomock.Any(), daoAddr).
		Return(authtypes.NewBaseAccountWithAddress(daoAddr)).Times(1)
	suite.acc.EXPECT().SetAccount(gomock.Any(), gomock.Any()).Times(1)
}

// createSampleDao runs MsgCreateDao with mocked account creation, returning
// the resulting (daoID, creator-address). The creator is also the admin
// and the single STATIC voting member at weight 1. Governance defaults to
// validGovernance(): WITHOUT_QUORUM, 50% threshold, 24h voting period,
// revote allowed.
func (suite *IntegrationTestSuite) createSampleDao(name string) (daoID uint64, creator string) {
	creator = freshAddr()
	suite.expectAccountCreated(suite.k.GetDaoIDCounter(suite.ctx))
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata(name),
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)
	return resp.DaoId, creator
}

// createDaoWithMembers builds a STATIC DAO with the supplied member list
// and governance config. Returns the DAO id and the creator address (used
// as admin). The creator is included as a member by construction.
//
// Epic 3 ergonomics: most tally / vote tests need explicit weights and a
// specific governance config (revote on/off, quorum yes/no). This helper
// keeps those call sites tight.
func (suite *IntegrationTestSuite) createDaoWithMembers(
	name string,
	members []types.StaticMember,
	gov types.GovernanceConfig,
) (daoID uint64, creator string) {
	creator = members[0].Address
	suite.expectAccountCreated(suite.k.GetDaoIDCounter(suite.ctx))
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata(name),
		VotingConfig: staticConfigWithMembers(members),
		Governance:   gov,
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)
	return resp.DaoId, creator
}

// advanceTime moves the suite's block-time forward by `d`. Used by tests
// that need to drive end-blocker behavior past a proposal's voting_end.
// Block height is bumped by one alongside the time advance to keep
// CreatedHeight diffs visible if any test inspects them.
func (suite *IntegrationTestSuite) advanceTime(d time.Duration) {
	suite.ctx = suite.ctx.
		WithBlockTime(suite.ctx.BlockTime().Add(d)).
		WithBlockHeight(suite.ctx.BlockHeight() + 1)
}

// expectInitialDepositSend installs the bank.SendCoins mock for the
// proposer-→-escrow transfer at MsgCreateProposal time. Use this when a
// test creates a proposal with a non-zero initial_deposit.
func (suite *IntegrationTestSuite) expectInitialDepositSend(daoID uint64, proposer string, amount sdk.Coin) {
	escrow := types.DepositEscrowAddress(daoID)
	proposerAddr := suite.mustAcc(proposer)
	suite.bank.EXPECT().
		SendCoins(gomock.Any(), proposerAddr, escrow, sdk.NewCoins(amount)).
		Return(nil).
		Times(1)
}

// expectRefundOnTerminal installs the bank.SendCoins mock for the
// escrow-→-depositor refund that handleTerminalDeposits issues when the
// proposal's refund policy is ALWAYS, or ON_PASS with outcome PASSED.
func (suite *IntegrationTestSuite) expectRefundOnTerminal(daoID uint64, depositor string, amount sdk.Coin) {
	escrow := types.DepositEscrowAddress(daoID)
	depositorAddr := suite.mustAcc(depositor)
	suite.bank.EXPECT().
		SendCoins(gomock.Any(), escrow, depositorAddr, sdk.NewCoins(amount)).
		Return(nil).
		Times(1)
}

// expectTreasuryForfeit installs the bank.SendCoins mock for the
// escrow-→-dao-treasury batch forfeit. `total` should equal the sum of
// all deposit records on the proposal.
func (suite *IntegrationTestSuite) expectTreasuryForfeit(daoID uint64, total sdk.Coin) {
	escrow := types.DepositEscrowAddress(daoID)
	dao, ok := suite.k.GetDao(suite.ctx, daoID)
	suite.Require().True(ok)
	treasury := suite.mustAcc(dao.AccountAddress)
	suite.bank.EXPECT().
		SendCoins(gomock.Any(), escrow, treasury, sdk.NewCoins(total)).
		Return(nil).
		Times(1)
}

// createTestProposal is the shared "create a proposal as a member, skip
// the deposit phase" helper used across Epic-3 and Epic-4 tests. The
// proposer attaches an initial deposit equal to the DAO's min_deposit
// (which validDeposit() sets to 1ubze) so the proposal lands directly in
// VOTING. The helper installs the proposer-→-escrow bank send mock
// itself; callers reaching terminal status should additionally call
// expectRefundOnTerminal or expectTreasuryForfeit before the
// terminating action.
func (suite *IntegrationTestSuite) createTestProposal(daoID uint64, proposer string) uint64 {
	deposit := sdk.NewInt64Coin("ubze", 1)
	suite.expectInitialDepositSend(daoID, proposer, deposit)
	resp, err := suite.msgServer.CreateProposal(suite.ctx, &types.MsgCreateProposal{
		Proposer:       proposer,
		DaoId:          daoID,
		Title:          "test proposal",
		Description:    "lifecycle test",
		InitialDeposit: deposit,
	})
	suite.Require().NoError(err)
	return resp.ProposalId
}
