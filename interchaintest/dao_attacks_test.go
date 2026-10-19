package interchaintest_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

func adminExecMsg(admin string, daoID uint64, msgs ...map[string]any) map[string]any {
	anys := make([]any, len(msgs))
	for i, m := range msgs {
		anys[i] = m
	}
	return daoMsg("MsgAdminExecute", map[string]any{"admin": admin, "dao_id": daoU64(daoID), "msgs": anys})
}

func authzExec(grantee string, msgs ...map[string]any) map[string]any {
	anys := make([]any, len(msgs))
	for i, m := range msgs {
		anys[i] = m
	}
	return map[string]any{"@type": "/cosmos.authz.v1beta1.MsgExec", "grantee": grantee, "msgs": anys}
}

// E1. Multi-message txs: the orderings only a real tx exercises (a join and a
// submit in one tx is in C, D3).
func (s *daoSuite) multiMessageTxs(t *testing.T) {
	src := stakeSource{s: s, sr: s.newSR(t, daoSrLock, s.stk, daoSrDuration)}
	a, b := s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0], s.users(t, 1, daoUserFunds, coins(2_000, s.stk)...)[0]
	src.join(t, a, 1_000)
	src.join(t, b, 1_000)
	cfg := defaultCfg()
	cfg.voting = "40s"
	d := s.createStake(t, a, cfg, src.electorate(), "")
	pid := s.submit(t, a, d, "multi", nil)

	t.Run("vote and exit in one tx: the ballot weighs 0", func(t *testing.T) {
		s.runMsgs(t, a, 2_500_000, nil, nil, voteMsg(a.FormattedAddress(), pid, "YES"), src.exitMsg(a.FormattedAddress()))
		b := s.mustBallot(t, pid, a.FormattedAddress())
		require.Equal(t, []string{"YES", "0"}, []string{b.Option, b.Weight})
		s.requireTallyOf(t, pid, 0, 0, 0)
	})
	t.Run("exit, rejoin and vote in one tx: the vote weighs 0, the tx fails", func(t *testing.T) {
		staked := src.staked(t)
		s.refuseMsgs(t, b, daoCode(1119), src.exitMsg(b.FormattedAddress()), src.joinMsg(b.FormattedAddress(), 1_000), voteMsg(b.FormattedAddress(), pid, "YES"))
		require.Equal(t, staked.String(), src.staked(t).String(), "the failed tx reverted the exit too")
		_, found := s.ballot(t, pid, b.FormattedAddress())
		require.False(t, found)
	})
	t.Run("create-dao and submit in one tx", func(t *testing.T) {
		c := s.user(t)
		next := s.daos + 1
		res := s.runMsgs(t, c, 1_500_000, []string{daoAddress(next)}, s.creationFlows(c.FormattedAddress()),
			s.createMsg(c, "DAO_TYPE_STATIC_LIST", "one tx", defaultCfg().gogo(), addrs(c), nil),
			submitMsg(c.FormattedAddress(), next, "right away", nil, false))
		require.Equal(t, daoU64(next), idOf(s.event(t, res, "EventDaoCreated")))
		s.daos = next
		s.submitted(t, res, daoRef{id: next, addr: daoAddress(next)}, c.FormattedAddress())
	})
	t.Run("a whale cannot settle expedited in its submit tx, only after the delay", func(t *testing.T) {
		wsrc := stakeSource{s: s, sr: s.newSR(t, daoSrLock, s.stk, daoSrDuration)}
		whale, small := s.users(t, 1, daoUserFunds, coins(7_000, s.stk)...)[0], s.users(t, 1, daoUserFunds, coins(3_000, s.stk)...)[0]
		wsrc.join(t, whale, 7_000)
		wsrc.join(t, small, 3_000)
		wcfg := defaultCfg()
		wcfg.allowExpedite = true
		wcfg.voting, wcfg.expedited = "40s", "30s"
		wd := s.createStake(t, whale, wcfg, wsrc.electorate(), "")
		s.fund(t, wd.addr, ubze(100))
		next := s.proposals + 1
		send := bankSend(wd.addr, whale.FormattedAddress(), ubze(100))
		submit := submitMsg(whale.FormattedAddress(), wd.id, "instant", []map[string]any{send}, true)
		s.refuseMsgs(t, whale, refusal{codespace: daoModule, code: 1132, watch: []string{wd.addr}},
			submit, voteMsg(whale.FormattedAddress(), next, "YES"), settleMsg(whale.FormattedAddress(), next))

		// submitted and voted in one tx, it settles early once the delay is over
		res := s.runMsgs(t, whale, 2_500_000, []string{wd.addr}, nil, submit, voteMsg(whale.FormattedAddress(), next, "YES"))
		require.Equal(t, daoU64(next), idOf(s.event(t, res, "EventProposalSubmitted")))
		s.proposals = next
		p := s.prop(t, next)
		require.Equal(t, res.Height, int64(p.SubmitHeight))
		s.waitTime(t, int64(p.SubmitTime)+daoExpediteDelaySecs)
		out := s.settle(t, small, next, "EXECUTED", []string{wd.addr}, []daoFlow{flow(wd.addr, whale.FormattedAddress(), ubze(100))})
		require.Equal(t, "true", s.event(t, out, "EventProposalSettled").Attributes["expedited_early"])
		_, now := s.blockTime(t)
		require.Less(t, now, int64(p.VotingEnd), "settled before the voting end")
	})
	s.waitVotingEnd(t, pid)
	s.settleFirst(t, a, pid, "REJECTED")
	src.exit(t, b)
}

