package interchaintest_test

import (
	"time"

	"fmt"
	"strings"
	"testing"

	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
)

func submitMsg(proposer string, daoID uint64, title string, msgs []map[string]any, expedited bool) map[string]any {
	anys := make([]any, len(msgs))
	for i, m := range msgs {
		anys[i] = m
	}
	return daoMsg("MsgSubmitProposal", map[string]any{
		"proposer": proposer, "dao_id": daoU64(daoID), "title": title, "msgs": anys, "expedited": expedited,
	})
}

func voteMsg(voter string, pid uint64, option string) map[string]any {
	return daoMsg("MsgVote", map[string]any{"voter": voter, "proposal_id": daoU64(pid), "option": option})
}

func settleMsg(sender string, pid uint64) map[string]any {
	return daoMsg("MsgSettleProposal", map[string]any{"sender": sender, "proposal_id": daoU64(pid)})
}

// A4. submit-proposal: the cost to the burner or the treasury, the deposit
// into the escrow, the DAO itself submitting, and every refusal.
func (s *daoSuite) submitProposal(t *testing.T) {
	m := s.users(t, 3, daoUserFunds)
	admin, outsider, rcpt := s.user(t), s.user(t), s.user(t)
	cfg := defaultCfg()
	cfg.cost, cfg.deposit = coinPtr(100, chain.Denom), coinPtr(1_000, chain.Denom)
	d := s.createStatic(t, m[0], cfg, addrs(m...), admin.FormattedAddress())
	s.fund(t, d.addr, ubze(10_000))

	pid := s.submit(t, m[0], d, "cost to the burner", nil, "--summary", "a summary")
	p := s.prop(t, pid)
	require.Equal(t, "a summary", p.Summary)
	require.Equal(t, coinStr(1_000, chain.Denom), p.PinnedDeposit.coins().String())
	require.Equal(t, "3", p.PinnedDenominator, "a static dao pins its member count")
	require.Equal(t, int64(p.SubmitTime)+15, int64(p.VotingEnd), "voting ends voting_period after the submit block")
	require.False(t, p.Expedited)
	require.Equal(t, s.dao(t, d.id).Config, p.PinnedConfig, "the config is pinned at submit")

	treasury := cfg
	treasury.costDest = "TREASURY"
	dt := s.createStatic(t, m[0], treasury, addrs(m...), admin.FormattedAddress())
	s.fund(t, dt.addr, ubze(10_000))
	s.submit(t, m[1], dt, "cost to the treasury", nil)

	t.Run("the dao's own account submits", func(t *testing.T) {
		// through its admin: the treasury cost is skipped, the deposit escrowed
		sub := submitMsg(dt.addr, dt.id, "by the dao", nil, false)
		res := s.adminExec(t, admin, dt, []daoFlow{flow(dt.addr, s.escrow, ubze(1_000))}, sub)
		s.submitted(t, res, dt, dt.addr)
	})

	t.Run("D22 non-member", func(t *testing.T) {
		s.refuse(t, outsider, refusal{codespace: daoModule, code: 1118, watch: []string{d.addr}}, "dao", "submit-proposal", d.idArg(), "outsider")
		open := cfg
		open.open = true
		do := s.createStatic(t, m[0], open, addrs(m...), admin.FormattedAddress())
		s.submit(t, outsider, do, "an outsider pays cost and deposit", nil)
	})

	t.Run("caps", func(t *testing.T) {
		// d holds one proposal: two more fill max_open_proposals
		s.submit(t, m[1], d, "second", nil)
		s.submit(t, m[2], d, "third", nil)
		s.refuse(t, m[0], refusal{codespace: daoModule, code: 1120, watch: []string{d.addr}}, "dao", "submit-proposal", d.idArg(), "fourth")
		// the cap is per dao
		other := s.createStatic(t, m[0], cfg, addrs(m...), "")
		s.submit(t, m[0], other, "another dao is not affected", nil)

		s.refuse(t, m[0], daoCode(1117), "dao", "submit-proposal", other.idArg(), strings.Repeat("t", daoMaxTitle+1))
		s.refuse(t, m[0], daoCode(1117), "dao", "submit-proposal", other.idArg(), "long summary", "--summary", strings.Repeat("s", daoMaxSummary+1))
		s.refuseMsgs(t, m[0], daoCode(1117), submitMsg(m[0].FormattedAddress(), other.id, "", nil, false))
		send := bankSend(other.addr, rcpt.FormattedAddress(), ubze(1))
		s.refuse(t, m[0], daoCode(1110), append([]string{"dao", "submit-proposal", other.idArg(), "four messages"}, msgArgs(send, send, send, send)...)...)
		big := daoMsg("MsgUpdateMetadata", map[string]any{"authority": other.addr, "dao_id": other.idArg(), "name": "x", "description": strings.Repeat("d", 2_000)})
		s.refuse(t, m[0], daoCode(1110), append([]string{"dao", "submit-proposal", other.idArg(), "too many bytes"}, msgArgs(big, big, big)...)...)
		// the bounds are inclusive
		s.submit(t, m[1], other, strings.Repeat("t", daoMaxTitle), []map[string]any{send, send, send}, "--summary", strings.Repeat("s", daoMaxSummary))
	})

	t.Run("expedite not allowed", func(t *testing.T) {
		fresh := s.createStatic(t, m[0], cfg, addrs(m...), "")
		s.refuse(t, m[0], daoCode(1121), "dao", "submit-proposal", fresh.idArg(), "fast", "--expedited")
	})

	t.Run("D1 and D24 signers", func(t *testing.T) {
		fresh := s.createStatic(t, m[0], cfg, addrs(m...), "")
		s.refuse(t, m[0], daoCode(1111), "dao", "submit-proposal", fresh.idArg(), "victim", msgArgs(bankSend(rcpt.FormattedAddress(), m[0].FormattedAddress(), ubze(1)))[0])
		multi := map[string]any{
			"@type":   "/cosmos.bank.v1beta1.MsgMultiSend",
			"inputs":  []map[string]any{{"address": fresh.addr, "coins": ubze(1)}, {"address": rcpt.FormattedAddress(), "coins": ubze(1)}},
			"outputs": []map[string]any{{"address": m[0].FormattedAddress(), "coins": ubze(2)}},
		}
		s.refuse(t, m[0], daoCode(1111), "dao", "submit-proposal", fresh.idArg(), "multi", msgArgs(multi)[0])
	})

	t.Run("insufficient funds store nothing", func(t *testing.T) {
		poor := s.users(t, 2, 50+daoRefuseGas/100)
		fresh := s.createStatic(t, m[0], cfg, addrs(poor...), "")
		// cannot pay the cost
		s.refuse(t, poor[0], refusal{codespace: "sdk", code: 5, reason: "proposal cost", watch: []string{fresh.addr}}, "dao", "submit-proposal", fresh.idArg(), "poor")
		// can pay the cost, not the deposit: the cost is not charged either
		s.fund(t, poor[1].FormattedAddress(), ubze(500))
		s.refuse(t, poor[1], refusal{codespace: "sdk", code: 5, reason: "proposal deposit", watch: []string{fresh.addr}}, "dao", "submit-proposal", fresh.idArg(), "poor")
	})

	t.Run("unknown dao and a dao without voters", func(t *testing.T) {
		s.refuse(t, m[0], daoCode(1108), "dao", "submit-proposal", "999999", "nobody")
		idle := s.createStake(t, admin, defaultCfg(), daoElectorate{SrIDs: []string{s.fixtures(t).srIdle}}, admin.FormattedAddress())
		// the dao itself may submit, but its electorate holds no stake: denominator 0
		s.refuse(t, admin, daoCode(1119), "dao", "admin-execute", idle.idArg(), msgArgs(submitMsg(idle.addr, idle.id, "no voters", nil, false))[0])
	})

	// leave nothing open: the proposals of this case are cancelled (ALWAYS
	// refunds), or settled once their voting ended
	for _, ref := range []daoRef{d, dt} {
		for _, p := range s.daoProposals(t, ref.id, "VOTING") {
			s.cancelBy(t, uint64(p.ID), map[string]ibc.Wallet{m[0].FormattedAddress(): m[0], m[1].FormattedAddress(): m[1], m[2].FormattedAddress(): m[2]}, admin, dt)
		}
	}
}

