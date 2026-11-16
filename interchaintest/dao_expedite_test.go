package interchaintest_test

import (
	"context"
	"testing"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
)

const (
	// a min_expedite_delay longer than any case below, so a refusal by the
	// delay cannot be mistaken for one by the bars and the static-list settle
	// can only be early because the delay does not apply
	dxDelay     = "200s"
	dxDelaySecs = 200

	dxVoting    = "300s"
	dxExpedited = "240s"

	dxSend = 100 // ubze the council sends from its treasury
)

// TestStaticDaoExpedite checks that a static-list DAO acts at once while a stake DAO
// keeps min_expedite_delay: a council of five members with expedite bars 0.5
// (threshold) and 0.6 (quorum) runs an expedited treasury send as soon as the
// third member voted yes, while a stake DAO whose single voter meets the same
// bars is refused as too early. It runs on its own network, with the x/dao
// genesis of TestDao and a long min_expedite_delay.
func TestStaticDaoExpedite(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	genesis := append(daoGenesis(), cosmos.NewGenesisKV("app_state.dao.params.min_expedite_delay", dxDelay))
	net := chain.Start(t, ctx, chain.WithGenesis(genesis...))
	s := setupDao(t, ctx, net)
	require.Equal(t, dxDelay, s.queryParams(t).MinExpediteDelay)

	t.Run("1 a static-list DAO settles at once", s.staticListActsAtOnce)
	t.Run("2 a stake DAO keeps the delay", s.stakeDaoKeepsDelay)
}

func expediteCfg() daoCfg {
	cfg := defaultCfg()
	cfg.allowExpedite = true
	cfg.voting, cfg.expedited = dxVoting, dxExpedited
	cfg.expThreshold, cfg.expQuorum = "0.5", "0.6"
	return cfg
}

// 1. Five members, three yes: the proposal is settleable right after the
// third vote and the next tx settles and runs it, long before
// submit_time + min_expedite_delay.
func (s *daoSuite) staticListActsAtOnce(t *testing.T) {
	m := s.users(t, 5, daoUserFunds)
	anyone, rcpt := s.user(t), s.user(t)
	d := s.createStatic(t, m[0], expediteCfg(), addrs(m...), "")
	s.fund(t, d.addr, ubze(dxSend))

	send := bankSend(d.addr, rcpt.FormattedAddress(), ubze(dxSend))
	pid := s.submit(t, m[0], d, "council send", []map[string]any{send}, "--expedited")
	p := s.prop(t, pid)
	require.True(t, p.Expedited)

	s.vote(t, m[0], pid, "yes")
	s.vote(t, m[1], pid, "yes")
	require.False(t, s.proposal(t, pid).ExpediteSettleable, "2 of 5 misses the expedite quorum")
	s.refuse(t, anyone, daoCode(1130), "dao", "settle-proposal", daoU64(pid))

	s.vote(t, m[2], pid, "yes")
	require.True(t, s.proposal(t, pid).ExpediteSettleable, "3 of 5 clears both bars (quorum >= 0.6, yes > 0.5)")
	out := s.settleFirst(t, anyone, pid, "EXECUTED", flow(d.addr, rcpt.FormattedAddress(), ubze(dxSend)))
	require.Equal(t, "true", s.event(t, out, "EventProposalSettled").Attributes["expedited_early"])

	_, now := s.blockTime(t)
	require.Less(t, now, int64(p.SubmitTime)+dxDelaySecs, "settled before min_expedite_delay ran out")
	s.requireInvariants(t)
}

// 2. A stake DAO whose only voter holds all the power meets the bars at once
// but is refused as too early (1132) while the delay runs.
func (s *daoSuite) stakeDaoKeepsDelay(t *testing.T) {
	src := stakeSource{s: s, sr: s.newSR(t, daoSrLock, s.stk, daoSrDuration)}
	voter := s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0]
	anyone := s.user(t)
	src.join(t, voter, 1_000)
	d := s.createStake(t, voter, expediteCfg(), src.electorate(), "")

	pid := s.submit(t, voter, d, "stake send", nil, "--expedited")
	p := s.prop(t, pid)
	s.vote(t, voter, pid, "yes")
	require.False(t, s.proposal(t, pid).ExpediteSettleable, "the bars are met, the delay is not over")
	s.refuse(t, anyone, daoCode(1132), "dao", "settle-proposal", daoU64(pid))
	s.requireStatus(t, pid, "VOTING")

	_, now := s.blockTime(t)
	require.Less(t, now, int64(p.SubmitTime)+dxDelaySecs, "refused within the delay")
	s.requireInvariants(t)
}