// E2. Settle gas griefing: a settle without enough gas fails as a whole and
// leaves the proposal as it was; no gas limit turns a passing proposal into
// EXECUTION_FAILED or EXPIRED.
func (s *daoSuite) settleGasGriefing(t *testing.T) {
	m := s.users(t, 2, daoUserFunds)
	griefer, rcpt := s.user(t), s.user(t)
	r := rcpt.FormattedAddress()
	cfg := defaultCfg()
	cfg.deposit, cfg.policy = coinPtr(1_000, chain.Denom), "ONLY_PASSED"
	d := s.createStatic(t, m[0], cfg, addrs(m...), "")
	s.fund(t, d.addr, ubze(1_000))

	light := []map[string]any{bankSend(d.addr, r, ubze(10))}
	heavy := []map[string]any{bankSend(d.addr, r, ubze(10)), bankSend(d.addr, r, ubze(10)), bankSend(d.addr, r, ubze(10))}
	twin := s.submit(t, m[0], d, "light twin", light)
	target := s.submit(t, m[0], d, "light target", light)
	heavyTwin := s.submit(t, m[1], d, "heavy twin", heavy)
	for _, pid := range []uint64{twin, target, heavyTwin} {
		s.vote(t, m[0], pid, "yes")
		s.vote(t, m[1], pid, "yes")
	}
	s.waitVotingEnd(t, heavyTwin)
	g := s.settleFirst(t, griefer, twin, "EXECUTED", flow(d.addr, r, ubze(10))).GasUsed
	s.grief(t, griefer, target, g, flow(d.addr, r, ubze(10)))

	gh := s.settleFirst(t, griefer, heavyTwin, "EXECUTED", flow(d.addr, r, ubze(10)), flow(d.addr, r, ubze(10)), flow(d.addr, r, ubze(10))).GasUsed
	heavyTarget := s.submit(t, m[1], d, "heavy target", heavy)
	s.vote(t, m[0], heavyTarget, "yes")
	s.vote(t, m[1], heavyTarget, "yes")
	s.waitVotingEnd(t, heavyTarget)
	s.grief(t, griefer, heavyTarget, gh, flow(d.addr, r, ubze(10)), flow(d.addr, r, ubze(10)), flow(d.addr, r, ubze(10)))
}

// grief settles a passing proposal with gas limits from half of what its
// twin used up to all of it. Every attempt either fails as a whole (the
// proposal, its deposit and the escrow unchanged) or executes it.
func (s *daoSuite) grief(t *testing.T, griefer ibc.Wallet, pid uint64, used int64, exec ...daoFlow) {
	t.Helper()
	before := s.prop(t, pid)
	for gas := used / 2; gas <= used+used/10; gas += used / 10 {
		escrow := s.allBalances(t, s.escrow)
		res, err := helpers.ExecTxWith(s.ctx, s.bze, griefer, []helpers.TxOption{helpers.WithGas(uint64(gas))}, "dao", "settle-proposal", daoU64(pid))
		require.NoError(t, err)
		status := s.prop(t, pid).status()
		if res.Code == 0 {
			require.Equal(t, "EXECUTED", status, "a settle that went through at gas %d must execute", gas)
			t.Logf("proposal %d: settles at %d gas (twin used %d)", pid, gas, used)
			return
		}
		require.Equal(t, "sdk", res.Codespace, res.RawLog)
		require.EqualValues(t, 11, res.Code, "gas %d: %s", gas, res.RawLog)
		require.Equal(t, "VOTING", status, "an out-of-gas settle at %d gas changes nothing", gas)
		require.Equal(t, mustJSON(before), mustJSON(s.prop(t, pid)))
		require.Equal(t, escrow.String(), s.allBalances(t, s.escrow).String())
	}
	s.settleFirst(t, griefer, pid, "EXECUTED", exec...)
}

