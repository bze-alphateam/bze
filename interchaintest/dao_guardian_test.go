package interchaintest_test

import (
	"context"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

const (
	// a min_expedite_delay longer than the time from a council proposal's
	// submission to its early settle (a few votes, under a minute), which
	// each early settle asserts: it is early only because a static-list DAO
	// does not wait for it
	gcdDelay     = "200s"
	gcdDelaySecs = 200

	gcdSend     = 100 // ubze the council sends from its treasury
	gcdSendURL  = "/cosmos.bank.v1beta1.MsgSend"
	gcdMultiURL = "/cosmos.bank.v1beta1.MsgMultiSend" // the control of step 5
	gcdVoteURL  = "/bze.dao.MsgVote"
	gcdCircuit  = "circuit breaker disables execution of this message: " + gcdSendURL
	gcdExecFail = "message failed" // the fixed EventProposalExecuted reason of a failed handler
)

// TestGuardianCouncilDao runs the guardian as a static-list DAO through the real
// binary. A governance MsgUpdateParams of x/guardian makes a council of five
// the guardian, and the wallet guardian set in genesis is refused from then
// on. The council disables MsgSend with an expedited proposal that settles as
// soon as three of five voted yes. A second council proposal carrying a
// treasury MsgSend then settles EXECUTION_FAILED on the circuit breaker, with
// the treasury untouched, while the council still submits, votes and settles
// (those dao messages are protected and cannot be disabled, not even by the
// council itself). Governance resets the breaker, and a retry of the failed
// proposal sends the coins: the breaker was the only reason it failed. The
// state carries over between the subtests, which run in order.
func TestGuardianCouncilDao(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()

	walletMnemonic, walletAddr := newAccount(t, chain.Bech32Prefix)
	genesis := append(daoGenesis(),
		cosmos.NewGenesisKV("app_state.dao.params.min_expedite_delay", gcdDelay),
		cosmos.NewGenesisKV("app_state.guardian.params.guardian", walletAddr),
	)
	net := chain.Start(t, ctx, chain.WithGenesis(genesis...))
	s := setupDao(t, ctx, net)
	require.Equal(t, gcdDelay, s.queryParams(t).MinExpediteDelay)

	c := &councilSuite{daoSuite: s}
	var err error
	c.wallet, err = interchaintest.GetAndFundTestUserWithMnemonic(ctx, "gcd-wallet", walletMnemonic, sdkmath.NewInt(daoUserFunds), s.bze)
	require.NoError(t, err)
	require.Equal(t, walletAddr, c.wallet.FormattedAddress())
	guardian, err := helpers.QueryGuardian(ctx, s.bze)
	require.NoError(t, err)
	require.Equal(t, walletAddr, guardian, "the genesis guardian")

	c.members = s.users(t, 5, daoUserFunds)
	c.rcpt, c.anyone = s.user(t), s.user(t)
	c.dao = s.createStatic(t, c.members[0], expediteCfg(), addrs(c.members...), "")
	require.Equal(t, c.dao.addr, s.dao(t, c.dao.id).Admin, "the council governs itself")
	// funded now: MsgSend is disabled from step 3 on
	s.fund(t, c.dao.addr, ubze(gcdSend))

	t.Run("1 governance makes the council the guardian", c.govSetsCouncil)
	t.Run("2 the wallet guardian is refused", c.walletRefused)
	t.Run("3 the council disables MsgSend at once", c.councilDisables)
	t.Run("4 a council send fails on the breaker", c.sendFails)
	t.Run("5 the council cannot disable its own vote", c.voteProtected)
	t.Run("6 governance resets and the failed send runs on retry", c.govResetsAndRetry)
}

type councilSuite struct {
	*daoSuite
	wallet         ibc.Wallet // the genesis guardian
	members        []ibc.Wallet
	rcpt, anyone   ibc.Wallet
	dao            daoRef
	failedSend     uint64 // the proposal of step 4
	treasuryBefore sdkmath.Int
}

// 1. A passed gov proposal with x/guardian's MsgUpdateParams sets the guardian
// to the council's address.
func (c *councilSuite) govSetsCouncil(t *testing.T) {
	_, err := helpers.SubmitAndPassProposal(c.ctx, c.bze, c.funder, helpers.Proposal{
		Messages: []any{map[string]any{
			"@type":     "/bze.guardian.v1.MsgUpdateParams",
			"authority": c.gov,
			"params":    map[string]any{"guardian": c.dao.addr},
		}},
		Title:   "council guardian",
		Summary: "e2e: a static-list DAO becomes the guardian",
	})
	require.NoError(t, err)
	guardian, err := helpers.QueryGuardian(c.ctx, c.bze)
	require.NoError(t, err)
	require.Equal(t, c.dao.addr, guardian)
	require.Empty(t, c.circuitDisabled(t))
}

// 2. The wallet that was the guardian is now a stranger: its disable is
// refused as unauthorized (1101), charged only the fee, and nothing is
// disabled.
func (c *councilSuite) walletRefused(t *testing.T) {
	c.refuse(t, c.wallet, refusal{codespace: gdnCodespace, code: gdnUnauthorizedErr}, "guardian", "disable-messages", gcdSendURL)
	require.Empty(t, c.circuitDisabled(t))
}

// 3. The council passes an expedited proposal with MsgDisableMessages signed
// by its own address: refused as not settleable at two yes of five (1130),
// settled and executed by anyone right after the third, long before
// min_expedite_delay. MsgSend is then on the disable list, put there by the
// council.
func (c *councilSuite) councilDisables(t *testing.T) {
	pid := c.submit(t, c.members[0], c.dao, "disable sends", []map[string]any{c.disableMsg(gcdSendURL)}, "--expedited")
	p := c.prop(t, pid)
	require.True(t, p.Expedited)

	c.vote(t, c.members[0], pid, "yes")
	c.vote(t, c.members[1], pid, "yes")
	require.False(t, c.proposal(t, pid).ExpediteSettleable, "2 of 5 misses the expedite quorum")
	c.refuse(t, c.anyone, daoCode(1130), "dao", "settle-proposal", daoU64(pid))
	require.Empty(t, c.circuitDisabled(t))

	c.vote(t, c.members[2], pid, "yes")
	require.True(t, c.proposal(t, pid).ExpediteSettleable, "3 of 5 clears both bars")
	res := c.settleFirst(t, c.anyone, pid, "EXECUTED")
	require.Equal(t, "true", c.event(t, res, "EventProposalSettled").Attributes["expedited_early"])
	executed := c.event(t, res, "EventProposalExecuted")
	require.Equal(t, "true", executed.Attributes["success"])
	require.Empty(t, executed.Attributes["error"])

	disabled := helpers.EventsWith(helpers.TxEvents(res, "bze.guardian.v1.MessagesDisabledEvent"), map[string]string{
		"signer": c.dao.addr,
	})
	require.Len(t, disabled, 1, "one MessagesDisabledEvent signed by the council")
	require.Contains(t, disabled[0].Attributes["msg_type_urls"], gcdSendURL)
	require.Equal(t, []string{gcdSendURL}, c.circuitDisabled(t))

	_, now := c.blockTime(t)
	require.Less(t, now, int64(p.SubmitTime)+gcdDelaySecs, "settled before min_expedite_delay ran out")
	c.requireInvariants(t)
}

// 4. A user's MsgSend is refused by the breaker with its reason. A council
// proposal sending from the treasury is still submitted, voted and settled
// early (protected dao messages), but its execution fails: EXECUTION_FAILED
// with the fixed reason of a failed handler. The settle moves nothing (funds
// audit), so the treasury, which holds the amount, keeps it. "message
// failed" is the reason of any handler error, so it does not prove the
// breaker by itself: the retry after the reset in step 6 does.
func (c *councilSuite) sendFails(t *testing.T) {
	res, err := helpers.ExecMsgs(c.ctx, c.bze, c.rcpt, []helpers.TxOption{helpers.WithGas(daoRefuseGas)},
		bankSend(c.rcpt.FormattedAddress(), c.anyone.FormattedAddress(), ubze(1)))
	reason, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, reason, gcdCircuit, "a user's MsgSend is refused by the breaker")

	send := bankSend(c.dao.addr, c.rcpt.FormattedAddress(), ubze(gcdSend))
	pid := c.submit(t, c.members[0], c.dao, "council send", []map[string]any{send}, "--expedited")
	for _, m := range c.members[:3] {
		c.vote(t, m, pid, "yes")
	}
	require.True(t, c.proposal(t, pid).ExpediteSettleable)

	c.treasuryBefore = c.bal(t, c.dao.addr, chain.Denom)
	require.True(t, c.treasuryBefore.GTE(sdkmath.NewInt(gcdSend)), "the treasury holds the amount: only the breaker can fail the send")
	rcptBefore := c.bal(t, c.rcpt.FormattedAddress(), chain.Denom)

	res = c.settleFirst(t, c.anyone, pid, "EXECUTION_FAILED")
	require.Equal(t, "true", c.event(t, res, "EventProposalSettled").Attributes["expedited_early"])
	executed := c.event(t, res, "EventProposalExecuted")
	require.Equal(t, "false", executed.Attributes["success"])
	require.Equal(t, gcdExecFail, executed.Attributes["error"])

	require.Equal(t, c.treasuryBefore.String(), c.bal(t, c.dao.addr, chain.Denom).String(), "the treasury is unchanged")
	require.Equal(t, rcptBefore.String(), c.bal(t, c.rcpt.FormattedAddress(), chain.Denom).String(), "the recipient got nothing")
	require.Equal(t, []string{gcdSendURL}, c.circuitDisabled(t))
	c.failedSend = pid
	c.requireInvariants(t)
}

