package interchaintest_test

import (
	"fmt"
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// ---------------------------------------------------------------------------
// DAO actions, each with its funds audit

// creationFlows is what a create-dao tx moves: the creation fee to its
// destination collector, when it is positive.
func (s *daoSuite) creationFlows(creator string) []daoFlow {
	fee := s.params.DaoCreationFee
	if !fee.IsPositive() {
		return nil
	}
	dest := s.burner
	if s.params.DaoCreationFeeDestination == "community_pool" {
		dest = s.cp
	}
	return []daoFlow{flow(creator, dest, sdk.NewCoins(fee))}
}

// create sends create-dao and checks the new DAO's id (the next one, I5), its
// derived address and its admin.
func (s *daoSuite) create(t *testing.T, creator ibc.Wallet, typ string, cfg daoCfg, extra ...string) daoRef {
	t.Helper()
	return s.createNamed(t, creator, typ, fmt.Sprintf("DAO %d", s.daos+1), cfg, extra...)
}

func (s *daoSuite) createNamed(t *testing.T, creator ibc.Wallet, typ, name string, cfg daoCfg, extra ...string) daoRef {
	t.Helper()
	next := s.daos + 1
	args := append([]string{"dao", "create-dao", typ, name, cfg.cli()}, extra...)
	res := s.run(t, creator, []string{daoAddress(next)}, s.creationFlows(creator.FormattedAddress()), args...)
	e := s.event(t, res, "EventDaoCreated")
	id := atoiU(t, idOf(e))
	require.Equal(t, next, id, "dao ids are handed out in order and never reused (I5)")
	require.Equal(t, daoAddress(id), e.Attributes["address"], "the dao address is address.Module(dao, id)")
	s.daos = id
	ref := daoRef{id: id, addr: e.Attributes["address"]}
	d := s.dao(t, id)
	require.Equal(t, ref.addr, d.Address)
	admin := ref.addr
	for i, a := range extra {
		if a == "--admin" {
			admin = extra[i+1]
		}
	}
	require.Equal(t, admin, d.Admin)
	require.Equal(t, admin, e.Attributes["admin"])
	require.Empty(t, d.PendingAdmin)
	return ref
}

func (s *daoSuite) createStatic(t *testing.T, creator ibc.Wallet, cfg daoCfg, members []string, admin string) daoRef {
	t.Helper()
	extra := []string{"--members", strings.Join(members, ",")}
	if admin != "" {
		extra = append(extra, "--admin", admin)
	}
	ref := s.create(t, creator, "static-list", cfg, extra...)
	d := s.dao(t, ref.id)
	require.Equal(t, "DAO_TYPE_STATIC_LIST", d.daoType())
	require.Equal(t, members, d.Members)
	return ref
}

func (s *daoSuite) createStake(t *testing.T, creator ibc.Wallet, cfg daoCfg, el daoElectorate, admin string) daoRef {
	t.Helper()
	var extra []string
	if len(el.SrIDs) > 0 || el.DrDenom != "" {
		extra = append(extra, "--electorate", mustJSON(el))
	}
	if admin != "" {
		extra = append(extra, "--admin", admin)
	}
	ref := s.create(t, creator, "stake", cfg, extra...)
	require.Equal(t, "DAO_TYPE_STAKE", s.dao(t, ref.id).daoType())
	return ref
}

// submitFlows is what a submit by proposer to the DAO moves: the cost to the
// burner or the treasury (skipped when the DAO pays itself) and the deposit
// into the escrow.
func (s *daoSuite) submitFlows(t *testing.T, d daoRec, proposer string) []daoFlow {
	t.Helper()
	var flows []daoFlow
	if cost := d.Config.ProposalCost.coins(); cost.IsAllPositive() {
		switch {
		case enumOr(d.Config.CostDestination, "BURN") == "TREASURY" && proposer == d.Address:
		case enumOr(d.Config.CostDestination, "BURN") == "TREASURY":
			flows = append(flows, flow(proposer, d.Address, cost))
		default:
			flows = append(flows, flow(proposer, s.burner, cost))
		}
	}
	if deposit := d.Config.ProposalDeposit.coins(); deposit.IsAllPositive() {
		flows = append(flows, flow(proposer, s.escrow, deposit))
	}
	return flows
}

// submit sends submit-proposal and returns the new proposal's id (the next
// one, I5). flags are e.g. --summary or --expedited.
func (s *daoSuite) submit(t *testing.T, proposer ibc.Wallet, ref daoRef, title string, msgs []map[string]any, flags ...string) uint64 {
	t.Helper()
	d := s.dao(t, ref.id)
	args := append([]string{"dao", "submit-proposal", ref.idArg(), title}, msgArgs(msgs...)...)
	args = append(args, flags...)
	res := s.run(t, proposer, []string{ref.addr}, s.submitFlows(t, d, proposer.FormattedAddress()), args...)
	return s.submitted(t, res, ref, proposer.FormattedAddress())
}

// submitted checks the EventProposalSubmitted of res and the stored proposal.
func (s *daoSuite) submitted(t *testing.T, res helpers.TxResult, ref daoRef, proposer string) uint64 {
	t.Helper()
	e := s.event(t, res, "EventProposalSubmitted")
	pid := atoiU(t, idOf(e))
	require.Equal(t, s.proposals+1, pid, "proposal ids are handed out in order and never reused (I5)")
	s.proposals = pid
	require.Equal(t, ref.idArg(), e.Attributes["dao_id"])
	require.Equal(t, proposer, e.Attributes["proposer"])
	p := s.prop(t, pid)
	require.Equal(t, "VOTING", p.status())
	require.Equal(t, proposer, p.Proposer)
	require.Equal(t, res.Height, int64(p.SubmitHeight))
	require.Equal(t, e.Attributes["denominator"], p.PinnedDenominator)
	return pid
}

func (s *daoSuite) vote(t *testing.T, voter ibc.Wallet, pid uint64, option string) helpers.TxResult {
	t.Helper()
	return s.run(t, voter, nil, nil, "dao", "vote", daoU64(pid), option)
}

// voteWeight votes and returns the weight the EventVote counted.
func (s *daoSuite) voteWeight(t *testing.T, voter ibc.Wallet, pid uint64, option string) sdkmath.Int {
	t.Helper()
	e := s.event(t, s.vote(t, voter, pid, option), "EventVote")
	require.Equal(t, voter.FormattedAddress(), e.Attributes["voter"])
	return parseInt(e.Attributes["weight"])
}

// depositFlow is the deposit release of a proposal: back to the proposer on a
// refund, else to the pinned kept_deposit_destination.
func (s *daoSuite) depositFlow(p daoProposal, refund bool) []daoFlow {
	deposit := p.PinnedDeposit.coins()
	if !deposit.IsAllPositive() {
		return nil
	}
	switch {
	case refund:
		return []daoFlow{flow(s.escrow, p.Proposer, deposit)}
	case enumOr(p.PinnedConfig.KeptDepositDestination, "BURN") == "TREASURY":
		return []daoFlow{flow(s.escrow, daoAddress(uint64(p.DaoID)), deposit)}
	default:
		return []daoFlow{flow(s.escrow, s.burner, deposit)}
	}
}

// refunds reports whether a first settle refunds the deposit (D13).
func refunds(p daoProposal, passed bool) bool {
	switch enumOr(p.PinnedConfig.DepositRefundPolicy, "ALWAYS") {
	case "ALWAYS":
		return true
	case "ONLY_PASSED":
		return passed
	default:
		return false
	}
}

// settle sends settle-proposal and requires the status it reports. want are
// the transfers it must make (the deposit release and the executed messages).
func (s *daoSuite) settle(t *testing.T, sender ibc.Wallet, pid uint64, status string, watch []string, want []daoFlow) helpers.TxResult {
	t.Helper()
	p := s.prop(t, pid)
	watch = append(watch, p.Proposer, daoAddress(uint64(p.DaoID)))
	res := s.run(t, sender, watch, want, "dao", "settle-proposal", daoU64(pid))
	e := s.event(t, res, "EventProposalSettled")
	require.Equal(t, status, statusOf(e), "settle of proposal %d", pid)
	require.Equal(t, status, s.prop(t, pid).status())
	return res
}

// settleFirst settles a proposal in voting and derives its transfers: the
// deposit release by policy and outcome, plus exec (the messages' moves) when
// it executes.
func (s *daoSuite) settleFirst(t *testing.T, sender ibc.Wallet, pid uint64, status string, exec ...daoFlow) helpers.TxResult {
	t.Helper()
	p := s.prop(t, pid)
	require.Equal(t, "VOTING", p.status())
	passed := status != "REJECTED"
	want := s.depositFlow(p, refunds(p, passed))
	if status == "EXECUTED" {
		want = append(want, exec...)
	}
	return s.settle(t, sender, pid, status, nil, want)
}

func (s *daoSuite) cancel(t *testing.T, proposer ibc.Wallet, pid uint64) helpers.TxResult {
	t.Helper()
	p := s.prop(t, pid)
	refund := enumOr(p.PinnedConfig.DepositRefundPolicy, "ALWAYS") == "ALWAYS"
	res := s.run(t, proposer, []string{daoAddress(uint64(p.DaoID))}, s.depositFlow(p, refund), "dao", "cancel-proposal", daoU64(pid))
	s.event(t, res, "EventProposalCancelled")
	require.Equal(t, "CANCELLED", s.prop(t, pid).status())
	return res
}

// cleanup sends one cleanup-proposal call and returns what it reports.
func (s *daoSuite) cleanup(t *testing.T, sender ibc.Wallet, pid uint64, limit int) (uint64, bool) {
	t.Helper()
	args := []string{"dao", "cleanup-proposal", daoU64(pid)}
	if limit > 0 {
		args = append(args, "--limit", fmt.Sprint(limit))
	}
	res := s.run(t, sender, nil, nil, args...)
	e := s.event(t, res, "EventProposalCleaned")
	completed := e.Attributes["completed"] == "true"
	if completed {
		s.cleaned[pid] = true
	}
	return atoiU(t, e.Attributes["processed"]), completed
}

// cleanupAll cleans a finished proposal up completely.
func (s *daoSuite) cleanupAll(t *testing.T, sender ibc.Wallet, pid uint64) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if _, done := s.cleanup(t, sender, pid, 0); done {
			require.False(t, s.proposalExists(t, pid))
			return
		}
	}
	t.Fatalf("proposal %d not cleaned after 100 calls", pid)
}