// E3. The escrow (the dao module account) takes no sends and signs nothing;
// deposits leave it only by the refund or destination rule (I1 after every
// case).
func (s *daoSuite) escrowAccount(t *testing.T) {
	u, admin := s.user(t), s.user(t)
	m := s.users(t, 2, daoUserFunds)
	s.refuse(t, u, refusal{codespace: "sdk", code: 4, watch: []string{s.escrow}}, "bank", "send", u.FormattedAddress(), s.escrow, coinStr(1, chain.Denom))

	d := s.createStatic(t, m[0], defaultCfg(), addrs(m...), admin.FormattedAddress())
	s.fund(t, d.addr, ubze(100))
	s.adminRefuse(t, admin, d, daoCode(1111), bankSend(s.escrow, admin.FormattedAddress(), ubze(1)))
	s.refuse(t, m[0], daoCode(1111), "dao", "submit-proposal", d.idArg(), "steal the escrow", msgArgs(bankSend(s.escrow, m[0].FormattedAddress(), ubze(1)))[0])
	s.adminRefuse(t, admin, d, refusal{codespace: "sdk", code: 4}, bankSend(d.addr, s.escrow, ubze(1)))

	self := s.createStatic(t, m[0], defaultCfg(), addrs(m...), "")
	s.fund(t, self.addr, ubze(10))
	pid := s.submit(t, m[0], self, "into the escrow", []map[string]any{bankSend(self.addr, s.escrow, ubze(10))})
	s.pass(t, pid, m, "EXECUTION_FAILED")
	require.Equal(t, "10", s.bal(t, self.addr, chain.Denom).String())
}

// E4. Cross-DAO: a DAO acts on another only as its admin, at execution time;
// chains and cycles of admins.
func (s *daoSuite) crossDao(t *testing.T) {
	a := s.users(t, 2, daoUserFunds)
	rcpt := s.user(t)
	r := rcpt.FormattedAddress()
	cfg := defaultCfg()
	A := s.createStatic(t, a[0], cfg, addrs(a...), "")
	B := s.createStatic(t, a[0], cfg, addrs(a[0]), A.addr)
	C := s.createStatic(t, a[0], cfg, addrs(a[0]), B.addr)
	for _, d := range []daoRef{A, B, C} {
		s.fund(t, d.addr, ubze(100))
	}

	t.Run("a message signed by another dao is refused at submit", func(t *testing.T) {
		s.refuse(t, a[0], daoCode(1111), "dao", "submit-proposal", A.idArg(), "spend B directly", msgArgs(bankSend(B.addr, r, ubze(1)))[0])
	})
	t.Run("the parent's members cannot act on the child themselves", func(t *testing.T) {
		s.refuse(t, a[0], refusal{codespace: daoModule, code: 1109, watch: []string{B.addr}}, "dao", "admin-execute", B.idArg(), msgArgs(bankSend(B.addr, r, ubze(1)))[0])
	})
	t.Run("A spends B, and C through B", func(t *testing.T) {
		viaB := adminExecMsg(A.addr, B.id, bankSend(B.addr, r, ubze(10)))
		viaC := adminExecMsg(A.addr, B.id, adminExecMsg(B.addr, C.id, bankSend(C.addr, r, ubze(10))))
		pid := s.submit(t, a[0], A, "spend B and C", []map[string]any{viaB, viaC})
		s.pass(t, pid, a, "EXECUTED", flow(B.addr, r, ubze(10)), flow(C.addr, r, ubze(10)))
		ctl := s.controller(t, C.id)
		require.Equal(t, []string{A.addr, "SELF_OWNED"}, []string{ctl.Controller, ctl.EndReason})
		require.Equal(t, []string{C.addr, B.addr, A.addr}, ctl.Path)
	})
	t.Run("the child cannot reach its parent", func(t *testing.T) {
		s.refuse(t, a[0], daoCode(1111), "dao", "submit-proposal", B.idArg(), "spend A", msgArgs(bankSend(A.addr, r, ubze(1)))[0])
		pid := s.submit(t, a[0], B, "admin-execute A", []map[string]any{adminExecMsg(B.addr, A.id, bankSend(A.addr, r, ubze(1)))})
		s.pass(t, pid, a[:1], "EXECUTION_FAILED")
	})
	t.Run("the admin is checked at execution time", func(t *testing.T) {
		u, v := s.user(t), s.user(t)
		pcfg := defaultCfg()
		pcfg.voting = "30s"
		pcfg.deposit, pcfg.policy = coinPtr(1_000, chain.Denom), "ONLY_PASSED"
		A2 := s.createStatic(t, a[0], pcfg, addrs(a...), u.FormattedAddress())
		B2 := s.createStatic(t, a[0], cfg, addrs(a[0]), A2.addr)
		s.fund(t, B2.addr, ubze(100))
		pid := s.submit(t, a[0], A2, "spend B2", []map[string]any{adminExecMsg(A2.addr, B2.id, bankSend(B2.addr, r, ubze(10)))})
		s.vote(t, a[0], pid, "yes")
		s.vote(t, a[1], pid, "yes")
		s.adminExec(t, u, A2, nil, daoMsg("MsgNominateAdmin", map[string]any{"authority": A2.addr, "dao_id": B2.idArg(), "nominee": v.FormattedAddress()}))
		s.run(t, v, nil, nil, "dao", "accept-admin-nomination", B2.idArg())
		s.waitVotingEnd(t, pid)
		s.settleFirst(t, a[0], pid, "EXECUTION_FAILED") // passed: ONLY_PASSED refunds
		require.Equal(t, "100", s.bal(t, B2.addr, chain.Denom).String())
	})
	t.Run("a cycle", func(t *testing.T) {
		xID := s.daos + 1
		X := s.createStatic(t, a[0], cfg, addrs(a...), daoAddress(xID+1))
		Y := s.createStatic(t, a[1], cfg, addrs(a...), X.addr)
		ctl := s.controller(t, X.id)
		require.Equal(t, []string{"CYCLE", X.addr}, []string{ctl.EndReason, ctl.Controller})
		require.Equal(t, []string{X.addr, Y.addr}, ctl.Path)
		s.fund(t, X.addr, ubze(20))
		// both stay governed by their own voters; a nested admin-execute
		// around the cycle runs because each step is signed by the current admin
		pid := s.submit(t, a[0], X, "around the cycle", []map[string]any{adminExecMsg(X.addr, Y.id, adminExecMsg(Y.addr, X.id, bankSend(X.addr, r, ubze(20))))})
		s.pass(t, pid, a, "EXECUTED", flow(X.addr, r, ubze(20)))
	})
}

