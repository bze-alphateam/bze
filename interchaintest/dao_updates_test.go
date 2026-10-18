package interchaintest_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// configMsg is a MsgUpdateDaoConfig signed by the DAO, for the CLI (ratios as
// 18-decimal integers).
func configMsg(d daoRef, cfg daoCfg) map[string]any {
	return daoMsg("MsgUpdateDaoConfig", map[string]any{"authority": d.addr, "dao_id": d.idArg(), "config": cfg.fields(ratioInt)})
}

func metadataMsg(d daoRef, name, description string) map[string]any {
	return daoMsg("MsgUpdateMetadata", map[string]any{"authority": d.addr, "dao_id": d.idArg(), "name": name, "description": description})
}

func membersMsg(d daoRef, add, remove []string) map[string]any {
	return daoMsg("MsgUpdateMembers", map[string]any{"authority": d.addr, "dao_id": d.idArg(), "add": add, "remove": remove})
}

func electorateMsg(d daoRef, el *daoElectorate) map[string]any {
	m := daoMsg("MsgUpdateElectorate", map[string]any{"authority": d.addr, "dao_id": d.idArg()})
	if el != nil {
		m["electorate"] = el
	}
	return m
}

// adminRefuse requires an admin-execute of msgs to be refused with r.
func (s *daoSuite) adminRefuse(t *testing.T, admin ibc.Wallet, d daoRef, r refusal, msgs ...map[string]any) {
	t.Helper()
	r.watch = append(r.watch, d.addr)
	s.refuse(t, admin, r, append([]string{"dao", "admin-execute", d.idArg()}, msgArgs(msgs...)...)...)
}