// A5. vote: lowercase options, a revote moves the tally (D23), the same
// option twice, and the refusals.
func (s *daoSuite) voting(t *testing.T) {
	m := s.users(t, 3, daoUserFunds)
	outsider := s.user(t)
	d := s.createStatic(t, m[0], defaultCfg(), addrs(m...), "")
	pid := s.submit(t, m[0], d, "vote on me", nil)
	id := daoU64(pid)

	require.Equal(t, "1", s.voteWeight(t, m[0], pid, "yes").String())
	s.requireTallyOf(t, pid, 1, 0, 0)
	s.refuse(t, m[0], daoCode(1125), "dao", "vote", id, "yes")
	require.Equal(t, "1", s.voteWeight(t, m[0], pid, "no").String(), "D23 revote")
	s.requireTallyOf(t, pid, 0, 1, 0)
	require.Equal(t, "NO", s.mustBallot(t, pid, m[0].FormattedAddress()).Option)
	s.voteWeight(t, m[1], pid, "abstain")
	s.requireTallyOf(t, pid, 0, 1, 1)

	s.refuse(t, m[2], daoCode(1122), "dao", "vote", id, "unspecified")
	s.refuseMsgs(t, m[2], daoCode(1122), daoMsg("MsgVote", map[string]any{"voter": m[2].FormattedAddress(), "proposal_id": id}))
	s.refuseMsgs(t, m[2], daoCode(1122), daoMsg("MsgVote", map[string]any{"voter": m[2].FormattedAddress(), "proposal_id": id, "option": 7}))
	s.refuse(t, outsider, daoCode(1119), "dao", "vote", id, "yes")
	s.refuse(t, m[2], daoCode(1123), "dao", "vote", "999999", "yes")
	s.refuseMsgs(t, m[2], daoCode(1123), voteMsg(m[2].FormattedAddress(), 0, "YES"))

	// D18: voting closes at voting_end itself
	s.waitVotingEnd(t, pid)
	s.refuse(t, m[2], daoCode(1124), "dao", "vote", id, "yes")
	s.requireTallyOf(t, pid, 0, 1, 1)
	s.settleFirst(t, outsider, pid, "REJECTED")
	s.refuse(t, m[2], daoCode(1124), "dao", "vote", id, "yes")
	s.terminal["REJECTED"] = pid
}