// E5. A DAO voting in another DAO: through its admin and its own proposal,
// counted once (I2), zeroed when removed.
func (s *daoSuite) daoAsVoter(t *testing.T) {
	u, w, x1 := s.user(t), s.user(t), s.user(t)
	y := s.users(t, 2, daoUserFunds)
	X := s.createStatic(t, x1, defaultCfg(), addrs(x1), u.FormattedAddress())
	ycfg := defaultCfg()
	ycfg.voting = "120s"
	Y := s.createStatic(t, y[0], ycfg, []string{y[0].FormattedAddress(), y[1].FormattedAddress(), X.addr}, w.FormattedAddress())
	pid := s.submit(t, y[0], Y, "a dao votes", nil)

	res := s.adminExec(t, u, X, nil, voteMsg(X.addr, pid, "YES"))
	require.Equal(t, "1", s.event(t, res, "EventVote").Attributes["weight"])
	s.requireTallyOf(t, pid, 1, 0, 0)

	again := s.submit(t, x1, X, "vote yes again", []map[string]any{voteMsg(X.addr, pid, "YES")})
	s.pass(t, again, []ibc.Wallet{x1}, "EXECUTION_FAILED")
	s.requireTallyOf(t, pid, 1, 0, 0)
	revote := s.submit(t, x1, X, "vote no", []map[string]any{voteMsg(X.addr, pid, "NO")})
	s.pass(t, revote, []ibc.Wallet{x1}, "EXECUTED")
	s.requireTallyOf(t, pid, 0, 1, 0)

	s.adminExec(t, w, Y, nil, membersMsg(Y, nil, []string{X.addr}))
	require.Equal(t, "0", s.mustBallot(t, pid, X.addr).Weight)
	s.requireTallyOf(t, pid, 0, 0, 0)
	s.cancel(t, y[0], pid)
}