// A10. update-dao-config and update-metadata: through admin-execute and a
// proposal, never signed directly, D9 and every invalid field.
func (s *daoSuite) updateConfigAndMetadata(t *testing.T) {
	admin := s.user(t)
	m := s.users(t, 3, daoUserFunds)
	d := s.createStatic(t, m[0], defaultCfg(), addrs(m...), admin.FormattedAddress())

	t.Run("D9 open proposals keep their pinned config", func(t *testing.T) {
		old := s.submit(t, m[0], d, "old config", nil)
		strict := defaultCfg()
		strict.quorum, strict.threshold, strict.expQuorum, strict.expThreshold = "1", "1", "1", "1"
		res := s.adminExec(t, admin, d, nil, configMsg(d, strict))
		var fields []string
		require.NoError(t, json.Unmarshal([]byte(s.event(t, res, "EventDaoConfigUpdated").Attributes["fields"]), &fields))
		require.Equal(t, []string{"min_quorum", "threshold", "expedite_quorum", "expedite_threshold"}, fields)
		require.Equal(t, "1", ratio(t, s.dao(t, d.id).Config.Threshold).TruncateInt().String())
		require.Equal(t, "0.500000000000000000", ratio(t, s.prop(t, old).PinnedConfig.Threshold).String())
		next := s.submit(t, m[0], d, "new config", nil)
		require.Equal(t, s.dao(t, d.id).Config, s.prop(t, next).PinnedConfig)
		for _, pid := range []uint64{old, next} {
			s.vote(t, m[0], pid, "yes")
			s.vote(t, m[1], pid, "yes")
			s.vote(t, m[2], pid, "no")
		}
		s.waitVotingEnd(t, next)
		s.settleFirst(t, m[0], old, "EXECUTED")
		s.settleFirst(t, m[0], next, "REJECTED")
		s.adminExec(t, admin, d, nil, configMsg(d, defaultCfg()))
	})

	t.Run("invalid configs", func(t *testing.T) {
		for name, mutate := range map[string]func(c *daoCfg){
			"zero quorum":                   func(c *daoCfg) { c.quorum = "0" },
			"expedite quorum below quorum":  func(c *daoCfg) { c.expQuorum = "0.4" },
			"expedited period above voting": func(c *daoCfg) { c.expedited = "16s" },
			"voting period below min":       func(c *daoCfg) { c.voting, c.expedited = "3s", "3s" },
			"voting period above max":       func(c *daoCfg) { c.voting = "601s" },
			"negative deposit":              func(c *daoCfg) { c.deposit = coinPtr(-5, chain.Denom) },
		} {
			t.Run(name, func(t *testing.T) {
				cfg := defaultCfg()
				mutate(&cfg)
				s.adminRefuse(t, admin, d, daoCode(1103), configMsg(d, cfg))
			})
		}
	})

	t.Run("signed directly", func(t *testing.T) {
		s.refuse(t, admin, daoCode(1111), "dao", "update-dao-config", d.idArg(), defaultCfg().cli())
		s.refuse(t, m[0], daoCode(1111), "dao", "update-metadata", d.idArg(), "name", "description")
		s.refuse(t, admin, daoCode(1108), "dao", "update-dao-config", "999999", defaultCfg().cli())
	})

	t.Run("metadata", func(t *testing.T) {
		s.adminExec(t, admin, d, nil, metadataMsg(d, strings.Repeat("n", 64), strings.Repeat("d", 2_000)))
		rec := s.dao(t, d.id)
		require.Len(t, rec.Name, 64)
		require.Len(t, rec.Description, 2_000)
		s.adminRefuse(t, admin, d, daoCode(1102), metadataMsg(d, strings.Repeat("n", 65), ""))
		s.adminRefuse(t, admin, d, daoCode(1102), metadataMsg(d, "", ""))
		s.adminRefuse(t, admin, d, daoCode(1102), metadataMsg(d, "x", strings.Repeat("d", 2_001)))
		res := s.adminExec(t, admin, d, nil, metadataMsg(d, "renamed", ""))
		s.event(t, res, "EventDaoMetadataUpdated")
		require.Equal(t, "renamed", s.dao(t, d.id).Name)
		require.Empty(t, s.dao(t, d.id).Description)
	})

	t.Run("through a proposal", func(t *testing.T) {
		self := s.createStatic(t, m[0], defaultCfg(), addrs(m...), "")
		cfg := defaultCfg()
		cfg.voting = "20s"
		pid := s.submit(t, m[0], self, "config and name", []map[string]any{configMsg(self, cfg), metadataMsg(self, "voted", "by its members")})
		s.pass(t, pid, m[:2], "EXECUTED")
		rec := s.dao(t, self.id)
		require.Equal(t, "20s", rec.Config.VotingPeriod)
		require.Equal(t, "voted", rec.Name)
		// an invalid config is accepted at submit (inner messages are not
		// validated then) and fails at execution, changing nothing
		bad := defaultCfg()
		bad.quorum = "0"
		pid = s.submit(t, m[0], self, "bad config", []map[string]any{configMsg(self, bad), metadataMsg(self, "not applied", "")})
		s.pass(t, pid, m[:2], "EXECUTION_FAILED")
		require.Equal(t, "voted", s.dao(t, self.id).Name, "a failed execution applies none of its messages")
	})
}