// cancelBy cancels a proposal with the wallet of its proposer (a member, or
// the DAO through its admin); one whose voting ended is settled instead.
func (s *daoSuite) cancelBy(t *testing.T, pid uint64, members map[string]ibc.Wallet, admin ibc.Wallet, viaDao daoRef) {
	t.Helper()
	if s.settledAfterVotingEnd(t, admin, pid) {
		return
	}
	p := s.prop(t, pid)
	if w, ok := members[p.Proposer]; ok {
		s.cancel(t, w, pid)
		return
	}
	require.Equal(t, viaDao.addr, p.Proposer)
	s.adminExec(t, admin, viaDao, s.depositFlow(p, true), daoMsg("MsgCancelProposal", map[string]any{"proposer": viaDao.addr, "proposal_id": daoU64(pid)}))
	s.requireStatus(t, pid, "CANCELLED")
}

// A6. cancel-proposal: the 3 refund policies × 2 kept destinations, a cancel
// with ballots, the refusal from voting_end, the other refusals and the freed
// slot.
func (s *daoSuite) cancelProposal(t *testing.T) {
	m := s.users(t, 2, daoUserFunds)
	for _, policy := range []string{"ALWAYS", "ONLY_PASSED", "NEVER"} {
		for _, dest := range []string{"BURN", "TREASURY"} {
			t.Run(policy+" "+dest, func(t *testing.T) {
				cfg := defaultCfg()
				cfg.deposit, cfg.policy, cfg.keptDest = coinPtr(1_000, chain.Denom), policy, dest
				d := s.createStatic(t, m[0], cfg, addrs(m...), "")
				pid := s.submit(t, m[0], d, "cancel me", nil)
				s.vote(t, m[1], pid, "yes")
				res := s.cancel(t, m[0], pid)
				e := s.event(t, res, "EventDepositReleased")
				require.Equal(t, fmt.Sprint(policy == "ALWAYS"), e.Attributes["refunded"])
				require.Equal(t, coinStr(1_000, chain.Denom), e.Attributes["amount"])
				p := s.prop(t, pid)
				require.Equal(t, "1", parseInt(p.Yes).String(), "a cancelled proposal keeps its tally")
				require.Len(t, s.ballots(t, pid), 1, "and its ballots")
				s.refuse(t, m[1], daoCode(1124), "dao", "vote", daoU64(pid), "no")
				s.refuse(t, m[0], daoCode(1124), "dao", "cancel-proposal", daoU64(pid))
			})
		}
	}

	cfg := defaultCfg()
	cfg.deposit = coinPtr(1_000, chain.Denom)
	d := s.createStatic(t, m[0], cfg, addrs(m...), "")
	t.Run("refusals", func(t *testing.T) {
		pid := s.submit(t, m[0], d, "not yours", nil)
		s.refuse(t, m[1], daoCode(1126), "dao", "cancel-proposal", daoU64(pid))
		s.refuse(t, m[1], daoCode(1123), "dao", "cancel-proposal", "999999")
		s.cancel(t, m[0], pid)
		s.terminal["CANCELLED-0-ballots"] = pid
	})
	t.Run("with ballots, before voting_end", func(t *testing.T) {
		pid := s.submit(t, m[0], d, "cancel with votes", nil)
		s.vote(t, m[1], pid, "yes")
		s.cancel(t, m[0], pid)
		s.terminal["CANCELLED"] = pid
	})
	t.Run("refused from voting_end", func(t *testing.T) {
		pid := s.submit(t, m[0], d, "late cancel", nil)
		s.vote(t, m[1], pid, "yes")
		s.waitVotingEnd(t, pid)
		require.True(t, s.proposal(t, pid).VotingEnded)
		s.refuse(t, m[0], daoCode(1124), "dao", "cancel-proposal", daoU64(pid))
		s.requireStatus(t, pid, "VOTING")
		status := "REJECTED"
		if passes(t, s.prop(t, pid)) {
			status = "EXECUTED"
		}
		s.settleFirst(t, m[1], pid, status)
	})
	t.Run("after settle", func(t *testing.T) {
		pid := s.submit(t, m[0], d, "settled", nil)
		s.waitVotingEnd(t, pid)
		s.settleFirst(t, m[1], pid, "REJECTED")
		s.refuse(t, m[0], daoCode(1124), "dao", "cancel-proposal", daoU64(pid))
	})
	t.Run("the slot is freed", func(t *testing.T) {
		var open []uint64
		for i := 0; i < daoMaxOpen; i++ {
			open = append(open, s.submit(t, m[i%2], d, fmt.Sprint("slot ", i), nil))
		}
		s.refuse(t, m[0], daoCode(1120), "dao", "submit-proposal", d.idArg(), "no slot")
		s.cancel(t, m[0], open[0])
		open[0] = s.submit(t, m[0], d, "slot freed", nil)
		for _, pid := range open {
			s.cancel(t, map[bool]ibc.Wallet{true: m[0], false: m[1]}[s.prop(t, pid).Proposer == m[0].FormattedAddress()], pid)
		}
	})
}

