package interchaintest_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// cliQuery runs `bzed q dao <args>` and decodes the answer.
func cliQuery[T any](t *testing.T, s *daoSuite, args ...string) T {
	t.Helper()
	out, err := helpers.Query[T](s.ctx, s.bze, append([]string{"dao"}, args...)...)
	require.NoError(t, err, "q dao %v", args)
	return out
}

// cliQueryFails requires `bzed q dao <args>` to fail with reason.
func (s *daoSuite) cliQueryFails(t *testing.T, reason string, args ...string) {
	t.Helper()
	_, err := helpers.Query[json.RawMessage](s.ctx, s.bze, append([]string{"dao"}, args...)...)
	require.Error(t, err, "q dao %v", args)
	require.Contains(t, err.Error(), reason)
}

type daoListPage struct {
	Daos       []daoRec      `json:"daos"`
	Proposals  []daoProposal `json:"proposals"`
	Ballots    []daoBallot   `json:"ballots"`
	Pagination daoPage       `json:"pagination"`
}

func ids[T any](items []T, id func(T) helpers.Int64) []uint64 {
	out := make([]uint64, len(items))
	for i, it := range items {
		out[i] = uint64(id(it))
	}
	return out
}

func daoIDs(ds []daoRec) []uint64 { return ids(ds, func(d daoRec) helpers.Int64 { return d.ID }) }

func propIDs(ps []daoProposal) []uint64 {
	return ids(ps, func(p daoProposal) helpers.Int64 { return p.ID })
}