// A10. update-members: repeated add/remove, D21 and the member bounds.
func (s *daoSuite) updateMembers(t *testing.T) {
	admin := s.user(t)
	m := s.users(t, 4, daoUserFunds)
	d := s.createStatic(t, m[0], defaultCfg(), addrs(m[0], m[1], m[2]), admin.FormattedAddress())
	other := s.createStatic(t, m[1], defaultCfg(), addrs(m[1], m[2]), "")
	pid := s.submit(t, m[0], d, "members vote", nil)
	otherPid := s.submit(t, m[1], other, "other dao", nil)
	s.vote(t, m[0], pid, "yes")
	s.vote(t, m[1], pid, "yes")
	s.vote(t, m[2], pid, "no")
	s.vote(t, m[1], otherPid, "yes")
	s.requireTallyOf(t, pid, 2, 1, 0)

	// D21: a removed member's ballot drops to 0 at once
	res := s.adminExec(t, admin, d, nil, membersMsg(d, nil, addrs(m[1])))
	updated := s.event(t, res, "EventDaoMembersUpdated")
	require.Equal(t, mustJSON(addrs(m[1])), updated.Attributes["removed"])
	reduced := s.event(t, res, "EventBallotReduced")
	require.Equal(t, []string{daoU64(pid), m[1].FormattedAddress(), "", "1"},
		[]string{reduced.Attributes["proposal_id"], reduced.Attributes["voter"], reduced.Attributes["source"], reduced.Attributes["amount"]})
	b := s.mustBallot(t, pid, m[1].FormattedAddress())
	require.Equal(t, []string{"YES", "0"}, []string{b.Option, b.Weight})
	s.requireTallyOf(t, pid, 1, 1, 0)
	require.Equal(t, "1", s.mustBallot(t, otherPid, m[1].FormattedAddress()).Weight, "other daos' ballots are untouched")
	s.refuse(t, m[1], daoCode(1119), "dao", "vote", daoU64(pid), "yes")

	// re-added, the member votes the same option again
	s.adminExec(t, admin, d, nil, membersMsg(d, addrs(m[1]), nil))
	require.Equal(t, "1", s.voteWeight(t, m[1], pid, "yes").String())
	s.requireTallyOf(t, pid, 2, 1, 0)

	// an added member votes on the open proposal, whose denominator stays pinned
	s.adminExec(t, admin, d, nil, membersMsg(d, addrs(m[3]), addrs(m[2])))
	require.Equal(t, "1", s.voteWeight(t, m[3], pid, "yes").String())
	p := s.prop(t, pid)
	s.requireTallyOf(t, pid, 3, 0, 0)
	require.Equal(t, "3", p.PinnedDenominator)
	t.Logf("a member added mid-vote voted: yes %s, pinned denominator %s", p.Yes, p.PinnedDenominator)

	t.Run("bounds", func(t *testing.T) {
		cur := s.dao(t, d.id).Members
		s.adminRefuse(t, admin, d, daoCode(1104), membersMsg(d, nil, cur))
		s.adminRefuse(t, admin, d, daoCode(1104), membersMsg(d, cur[:1], nil))
		s.adminRefuse(t, admin, d, daoCode(1104), membersMsg(d, nil, addrs(admin)))
		s.adminRefuse(t, admin, d, daoCode(1104), membersMsg(d, nil, nil))
		s.adminRefuse(t, admin, d, daoCode(1104), membersMsg(d, addrs(admin), addrs(admin)))
		s.adminRefuse(t, admin, d, daoCode(1104), membersMsg(d, addrs(admin, admin), nil))
		s.adminRefuse(t, admin, d, daoCode(1104), membersMsg(d, addrs(s.users(t, daoMaxMembers-len(cur)+1, 1)...), nil))
		fill := addrs(s.users(t, daoMaxMembers-len(cur), 1)...)
		s.adminExec(t, admin, d, nil, membersMsg(d, fill, nil))
		require.Len(t, s.dao(t, d.id).Members, daoMaxMembers)
		s.adminExec(t, admin, d, nil, membersMsg(d, nil, fill))
	})
	t.Run("signed directly", func(t *testing.T) {
		s.refuse(t, m[0], daoCode(1111), "dao", "update-members", d.idArg(), "--add", admin.FormattedAddress())
	})
	t.Run("a stake dao", func(t *testing.T) {
		st := s.createStake(t, admin, defaultCfg(), daoElectorate{}, admin.FormattedAddress())
		s.adminRefuse(t, admin, st, daoCode(1101), membersMsg(st, addrs(m[0]), nil))
	})
	t.Run("through a proposal", func(t *testing.T) {
		self := s.createStatic(t, m[0], defaultCfg(), addrs(m[0], m[1]), "")
		id := s.submit(t, m[0], self, "add a member", []map[string]any{membersMsg(self, addrs(m[2], m[3]), addrs(m[1]))})
		s.pass(t, id, m[:2], "EXECUTED")
		require.Equal(t, addrs(m[0], m[2], m[3]), s.dao(t, self.id).Members)
	})
	s.waitVotingEnd(t, pid)
	s.settleFirst(t, m[0], pid, "EXECUTED")
	s.waitVotingEnd(t, otherPid)
	s.settleFirst(t, m[1], otherPid, "EXECUTED") // 1 of 2 meets the quorum
}