// E6. Self-referencing electorates: a DAO cannot list its own address as a
// member, and a stake DAO's treasury staked in its own source never votes on
// its proposals nor counts in their denominator. Other DAOs as voters are E5.
func (s *daoSuite) selfReferencing(t *testing.T) {
	u, z1 := s.user(t), s.user(t)
	t.Run("own address as a static member is refused", func(t *testing.T) {
		next := s.daos + 1
		s.refuse(t, z1, daoCode(1104), "dao", "create-dao", "static-list", "self member", defaultCfg().cli(),
			"--members", z1.FormattedAddress()+","+daoAddress(next), "--admin", u.FormattedAddress())
		require.Equal(t, next-1, s.daos, "nothing was created")

		Z := s.createStatic(t, z1, defaultCfg(), addrs(z1), u.FormattedAddress())
		s.adminRefuse(t, u, Z, daoCode(1104), membersMsg(Z, []string{Z.addr}, nil))
		require.Equal(t, addrs(z1), s.dao(t, Z.id).Members)
	})
	t.Run("treasury stake in its own source never votes", func(t *testing.T) {
		sr := s.newSR(t, daoSrLock, s.stk, daoSrDuration)
		st1 := s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0]
		s.joinSR(t, st1, sr, 1_000)
		S := s.createStake(t, u, defaultCfg(), daoElectorate{SrIDs: []string{sr}}, u.FormattedAddress())
		s.fund(t, S.addr, coins(3_000, s.stk))
		join := map[string]any{"@type": "/bze.rewards.MsgJoinStaking", "creator": S.addr, "reward_id": sr, "amount": "3000"}
		s.adminExec(t, u, S, []daoFlow{flow(S.addr, s.rewards, coins(3_000, s.stk))}, join)
		pid := s.submit(t, st1, S, "treasury stake votes", nil)
		require.Equal(t, "1000", s.prop(t, pid).PinnedDenominator, "the treasury's stake stays out of the denominator")
		s.adminRefuse(t, u, S, daoCode(1119), voteMsg(S.addr, pid, "YES"))
		s.vote(t, st1, pid, "yes")
		s.waitVotingEnd(t, pid)
		s.requireTallyOf(t, pid, 1_000, 0, 0)
		s.settleFirst(t, u, pid, "EXECUTED")
	})
}