// 5. A council proposal disabling MsgVote passes the vote, but x/guardian
// refuses the protected type on execution: EXECUTION_FAILED, and the disable
// list keeps only MsgSend. The guardian DAO cannot lock itself out. The
// control is the same proposal with MsgMultiSend, which is not protected: it
// executes and MsgMultiSend is disabled, so only the protected type made the
// first one fail. (The protected-type error code itself is pinned for a
// wallet guardian by TestGuardian; a DAO has no route to sign the message
// directly, outside a proposal.)
func (c *councilSuite) voteProtected(t *testing.T) {
	pid := c.submit(t, c.members[0], c.dao, "disable votes", []map[string]any{c.disableMsg(gcdVoteURL)}, "--expedited")
	for _, m := range c.members[:3] {
		c.vote(t, m, pid, "yes")
	}
	res := c.settleFirst(t, c.anyone, pid, "EXECUTION_FAILED")
	executed := c.event(t, res, "EventProposalExecuted")
	require.Equal(t, "false", executed.Attributes["success"])
	require.Equal(t, gcdExecFail, executed.Attributes["error"])
	require.Empty(t, helpers.TxEvents(res, "bze.guardian.v1.MessagesDisabledEvent"))
	require.Equal(t, []string{gcdSendURL}, c.circuitDisabled(t))

	pid = c.submit(t, c.members[0], c.dao, "disable multi-sends", []map[string]any{c.disableMsg(gcdMultiURL)}, "--expedited")
	for _, m := range c.members[:3] {
		c.vote(t, m, pid, "yes")
	}
	res = c.settleFirst(t, c.anyone, pid, "EXECUTED")
	executed = c.event(t, res, "EventProposalExecuted")
	require.Equal(t, "true", executed.Attributes["success"])
	disabled := helpers.EventsWith(helpers.TxEvents(res, "bze.guardian.v1.MessagesDisabledEvent"), map[string]string{
		"signer": c.dao.addr,
	})
	require.Len(t, disabled, 1, "one MessagesDisabledEvent signed by the council")
	require.Contains(t, disabled[0].Attributes["msg_type_urls"], gcdMultiURL)
	require.ElementsMatch(t, []string{gcdSendURL, gcdMultiURL}, c.circuitDisabled(t))

	// the council still votes after the attempt
	pid = c.submit(t, c.members[0], c.dao, "still voting", nil, "--expedited")
	for _, m := range c.members[:3] {
		c.vote(t, m, pid, "yes")
	}
	c.requireTallyOf(t, pid, 3, 0, 0)
	c.settleFirst(t, c.anyone, pid, "EXECUTED")
	c.requireInvariants(t)
}