// A10. update-electorate: add and remove sources, D11, D14, D25, D28, D29.
func (s *daoSuite) updateElectorate(t *testing.T) {
	fx := s.fixtures(t)
	admin := s.user(t)
	stakers := s.users(t, 3, daoUserFunds, coins(10_000, s.stk)...)
	srX := s.newSR(t, daoSrLock, s.stk, daoSrDuration)
	s.joinSR(t, stakers[0], srX, 1_000)
	d := s.createStake(t, admin, defaultCfg(), daoElectorate{SrIDs: []string{srX}}, admin.FormattedAddress())

	res := s.adminExec(t, admin, d, nil, electorateMsg(d, &daoElectorate{SrIDs: []string{srX, fx.srB}, DrDenom: s.stk}))
	e := s.event(t, res, "EventDaoElectorateUpdated")
	require.Equal(t, mustJSON([]string{srX, fx.srB}), e.Attributes["sr_ids"])
	require.Equal(t, s.stk, e.Attributes["dr_denom"])
	require.Equal(t, []string{srX, fx.srB}, s.dao(t, d.id).Electorate.SrIDs)
	s.adminExec(t, admin, d, nil, electorateMsg(d, &daoElectorate{SrIDs: []string{srX}}))
	require.Equal(t, []string{srX}, s.dao(t, d.id).Electorate.SrIDs)
	require.Empty(t, s.dao(t, d.id).Electorate.DrDenom)

	for name, el := range map[string]daoElectorate{
		"D14 duplicate ids":    {SrIDs: []string{srX, srX}},
		"D28 lock below min":   {SrIDs: []string{srX, fx.srLow}},
		"D29 other denom":      {SrIDs: []string{srX, fx.srOth}},
		"D29 other dr denom":   {SrIDs: []string{srX}, DrDenom: s.oth},
		"over max_sr_per_dao":  {SrIDs: []string{srX, fx.srA, fx.srB}},
		"missing reward":       {SrIDs: []string{"999999"}},
		"missing denom reward": {DrDenom: s.tok},
	} {
		t.Run(name, func(t *testing.T) {
			s.adminRefuse(t, admin, d, daoCode(1105), electorateMsg(d, &el))
		})
	}
	t.Run("a static dao", func(t *testing.T) {
		st := s.createStatic(t, admin, defaultCfg(), addrs(admin), admin.FormattedAddress())
		s.adminRefuse(t, admin, st, daoCode(1101), electorateMsg(st, &daoElectorate{SrIDs: []string{srX}}))
	})
	t.Run("signed directly", func(t *testing.T) {
		s.refuse(t, admin, daoCode(1111), "dao", "update-electorate", d.idArg(), mustJSON(daoElectorate{SrIDs: []string{srX}}))
	})
	t.Run("an external-admin dao may empty its electorate", func(t *testing.T) {
		s.adminExec(t, admin, d, nil, electorateMsg(d, nil))
		require.Nil(t, s.dao(t, d.id).Electorate)
		s.adminExec(t, admin, d, nil, electorateMsg(d, &daoElectorate{SrIDs: []string{srX}}))
	})

	t.Run("through a proposal, and D11", func(t *testing.T) {
		self := s.createStake(t, stakers[0], defaultCfg(), daoElectorate{SrIDs: []string{srX}}, "")
		pid := s.submit(t, stakers[0], self, "add a source", []map[string]any{electorateMsg(self, &daoElectorate{SrIDs: []string{srX, fx.srA}})})
		s.pass(t, pid, stakers[:1], "EXECUTED")
		require.Equal(t, []string{srX, fx.srA}, s.dao(t, self.id).Electorate.SrIDs)
		// a self-owned dao signs only through its proposals; emptying its
		// electorate fails there (ErrElectorateNotLive inside the execution)
		pid = s.submit(t, stakers[0], self, "empty myself", []map[string]any{electorateMsg(self, nil)})
		s.pass(t, pid, stakers[:1], "EXECUTION_FAILED")
		require.Equal(t, []string{srX, fx.srA}, s.dao(t, self.id).Electorate.SrIDs)
		// a source without stake leaves it as dead as an empty one
		pid = s.submit(t, stakers[0], self, "only idle", []map[string]any{electorateMsg(self, &daoElectorate{SrIDs: []string{fx.srIdle}})})
		s.pass(t, pid, stakers[:1], "EXECUTION_FAILED")
	})

	t.Run("D25 a removed source stays pinned by open proposals", func(t *testing.T) {
		srD := s.newSR(t, daoSrLock, s.stk, 1)
		srK := s.newSR(t, daoSrLock, s.stk, daoSrDuration)
		whale := s.users(t, 1, daoUserFunds, coins(1_000_000, s.stk)...)[0]
		s.joinSR(t, stakers[1], srD, 1_000)
		s.joinSR(t, stakers[2], srK, 1_000)
		cfg := defaultCfg()
		cfg.voting = "90s"
		dd := s.createStake(t, admin, cfg, daoElectorate{SrIDs: []string{srD, srK}}, admin.FormattedAddress())
		pid := s.submit(t, stakers[1], dd, "pins srD", nil)
		s.adminExec(t, admin, dd, nil, electorateMsg(dd, &daoElectorate{SrIDs: []string{srK}}))
		require.Equal(t, []string{srD, srK}, s.prop(t, pid).PinnedElectorate.SrIDs, "the proposal keeps its pinned electorate")

		join := s.joinSR(t, whale, srD, 1_000_000)
		h, found := s.joinRecord(t, srSource(srD), whale.FormattedAddress())
		require.True(t, found, "the proposal still pins srD, so the join is recorded")
		require.Equal(t, join.Height, h)
		s.refuse(t, whale, daoCode(1119), "dao", "vote", daoU64(pid), "yes")
		require.Equal(t, "1000", s.voteWeight(t, stakers[1], pid, "yes").String(), "stake from before the submit still counts")

		cur, err := helpers.CurrentEpoch(s.ctx, s.bze, "day")
		require.NoError(t, err)
		require.NoError(t, helpers.WaitForEpoch(s.ctx, s.bze, "day", cur+1))
		s.exitSR(t, stakers[1], srD)
		s.exitSR(t, whale, srD)
		require.True(t, s.srExists(t, srD), "the open proposal vetoes the deletion")
		s.refuse(t, s.funder, daoCode(1107), "rewards", "delete-staking-reward", srD)
		s.cancel(t, stakers[1], pid)
		s.mustExec(t, s.funder, "rewards", "delete-staking-reward", srD)
		require.False(t, s.srExists(t, srD), "the deletion is allowed once nothing pins the reward")
	})
}