// E7. authz and feegrant: a grantee acts only for its granter and only
// within the grant; a DAO never acts for an address that did not grant it.
func (s *daoSuite) authzAndFeegrant(t *testing.T) {
	m := s.users(t, 3, daoUserFunds)
	grantee, attacker, victim, admin := s.user(t), s.user(t), s.user(t), s.user(t)
	g := grantee.FormattedAddress()
	cfg := defaultCfg()
	cfg.voting = "60s"
	d := s.createStatic(t, m[0], cfg, addrs(m...), "")
	pid := s.submit(t, m[0], d, "authz votes", nil)

	t.Run("a grantee votes for its granter, once", func(t *testing.T) {
		s.run(t, m[1], nil, nil, "authz", "grant", g, "generic", "--msg-type", "/bze.dao.MsgVote")
		s.runMsgs(t, grantee, 500_000, nil, nil, authzExec(g, voteMsg(m[1].FormattedAddress(), pid, "YES")))
		require.Equal(t, "1", s.mustBallot(t, pid, m[1].FormattedAddress()).Weight, "the vote counts as the granter's")
		_, found := s.ballot(t, pid, g)
		require.False(t, found, "the grantee casts no ballot of its own")
		s.requireTallyOf(t, pid, 1, 0, 0)
		s.refuseMsgs(t, grantee, daoCode(1125), authzExec(g, voteMsg(m[1].FormattedAddress(), pid, "YES")))
		s.refuse(t, grantee, daoCode(1119), "dao", "vote", daoU64(pid), "yes")
	})
	ad := s.createStatic(t, m[0], defaultCfg(), addrs(m[0]), admin.FormattedAddress())
	s.fund(t, ad.addr, ubze(100))
	t.Run("without a grant", func(t *testing.T) {
		a := attacker.FormattedAddress()
		s.refuseMsgs(t, attacker, refusal{codespace: "authz", watch: []string{ad.addr}}, authzExec(a, bankSend(ad.addr, a, ubze(10))))
		s.refuseMsgs(t, attacker, refusal{codespace: "authz", watch: []string{ad.addr}},
			authzExec(a, adminExecMsg(admin.FormattedAddress(), ad.id, bankSend(ad.addr, a, ubze(10)))))
	})
	t.Run("a proposal cannot spend for an address that granted nothing", func(t *testing.T) {
		theft := s.submit(t, m[0], d, "authz theft", []map[string]any{authzExec(d.addr, bankSend(victim.FormattedAddress(), m[0].FormattedAddress(), ubze(1)))})
		s.vote(t, m[0], theft, "yes")
		s.vote(t, m[2], theft, "yes")
		s.waitVotingEnd(t, theft)
		s.settleFirst(t, m[0], theft, "EXECUTION_FAILED")
	})
	t.Run("a dao grants authz and a fee allowance by proposal", func(t *testing.T) {
		gd := s.createStatic(t, m[0], defaultCfg(), addrs(m...), "")
		s.fund(t, gd.addr, ubze(1_000_000))
		grant := map[string]any{
			"@type": "/cosmos.authz.v1beta1.MsgGrant", "granter": gd.addr, "grantee": g,
			"grant": map[string]any{"authorization": map[string]any{"@type": "/cosmos.bank.v1beta1.SendAuthorization", "spend_limit": ubze(100)}},
		}
		allowance := map[string]any{
			"@type": "/cosmos.feegrant.v1beta1.MsgGrantAllowance", "granter": gd.addr, "grantee": g,
			"allowance": map[string]any{"@type": "/cosmos.feegrant.v1beta1.BasicAllowance", "spend_limit": ubze(50_000)},
		}
		gp := s.submit(t, m[0], gd, "grants", []map[string]any{grant, allowance})
		s.pass(t, gp, m[:2], "EXECUTED")

		s.runMsgs(t, grantee, 300_000, []string{gd.addr}, []daoFlow{flow(gd.addr, g, ubze(60))}, authzExec(g, bankSend(gd.addr, g, ubze(60))))
		s.refuseMsgs(t, grantee, refusal{codespace: "sdk", code: 5, watch: []string{gd.addr}}, authzExec(g, bankSend(gd.addr, g, ubze(60))))
		a := attacker.FormattedAddress()
		s.refuseMsgs(t, attacker, refusal{codespace: "authz", watch: []string{gd.addr}}, authzExec(a, bankSend(gd.addr, a, ubze(1))))

		// the fee of the grantee's own tx comes out of the treasury, within the allowance
		res := s.runWith(t, grantee, []helpers.TxOption{helpers.WithFlags("--fee-granter", gd.addr)}, []string{gd.addr},
			[]daoFlow{flow(g, victim.FormattedAddress(), ubze(1))}, "bank", "send", g, victim.FormattedAddress(), coinStr(1, chain.Denom))
		payer, _ := s.feeOf(t, res)
		require.Equal(t, gd.addr, payer, "the dao paid the fee")
		// 10,000,000 gas costs a 100,000 fee, above what is left of the allowance
		over := []string{"bank", "send", g, victim.FormattedAddress(), coinStr(1, chain.Denom)}
		s.refuseSend(t, grantee, refusal{codespace: "feegrant", watch: []string{gd.addr}}, over, func() (helpers.TxResult, error) {
			return helpers.ExecTxWith(s.ctx, s.bze, grantee, []helpers.TxOption{helpers.WithGas(10_000_000), helpers.WithFlags("--fee-granter", gd.addr)}, over...)
		})
		// no allowance at all: the ante returns the SDK's not-found error
		s.refuse(t, attacker, refusal{codespace: "sdk", reason: "fee-grant not found", watch: []string{gd.addr}}, "bank", "send", attacker.FormattedAddress(), victim.FormattedAddress(), coinStr(1, chain.Denom),
			"--fee-granter", gd.addr)
	})
	s.waitVotingEnd(t, pid)
	s.settleFirst(t, m[0], pid, "REJECTED")
}