// B. Every query through the CLI, and one REST GET per route.
func (s *daoSuite) queries(t *testing.T) {
	m := s.users(t, 3, daoUserFunds)
	admin, stranger := s.user(t), s.user(t)

	t.Run("params", func(t *testing.T) {
		res := cliQuery[struct {
			Params daoParams `json:"params"`
		}](t, s, "params")
		// REST prints durations as Go durations ("10m0s"), the CLI as seconds ("600s")
		norm := func(p daoParams) string {
			for _, d := range []*string{&p.MinVotingPeriod, &p.MaxVotingPeriod, &p.ExecutionWindow} {
				v, err := time.ParseDuration(*d)
				require.NoError(t, err)
				*d = v.String()
			}
			return mustJSON(p)
		}
		require.Equal(t, norm(s.params), norm(res.Params))
		require.Equal(t, norm(s.params), norm(s.queryParams(t)))
	})

	d := s.createStatic(t, m[0], defaultCfg(), addrs(m...), admin.FormattedAddress())
	t.Run("dao and dao-by-address", func(t *testing.T) {
		res := cliQuery[struct {
			Dao daoRec `json:"dao"`
		}](t, s, "dao", d.idArg())
		rest := s.dao(t, d.id)
		require.Equal(t, rest.Address, res.Dao.Address)
		require.Equal(t, rest.Admin, res.Dao.Admin)
		require.Equal(t, rest.Members, res.Dao.Members)
		require.True(t, ratio(t, res.Dao.Config.MinQuorum).Equal(ratio(t, rest.Config.MinQuorum)), "the CLI prints 18-decimal integers, REST decimals")
		byAddr := cliQuery[struct {
			Dao daoRec `json:"dao"`
		}](t, s, "dao-by-address", d.addr)
		require.EqualValues(t, d.id, byAddr.Dao.ID)
		s.cliQueryFails(t, "not found", "dao", "999999")
		s.cliQueryFails(t, "no dao owns", "dao-by-address", stranger.FormattedAddress())
		_, err := helpers.Query[json.RawMessage](s.ctx, s.bze, "dao", "dao-by-address", "bze1invalid")
		require.Error(t, err)
		require.Equal(t, http.StatusNotFound, s.restGet(t, "/bze/dao/dao/999999", nil))
		require.Equal(t, http.StatusNotFound, s.restGet(t, "/bze/dao/dao_by_address/"+stranger.FormattedAddress(), nil))
		require.Equal(t, http.StatusBadRequest, s.restGet(t, "/bze/dao/dao_by_address/bze1invalid", nil))
		var rb struct {
			Dao daoRec `json:"dao"`
		}
		s.mustRest(t, "/bze/dao/dao_by_address/"+d.addr, &rb)
		require.EqualValues(t, d.id, rb.Dao.ID)
	})

	t.Run("dao-controller", func(t *testing.T) {
		ctl := cliQuery[daoController](t, s, "dao-controller", d.idArg())
		require.Equal(t, []string{admin.FormattedAddress(), "ACCOUNT"}, []string{ctl.Controller, enumOr(ctl.EndReason, "ACCOUNT")})
		require.Equal(t, []string{d.addr}, ctl.Path)
		self := s.createStatic(t, m[0], defaultCfg(), addrs(m[0]), "")
		ctl = cliQuery[daoController](t, s, "dao-controller", self.idArg())
		require.Equal(t, []string{self.addr, "SELF_OWNED"}, []string{ctl.Controller, ctl.EndReason})
		child := s.createStatic(t, m[0], defaultCfg(), addrs(m[0]), self.addr)
		rest := s.controller(t, child.id)
		require.Equal(t, []string{self.addr, "SELF_OWNED"}, []string{rest.Controller, rest.EndReason})
		require.Equal(t, []string{child.addr, self.addr}, rest.Path)
		s.cliQueryFails(t, "not found", "dao-controller", "999999")

		// DEPTH_CAP: a chain of 17 daos, each the admin of the one before it
		first := s.daos + 1
		var chainRefs []daoRef
		for i := uint64(0); i < 17; i++ {
			adminAddr := daoAddress(first + i + 1)
			if i == 16 {
				adminAddr = admin.FormattedAddress()
			}
			chainRefs = append(chainRefs, s.createStatic(t, m[0], defaultCfg(), addrs(m[0]), adminAddr))
		}
		ctl = cliQuery[daoController](t, s, "dao-controller", chainRefs[0].idArg())
		require.Equal(t, "DEPTH_CAP", ctl.EndReason)
		require.Len(t, ctl.Path, 16)
		require.Equal(t, chainRefs[15].addr, ctl.Controller, "the walk stops at the 16th dao")
		ctl = cliQuery[daoController](t, s, "dao-controller", chainRefs[1].idArg())
		require.Equal(t, []string{admin.FormattedAddress(), "ACCOUNT"}, []string{ctl.Controller, enumOr(ctl.EndReason, "ACCOUNT")})
		require.Len(t, ctl.Path, 16, "16 daos reach the account before the cap")
	})

	t.Run("proposal, ballot, ballots and their pages", func(t *testing.T) {
		pid := s.submit(t, m[0], d, "query me", nil)
		res := cliQuery[daoProposalResp](t, s, "proposal", daoU64(pid))
		require.Equal(t, "VOTING", res.Proposal.status())
		require.False(t, res.VotingEnded)
		require.Zero(t, int64(res.RetryDeadline))
		require.False(t, res.ExpediteSettleable)
		for i, v := range m {
			s.vote(t, v, pid, []string{"yes", "no", "abstain"}[i])
		}
		b := cliQuery[struct {
			Ballot daoBallot `json:"ballot"`
		}](t, s, "ballot", daoU64(pid), m[1].FormattedAddress())
		require.Equal(t, []string{"NO", "1"}, []string{b.Ballot.Option, b.Ballot.Weight})
		s.cliQueryFails(t, "no ballot", "ballot", daoU64(pid), stranger.FormattedAddress())
		s.cliQueryFails(t, "not found", "proposal", "999999")
		var rb struct {
			Ballot daoBallot `json:"ballot"`
		}
		s.mustRest(t, fmt.Sprintf("/bze/dao/ballot/%d/%s", pid, m[2].FormattedAddress()), &rb)
		require.Equal(t, "ABSTAIN", rb.Ballot.Option)

		all := cliQuery[daoListPage](t, s, "ballots", daoU64(pid), "--page-count-total")
		require.Len(t, all.Ballots, 3)
		require.Equal(t, "3", all.Pagination.Total)
		page1 := cliQuery[daoListPage](t, s, "ballots", daoU64(pid), "--page-limit", "2")
		require.Len(t, page1.Ballots, 2)
		require.NotEmpty(t, page1.Pagination.NextKey)
		page2 := cliQuery[daoListPage](t, s, "ballots", daoU64(pid), "--page-limit", "2", "--page-key", page1.Pagination.NextKey)
		require.Len(t, page2.Ballots, 1)
		require.Empty(t, page2.Pagination.NextKey)
		var voters []string
		for _, b := range append(page1.Ballots, page2.Ballots...) {
			voters = append(voters, b.Voter)
		}
		var allVoters []string
		for _, b := range all.Ballots {
			allVoters = append(allVoters, b.Voter)
		}
		require.Equal(t, allVoters, voters, "two pages joined by next_key list every ballot once, in order")
		rev := cliQuery[daoListPage](t, s, "ballots", daoU64(pid), "--page-reverse")
		require.Equal(t, allVoters[2], rev.Ballots[0].Voter)
		var empty daoListPage
		code := s.restGet(t, "/bze/dao/ballots/999999", &empty)
		require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, code)
		require.Empty(t, empty.Ballots)

		s.waitVotingEnd(t, pid)
		res = cliQuery[daoProposalResp](t, s, "proposal", daoU64(pid))
		require.True(t, res.VotingEnded, "voting_ended is set from the voting end on, while the proposal is in voting")
		s.settleFirst(t, m[0], pid, "EXECUTED") // 1 yes of 2 decided meets the 0.5 threshold, 3 of 3 the quorum
	})

	t.Run("voting-power and join-record", func(t *testing.T) {
		sr := s.newSR(t, daoSrLock, s.stk, daoSrDuration)
		early, late := s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0], s.users(t, 1, daoUserFunds, coins(500, s.stk)...)[0]
		s.joinSR(t, early, sr, 1_000)
		cfg := defaultCfg()
		cfg.voting = "30s"
		sd := s.createStake(t, early, cfg, daoElectorate{SrIDs: []string{sr}}, "")
		pid := s.submit(t, early, sd, "power", nil)
		join := s.joinSR(t, late, sr, 500)

		vp := cliQuery[daoVotingPower](t, s, "voting-power", sd.idArg(), late.FormattedAddress())
		require.Equal(t, "500", vp.Current)
		require.True(t, vp.eligible(t, pid).IsZero(), "an ineligible join shows 0 on the open proposal")
		jr := cliQuery[struct {
			Height helpers.Int64 `json:"height"`
		}](t, s, "join-record", srSource(sr), late.FormattedAddress())
		require.Equal(t, join.Height, int64(jr.Height))
		s.cliQueryFails(t, "no join record", "join-record", srSource(sr), early.FormattedAddress())
		require.Equal(t, http.StatusBadRequest, s.restGet(t, "/bze/dao/join_record?source=&address="+early.FormattedAddress(), nil))
		h, ok := s.joinRecord(t, srSource(sr), late.FormattedAddress())
		require.True(t, ok)
		require.Equal(t, join.Height, h)

		later := s.submit(t, early, sd, "later power", nil)
		vp = s.votingPower(t, sd.id, late.FormattedAddress())
		require.True(t, vp.eligible(t, pid).IsZero())
		require.Equal(t, "500", vp.eligible(t, later).String(), "the same stake has full weight on a later proposal")
		static := cliQuery[daoVotingPower](t, s, "voting-power", d.idArg(), m[0].FormattedAddress())
		require.Equal(t, "1", static.Current)
		require.Equal(t, "0", enumOr(cliQuery[daoVotingPower](t, s, "voting-power", d.idArg(), stranger.FormattedAddress()).Current, "0"))
		s.waitVotingEnd(t, later)
		s.settleFirst(t, early, pid, "REJECTED")
		s.settleFirst(t, early, later, "REJECTED")
	})

	t.Run("daos, daos-by-admin and proposals listings", func(t *testing.T) {
		total := cliQuery[daoListPage](t, s, "daos", "--page-count-total", "--page-limit", "1")
		require.Equal(t, daoU64(s.daos), total.Pagination.Total)
		require.EqualValues(t, 1, total.Daos[0].ID)
		p1 := cliQuery[daoListPage](t, s, "daos", "--page-limit", "2")
		p2 := cliQuery[daoListPage](t, s, "daos", "--page-limit", "2", "--page-key", p1.Pagination.NextKey)
		require.Equal(t, []uint64{1, 2}, daoIDs(p1.Daos))
		require.Equal(t, []uint64{3, 4}, daoIDs(p2.Daos))
		rev := cliQuery[daoListPage](t, s, "daos", "--page-reverse", "--page-limit", "1")
		require.EqualValues(t, s.daos, rev.Daos[0].ID)
		off := cliQuery[daoListPage](t, s, "daos", "--page-offset", "2", "--page-limit", "1")
		require.EqualValues(t, 3, off.Daos[0].ID)

		mine := cliQuery[daoListPage](t, s, "daos-by-admin", admin.FormattedAddress(), "--page-count-total")
		require.Equal(t, daoIDs(s.daosByAdmin(t, admin.FormattedAddress())), daoIDs(mine.Daos))
		require.Equal(t, fmt.Sprint(len(mine.Daos)), mine.Pagination.Total)
		require.Empty(t, cliQuery[daoListPage](t, s, "daos-by-admin", stranger.FormattedAddress()).Daos)

		// d now holds proposals in several statuses
		voting := s.submit(t, m[0], d, "still voting", nil)
		var rejected, executed []uint64
		for _, p := range s.daoProposals(t, d.id) {
			switch p.status() {
			case "REJECTED":
				rejected = append(rejected, uint64(p.ID))
			case "EXECUTED":
				executed = append(executed, uint64(p.ID))
			}
		}
		got := cliQuery[daoListPage](t, s, "proposals", d.idArg(), "--statuses", "voting")
		require.Equal(t, []uint64{voting}, propIDs(got.Proposals))
		// the command's help says the flag may be comma-separated: it may not
		s.cliQueryFails(t, "is not a valid value for enum", "proposals", d.idArg(), "--statuses", "rejected,executed")
		got = cliQuery[daoListPage](t, s, "proposals", d.idArg(), "--statuses", "rejected", "--statuses", "executed")
		want := append(append([]uint64{}, rejected...), executed...)
		slices.Sort(want)
		require.Equal(t, want, propIDs(got.Proposals))
		require.Equal(t, propIDs(s.daoProposals(t, d.id, "VOTING")), []uint64{voting}, "REST takes the statuses too")
		fresh := s.createStatic(t, m[0], defaultCfg(), addrs(m[0]), "")
		require.Empty(t, cliQuery[daoListPage](t, s, "proposals", fresh.idArg()).Proposals)
		s.cancel(t, m[0], voting)
		got = cliQuery[daoListPage](t, s, "proposals", d.idArg(), "--statuses", "cancelled")
		require.Equal(t, []uint64{voting}, propIDs(got.Proposals))
		s.cleanupAll(t, m[0], voting)
		got = cliQuery[daoListPage](t, s, "proposals", d.idArg())
		require.NotContains(t, propIDs(got.Proposals), voting, "a cleaned proposal is not listed")
	})

	t.Run("one REST GET per route", func(t *testing.T) {
		pid := s.submit(t, m[0], d, "rest", nil)
		s.vote(t, m[0], pid, "yes")
		for _, route := range []string{
			"/bze/dao/params",
			"/bze/dao/dao/" + d.idArg(),
			"/bze/dao/dao_by_address/" + d.addr,
			"/bze/dao/dao_controller/" + d.idArg(),
			"/bze/dao/proposal/" + daoU64(pid),
			"/bze/dao/ballot/" + daoU64(pid) + "/" + m[0].FormattedAddress(),
			"/bze/dao/voting_power/" + d.idArg() + "/" + m[0].FormattedAddress(),
			"/bze/dao/daos",
			"/bze/dao/daos_by_admin/" + admin.FormattedAddress(),
			"/bze/dao/proposals/" + d.idArg() + "?statuses=VOTING",
			"/bze/dao/ballots/" + daoU64(pid),
		} {
			var raw json.RawMessage
			s.mustRest(t, route, &raw)
			require.NotEmpty(t, raw, route)
		}
		require.Equal(t, http.StatusNotFound, s.restGet(t, "/bze/dao/join_record?source=sr/1&address="+stranger.FormattedAddress(), nil))
		s.cancel(t, m[0], pid)
	})
}