// adminExec sends admin-execute with msgs and requires it to succeed with the
// transfers want.
func (s *daoSuite) adminExec(t *testing.T, admin ibc.Wallet, ref daoRef, want []daoFlow, msgs ...map[string]any) helpers.TxResult {
	t.Helper()
	args := append([]string{"dao", "admin-execute", ref.idArg()}, msgArgs(msgs...)...)
	res := s.run(t, admin, []string{ref.addr}, want, args...)
	e := s.event(t, res, "EventAdminExecuted")
	require.Equal(t, fmt.Sprint(len(msgs)), e.Attributes["msg_count"])
	return res
}

// pass votes yes with every voter, waits for the voting end and settles the
// proposal, which must reach status; exec are its messages' transfers.
func (s *daoSuite) pass(t *testing.T, pid uint64, voters []ibc.Wallet, status string, exec ...daoFlow) helpers.TxResult {
	t.Helper()
	for _, v := range voters {
		s.vote(t, v, pid, "yes")
	}
	s.waitVotingEnd(t, pid)
	return s.settleFirst(t, voters[0], pid, status, exec...)
}

// fund sends coins from the funder to addr.
func (s *daoSuite) fund(t *testing.T, addr string, amt sdk.Coins) {
	t.Helper()
	s.run(t, s.funder, []string{addr}, []daoFlow{flow(s.funder.FormattedAddress(), addr, amt)},
		"bank", "send", s.funder.FormattedAddress(), addr, amt.String())
}