// 6. A gov proposal with x/circuit's MsgResetCircuitBreaker takes MsgSend
// (and the control's MsgMultiSend) off the list. The failed proposal of step 4 is retried within its execution
// window and now sends exactly the amount from the treasury.
func (c *councilSuite) govResetsAndRetry(t *testing.T) {
	require.NotZero(t, c.failedSend, "step 4 left a failed proposal")
	_, err := helpers.SubmitAndPassProposal(c.ctx, c.bze, c.funder, helpers.Proposal{
		Messages: []any{map[string]any{
			"@type":         gdnResetURL,
			"authority":     c.gov,
			"msg_type_urls": []string{gcdSendURL, gcdMultiURL},
		}},
		Title:   "re-enable sends",
		Summary: "e2e: reset the circuit breaker for MsgSend and MsgMultiSend",
	})
	require.NoError(t, err)
	require.Empty(t, c.circuitDisabled(t))

	resp := c.proposal(t, c.failedSend)
	require.Equal(t, "EXECUTION_FAILED", resp.Proposal.status())
	_, now := c.blockTime(t)
	require.Less(t, now, int64(resp.RetryDeadline), "the retry is within the execution window")

	res := c.settle(t, c.anyone, c.failedSend, "EXECUTED", []string{c.rcpt.FormattedAddress()},
		[]daoFlow{flow(c.dao.addr, c.rcpt.FormattedAddress(), ubze(gcdSend))})
	executed := c.event(t, res, "EventProposalExecuted")
	require.Equal(t, "true", executed.Attributes["success"])
	require.Equal(t, c.treasuryBefore.SubRaw(gcdSend).String(), c.bal(t, c.dao.addr, chain.Denom).String())
	c.requireInvariants(t)
}

func (c *councilSuite) disableMsg(urls ...string) map[string]any {
	return map[string]any{
		"@type":         "/bze.guardian.v1.MsgDisableMessages",
		"signer":        c.dao.addr,
		"msg_type_urls": urls,
	}
}

// circuitDisabled is x/circuit's disable list.
func (c *councilSuite) circuitDisabled(t *testing.T) []string {
	t.Helper()
	res, err := helpers.Query[struct {
		DisabledList []string `json:"disabled_list"`
	}](c.ctx, c.bze, "circuit", "disabled-list")
	require.NoError(t, err)
	return res.DisabledList
}