// F. CLI facts the docs state.
func (s *daoSuite) cliFacts(t *testing.T) {
	m := s.users(t, 2, daoUserFunds)
	admin := s.user(t)
	d := s.createStatic(t, m[0], defaultCfg(), addrs(m...), admin.FormattedAddress())
	pid := s.submit(t, m[0], d, "cli", nil)

	t.Run("vote options are lowercase names", func(t *testing.T) {
		for _, bad := range []string{"YES", "1"} {
			before := s.allBalances(t, m[0].FormattedAddress())
			_, err := helpers.ExecTxWith(s.ctx, s.bze, m[0], []helpers.TxOption{helpers.WithGas(daoRefuseGas)}, "dao", "vote", daoU64(pid), bad)
			require.Error(t, err, "vote %s must not parse", bad)
			require.Equal(t, before.String(), s.allBalances(t, m[0].FormattedAddress()).String())
		}
		s.vote(t, m[0], pid, "yes")
	})

	t.Run("update-members takes repeated flags, not a comma list", func(t *testing.T) {
		gen := func(flags ...string) ([]byte, error) {
			args := append([]string{"tx", "dao", "update-members", d.idArg()}, flags...)
			return helpers.CLI(s.ctx, s.bze, append(args, "--from", d.addr, "--generate-only", "--chain-id", s.bze.Config().ChainID, "--output", "json")...)
		}
		a, b := s.user(t), s.user(t)
		_, err := gen("--add", a.FormattedAddress()+","+b.FormattedAddress())
		require.Error(t, err, "a comma-separated --add must not parse")
		out, err := gen("--add", a.FormattedAddress(), "--add", b.FormattedAddress())
		require.NoError(t, err)
		var tx struct {
			Body struct {
				Messages []map[string]any `json:"messages"`
			} `json:"body"`
		}
		require.NoError(t, json.Unmarshal(out, &tx), "%s", out)
		require.Len(t, tx.Body.Messages, 1)
		msg := tx.Body.Messages[0]
		require.Equal(t, "/bze.dao.MsgUpdateMembers", msg["@type"])
		require.Equal(t, d.addr, msg["authority"])
		s.adminExec(t, admin, d, nil, msg)
		require.Equal(t, append(addrs(m...), a.FormattedAddress(), b.FormattedAddress()), s.dao(t, d.id).Members)
	})

	t.Run("ratios are 18-decimal integers on the CLI", func(t *testing.T) {
		decimal := strings.Replace(defaultCfg().cli(), ratioInt("0.5"), "0.5", 1)
		require.NotEqual(t, defaultCfg().cli(), decimal)
		before := s.daos
		res, err := helpers.ExecTxWith(s.ctx, s.bze, m[0], []helpers.TxOption{helpers.WithGas(daoRefuseGas)},
			"dao", "create-dao", "static-list", "decimal", decimal, "--members", m[0].FormattedAddress())
		if err == nil {
			require.NotZero(t, res.Code, "a decimal ratio must not create a dao")
		}
		require.Len(t, s.allDaos(t), int(before), "nothing is stored")
		t.Logf("a decimal ratio on the CLI: err %v, code %d %s", err, res.Code, res.RawLog)
	})

	s.waitVotingEnd(t, pid)
	s.settleFirst(t, m[1], pid, "EXECUTED") // 1 of 2 votes is the quorum; 1 yes of 1 decided passes
}