// E8. Denoms and amounts: costs and deposits in a factory denom, a
// multi-coin spend, zero and negative amounts, spends beyond the treasury.
func (s *daoSuite) denomsAndAmounts(t *testing.T) {
	m := s.users(t, 2, daoUserFunds, coins(10_000, s.tok)...)
	cfg := defaultCfg()
	cfg.cost, cfg.deposit, cfg.policy = coinPtr(100, s.tok), coinPtr(500, s.tok), "ONLY_PASSED"
	d := s.createStatic(t, m[0], cfg, addrs(m...), "")
	pid := s.submit(t, m[0], d, "tok cost", nil)
	s.vote(t, m[0], pid, "no")
	s.vote(t, m[1], pid, "no")
	s.waitVotingEnd(t, pid)
	s.settleFirst(t, m[1], pid, "REJECTED") // the tok deposit goes to the burner

	tcfg := defaultCfg()
	tcfg.cost, tcfg.costDest, tcfg.deposit = coinPtr(100, s.tok), "TREASURY", coinPtr(500, s.tok)
	dt := s.createStatic(t, m[0], tcfg, addrs(m...), "")
	s.fund(t, dt.addr, sdk.NewCoins(sdk.NewInt64Coin(chain.Denom, 50), sdk.NewInt64Coin(s.tok, 70)))
	multi := sdk.NewCoins(sdk.NewInt64Coin(chain.Denom, 50), sdk.NewInt64Coin(s.tok, 170))
	pid = s.submit(t, m[0], dt, "multi-coin spend", []map[string]any{bankSend(dt.addr, m[1].FormattedAddress(), multi)})
	s.pass(t, pid, m, "EXECUTED", flow(dt.addr, m[1].FormattedAddress(), multi)) // 70 tok + the 100 tok cost
	require.True(t, s.allBalances(t, dt.addr).IsZero())

	zcfg := defaultCfg()
	zcfg.cost, zcfg.deposit = coinPtr(0, chain.Denom), coinPtr(0, chain.Denom)
	dz := s.createStatic(t, m[0], zcfg, addrs(m...), "")
	pid = s.submit(t, m[0], dz, "zero cost and deposit", nil)
	require.True(t, s.prop(t, pid).PinnedDeposit.coins().IsZero(), "a zero deposit escrows nothing")

	s.fund(t, dz.addr, ubze(10))
	zero := map[string]any{"@type": "/cosmos.bank.v1beta1.MsgSend", "from_address": dz.addr, "to_address": m[0].FormattedAddress(),
		"amount": []map[string]string{{"denom": chain.Denom, "amount": "0"}}}
	negative := map[string]any{"@type": "/cosmos.bank.v1beta1.MsgSend", "from_address": dz.addr, "to_address": m[0].FormattedAddress(),
		"amount": []map[string]string{{"denom": chain.Denom, "amount": "-5"}}}
	partial := []map[string]any{bankSend(dz.addr, m[0].FormattedAddress(), ubze(5)), bankSend(dz.addr, m[0].FormattedAddress(), ubze(1_000))}
	var pids []uint64
	for _, msgs := range [][]map[string]any{{zero}, {negative}, partial} {
		pids = append(pids, s.submitOrRefuse(t, m[0], dz, msgs))
	}
	for _, p := range pids {
		if p == 0 {
			continue
		}
		s.vote(t, m[0], p, "yes")
		s.vote(t, m[1], p, "yes")
	}
	for _, p := range pids {
		if p == 0 {
			continue
		}
		s.waitVotingEnd(t, p)
		s.settleFirst(t, m[0], p, "EXECUTION_FAILED")
	}
	require.Equal(t, "10", s.bal(t, dz.addr, chain.Denom).String(), "no failed execution moved anything")
	s.closeOpen(t, m[0], pid)
}

// submitOrRefuse submits a proposal whose messages the CLI may refuse to
// encode (a zero or negative coin); it returns 0 when nothing was submitted.
func (s *daoSuite) submitOrRefuse(t *testing.T, proposer ibc.Wallet, d daoRef, msgs []map[string]any) uint64 {
	t.Helper()
	args := append([]string{"dao", "submit-proposal", d.idArg(), "odd amounts"}, msgArgs(msgs...)...)
	before := s.allBalances(t, proposer.FormattedAddress())
	res, err := helpers.ExecTxWith(s.ctx, s.bze, proposer, []helpers.TxOption{helpers.WithGas(daoRefuseGas)}, args...)
	if err != nil || res.Code != 0 {
		t.Logf("refused before a proposal was stored: %v %s", err, res.RawLog)
		if err != nil {
			require.Equal(t, before.String(), s.allBalances(t, proposer.FormattedAddress()).String())
		}
		return 0
	}
	return s.submitted(t, res, d, proposer.FormattedAddress())
}

// E10. The proposer leaves mid-vote: the proposal goes on, their ballot is
// zeroed, and a refund still reaches them.
func (s *daoSuite) proposerLeaves(t *testing.T) {
	admin := s.user(t)
	m := s.users(t, 3, daoUserFunds)
	cfg := defaultCfg()
	cfg.deposit = coinPtr(1_000, chain.Denom)
	// the removal and the exit must land before voting_end: from then on the tally is final
	cfg.voting = "40s"
	d := s.createStatic(t, m[0], cfg, addrs(m...), admin.FormattedAddress())
	pid := s.submit(t, m[1], d, "the proposer leaves", nil)
	s.vote(t, m[1], pid, "yes")
	s.vote(t, m[0], pid, "yes")
	s.adminExec(t, admin, d, nil, membersMsg(d, nil, addrs(m[1])))
	require.Equal(t, "0", s.mustBallot(t, pid, m[1].FormattedAddress()).Weight)
	s.waitVotingEnd(t, pid)
	s.settleFirst(t, m[0], pid, "REJECTED") // 1 of 3 misses the quorum; ALWAYS refunds the former member

	sr := s.newSR(t, daoSrLock, s.stk, daoSrDuration)
	p1, p2 := s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0], s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0]
	s.joinSR(t, p1, sr, 1_000)
	s.joinSR(t, p2, sr, 1_000)
	ds := s.createStake(t, p1, cfg, daoElectorate{SrIDs: []string{sr}}, "")
	pid = s.submit(t, p1, ds, "the staker leaves", nil)
	s.vote(t, p1, pid, "yes")
	s.vote(t, p2, pid, "yes")
	s.exitSR(t, p1, sr)
	require.Equal(t, "0", s.mustBallot(t, pid, p1.FormattedAddress()).Weight)
	s.waitVotingEnd(t, pid)
	s.settleFirst(t, p2, pid, "EXECUTED") // 1000 of 2000 meets the quorum
}