// validatorWallet is the first validator's own key, which votes on gov
// proposals.
func (s *daoSuite) validatorWallet(t *testing.T) ibc.Wallet {
	t.Helper()
	v := s.bze.Validators[0]
	bech32, err := v.AccountKeyBech32(s.ctx, "validator")
	require.NoError(t, err)
	addr, err := sdk.GetFromBech32(bech32, chain.Bech32Prefix)
	require.NoError(t, err)
	return cosmos.NewWallet("validator", addr, "", s.bze.Config())
}

// govParams passes a MsgUpdateParams with p through gov and checks the chain holds p.
func (s *daoSuite) govParams(t *testing.T, proposer ibc.Wallet, p daoParams) {
	t.Helper()
	msg := map[string]any{"@type": "/bze.dao.MsgUpdateParams", "authority": s.gov, "params": p}
	_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, proposer, helpers.Proposal{
		Messages: []any{msg}, Title: "dao params", Summary: "dao params",
	})
	require.NoError(t, err)
	s.params = p
	require.Equal(t, mustJSON(p), mustJSON(s.queryParams(t)))
}

// A11 and E14. update-params: only the gov authority; through gov the new
// bounds apply to later messages only; caps, the execution window and the
// minimum lock changed while proposals are open.
func (s *daoSuite) updateParams(t *testing.T) {
	genesis := s.params
	u := s.user(t)
	m := s.users(t, 2, daoUserFunds)
	fx := s.fixtures(t)

	t.Run("not the gov authority", func(t *testing.T) {
		s.refuseMsgs(t, u, daoCode(1100), map[string]any{"@type": "/bze.dao.MsgUpdateParams", "authority": u.FormattedAddress(), "params": s.params})
	})

	t.Run("through gov with the CLI command: a zero creation fee", func(t *testing.T) {
		p := s.params
		p.DaoCreationFee = sdk.NewInt64Coin(chain.Denom, 0)
		res := s.mustExec(t, u, "dao", "update-params", mustJSON(p),
			"--deposit", chain.MinDeposit+chain.Denom, "--title", "free daos", "--summary", "zero creation fee")
		var id uint64
		for _, e := range helpers.TxEvents(res, "submit_proposal") {
			if v, ok := e.Attributes["proposal_id"]; ok {
				id = atoiU(t, v)
			}
		}
		require.NotZero(t, id)
		s.mustExec(t, s.validatorWallet(t), "gov", "vote", daoU64(id), "yes")
		require.NoError(t, helpers.WaitForProposalPassed(s.ctx, s.bze, id))
		s.params = p
		require.Equal(t, mustJSON(p), mustJSON(s.queryParams(t)))
		// A1: a zero fee moves nothing
		require.Empty(t, s.creationFlows(u.FormattedAddress()))
		s.createStatic(t, u, defaultCfg(), addrs(u), "")
	})

	t.Run("A1 the creation fee to the community pool", func(t *testing.T) {
		p := s.params
		p.DaoCreationFee = sdk.NewInt64Coin(chain.Denom, 2*daoCreationFee)
		p.DaoCreationFeeDestination = "community_pool"
		s.govParams(t, u, p)
		s.createStatic(t, u, defaultCfg(), addrs(u), "")
	})

	t.Run("E14 changes while proposals are open", func(t *testing.T) {
		d := s.createStatic(t, m[0], defaultCfg(), addrs(m...), "")
		cfgLong := defaultCfg()
		cfgLong.voting = "90s"
		long := s.createStatic(t, m[0], cfgLong, addrs(m...), "")
		var open []uint64
		for i := 0; i < daoMaxOpen; i++ {
			open = append(open, s.submit(t, m[0], long, "open "+strconv.Itoa(i)+" with a long title", nil))
		}
		s.fund(t, d.addr, ubze(10))
		failed := s.submit(t, m[0], d, "fails", []map[string]any{bankSend(d.addr, u.FormattedAddress(), ubze(1_000))})
		s.pass(t, failed, m, "EXECUTION_FAILED")
		before := int64(s.proposal(t, failed).RetryDeadline)
		voteEnd := int64(s.prop(t, failed).VotingEnd)
		require.Equal(t, voteEnd+daoExecWindowSecs, before)

		staker := s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0]
		s.joinSR(t, staker, fx.srC, 1_000)
		st := s.createStake(t, staker, defaultCfg(), daoElectorate{SrIDs: []string{fx.srC}}, staker.FormattedAddress())

		p := s.params
		p.MaxOpenProposals = 2
		p.ExecutionWindow = "5s"
		p.MinElectorateLockDays = 2
		p.MaxTitleLen = 20
		s.govParams(t, u, p)

		// the retry window is read at settle
		require.Equal(t, voteEnd+5, int64(s.proposal(t, failed).RetryDeadline))
		s.waitTime(t, voteEnd+5)
		s.settle(t, u, failed, "EXPIRED", nil, nil)

		// open proposals keep voting; new ones wait until the dao is under the new cap
		for _, pid := range open {
			s.requireStatus(t, pid, "VOTING")
		}
		s.refuse(t, m[0], daoCode(1120), "dao", "submit-proposal", long.idArg(), "capped")
		s.refuse(t, m[0], daoCode(1117), "dao", "submit-proposal", d.idArg(), strings.Repeat("t", 21))
		s.cancel(t, m[0], open[0])
		s.cancel(t, m[0], open[1])
		s.submit(t, m[0], long, "under the new cap", nil)
		s.vote(t, m[0], open[2], "yes")
		s.vote(t, m[1], open[2], "yes")

		// a raised minimum lock: a kept source stays, a new one is refused
		s.adminExec(t, staker, st, nil, electorateMsg(st, &daoElectorate{SrIDs: []string{fx.srC}, DrDenom: ""}))
		s.adminRefuse(t, staker, st, daoCode(1105), electorateMsg(st, &daoElectorate{SrIDs: []string{fx.srC, fx.srA}}))
		s.refuse(t, staker, daoCode(1105), "dao", "create-dao", "stake", "x", defaultCfg().cli(), "--electorate", mustJSON(daoElectorate{SrIDs: []string{fx.srC}}))
		s.submit(t, staker, st, "still works", nil)

		s.waitVotingEnd(t, open[2])
		s.settleFirst(t, m[0], open[2], "EXECUTED")
	})

	t.Run("back to the genesis params", func(t *testing.T) {
		s.govParams(t, u, genesis)
		time.Sleep(time.Second)
	})
}