// D. At the end: stop the node, export the state, check I4 (pins) and I6
// (the vote index of proposals in voting) on the export, and validate it.
func (s *daoSuite) exportAndValidate(t *testing.T) {
	s.requireInvariants(t)
	node := s.bze.Validators[0]
	require.NoError(t, s.bze.StopAllNodes(s.ctx))
	exported, err := node.ExportState(s.ctx, -1)
	require.NoError(t, err)

	var doc struct {
		AppState struct {
			Dao struct {
				Daos          []daoRec      `json:"daos"`
				DaoCount      helpers.Int64 `json:"dao_count"`
				ProposalCount helpers.Int64 `json:"proposal_count"`
				Proposals     []daoProposal `json:"proposals"`
				Ballots       []daoBallot   `json:"ballots"`
				VoteIndex     []struct {
					Source     string        `json:"source"`
					Address    string        `json:"address"`
					ProposalID helpers.Int64 `json:"proposal_id"`
				} `json:"vote_index"`
				JoinRecords []json.RawMessage `json:"join_records"`
				Pins        []struct {
					Source string        `json:"source"`
					Count  helpers.Int64 `json:"count"`
				} `json:"pins"`
			} `json:"dao"`
			Bank struct {
				Balances []struct {
					Address string    `json:"address"`
					Coins   []daoCoin `json:"coins"`
				} `json:"balances"`
			} `json:"bank"`
		} `json:"app_state"`
	}
	require.NoError(t, json.Unmarshal([]byte(exported), &doc))
	g := doc.AppState.Dao
	require.EqualValues(t, s.daos, g.DaoCount)
	require.EqualValues(t, s.proposals, g.ProposalCount)
	require.Len(t, g.Daos, int(s.daos))

	// I4: a source's pins are the daos and the proposals in voting using it
	want := map[string]int64{}
	voting := map[uint64]bool{}
	deposits := sdk.NewCoins()
	sources := func(el *daoElectorate) []string {
		if el == nil {
			return nil
		}
		var out []string
		for _, id := range el.SrIDs {
			out = append(out, srSource(id))
		}
		if el.DrDenom != "" {
			out = append(out, drSource(el.DrDenom))
		}
		return out
	}
	for _, d := range g.Daos {
		for _, src := range sources(d.Electorate) {
			want[src]++
		}
	}
	for _, p := range g.Proposals {
		if p.status() != "VOTING" {
			continue
		}
		voting[uint64(p.ID)] = true
		deposits = deposits.Add(p.PinnedDeposit.coins()...)
		for _, src := range sources(p.PinnedElectorate) {
			want[src]++
		}
	}
	got := map[string]int64{}
	for _, pin := range g.Pins {
		got[pin.Source] = int64(pin.Count)
	}
	require.Equal(t, want, got, "I4 on the export")

	// I6: every share of a ballot on a proposal in voting has its vote-index
	// entry, and every entry of a proposal in voting names such a share
	shares := map[string]bool{}
	for _, b := range g.Ballots {
		if !voting[uint64(b.ProposalID)] {
			continue
		}
		for _, sh := range b.Shares {
			shares[sh.Source+"/"+b.Voter+"/"+fmt.Sprint(b.ProposalID)] = true
		}
	}
	entries := map[string]bool{}
	for _, e := range g.VoteIndex {
		if voting[uint64(e.ProposalID)] {
			entries[e.Source+"/"+e.Address+"/"+fmt.Sprint(e.ProposalID)] = true
		}
	}
	require.Equal(t, shares, entries, "I6 on the export")

	// I1 on the export: the escrow holds the deposits of the proposals in voting
	held := sdk.NewCoins()
	for _, b := range doc.AppState.Bank.Balances {
		if b.Address == s.escrow {
			for _, c := range b.Coins {
				held = held.Add(sdk.NewCoin(c.Denom, parseInt(c.Amount)))
			}
		}
	}
	require.Equal(t, deposits.String(), held.String(), "I1 on the export")
	t.Logf("export: %d daos, %d proposals (%d in voting), %d ballots, %d vote-index entries, %d join records, %d pins",
		len(g.Daos), len(g.Proposals), len(voting), len(g.Ballots), len(g.VoteIndex), len(g.JoinRecords), len(g.Pins))

	_, stderr, err := node.ExecBin(s.ctx, "genesis", "validate", path.Join(node.HomeDir(), "state_export.json"))
	require.NoError(t, err, "genesis validate: %s", stderr)
	t.Log("importing the export into a fresh node is not covered: the harness starts chains from generated genesis only")
}