// ---------------------------------------------------------------------------
// invariants on the live chain (Security & Regression Tests, I1–I7)

// requireInvariants checks what queries can show after every sub-test:
// I1 the escrow holds exactly the deposits of the proposals in voting, I2
// each proposal's tally is the sum of its ballots, I3 each ballot's weight is
// the sum of its shares, I5 ids are handed out in order and a cleaned id
// never returns, I7 every DAO is listed under its admin and only there.
func (s *daoSuite) requireInvariants(t *testing.T) {
	t.Helper()
	daos := s.allDaos(t)
	require.Len(t, daos, int(s.daos), "every created dao is listed (I5)")
	byAdmin := map[string][]uint64{}
	held := sdk.NewCoins()
	seen := map[uint64]bool{}
	for i, d := range daos {
		id := uint64(d.ID)
		require.Equal(t, uint64(i+1), id, "dao ids are 1..n in order (I5)")
		require.Equal(t, daoAddress(id), d.Address)
		byAdmin[d.Admin] = append(byAdmin[d.Admin], id)
		require.NotEqual(t, d.Admin, d.PendingAdmin, "dao %d: the pending admin is never the admin (I7)", id)

		last := uint64(0)
		for _, p := range s.daoProposals(t, id) {
			pid := uint64(p.ID)
			require.Greater(t, pid, last, "dao %d: proposals are listed in id order", id)
			last = pid
			require.LessOrEqual(t, pid, s.proposals, "no proposal id beyond the ones handed out (I5)")
			require.False(t, s.cleaned[pid], "a cleaned proposal %d never comes back (I5)", pid)
			require.False(t, seen[pid], "proposal %d listed twice", pid)
			seen[pid] = true
			require.Equal(t, id, uint64(p.DaoID))
			if p.status() == "VOTING" {
				held = held.Add(p.PinnedDeposit.coins()...)
			}
			if p.CleanupCursor == "" {
				s.requireTally(t, p)
			}
		}
	}
	require.Equal(t, held.String(), s.allBalances(t, s.escrow).String(), "the escrow holds exactly the deposits of the proposals in voting (I1)")
	for admin, ids := range byAdmin {
		var listed []uint64
		for _, d := range s.daosByAdmin(t, admin) {
			listed = append(listed, uint64(d.ID))
		}
		require.Equal(t, ids, listed, "daos-by-admin %s lists exactly the daos it administers (I7)", admin)
	}
}