// A7. settle-proposal: every outcome, the deposit matrix against the tally
// (D13), the refusals, D18 at voting_end and D8.
func (s *daoSuite) settleProposal(t *testing.T) {
	m := s.users(t, 3, daoUserFunds)
	anyone, rcpt := s.user(t), s.user(t)
	r := rcpt.FormattedAddress()

	t.Run("D13 deposit matrix", func(t *testing.T) {
		type pair struct {
			d              daoRef
			passed, failed uint64
		}
		var pairs []pair
		for _, policy := range []string{"ALWAYS", "ONLY_PASSED", "NEVER"} {
			cfg := defaultCfg()
			cfg.deposit, cfg.policy, cfg.keptDest = coinPtr(1_000, chain.Denom), policy, "TREASURY"
			d := s.createStatic(t, m[0], cfg, addrs(m...), "")
			pairs = append(pairs, pair{d: d, passed: s.submit(t, m[0], d, "pass", nil), failed: s.submit(t, m[1], d, "reject", nil)})
		}
		for _, p := range pairs {
			s.vote(t, m[0], p.passed, "yes")
			s.vote(t, m[1], p.passed, "yes")
			s.vote(t, m[0], p.failed, "no")
			s.vote(t, m[1], p.failed, "no")
		}
		for _, p := range pairs {
			s.waitVotingEnd(t, p.failed)
			s.settleFirst(t, anyone, p.passed, "EXECUTED")
			s.settleFirst(t, anyone, p.failed, "REJECTED")
		}
		s.terminal["EXECUTED"] = pairs[0].passed
	})

	cfg := defaultCfg()
	cfg.deposit, cfg.policy = coinPtr(1_000, chain.Denom), "ONLY_PASSED"
	d := s.createStatic(t, m[0], cfg, addrs(m...), "")
	s.fund(t, d.addr, ubze(100))

	t.Run("executed", func(t *testing.T) {
		pid := s.submit(t, m[0], d, "pay", []map[string]any{bankSend(d.addr, r, ubze(10))})
		s.refuse(t, anyone, daoCode(1127), "dao", "settle-proposal", daoU64(pid))
		s.vote(t, m[0], pid, "yes")
		s.vote(t, m[1], pid, "no")
		s.vote(t, m[2], pid, "yes")
		// D18: settle is accepted in the first block at or after voting_end
		s.waitVotingEnd(t, pid)
		res := s.settleFirst(t, anyone, pid, "EXECUTED", flow(d.addr, r, ubze(10)))
		e := s.event(t, res, "EventProposalSettled")
		require.Equal(t, []string{"2", "1", "0", "3", "false"}, []string{e.Attributes["yes"], e.Attributes["no"], e.Attributes["abstain"], e.Attributes["denominator"], e.Attributes["expedited_early"]})
		require.Equal(t, "true", s.event(t, res, "EventProposalExecuted").Attributes["success"])
		s.refuse(t, anyone, daoCode(1128), "dao", "settle-proposal", daoU64(pid))
		s.refuse(t, anyone, daoCode(1123), "dao", "settle-proposal", "999999")
	})
	t.Run("D16 all abstain is rejected", func(t *testing.T) {
		pid := s.submit(t, m[0], d, "abstain", []map[string]any{bankSend(d.addr, r, ubze(10))})
		for _, v := range m {
			s.vote(t, v, pid, "abstain")
		}
		s.waitVotingEnd(t, pid)
		s.settleFirst(t, anyone, pid, "REJECTED")
	})
	t.Run("quorum not met is rejected", func(t *testing.T) {
		pid := s.submit(t, m[0], d, "quorum", nil)
		s.vote(t, m[0], pid, "yes")
		s.waitVotingEnd(t, pid)
		s.settleFirst(t, anyone, pid, "REJECTED")
	})
	t.Run("execution failed, retried, executed", func(t *testing.T) {
		pid := s.submit(t, m[0], d, "too much", []map[string]any{bankSend(d.addr, r, ubze(1_000))})
		s.pass(t, pid, m[:2], "EXECUTION_FAILED")
		res := s.proposal(t, pid)
		require.Equal(t, int64(res.Proposal.VotingEnd)+daoExecWindowSecs, int64(res.RetryDeadline))
		require.False(t, res.VotingEnded, "voting_ended is only set while the proposal is in voting")
		s.fund(t, d.addr, ubze(1_000))
		// the deposit was released (refunded: it passed, ONLY_PASSED) by the first settle
		s.settle(t, anyone, pid, "EXECUTED", nil, []daoFlow{flow(d.addr, r, ubze(1_000))})
		require.Zero(t, int64(s.proposal(t, pid).RetryDeadline))
	})
	t.Run("execution failed, expired after the window", func(t *testing.T) {
		pid := s.submit(t, m[0], d, "expire", []map[string]any{bankSend(d.addr, r, ubze(1_000_000))})
		s.pass(t, pid, m[:2], "EXECUTION_FAILED")
		deadline := int64(s.proposal(t, pid).RetryDeadline)
		s.waitTime(t, deadline)
		s.fund(t, d.addr, ubze(1_000_000))
		s.settle(t, anyone, pid, "EXPIRED", nil, nil)
		s.refuse(t, anyone, daoCode(1128), "dao", "settle-proposal", daoU64(pid))
		s.terminal["EXPIRED"] = pid
		s.adminlessDrain(t, d, m, r)
	})
	t.Run("D8 a proposal that settles itself", func(t *testing.T) {
		next := s.proposals + 1
		pid := s.submit(t, m[0], d, "settle myself", []map[string]any{settleMsg(d.addr, next)})
		require.Equal(t, next, pid)
		res := s.pass(t, pid, m[:2], "EXECUTION_FAILED")
		executed := s.event(t, res, "EventProposalExecuted")
		require.Equal(t, "false", executed.Attributes["success"])
		require.Equal(t, "message failed", executed.Attributes["error"])
		s.settle(t, anyone, pid, "EXECUTION_FAILED", nil, nil)
		s.failed = pid
	})
}