// E11. Cancel games: from voting_end on the outcome is decided, so the
// proposer can neither withdraw a passed proposal (vetoing the electorate)
// nor rescue the deposit of a losing one; both settle under the pinned
// policy. A cancel that frees a slot costs the deposit.
func (s *daoSuite) cancelGames(t *testing.T) {
	m := s.users(t, 3, daoUserFunds)
	for _, policy := range []string{"ONLY_PASSED", "ALWAYS"} {
		t.Run(policy, func(t *testing.T) {
			cfg := defaultCfg()
			cfg.deposit, cfg.policy = coinPtr(1_000, chain.Denom), policy
			d := s.createStatic(t, m[0], cfg, addrs(m...), "")
			losing := s.submit(t, m[0], d, "losing", nil)
			passed := s.submit(t, m[0], d, "passed", nil)
			s.vote(t, m[1], losing, "no")
			s.vote(t, m[2], losing, "no")
			s.vote(t, m[1], passed, "yes")
			s.vote(t, m[2], passed, "yes")
			s.waitVotingEnd(t, passed)
			for _, pid := range []uint64{losing, passed} {
				s.refuse(t, m[0], daoCode(1124), "dao", "cancel-proposal", daoU64(pid))
				s.requireStatus(t, pid, "VOTING")
			}
			s.settleFirst(t, m[1], losing, "REJECTED")
			s.settleFirst(t, m[1], passed, "EXECUTED")
		})
	}
	t.Run("freeing a slot costs the deposit", func(t *testing.T) {
		cfg := defaultCfg()
		cfg.deposit, cfg.policy = coinPtr(1_000, chain.Denom), "NEVER"
		d := s.createStatic(t, m[0], cfg, addrs(m...), "")
		var open []uint64
		for i := 0; i < daoMaxOpen; i++ {
			open = append(open, s.submit(t, m[0], d, "slot "+strconv.Itoa(i), nil))
		}
		s.refuse(t, m[0], daoCode(1120), "dao", "submit-proposal", d.idArg(), "no slot")
		s.cancel(t, m[0], open[0]) // the 1000 deposit is burned
		open[0] = s.submit(t, m[0], d, "resubmitted", nil)
		for _, pid := range open {
			s.cancel(t, m[0], pid)
		}
	})
}

// E12. Open-proposal slot griefing: an outsider fills every slot of an
// open-submission DAO; cost and deposit price it, and the members act again
// once the outsider's proposals are settled.
func (s *daoSuite) slotGriefing(t *testing.T) {
	m := s.users(t, 2, daoUserFunds)
	outsider := s.user(t)
	cfg := defaultCfg()
	cfg.open = true
	cfg.cost, cfg.deposit, cfg.policy, cfg.keptDest = coinPtr(1_000, chain.Denom), coinPtr(5_000, chain.Denom), "ONLY_PASSED", "TREASURY"
	d := s.createStatic(t, m[0], cfg, addrs(m...), "")
	var spam []uint64
	for i := 0; i < daoMaxOpen; i++ {
		spam = append(spam, s.submit(t, outsider, d, "spam "+strconv.Itoa(i), nil))
	}
	s.refuse(t, m[0], daoCode(1120), "dao", "submit-proposal", d.idArg(), "blocked")
	for _, pid := range spam {
		s.vote(t, m[0], pid, "no")
		s.vote(t, m[1], pid, "no")
	}
	s.waitVotingEnd(t, spam[len(spam)-1])
	for _, pid := range spam {
		s.settleFirst(t, m[0], pid, "REJECTED")
	}
	require.Equal(t, strconv.Itoa(daoMaxOpen*5_000), s.bal(t, d.addr, chain.Denom).String(), "the kept deposits fund the treasury")
	pid := s.submit(t, m[0], d, "members act again", nil)
	s.cancel(t, m[0], pid)
}