// requireTally checks I2 and I3 on one proposal.
func (s *daoSuite) requireTally(t *testing.T, p daoProposal) {
	t.Helper()
	sum := map[string]sdkmath.Int{"YES": sdkmath.ZeroInt(), "NO": sdkmath.ZeroInt(), "ABSTAIN": sdkmath.ZeroInt()}
	for _, b := range s.ballots(t, uint64(p.ID)) {
		w := parseInt(b.Weight)
		opt := enumOr(b.Option, "VOTE_OPTION_UNSPECIFIED")
		require.Contains(t, sum, opt, "proposal %d: ballot of %s has option %s", p.ID, b.Voter, opt)
		sum[opt] = sum[opt].Add(w)
		if len(b.Shares) > 0 {
			shares := sdkmath.ZeroInt()
			for _, sh := range b.Shares {
				shares = shares.Add(parseInt(sh.Amount))
			}
			require.Equal(t, shares.String(), w.String(), "proposal %d: ballot of %s weighs the sum of its shares (I3)", p.ID, b.Voter)
		} else {
			// a static ballot weighs 1 (0 once its member is removed); a stake
			// ballot without shares has lost all of them to exits
			require.True(t, w.IsZero() || w.Equal(sdkmath.OneInt()), "proposal %d: ballot of %s without shares weighs %s (I3)", p.ID, b.Voter, w)
		}
	}
	require.Equal(t, sum["YES"].String(), parseInt(p.Yes).String(), "proposal %d: yes is the sum of the yes ballots (I2)", p.ID)
	require.Equal(t, sum["NO"].String(), parseInt(p.No).String(), "proposal %d: no is the sum of the no ballots (I2)", p.ID)
	require.Equal(t, sum["ABSTAIN"].String(), parseInt(p.Abstain).String(), "proposal %d: abstain is the sum of the abstain ballots (I2)", p.ID)
}

// requireStatusIn checks a proposal's stored status.
func (s *daoSuite) requireStatus(t *testing.T, pid uint64, status string) {
	t.Helper()
	require.Equal(t, status, s.prop(t, pid).status(), "proposal %d", pid)
}

// requireTallyOf checks a proposal's yes, no and abstain.
func (s *daoSuite) requireTallyOf(t *testing.T, pid uint64, yes, no, abstain int64) {
	t.Helper()
	p := s.prop(t, pid)
	require.Equal(t, []string{fmt.Sprint(yes), fmt.Sprint(no), fmt.Sprint(abstain)},
		[]string{parseInt(p.Yes).String(), parseInt(p.No).String(), parseInt(p.Abstain).String()}, "tally of proposal %d", pid)
}