// adminlessDrain spends the leftover treasury of a self-owned test DAO back
// to rcpt, so later balances in the suite stay small.
func (s *daoSuite) adminlessDrain(t *testing.T, d daoRef, voters []ibc.Wallet, rcpt string) {
	t.Helper()
	left := s.allBalances(t, d.addr)
	if left.IsZero() {
		return
	}
	pid := s.submit(t, voters[0], d, "drain", []map[string]any{bankSend(d.addr, rcpt, left)})
	s.pass(t, pid, voters[:2], "EXECUTED", flow(d.addr, rcpt, left))
}

// A8. Expedite (D7): an early settle once both bars are met past
// min_expedite_delay, 1130 below them, and the regular bars after the voting
// end. A settle within the delay (1132) is E1.
func (s *daoSuite) expedite(t *testing.T) {
	m := s.users(t, 5, daoUserFunds)
	anyone := s.user(t)
	cfg := defaultCfg()
	cfg.allowExpedite = true
	cfg.voting, cfg.expedited = "40s", "30s"
	cfg.deposit = coinPtr(1_000, chain.Denom)
	d := s.createStatic(t, m[0], cfg, addrs(m...), "")

	pid := s.submit(t, m[0], d, "fast", nil, "--expedited")
	p := s.prop(t, pid)
	require.True(t, p.Expedited)
	require.Equal(t, int64(p.SubmitTime)+30, int64(p.VotingEnd), "an expedited proposal uses expedited_period")
	s.vote(t, m[0], pid, "yes")
	s.vote(t, m[1], pid, "yes")
	// past min_expedite_delay, so the bars alone decide (the delay is E1)
	s.waitTime(t, int64(p.SubmitTime)+daoExpediteDelaySecs)
	require.False(t, s.proposal(t, pid).ExpediteSettleable)
	s.refuse(t, anyone, daoCode(1130), "dao", "settle-proposal", daoU64(pid))
	s.vote(t, m[2], pid, "yes")
	require.False(t, s.proposal(t, pid).ExpediteSettleable, "3 of 5 is exactly 0.6: the expedite yes bar must be exceeded")
	s.refuse(t, anyone, daoCode(1130), "dao", "settle-proposal", daoU64(pid))
	s.vote(t, m[3], pid, "yes")
	res := s.proposal(t, pid)
	require.True(t, res.ExpediteSettleable, "4 of 5 clears both expedite bars (quorum ≥ 0.6, yes > 0.6)")
	_, now := s.blockTime(t)
	require.Less(t, now, int64(res.Proposal.VotingEnd), "the case must settle before the voting end")
	out := s.settleFirst(t, anyone, pid, "EXECUTED")
	require.Equal(t, "true", s.event(t, out, "EventProposalSettled").Attributes["expedited_early"])

	// below the expedite bar, but the regular bars pass after the voting end
	pid = s.submit(t, m[0], d, "regular bars", nil, "--expedited")
	s.vote(t, m[0], pid, "yes")
	s.vote(t, m[1], pid, "yes")
	s.vote(t, m[2], pid, "no")
	s.waitTime(t, int64(s.prop(t, pid).SubmitTime)+daoExpediteDelaySecs)
	s.refuse(t, anyone, daoCode(1130), "dao", "settle-proposal", daoU64(pid))
	s.waitVotingEnd(t, pid)
	require.False(t, s.proposal(t, pid).ExpediteSettleable, "expedite_settleable is only set before the voting end")
	out = s.settleFirst(t, anyone, pid, "EXECUTED")
	require.Equal(t, "false", s.event(t, out, "EventProposalSettled").Attributes["expedited_early"])

	// a regular proposal never settles early, even unanimous
	pid = s.submit(t, m[0], d, "not expedited", nil)
	for _, v := range m {
		s.vote(t, v, pid, "yes")
	}
	s.refuse(t, anyone, daoCode(1127), "dao", "settle-proposal", daoU64(pid))
	s.waitVotingEnd(t, pid)
	s.settleFirst(t, anyone, pid, "EXECUTED")
}

// closeOpen leaves nothing of a test's proposal open: it is cancelled by its
// proposer, or settled once its voting ended (a cancel is refused from then
// on).
func (s *daoSuite) closeOpen(t *testing.T, proposer ibc.Wallet, pid uint64) {
	t.Helper()
	if !s.settledAfterVotingEnd(t, proposer, pid) {
		s.cancel(t, proposer, pid)
	}
}

// settledAfterVotingEnd settles the proposal under its tally when its voting
// ends within a few seconds (a cancel sent now could land after voting_end),
// waiting for the end first; it reports whether it settled.
func (s *daoSuite) settledAfterVotingEnd(t *testing.T, sender ibc.Wallet, pid uint64) bool {
	t.Helper()
	p := s.prop(t, pid)
	if time.Until(time.Unix(int64(p.VotingEnd), 0)) > 10*time.Second {
		return false
	}
	s.waitVotingEnd(t, pid)
	p = s.prop(t, pid)
	status := "REJECTED"
	if passes(t, p) {
		status = "EXECUTED"
	}
	s.settleFirst(t, sender, pid, status)

	return true
}
