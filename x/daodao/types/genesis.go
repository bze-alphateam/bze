package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state: default Params, no DAOs.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:       DefaultParams(),
		DaoIdCounter: 1,
		Daos:         nil,
	}
}

// validateGenesisProposalsAndVotes enforces the referential-integrity rules
// across the Epic-3 genesis fields (proposals, votes, snapshot tables, and
// per-DAO counters). Called from GenesisState.Validate after the DAO loop
// has populated `seenDaoIDs`.
//
// Rules enforced:
//
//   Proposals:
//   - Each proposal's dao_id matches a DAO present in `daos`.
//   - (dao_id, proposal_id) pairs are unique.
//   - proposal_id is non-zero.
//   - status is a known variant (UNSPECIFIED rejected).
//   - status VOTING implies voting_end is set and non-pre-epoch — a zero
//     time.Time has UnixNano() ≈ -62e18 which wraps when cast to uint64
//     and would park the proposal at the wrong end of the timer queue,
//     never being dequeued by the end-blocker.
//   - For every proposal, a matching SnapshotTotalEntry must exist (votes
//     would otherwise read 0 power on import).
//   - p.GovernanceSnapshot passes the same stateless + Param caps that
//     MsgCreateDao / MsgUpdateGovernanceConfig enforce at runtime; EndBlock
//     feeds this snapshot directly into computeOutcome and a bogus config
//     (UNSPECIFIED approval rule, zero threshold, etc.) would produce
//     outcomes runtime creation could never have produced.
//   - p.Tally is structurally consistent: yes+no+abstain doesn't overflow
//     and stays ≤ total_power, and total_power matches the corresponding
//     SnapshotTotalEntry. (Tighter "votes reconcile to tally" check
//     follows below in the vote loop.)
//
//   Votes:
//   - Each vote's (dao_id, proposal_id) resolves to a known proposal.
//   - voter is valid bech32.
//   - (dao_id, proposal_id, voter) is unique.
//   - option is a known variant.
//   - Vote.Power matches the SnapshotPowerEntry for (dao, snap, voter).
//     This is the load-bearing reconciliation: it prevents a crafted
//     genesis from claiming a voter cast more weight than they had in
//     the proposal's snapshot.
//
//   Tally reconciliation (final post-loop pass):
//   - Sum of votes' power per option matches the proposal's stored Tally.
//     With revote replacing the row in place, each surviving Vote is the
//     voter's effective contribution, so the sum is exact.
//
//   Snapshot tables:
//   - SnapshotPowerEntry: address is valid bech32; (dao, snap, address)
//     unique.
//   - SnapshotTotalEntry: (dao_id, snapshot_id) unique.
//
//   Counters:
//   - proposal_id_counters / snapshot_id_counters: dao_id present in `daos`;
//     unique per dao_id; counter strictly exceeds the maximum
//     proposal_id (resp. snapshot_id) present for that DAO.
//
// Returns the first violation encountered.
func (gs GenesisState) validateGenesisProposalsAndVotes(seenDaoIDs map[uint64]struct{}) error {
	// --------- Proposals ---------
	proposalsByDao := make(map[uint64]map[uint64]*Proposal, len(seenDaoIDs))
	for i := range gs.Proposals {
		p := &gs.Proposals[i]
		if _, ok := seenDaoIDs[p.DaoId]; !ok {
			return fmt.Errorf("proposal %d/%d: dao_id not in genesis daos", p.DaoId, p.ProposalId)
		}
		if p.ProposalId == 0 {
			return fmt.Errorf("proposal at dao %d has zero proposal_id", p.DaoId)
		}
		inner, ok := proposalsByDao[p.DaoId]
		if !ok {
			inner = make(map[uint64]*Proposal)
			proposalsByDao[p.DaoId] = inner
		}
		if _, dup := inner[p.ProposalId]; dup {
			return fmt.Errorf("duplicate proposal %d in dao %d", p.ProposalId, p.DaoId)
		}
		inner[p.ProposalId] = p
		switch p.Status {
		case ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD,
			ProposalStatus_PROPOSAL_STATUS_VOTING,
			ProposalStatus_PROPOSAL_STATUS_PASSED,
			ProposalStatus_PROPOSAL_STATUS_REJECTED,
			ProposalStatus_PROPOSAL_STATUS_EXECUTED,
			ProposalStatus_PROPOSAL_STATUS_REJECTED_NO_DEPOSIT:
			// ok
		default:
			return fmt.Errorf("proposal %d/%d: unknown status %v", p.DaoId, p.ProposalId, p.Status)
		}
		// DEPOSIT_PERIOD and VOTING proposals are re-enqueued on the
		// end-blocker timer using uint64(deadline.UnixNano()). A zero or
		// pre-epoch timestamp would wrap to a huge key and silently never
		// expire. Each status uses its own deadline field (mirroring
		// proposalDeadlineNs in the keeper).
		if p.Status == ProposalStatus_PROPOSAL_STATUS_VOTING {
			if p.VotingEnd.IsZero() {
				return fmt.Errorf("proposal %d/%d: VOTING status requires a non-zero voting_end",
					p.DaoId, p.ProposalId)
			}
			if p.VotingEnd.UnixNano() < 0 {
				return fmt.Errorf("proposal %d/%d: voting_end %s is before Unix epoch",
					p.DaoId, p.ProposalId, p.VotingEnd)
			}
		}
		if p.Status == ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD {
			if p.DepositDeadline.IsZero() {
				return fmt.Errorf("proposal %d/%d: DEPOSIT_PERIOD status requires a non-zero deposit_deadline",
					p.DaoId, p.ProposalId)
			}
			if p.DepositDeadline.UnixNano() < 0 {
				return fmt.Errorf("proposal %d/%d: deposit_deadline %s is before Unix epoch",
					p.DaoId, p.ProposalId, p.DepositDeadline)
			}
		}
		// Status-vs-deposit invariants. Without these, a crafted import
		// could put an underfunded proposal directly into VOTING (or beyond)
		// and bypass the economic deposit gate that MsgCreateProposal +
		// MsgDeposit enforce at runtime. Symmetrically, DEPOSIT_PERIOD
		// must mean "below threshold" — anything at-or-above should have
		// auto-promoted to VOTING.
		if !p.DepositCollected.Amount.IsNil() {
			minAmt := p.DepositSnapshot.MinDeposit.Amount
			collected := p.DepositCollected.Amount
			switch p.Status {
			case ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD:
				if collected.GTE(minAmt) {
					return fmt.Errorf("proposal %d/%d: DEPOSIT_PERIOD with collected=%s >= min_deposit=%s (should have transitioned to VOTING)",
						p.DaoId, p.ProposalId, collected, minAmt)
				}
			case ProposalStatus_PROPOSAL_STATUS_VOTING,
				ProposalStatus_PROPOSAL_STATUS_PASSED,
				ProposalStatus_PROPOSAL_STATUS_REJECTED,
				ProposalStatus_PROPOSAL_STATUS_EXECUTED:
				if collected.LT(minAmt) {
					return fmt.Errorf("proposal %d/%d: %s with collected=%s < min_deposit=%s (deposit gate bypassed)",
						p.DaoId, p.ProposalId, p.Status, collected, minAmt)
				}
			}
			// Open proposals: deposit_collected.denom must match the
			// frozen min_deposit denom — otherwise MsgDeposit's Coin.Add
			// would panic on a denom mismatch the first time someone
			// tops up.
			isOpen := p.Status == ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD ||
				p.Status == ProposalStatus_PROPOSAL_STATUS_VOTING
			if isOpen && p.DepositCollected.Denom != p.DepositSnapshot.MinDeposit.Denom {
				return fmt.Errorf("proposal %d/%d: open proposal deposit_collected.denom=%q != min_deposit.denom=%q",
					p.DaoId, p.ProposalId, p.DepositCollected.Denom, p.DepositSnapshot.MinDeposit.Denom)
			}
		}
		// The proposal's frozen GovernanceSnapshot must satisfy the same
		// brick-prevention caps that the tx path enforces. Without this,
		// computeOutcome could be called on an unknown approval_rule or
		// a zero threshold and produce surprising outcomes.
		if err := ValidateGovernanceConfigStateless(p.GovernanceSnapshot); err != nil {
			return fmt.Errorf("proposal %d/%d: governance_snapshot: %w",
				p.DaoId, p.ProposalId, err)
		}
		if err := ValidateGovernanceConfigAgainstParams(p.GovernanceSnapshot, gs.Params.MaxVotingPeriod); err != nil {
			return fmt.Errorf("proposal %d/%d: governance_snapshot: %w",
				p.DaoId, p.ProposalId, err)
		}
		// Epic 4: deposit_snapshot drives refund/forfeit routing at terminal
		// status; same brick-prevention caps must hold.
		if err := ValidateDepositConfigStateless(p.DepositSnapshot); err != nil {
			return fmt.Errorf("proposal %d/%d: deposit_snapshot: %w",
				p.DaoId, p.ProposalId, err)
		}
		if err := ValidateDepositConfigAgainstParams(p.DepositSnapshot, gs.Params.MaxDepositPeriod); err != nil {
			return fmt.Errorf("proposal %d/%d: deposit_snapshot: %w",
				p.DaoId, p.ProposalId, err)
		}
		// Tally structural invariants: no overflow when summing the three
		// option buckets, and the sum stays bounded by total_power.
		// `safeAddU64`-style explicit checks because uint64 silently wraps.
		yn, ok := safeAdd(p.Tally.YesPower, p.Tally.NoPower)
		if !ok {
			return fmt.Errorf("proposal %d/%d: tally yes+no overflows uint64",
				p.DaoId, p.ProposalId)
		}
		voted, ok := safeAdd(yn, p.Tally.AbstainPower)
		if !ok {
			return fmt.Errorf("proposal %d/%d: tally yes+no+abstain overflows uint64",
				p.DaoId, p.ProposalId)
		}
		if voted > p.Tally.TotalPower {
			return fmt.Errorf("proposal %d/%d: tally voted=%d exceeds total_power=%d",
				p.DaoId, p.ProposalId, voted, p.Tally.TotalPower)
		}

		// Reviewer Finding 1: terminal proposal statuses MUST be justified
		// by their stored tally. Without this, a crafted genesis can mark
		// a proposal PASSED with zero votes; MsgExecuteProposal then trusts
		// the status flag and dispatches the DAO-signed bundle. Recompute
		// the outcome and compare.
		//
		// REJECTED_NO_DEPOSIT proposals never reach VOTING, so there's no
		// tally to verify — covered separately by the deposit-phase
		// invariants and by the zero-tally check on votes.
		switch p.Status {
		case ProposalStatus_PROPOSAL_STATUS_PASSED, ProposalStatus_PROPOSAL_STATUS_EXECUTED:
			if ComputeOutcome(p.GovernanceSnapshot, p.Tally) != OutcomePass {
				return fmt.Errorf("proposal %d/%d: status %s but tally does not justify a pass outcome",
					p.DaoId, p.ProposalId, p.Status)
			}
		case ProposalStatus_PROPOSAL_STATUS_REJECTED:
			if ComputeOutcome(p.GovernanceSnapshot, p.Tally) != OutcomeReject {
				return fmt.Errorf("proposal %d/%d: status REJECTED but tally indicates a pass outcome",
					p.DaoId, p.ProposalId)
			}
		}
	}

	// --------- Snapshot totals ---------
	// (dao, snap) → total — used both for orphan-power detection and to
	// cross-check Proposal.Tally.TotalPower.
	snapTotalByKey := make(map[[2]uint64]uint64, len(gs.SnapshotTotals))
	for _, st := range gs.SnapshotTotals {
		if _, ok := seenDaoIDs[st.DaoId]; !ok {
			return fmt.Errorf("snapshot_total: dao_id %d not in genesis daos", st.DaoId)
		}
		key := [2]uint64{st.DaoId, st.SnapshotId}
		if _, dup := snapTotalByKey[key]; dup {
			return fmt.Errorf("duplicate snapshot_total for (dao=%d, snap=%d)", st.DaoId, st.SnapshotId)
		}
		snapTotalByKey[key] = st.Total
	}

	// Each proposal must have a matching snapshot_total whose value equals
	// Tally.TotalPower — the runtime stores them in lock-step, and divergence
	// would let outcomes use a different denominator than vote-time reads.
	for _, p := range gs.Proposals {
		total, ok := snapTotalByKey[[2]uint64{p.DaoId, p.SnapshotId}]
		if !ok {
			return fmt.Errorf("proposal %d/%d: missing snapshot_total for snapshot %d",
				p.DaoId, p.ProposalId, p.SnapshotId)
		}
		if total != p.Tally.TotalPower {
			return fmt.Errorf("proposal %d/%d: tally total_power=%d != snapshot_total=%d",
				p.DaoId, p.ProposalId, p.Tally.TotalPower, total)
		}
	}

	// --------- Snapshot powers ---------
	// (dao, snap, address) → power — keyed by string-stringified composite
	// so the vote loop can verify Vote.Power matches the snapshot row.
	snapPowerByKey := make(map[string]uint64, len(gs.SnapshotPowers))
	for _, sp := range gs.SnapshotPowers {
		if _, ok := seenDaoIDs[sp.DaoId]; !ok {
			return fmt.Errorf("snapshot_power: dao_id %d not in genesis daos", sp.DaoId)
		}
		if _, err := sdk.AccAddressFromBech32(sp.Address); err != nil {
			return fmt.Errorf("snapshot_power (dao=%d, snap=%d): invalid address: %w",
				sp.DaoId, sp.SnapshotId, err)
		}
		key := snapPowerKey(sp.DaoId, sp.SnapshotId, sp.Address)
		if _, dup := snapPowerByKey[key]; dup {
			return fmt.Errorf("duplicate snapshot_power for (dao=%d, snap=%d, addr=%s)",
				sp.DaoId, sp.SnapshotId, sp.Address)
		}
		snapPowerByKey[key] = sp.Power
		// Orphan check: every power row needs a matching snapshot_total.
		if _, ok := snapTotalByKey[[2]uint64{sp.DaoId, sp.SnapshotId}]; !ok {
			return fmt.Errorf("snapshot_power (dao=%d, snap=%d): no snapshot_total for snapshot",
				sp.DaoId, sp.SnapshotId)
		}
	}

	// --------- Votes ---------
	// While iterating votes, accumulate per-(dao, proposal, option) power
	// sums so we can reconcile against Proposal.Tally below.
	type tallyAggKey struct {
		daoID, proposalID uint64
	}
	type tallyAgg struct {
		yes, no, abstain uint64
	}
	talliesByProp := make(map[tallyAggKey]*tallyAgg, len(proposalsByDao))
	seenVotes := make(map[[3]string]struct{}, len(gs.Votes))
	for _, v := range gs.Votes {
		propsInDao, ok := proposalsByDao[v.DaoId]
		if !ok {
			return fmt.Errorf("vote (dao=%d, proposal=%d, voter=%s): dao not in genesis",
				v.DaoId, v.ProposalId, v.Voter)
		}
		p, ok := propsInDao[v.ProposalId]
		if !ok {
			return fmt.Errorf("vote (dao=%d, proposal=%d, voter=%s): proposal not in genesis",
				v.DaoId, v.ProposalId, v.Voter)
		}
		// Reviewer Finding 2: votes are only meaningful on proposals that
		// reached (or are in) VOTING. Allowing a vote on a DEPOSIT_PERIOD
		// proposal lets a crafted import preload votes that go live the
		// moment MsgDeposit promotes the proposal — a clear bypass of
		// runtime gating.
		switch p.Status {
		case ProposalStatus_PROPOSAL_STATUS_VOTING,
			ProposalStatus_PROPOSAL_STATUS_PASSED,
			ProposalStatus_PROPOSAL_STATUS_REJECTED,
			ProposalStatus_PROPOSAL_STATUS_EXECUTED:
			// ok — proposal has reached or passed through VOTING.
		default:
			return fmt.Errorf("vote (dao=%d, proposal=%d, voter=%s): proposal status %s never reached VOTING; vote not allowed",
				v.DaoId, v.ProposalId, v.Voter, p.Status)
		}
		if _, err := sdk.AccAddressFromBech32(v.Voter); err != nil {
			return fmt.Errorf("vote (dao=%d, proposal=%d): invalid voter bech32: %w",
				v.DaoId, v.ProposalId, err)
		}
		switch v.Option {
		case VoteOption_VOTE_OPTION_YES, VoteOption_VOTE_OPTION_NO, VoteOption_VOTE_OPTION_ABSTAIN:
			// ok
		default:
			return fmt.Errorf("vote (dao=%d, proposal=%d, voter=%s): unknown option %v",
				v.DaoId, v.ProposalId, v.Voter, v.Option)
		}
		key := [3]string{
			fmt.Sprintf("%d", v.DaoId),
			fmt.Sprintf("%d", v.ProposalId),
			v.Voter,
		}
		if _, dup := seenVotes[key]; dup {
			return fmt.Errorf("duplicate vote for (dao=%d, proposal=%d, voter=%s)",
				v.DaoId, v.ProposalId, v.Voter)
		}
		seenVotes[key] = struct{}{}

		// Vote.Power must equal the snapshot row for (dao, proposal-snap,
		// voter). Power == 0 from a missing snapshot row would also be
		// caught here: a Vote with power=0 against a missing snapshot
		// entry would only be "valid" if the snapshot says 0 too, which
		// means the address is not in the lookup.
		gotPower, hasSnap := snapPowerByKey[snapPowerKey(v.DaoId, p.SnapshotId, v.Voter)]
		if !hasSnap {
			// Vote.Power must then be 0 (a voter not in the snapshot has
			// no power and could not have been admitted at runtime).
			if v.Power != 0 {
				return fmt.Errorf("vote (dao=%d, proposal=%d, voter=%s): power=%d but no snapshot row exists",
					v.DaoId, v.ProposalId, v.Voter, v.Power)
			}
		} else if v.Power != gotPower {
			return fmt.Errorf("vote (dao=%d, proposal=%d, voter=%s): power=%d != snapshot=%d",
				v.DaoId, v.ProposalId, v.Voter, v.Power, gotPower)
		}

		// Accumulate for the final tally reconciliation pass.
		aggKey := tallyAggKey{v.DaoId, v.ProposalId}
		agg := talliesByProp[aggKey]
		if agg == nil {
			agg = &tallyAgg{}
			talliesByProp[aggKey] = agg
		}
		var bucket *uint64
		switch v.Option {
		case VoteOption_VOTE_OPTION_YES:
			bucket = &agg.yes
		case VoteOption_VOTE_OPTION_NO:
			bucket = &agg.no
		case VoteOption_VOTE_OPTION_ABSTAIN:
			bucket = &agg.abstain
		}
		next, ok := safeAdd(*bucket, v.Power)
		if !ok {
			return fmt.Errorf("vote aggregate for (dao=%d, proposal=%d, option=%v) overflows uint64",
				v.DaoId, v.ProposalId, v.Option)
		}
		*bucket = next
	}

	// Tally reconciliation: each proposal's stored Tally must equal the
	// sum of its votes. With revote replacing the row in place, each
	// surviving Vote is the voter's effective contribution, so the
	// equality is exact. A proposal with no votes must have an all-zero
	// tally (TotalPower is independently checked above).
	for daoID, props := range proposalsByDao {
		for pid, p := range props {
			agg := talliesByProp[tallyAggKey{daoID, pid}]
			var yes, no, abstain uint64
			if agg != nil {
				yes, no, abstain = agg.yes, agg.no, agg.abstain
			}
			if p.Tally.YesPower != yes || p.Tally.NoPower != no || p.Tally.AbstainPower != abstain {
				return fmt.Errorf("proposal %d/%d: tally (yes=%d, no=%d, abstain=%d) does not match votes (yes=%d, no=%d, abstain=%d)",
					daoID, pid, p.Tally.YesPower, p.Tally.NoPower, p.Tally.AbstainPower, yes, no, abstain)
			}
		}
	}

	// --------- Counters ---------
	if err := validateCounters("proposal_id_counters", gs.ProposalIdCounters, seenDaoIDs,
		maxIDByDao(gs.Proposals, func(p Proposal) (uint64, uint64) { return p.DaoId, p.ProposalId }),
	); err != nil {
		return err
	}
	if err := validateCounters("snapshot_id_counters", gs.SnapshotIdCounters, seenDaoIDs,
		maxIDByDao(gs.SnapshotTotals, func(s SnapshotTotalEntry) (uint64, uint64) { return s.DaoId, s.SnapshotId }),
	); err != nil {
		return err
	}

	// --------- Deposit records (Epic 4) ---------
	if err := gs.validateGenesisDepositRecords(proposalsByDao); err != nil {
		return err
	}

	// --------- STATIC members (post-review Finding 2 fix) ---------
	if err := gs.validateGenesisStaticMembers(); err != nil {
		return err
	}

	// --------- Polls + poll votes + poll deposit records (Epic 6) ---------
	if err := gs.validateGenesisPolls(seenDaoIDs); err != nil {
		return err
	}

	return nil
}

// validateGenesisPolls enforces referential integrity over the Epic 6
// fields. Independent of the proposal validators above (rebuilds its own
// snapshot lookups from gs.SnapshotPowers / SnapshotTotals) so the two
// validation phases stay decoupled.
//
// Rules:
//
//   Polls:
//   - dao_id resolves to a known DAO; poll_id non-zero; (dao, poll_id) unique.
//   - status is a known PollStatus variant (UNSPECIFIED rejected).
//   - VOTING polls require non-zero, post-epoch voting_end (mirrors the
//     proposal queue-key safety check).
//   - choices count, label length, NOTA reservation enforced.
//   - max_selections in [1, len(choices)] (after NOTA append in storage,
//     the cap can be up to choices_len; the user-input cap from
//     MsgCreatePoll is the stricter [1, user_choices_len]).
//   - quorum_bps in [0, MaxQuorumBps].
//   - deposit_snapshot satisfies ValidateDepositConfigStateless AND the
//     gs.Params.MaxDepositPeriod ceiling — same posture as proposals.
//   - voting_period_snapshot is in [MinVotingPeriod, gs.Params.MaxVotingPeriod].
//   - tally vectors and counts are internally consistent:
//       * len(tally.choice_power) == len(choices).
//       * tally.total_voted_power <= tally.total_power.
//       * (per-choice sum is approval-style — multi-select allows
//         sum(choice_power) > total_voted_power, so we don't bound that.)
//   - For every poll, a matching SnapshotTotalEntry must exist; total
//     matches tally.total_power.
//   - winning_choice_index is in [0, len(choices)-1] when status == CONCLUDED.
//
//   Poll votes:
//   - (dao_id, poll_id) resolves to a known poll.
//   - voter is valid bech32; (dao, poll, voter) unique.
//   - ValidatePollSelection: range / duplicates / NOTA exclusivity.
//   - Vote.Power matches SnapshotPower(dao, poll.snapshot_id, voter), or
//     0 if no snapshot row exists (mirrors the proposal-side check).
//
//   Poll deposit records:
//   - (dao, poll) resolves to a known poll.
//   - depositor is valid bech32; (dao, poll, depositor) unique.
//   - amount denom matches poll.deposit_snapshot.min_deposit.denom.
//   - Open (DEPOSIT_PERIOD / VOTING) polls: sum of records == deposit_collected.
//   - Terminal polls: no records may exist.
//
//   Counters:
//   - poll_id_counters: dao_id known; unique per dao_id; counter >
//     max(poll_id) for that DAO.
func (gs GenesisState) validateGenesisPolls(seenDaoIDs map[uint64]struct{}) error {
	// Rebuild snapshot lookups (cheap; genesis-size). Could be plumbed
	// from the proposal validator instead, but decoupling is worth the
	// linear extra cost.
	snapTotalByKey := make(map[[2]uint64]uint64, len(gs.SnapshotTotals))
	for _, st := range gs.SnapshotTotals {
		snapTotalByKey[[2]uint64{st.DaoId, st.SnapshotId}] = st.Total
	}
	snapPowerByKey := make(map[string]uint64, len(gs.SnapshotPowers))
	for _, sp := range gs.SnapshotPowers {
		snapPowerByKey[snapPowerKey(sp.DaoId, sp.SnapshotId, sp.Address)] = sp.Power
	}

	// --------- Polls ---------
	pollsByDao := make(map[uint64]map[uint64]*Poll, len(seenDaoIDs))
	for i := range gs.Polls {
		p := &gs.Polls[i]
		if _, ok := seenDaoIDs[p.DaoId]; !ok {
			return fmt.Errorf("poll %d/%d: dao_id not in genesis daos", p.DaoId, p.PollId)
		}
		if p.PollId == 0 {
			return fmt.Errorf("poll at dao %d has zero poll_id", p.DaoId)
		}
		inner, ok := pollsByDao[p.DaoId]
		if !ok {
			inner = make(map[uint64]*Poll)
			pollsByDao[p.DaoId] = inner
		}
		if _, dup := inner[p.PollId]; dup {
			return fmt.Errorf("duplicate poll %d in dao %d", p.PollId, p.DaoId)
		}
		inner[p.PollId] = p

		switch p.Status {
		case PollStatus_POLL_STATUS_DEPOSIT_PERIOD,
			PollStatus_POLL_STATUS_VOTING,
			PollStatus_POLL_STATUS_CONCLUDED,
			PollStatus_POLL_STATUS_REJECTED,
			PollStatus_POLL_STATUS_REJECTED_NO_DEPOSIT,
			PollStatus_POLL_STATUS_CLOSED:
			// ok
		default:
			return fmt.Errorf("poll %d/%d: unknown status %v", p.DaoId, p.PollId, p.Status)
		}
		if p.Status == PollStatus_POLL_STATUS_VOTING {
			if p.VotingEnd.IsZero() {
				return fmt.Errorf("poll %d/%d: VOTING status requires non-zero voting_end", p.DaoId, p.PollId)
			}
			if p.VotingEnd.UnixNano() < 0 {
				return fmt.Errorf("poll %d/%d: voting_end %s is before Unix epoch", p.DaoId, p.PollId, p.VotingEnd)
			}
		}
		// DEPOSIT_PERIOD polls are re-enqueued by InitGenesis at
		// uint64(deposit_deadline.UnixNano()) — guard the same way as
		// VOTING / proposals.
		if p.Status == PollStatus_POLL_STATUS_DEPOSIT_PERIOD {
			if p.DepositDeadline.IsZero() {
				return fmt.Errorf("poll %d/%d: DEPOSIT_PERIOD status requires non-zero deposit_deadline",
					p.DaoId, p.PollId)
			}
			if p.DepositDeadline.UnixNano() < 0 {
				return fmt.Errorf("poll %d/%d: deposit_deadline %s is before Unix epoch",
					p.DaoId, p.PollId, p.DepositDeadline)
			}
		}
		// Status-vs-deposit invariants (mirror the proposal-side checks).
		if !p.DepositCollected.Amount.IsNil() {
			minAmt := p.DepositSnapshot.MinDeposit.Amount
			collected := p.DepositCollected.Amount
			switch p.Status {
			case PollStatus_POLL_STATUS_DEPOSIT_PERIOD:
				if collected.GTE(minAmt) {
					return fmt.Errorf("poll %d/%d: DEPOSIT_PERIOD with collected=%s >= min_deposit=%s (should have transitioned to VOTING)",
						p.DaoId, p.PollId, collected, minAmt)
				}
			case PollStatus_POLL_STATUS_VOTING,
				PollStatus_POLL_STATUS_CONCLUDED,
				PollStatus_POLL_STATUS_REJECTED:
				if collected.LT(minAmt) {
					return fmt.Errorf("poll %d/%d: %s with collected=%s < min_deposit=%s (deposit gate bypassed)",
						p.DaoId, p.PollId, p.Status, collected, minAmt)
				}
			}
			isOpen := p.Status == PollStatus_POLL_STATUS_DEPOSIT_PERIOD ||
				p.Status == PollStatus_POLL_STATUS_VOTING
			if isOpen && p.DepositCollected.Denom != p.DepositSnapshot.MinDeposit.Denom {
				return fmt.Errorf("poll %d/%d: open poll deposit_collected.denom=%q != min_deposit.denom=%q",
					p.DaoId, p.PollId, p.DepositCollected.Denom, p.DepositSnapshot.MinDeposit.Denom)
			}
		}
		if err := ValidatePollChoices(stripNotaForCheck(p.Choices, p.IncludeNota)); err != nil {
			return fmt.Errorf("poll %d/%d: choices: %w", p.DaoId, p.PollId, err)
		}
		// max_selections cap on the in-storage `choices` (which may include
		// NOTA). The MsgCreatePoll path caps against user_choices_len; the
		// stricter check is equivalent when include_nota=false. When
		// include_nota=true the user_choices_len is len(choices)-1, and
		// MsgCreatePoll already enforced the stricter bound; we just
		// verify max_selections > 0 here.
		if p.MaxSelections == 0 || int(p.MaxSelections) > len(p.Choices) {
			return fmt.Errorf("poll %d/%d: max_selections %d not in [1, %d]",
				p.DaoId, p.PollId, p.MaxSelections, len(p.Choices))
		}
		if p.QuorumBps > MaxQuorumBps {
			return fmt.Errorf("poll %d/%d: quorum_bps %d exceeds cap %d",
				p.DaoId, p.PollId, p.QuorumBps, MaxQuorumBps)
		}
		if err := ValidateDepositConfigStateless(p.DepositSnapshot); err != nil {
			return fmt.Errorf("poll %d/%d: deposit_snapshot: %w", p.DaoId, p.PollId, err)
		}
		if err := ValidateDepositConfigAgainstParams(p.DepositSnapshot, gs.Params.MaxDepositPeriod); err != nil {
			return fmt.Errorf("poll %d/%d: deposit_snapshot: %w", p.DaoId, p.PollId, err)
		}
		if p.VotingPeriodSnapshot < MinVotingPeriod {
			return fmt.Errorf("poll %d/%d: voting_period_snapshot %s below floor %s",
				p.DaoId, p.PollId, p.VotingPeriodSnapshot, MinVotingPeriod)
		}
		if p.VotingPeriodSnapshot > gs.Params.MaxVotingPeriod {
			return fmt.Errorf("poll %d/%d: voting_period_snapshot %s exceeds chain cap %s",
				p.DaoId, p.PollId, p.VotingPeriodSnapshot, gs.Params.MaxVotingPeriod)
		}
		if len(p.Tally.ChoicePower) != len(p.Choices) {
			return fmt.Errorf("poll %d/%d: tally.choice_power length %d != choices length %d",
				p.DaoId, p.PollId, len(p.Tally.ChoicePower), len(p.Choices))
		}
		if p.Tally.TotalVotedPower > p.Tally.TotalPower {
			return fmt.Errorf("poll %d/%d: tally total_voted_power %d > total_power %d",
				p.DaoId, p.PollId, p.Tally.TotalVotedPower, p.Tally.TotalPower)
		}
		// Snapshot lock-step: total matches the stored SnapshotTotalEntry.
		totalSnap, hasSnap := snapTotalByKey[[2]uint64{p.DaoId, p.SnapshotId}]
		if !hasSnap {
			return fmt.Errorf("poll %d/%d: missing snapshot_total for snapshot %d",
				p.DaoId, p.PollId, p.SnapshotId)
		}
		if totalSnap != p.Tally.TotalPower {
			return fmt.Errorf("poll %d/%d: tally total_power=%d != snapshot_total=%d",
				p.DaoId, p.PollId, p.Tally.TotalPower, totalSnap)
		}
		if p.Status == PollStatus_POLL_STATUS_CONCLUDED {
			if int(p.WinningChoiceIndex) >= len(p.Choices) {
				return fmt.Errorf("poll %d/%d: winning_choice_index %d out of range [0,%d]",
					p.DaoId, p.PollId, p.WinningChoiceIndex, len(p.Choices)-1)
			}
		}
		// Reviewer Finding 3: terminal poll statuses MUST be justified by
		// the stored tally + quorum + NOTA settings. Without this, a
		// crafted genesis can mark the wrong option as the winner, or
		// mark a poll CONCLUDED when runtime would reject it. Recompute
		// the outcome and compare both status AND winning_choice_index.
		//
		// REJECTED_NO_DEPOSIT polls never reach VOTING and have no tally
		// to recompute.
		switch p.Status {
		case PollStatus_POLL_STATUS_CONCLUDED:
			got := ComputePollOutcome(*p)
			if got.Status != PollStatus_POLL_STATUS_CONCLUDED {
				return fmt.Errorf("poll %d/%d: status CONCLUDED but tally indicates %s",
					p.DaoId, p.PollId, got.Status)
			}
			if got.WinningChoiceIndex != p.WinningChoiceIndex {
				return fmt.Errorf("poll %d/%d: winning_choice_index %d does not match computed %d",
					p.DaoId, p.PollId, p.WinningChoiceIndex, got.WinningChoiceIndex)
			}
		case PollStatus_POLL_STATUS_REJECTED:
			got := ComputePollOutcome(*p)
			if got.Status != PollStatus_POLL_STATUS_REJECTED {
				return fmt.Errorf("poll %d/%d: status REJECTED but tally indicates a CONCLUDED outcome",
					p.DaoId, p.PollId)
			}
		}
	}

	// --------- Poll votes ---------
	// While iterating, accumulate the per-(dao, poll) tally (per-choice
	// power + distinct-voter total) so we can reconcile against the
	// stored Poll.Tally below. Mirrors the proposal-side reconciliation:
	// approval-style "each picked index gets full voter power; voter
	// counts once toward total_voted_power".
	type pollTallyAggKey struct {
		daoID, pollID uint64
	}
	type pollTallyAgg struct {
		choicePower     []uint64
		totalVotedPower uint64
	}
	pollTallyByPoll := make(map[pollTallyAggKey]*pollTallyAgg, len(pollsByDao))
	seenPollVotes := make(map[[3]string]struct{}, len(gs.PollVotes))
	for _, v := range gs.PollVotes {
		pollsInDao, ok := pollsByDao[v.DaoId]
		if !ok {
			return fmt.Errorf("poll_vote (dao=%d, poll=%d, voter=%s): dao not in genesis",
				v.DaoId, v.PollId, v.Voter)
		}
		poll, ok := pollsInDao[v.PollId]
		if !ok {
			return fmt.Errorf("poll_vote (dao=%d, poll=%d, voter=%s): poll not in genesis",
				v.DaoId, v.PollId, v.Voter)
		}
		// Reviewer Finding 4: votes are only meaningful on polls that
		// reached (or are in) VOTING. Pre-loading votes during
		// DEPOSIT_PERIOD lets a crafted import influence the tally before
		// voting could legitimately have happened.
		switch poll.Status {
		case PollStatus_POLL_STATUS_VOTING,
			PollStatus_POLL_STATUS_CONCLUDED,
			PollStatus_POLL_STATUS_REJECTED:
			// ok — poll has reached or passed through VOTING.
		default:
			return fmt.Errorf("poll_vote (dao=%d, poll=%d, voter=%s): poll status %s never reached VOTING; vote not allowed",
				v.DaoId, v.PollId, v.Voter, poll.Status)
		}
		if _, err := sdk.AccAddressFromBech32(v.Voter); err != nil {
			return fmt.Errorf("poll_vote (dao=%d, poll=%d): invalid voter bech32: %w",
				v.DaoId, v.PollId, err)
		}
		if err := ValidatePollSelection(v.ChoiceIndices, len(poll.Choices), poll.MaxSelections, poll.IncludeNota); err != nil {
			return fmt.Errorf("poll_vote (dao=%d, poll=%d, voter=%s): %w",
				v.DaoId, v.PollId, v.Voter, err)
		}
		key := [3]string{
			fmt.Sprintf("%d", v.DaoId),
			fmt.Sprintf("%d", v.PollId),
			v.Voter,
		}
		if _, dup := seenPollVotes[key]; dup {
			return fmt.Errorf("duplicate poll_vote for (dao=%d, poll=%d, voter=%s)",
				v.DaoId, v.PollId, v.Voter)
		}
		seenPollVotes[key] = struct{}{}

		gotPower, hasRow := snapPowerByKey[snapPowerKey(v.DaoId, poll.SnapshotId, v.Voter)]
		if !hasRow {
			if v.Power != 0 {
				return fmt.Errorf("poll_vote (dao=%d, poll=%d, voter=%s): power=%d but no snapshot row",
					v.DaoId, v.PollId, v.Voter, v.Power)
			}
		} else if v.Power != gotPower {
			return fmt.Errorf("poll_vote (dao=%d, poll=%d, voter=%s): power=%d != snapshot=%d",
				v.DaoId, v.PollId, v.Voter, v.Power, gotPower)
		}

		// Aggregate into the post-loop tally reconciliation.
		aggKey := pollTallyAggKey{v.DaoId, v.PollId}
		agg := pollTallyByPoll[aggKey]
		if agg == nil {
			agg = &pollTallyAgg{choicePower: make([]uint64, len(poll.Choices))}
			pollTallyByPoll[aggKey] = agg
		}
		next, ok := safeAdd(agg.totalVotedPower, v.Power)
		if !ok {
			return fmt.Errorf("poll_vote aggregate for (dao=%d, poll=%d): total_voted_power overflows uint64",
				v.DaoId, v.PollId)
		}
		agg.totalVotedPower = next
		for _, idx := range v.ChoiceIndices {
			if int(idx) >= len(agg.choicePower) {
				// Already caught by ValidatePollSelection above; defensive.
				return fmt.Errorf("poll_vote (dao=%d, poll=%d, voter=%s): index %d out of range",
					v.DaoId, v.PollId, v.Voter, idx)
			}
			next, ok := safeAdd(agg.choicePower[idx], v.Power)
			if !ok {
				return fmt.Errorf("poll_vote aggregate for (dao=%d, poll=%d): choice_power[%d] overflows uint64",
					v.DaoId, v.PollId, idx)
			}
			agg.choicePower[idx] = next
		}
	}

	// Tally reconciliation: aggregated vote sums must match each poll's
	// stored Tally. With revote replacing the row in place, surviving
	// votes encode the effective tally exactly.
	for daoID, polls := range pollsByDao {
		for pollID, p := range polls {
			agg := pollTallyByPoll[pollTallyAggKey{daoID, pollID}]
			var (
				gotTotalVoted uint64
				gotChoice     []uint64
			)
			if agg != nil {
				gotTotalVoted = agg.totalVotedPower
				gotChoice = agg.choicePower
			} else {
				gotChoice = make([]uint64, len(p.Choices))
			}
			if p.Tally.TotalVotedPower != gotTotalVoted {
				return fmt.Errorf("poll %d/%d: tally total_voted_power=%d != sum(votes)=%d",
					daoID, pollID, p.Tally.TotalVotedPower, gotTotalVoted)
			}
			for i, want := range p.Tally.ChoicePower {
				if gotChoice[i] != want {
					return fmt.Errorf("poll %d/%d: tally choice_power[%d]=%d != sum(votes)=%d",
						daoID, pollID, i, want, gotChoice[i])
				}
			}
		}
	}

	// --------- Counters ---------
	if err := validateCounters("poll_id_counters", gs.PollIdCounters, seenDaoIDs,
		maxIDByDao(gs.Polls, func(p Poll) (uint64, uint64) { return p.DaoId, p.PollId }),
	); err != nil {
		return err
	}

	// --------- Poll deposit records ---------
	type pollDepositKey struct{ daoID, pollID uint64 }
	sums := make(map[pollDepositKey]uint64)
	seenDeposits := make(map[[3]string]struct{}, len(gs.PollDepositRecords))
	for _, r := range gs.PollDepositRecords {
		pollsInDao, ok := pollsByDao[r.DaoId]
		if !ok {
			return fmt.Errorf("poll_deposit_record (dao=%d, poll=%d, depositor=%s): dao not in genesis",
				r.DaoId, r.ProposalId, r.Depositor)
		}
		poll, ok := pollsInDao[r.ProposalId]
		if !ok {
			return fmt.Errorf("poll_deposit_record (dao=%d, poll=%d, depositor=%s): poll not in genesis",
				r.DaoId, r.ProposalId, r.Depositor)
		}
		if _, err := sdk.AccAddressFromBech32(r.Depositor); err != nil {
			return fmt.Errorf("poll_deposit_record (dao=%d, poll=%d): invalid depositor bech32: %w",
				r.DaoId, r.ProposalId, err)
		}
		if err := r.Amount.Validate(); err != nil {
			return fmt.Errorf("poll_deposit_record (dao=%d, poll=%d, depositor=%s): amount: %w",
				r.DaoId, r.ProposalId, r.Depositor, err)
		}
		if r.Amount.Amount.IsNil() || !r.Amount.Amount.IsPositive() {
			return fmt.Errorf("poll_deposit_record (dao=%d, poll=%d, depositor=%s): amount must be > 0",
				r.DaoId, r.ProposalId, r.Depositor)
		}
		switch poll.Status {
		case PollStatus_POLL_STATUS_CONCLUDED,
			PollStatus_POLL_STATUS_REJECTED,
			PollStatus_POLL_STATUS_REJECTED_NO_DEPOSIT,
			PollStatus_POLL_STATUS_CLOSED:
			return fmt.Errorf("poll_deposit_record (dao=%d, poll=%d, depositor=%s): poll is terminal (%s); records should have been disbursed",
				r.DaoId, r.ProposalId, r.Depositor, poll.Status)
		}
		if r.Amount.Denom != poll.DepositSnapshot.MinDeposit.Denom {
			return fmt.Errorf("poll_deposit_record (dao=%d, poll=%d, depositor=%s): denom %q != poll min_deposit denom %q",
				r.DaoId, r.ProposalId, r.Depositor, r.Amount.Denom, poll.DepositSnapshot.MinDeposit.Denom)
		}
		key := [3]string{
			fmt.Sprintf("%d", r.DaoId),
			fmt.Sprintf("%d", r.ProposalId),
			r.Depositor,
		}
		if _, dup := seenDeposits[key]; dup {
			return fmt.Errorf("duplicate poll_deposit_record for (dao=%d, poll=%d, depositor=%s)",
				r.DaoId, r.ProposalId, r.Depositor)
		}
		seenDeposits[key] = struct{}{}
		next, ok := safeAdd(sums[pollDepositKey{r.DaoId, r.ProposalId}], r.Amount.Amount.Uint64())
		if !ok {
			return fmt.Errorf("poll_deposit_record aggregate for (dao=%d, poll=%d) overflows uint64",
				r.DaoId, r.ProposalId)
		}
		sums[pollDepositKey{r.DaoId, r.ProposalId}] = next
	}
	// Open polls: sum of records == deposit_collected.
	for daoID, polls := range pollsByDao {
		for pollID, p := range polls {
			isOpen := p.Status == PollStatus_POLL_STATUS_DEPOSIT_PERIOD ||
				p.Status == PollStatus_POLL_STATUS_VOTING
			if !isOpen {
				continue
			}
			if p.DepositCollected.Amount.IsNil() {
				return fmt.Errorf("poll %d/%d: open poll has nil deposit_collected.amount",
					daoID, pollID)
			}
			if !p.DepositCollected.Amount.IsUint64() {
				return fmt.Errorf("poll %d/%d: deposit_collected %s does not fit in uint64",
					daoID, pollID, p.DepositCollected.Amount)
			}
			if p.DepositCollected.Amount.Uint64() != sums[pollDepositKey{daoID, pollID}] {
				return fmt.Errorf("poll %d/%d: deposit_collected.amount=%s != sum(poll_deposit_records)=%d",
					daoID, pollID, p.DepositCollected.Amount, sums[pollDepositKey{daoID, pollID}])
			}
		}
	}
	return nil
}

// validateGenesisStaticMembers enforces referential integrity on the
// STATIC member round-trip, including the same runtime caps that
// MsgCreateDao / MsgUpdateMembers enforce:
//
//   - Each entry's dao_id resolves to a DAO in `daos` with
//     voting_backend == STATIC.
//   - address is valid bech32; weight > 0; (dao_id, address) is unique.
//   - Every STATIC DAO has 1..MaxStaticMembers entries.
//   - The per-DAO sum of weights does not overflow uint64 (the cached
//     StaticTotalPowerKey is recomputed on import; without a preflight
//     check here, InitGenesis would panic mid-import on overflow).
//
// The cached total is NOT validated against the per-DAO sum here — it's
// not serialized; InitGenesis recomputes it. The overflow check below
// is the same one applyStaticMembersInit performs at runtime.
func (gs GenesisState) validateGenesisStaticMembers() error {
	daosByID := make(map[uint64]*Dao, len(gs.Daos))
	for i := range gs.Daos {
		daosByID[gs.Daos[i].Id] = &gs.Daos[i]
	}

	// Group entries per DAO; defer the per-DAO cap + overflow check until
	// after the full pass. Per-entry rules are checked inline.
	membersPerDao := make(map[uint64][]StaticMember, len(daosByID))
	seen := make(map[[2]string]struct{}, len(gs.StaticMembers))
	for _, m := range gs.StaticMembers {
		dao, ok := daosByID[m.DaoId]
		if !ok {
			return fmt.Errorf("static_member (dao=%d, addr=%s): dao not in genesis", m.DaoId, m.Address)
		}
		if dao.VotingBackend != VotingBackendType_VOTING_BACKEND_STATIC {
			return fmt.Errorf("static_member (dao=%d, addr=%s): dao voting_backend is %s, not STATIC",
				m.DaoId, m.Address, dao.VotingBackend)
		}
		if _, err := sdk.AccAddressFromBech32(m.Address); err != nil {
			return fmt.Errorf("static_member (dao=%d): invalid bech32 %q: %w", m.DaoId, m.Address, err)
		}
		if m.Weight == 0 {
			return fmt.Errorf("static_member (dao=%d, addr=%s): weight must be > 0", m.DaoId, m.Address)
		}
		key := [2]string{fmt.Sprintf("%d", m.DaoId), m.Address}
		if _, dup := seen[key]; dup {
			return fmt.Errorf("duplicate static_member for (dao=%d, addr=%s)", m.DaoId, m.Address)
		}
		seen[key] = struct{}{}
		membersPerDao[m.DaoId] = append(membersPerDao[m.DaoId], StaticMember{
			Address: m.Address,
			Weight:  m.Weight,
		})
	}

	// Per-DAO caps + overflow-safe weight sum. Reviewer Finding 5:
	//   - MsgUpdateMembers' MaxStaticMembers cap also applies to imports.
	//     Without this, a crafted genesis could land a STATIC DAO well
	//     above the SnapshotAll iteration bound the keeper relies on.
	//   - Sum overflow check mirrors applyStaticMembersInit; otherwise
	//     InitGenesis would panic with "static total weight overflow"
	//     AFTER Validate said the input was fine — surfacing that here
	//     is the correct posture (Validate guarantees InitGenesis won't
	//     panic).
	for id, dao := range daosByID {
		if dao.VotingBackend != VotingBackendType_VOTING_BACKEND_STATIC {
			continue
		}
		members := membersPerDao[id]
		if len(members) == 0 {
			return fmt.Errorf("dao %d: STATIC DAO has no static_member entries in genesis", id)
		}
		if len(members) > MaxStaticMembers {
			return fmt.Errorf("dao %d: %d static_members exceeds cap %d",
				id, len(members), MaxStaticMembers)
		}
		var total uint64
		for _, m := range members {
			next, ok := safeAdd(total, m.Weight)
			if !ok {
				return fmt.Errorf("dao %d: per-DAO static-weight sum overflows uint64", id)
			}
			total = next
		}
	}
	return nil
}

// stripNotaForCheck removes the keeper-appended NOTA label (if any) from
// the choice slice so we can validate the user-supplied portion against
// the same ValidatePollChoices used at MsgCreatePoll time.
//
// We don't validate the NOTA label here; we trust that include_nota=true
// implies the tail is NotaLabel. The keeper writes that string itself
// at poll creation; a chain operator who hand-crafts genesis with a
// different tail would silently break UI display but not protocol logic.
func stripNotaForCheck(choices []string, includeNota bool) []string {
	if !includeNota || len(choices) == 0 {
		return choices
	}
	return choices[:len(choices)-1]
}

// validateGenesisDepositRecords enforces the referential-integrity rules for
// Epic 4 deposit records:
//
//   - Each row's (dao_id, proposal_id) resolves to a known proposal.
//   - depositor is valid bech32; (dao, proposal, depositor) unique.
//   - amount is a structurally valid Coin with positive amount and
//     matches the proposal's deposit_snapshot.min_deposit.denom.
//   - Open proposals (DEPOSIT_PERIOD / VOTING): sum of DepositRecord
//     amounts must equal Proposal.deposit_collected.
//   - Terminal proposals (PASSED / REJECTED / EXECUTED / REJECTED_NO_DEPOSIT):
//     no DepositRecord rows may exist — terminal proposals have already
//     disbursed their escrow.
func (gs GenesisState) validateGenesisDepositRecords(proposalsByDao map[uint64]map[uint64]*Proposal) error {
	type aggKey struct{ daoID, proposalID uint64 }
	sums := make(map[aggKey]uint64)
	seen := make(map[[3]string]struct{}, len(gs.DepositRecords))

	for _, r := range gs.DepositRecords {
		propsInDao, ok := proposalsByDao[r.DaoId]
		if !ok {
			return fmt.Errorf("deposit_record (dao=%d, proposal=%d, depositor=%s): dao not in genesis",
				r.DaoId, r.ProposalId, r.Depositor)
		}
		p, ok := propsInDao[r.ProposalId]
		if !ok {
			return fmt.Errorf("deposit_record (dao=%d, proposal=%d, depositor=%s): proposal not in genesis",
				r.DaoId, r.ProposalId, r.Depositor)
		}
		if _, err := sdk.AccAddressFromBech32(r.Depositor); err != nil {
			return fmt.Errorf("deposit_record (dao=%d, proposal=%d): invalid depositor bech32: %w",
				r.DaoId, r.ProposalId, err)
		}
		if err := r.Amount.Validate(); err != nil {
			return fmt.Errorf("deposit_record (dao=%d, proposal=%d, depositor=%s): amount: %w",
				r.DaoId, r.ProposalId, r.Depositor, err)
		}
		if r.Amount.Amount.IsNil() || !r.Amount.Amount.IsPositive() {
			return fmt.Errorf("deposit_record (dao=%d, proposal=%d, depositor=%s): amount must be > 0",
				r.DaoId, r.ProposalId, r.Depositor)
		}
		// Terminal proposals shouldn't have surviving records.
		switch p.Status {
		case ProposalStatus_PROPOSAL_STATUS_PASSED,
			ProposalStatus_PROPOSAL_STATUS_REJECTED,
			ProposalStatus_PROPOSAL_STATUS_EXECUTED,
			ProposalStatus_PROPOSAL_STATUS_REJECTED_NO_DEPOSIT:
			return fmt.Errorf("deposit_record (dao=%d, proposal=%d, depositor=%s): proposal is terminal (%s); records should have been disbursed",
				r.DaoId, r.ProposalId, r.Depositor, p.Status)
		}
		// Denom must match the proposal's frozen min_deposit denom.
		if r.Amount.Denom != p.DepositSnapshot.MinDeposit.Denom {
			return fmt.Errorf("deposit_record (dao=%d, proposal=%d, depositor=%s): denom %q != proposal min_deposit denom %q",
				r.DaoId, r.ProposalId, r.Depositor, r.Amount.Denom, p.DepositSnapshot.MinDeposit.Denom)
		}
		// Uniqueness.
		key := [3]string{
			fmt.Sprintf("%d", r.DaoId),
			fmt.Sprintf("%d", r.ProposalId),
			r.Depositor,
		}
		if _, dup := seen[key]; dup {
			return fmt.Errorf("duplicate deposit_record for (dao=%d, proposal=%d, depositor=%s)",
				r.DaoId, r.ProposalId, r.Depositor)
		}
		seen[key] = struct{}{}
		// Aggregate by proposal for the sum check below.
		next, ok := safeAdd(sums[aggKey{r.DaoId, r.ProposalId}], r.Amount.Amount.Uint64())
		if !ok {
			return fmt.Errorf("deposit_record aggregate for (dao=%d, proposal=%d) overflows uint64",
				r.DaoId, r.ProposalId)
		}
		sums[aggKey{r.DaoId, r.ProposalId}] = next
	}

	// Open proposals: sum of records == deposit_collected. We compare
	// uint64 representations since on-chain coin amounts are math.Int but
	// realistic deposit totals fit in uint64 (BZE supply is ~10^14). If
	// a proposal's deposit_collected doesn't fit in uint64, we surface
	// the error so the importer doesn't silently accept truncated data.
	for daoID, props := range proposalsByDao {
		for pid, p := range props {
			isOpen := p.Status == ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD ||
				p.Status == ProposalStatus_PROPOSAL_STATUS_VOTING
			recordedSum := sums[aggKey{daoID, pid}]
			if !isOpen {
				// Terminal proposals already checked above — no records
				// allowed. We don't enforce deposit_collected == 0 because
				// the runtime keeps that field as historical state after
				// disbursement.
				continue
			}
			if p.DepositCollected.Amount.IsNil() {
				return fmt.Errorf("proposal %d/%d: open proposal has nil deposit_collected.amount",
					daoID, pid)
			}
			collected := p.DepositCollected.Amount
			if !collected.IsUint64() {
				return fmt.Errorf("proposal %d/%d: deposit_collected %s does not fit in uint64",
					daoID, pid, collected)
			}
			if collected.Uint64() != recordedSum {
				return fmt.Errorf("proposal %d/%d: deposit_collected.amount=%s != sum(deposit_records)=%d",
					daoID, pid, collected, recordedSum)
			}
		}
	}
	return nil
}

// validateCounters enforces: dao_id in `daos`; unique per dao_id; counter
// strictly > maxIDPerDao[dao_id] (or counter >= 1 if the DAO has no
// matching entries).
func validateCounters(label string, counters []PerDaoUint64, seenDaoIDs map[uint64]struct{}, maxIDPerDao map[uint64]uint64) error {
	seen := make(map[uint64]struct{}, len(counters))
	for _, c := range counters {
		if _, ok := seenDaoIDs[c.DaoId]; !ok {
			return fmt.Errorf("%s: dao_id %d not in genesis daos", label, c.DaoId)
		}
		if _, dup := seen[c.DaoId]; dup {
			return fmt.Errorf("%s: duplicate entry for dao_id %d", label, c.DaoId)
		}
		seen[c.DaoId] = struct{}{}
		if c.Value == 0 {
			return fmt.Errorf("%s: counter for dao_id %d must be >= 1", label, c.DaoId)
		}
		if maxID, ok := maxIDPerDao[c.DaoId]; ok && c.Value <= maxID {
			return fmt.Errorf("%s: counter for dao_id %d is %d but max id present is %d",
				label, c.DaoId, c.Value, maxID)
		}
	}
	// Any DAO that has proposals/snapshots but no explicit counter is also
	// a bug — the export would later allow id reuse.
	for daoID := range maxIDPerDao {
		if _, ok := seen[daoID]; !ok {
			return fmt.Errorf("%s: dao_id %d has entries but no counter", label, daoID)
		}
	}
	return nil
}

// maxIDByDao builds a (dao_id → max(id)) map from a slice of records,
// keyed by a per-record extractor that returns (dao_id, id).
func maxIDByDao[T any](items []T, extract func(T) (daoID, id uint64)) map[uint64]uint64 {
	out := make(map[uint64]uint64)
	for _, it := range items {
		daoID, id := extract(it)
		if id > out[daoID] {
			out[daoID] = id
		}
	}
	return out
}

// safeAdd returns a+b and reports whether the sum stayed within uint64.
// Mirrors the keeper's safeAddU64 but lives here so the types package can
// reuse it for tally / aggregate-vote overflow checks without an import
// cycle.
func safeAdd(a, b uint64) (uint64, bool) {
	s := a + b
	if s < a {
		return 0, false
	}
	return s, true
}

// snapPowerKey packs (dao_id, snapshot_id, address) into a single string
// suitable for use as a Go map key. Address is already bech32-encoded so
// it's collision-free against the numeric ids when separated by "/".
func snapPowerKey(daoID, snapshotID uint64, address string) string {
	return fmt.Sprintf("%d/%d/%s", daoID, snapshotID, address)
}

// Validate performs basic genesis state validation. It returns an error on
// any structural problem (bad Params, duplicate DAO ids, dangling parent
// references, ID counter inconsistencies).
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// The id counter is monotonic and 0 is reserved as "unset" in messages
	// and queries. It must be at least 1, even on an empty genesis (otherwise
	// the first allocated id would be 0 and unreachable).
	if gs.DaoIdCounter == 0 {
		return fmt.Errorf("dao_id_counter must be >= 1")
	}

	seenIDs := make(map[uint64]struct{}, len(gs.Daos))
	seenAddrs := make(map[string]struct{}, len(gs.Daos))
	daosByID := make(map[uint64]*Dao, len(gs.Daos))
	maxID := uint64(0)

	for i := range gs.Daos {
		dao := gs.Daos[i]

		if dao.Id == 0 {
			return fmt.Errorf("dao at index %d has id 0", i)
		}
		if _, dup := seenIDs[dao.Id]; dup {
			return fmt.Errorf("duplicate dao id %d", dao.Id)
		}
		seenIDs[dao.Id] = struct{}{}
		daosByID[dao.Id] = &gs.Daos[i]
		if dao.Id > maxID {
			maxID = dao.Id
		}

		// account_address must match the deterministic derivation
		want := DaoAccountAddress(dao.Id).String()
		if dao.AccountAddress != want {
			return fmt.Errorf("dao %d: account_address %q does not match derived %q",
				dao.Id, dao.AccountAddress, want)
		}
		if _, dup := seenAddrs[dao.AccountAddress]; dup {
			return fmt.Errorf("duplicate account_address %q", dao.AccountAddress)
		}
		seenAddrs[dao.AccountAddress] = struct{}{}

		// admin must parse
		if _, err := sdk.AccAddressFromBech32(dao.Admin); err != nil {
			return fmt.Errorf("dao %d: invalid admin: %w", dao.Id, err)
		}
		// pending_admin is optional but must parse if set
		if dao.PendingAdmin != "" {
			if _, err := sdk.AccAddressFromBech32(dao.PendingAdmin); err != nil {
				return fmt.Errorf("dao %d: invalid pending_admin: %w", dao.Id, err)
			}
		}
		// creator must parse
		if _, err := sdk.AccAddressFromBech32(dao.Creator); err != nil {
			return fmt.Errorf("dao %d: invalid creator: %w", dao.Id, err)
		}
		// metadata must be valid
		if err := ValidateDaoMetadata(dao.Metadata); err != nil {
			return fmt.Errorf("dao %d: %w", dao.Id, err)
		}
		// voting_backend must be set to a known variant.
		//
		// Epic 2 only accepts STATIC DAOs in genesis. REWARD_STAKED is rejected
		// at the public message path (MsgCreateDao). Epic 3 implements the
		// SnapshotAll iterator path but the backend swap that legitimately
		// produces a REWARD_STAKED DAO still requires Epic 5
		// (MsgUpdateVotingBackend). Accepting REWARD_STAKED in genesis today
		// would be an asymmetric smuggling channel: a chain operator could
		// plant DAOs that no user tx can create. Relax this once Epic 5 lands.
		switch dao.VotingBackend {
		case VotingBackendType_VOTING_BACKEND_STATIC:
			// REWARD_STAKED DAOs carry a reward_id; STATIC DAOs must NOT.
			if dao.RewardId != "" {
				return fmt.Errorf("dao %d: STATIC backend must not set reward_id", dao.Id)
			}
		case VotingBackendType_VOTING_BACKEND_REWARD_STAKED:
			return fmt.Errorf("dao %d: REWARD_STAKED backend is not accepted in genesis yet "+
				"(create as STATIC and swap via MsgUpdateVotingBackend once Epic 5 lands)", dao.Id)
		default:
			return fmt.Errorf("dao %d: voting_backend must be a known variant, got %v", dao.Id, dao.VotingBackend)
		}
		// Epic 3: every DAO record must carry a valid GovernanceConfig. We
		// enforce BOTH the stateless caps (range / variant) and the
		// Param-driven voting_period ceiling here — gs.Params is already
		// loaded for this Validate call, and InitGenesis just persists DAOs
		// verbatim, so genesis is the only place to catch a config that
		// blows past Params.MaxVotingPeriod. (MsgCreateDao and
		// MsgUpdateGovernanceConfig already validate the same way against
		// the live keeper Params; matching the rule here keeps the tx and
		// genesis paths symmetric.)
		if err := ValidateGovernanceConfigStateless(dao.Governance); err != nil {
			return fmt.Errorf("dao %d: %w", dao.Id, err)
		}
		if err := ValidateGovernanceConfigAgainstParams(dao.Governance, gs.Params.MaxVotingPeriod); err != nil {
			return fmt.Errorf("dao %d: %w", dao.Id, err)
		}
		// Epic 4: deposit config validates analogously to governance —
		// stateless caps + Param-driven deposit_period ceiling. Without
		// this, an importer could plant DAOs that MsgCreateDao /
		// MsgUpdateDepositConfig would reject at runtime.
		if err := ValidateDepositConfigStateless(dao.Deposit); err != nil {
			return fmt.Errorf("dao %d: %w", dao.Id, err)
		}
		if err := ValidateDepositConfigAgainstParams(dao.Deposit, gs.Params.MaxDepositPeriod); err != nil {
			return fmt.Errorf("dao %d: %w", dao.Id, err)
		}
		// Note: Epic 2 does NOT serialize STATIC member lists into genesis.
		// A STATIC DAO imported from genesis lands with zero members and is
		// "frozen" until its admin runs MsgUpdateMembers. Member-in-genesis
		// support is a known limitation to be addressed in a follow-up
		// (would require extending GenesisState with a `members` field —
		// see cross-module-notes.md).
	}

	// parent_dao_id references must point to DAOs included in this genesis,
	// must not be self, and must not form a cycle. Runtime creation prevents
	// cycles; genesis is the only path that could otherwise smuggle one in.
	for i := range gs.Daos {
		dao := gs.Daos[i]
		if dao.ParentDaoId == 0 {
			continue
		}
		if _, ok := seenIDs[dao.ParentDaoId]; !ok {
			return fmt.Errorf("dao %d: parent_dao_id %d not in genesis daos", dao.Id, dao.ParentDaoId)
		}
		if dao.ParentDaoId == dao.Id {
			return fmt.Errorf("dao %d: parent_dao_id equals self", dao.Id)
		}
		// Walk the parent chain from dao.Id; reject if we revisit any id.
		// Bounded by len(daos) so it always terminates.
		visited := map[uint64]struct{}{dao.Id: {}}
		cur := dao.ParentDaoId
		for steps := 0; steps <= len(gs.Daos); steps++ {
			if cur == 0 {
				break
			}
			if _, seen := visited[cur]; seen {
				return fmt.Errorf("dao %d: parent chain contains a cycle at %d", dao.Id, cur)
			}
			visited[cur] = struct{}{}
			parent, ok := daosByID[cur]
			if !ok {
				// Already handled by the membership check above for the first
				// hop; multi-hop refs to missing daos are caught here too.
				return fmt.Errorf("dao %d: parent chain references missing dao %d", dao.Id, cur)
			}
			cur = parent.ParentDaoId
		}
	}

	// dao_id_counter must be strictly greater than every imported id.
	if gs.DaoIdCounter <= maxID {
		return fmt.Errorf("dao_id_counter (%d) must be > max dao id (%d)", gs.DaoIdCounter, maxID)
	}

	// Epic 3 fields — proposals / votes / snapshots / per-DAO counters.
	if err := gs.validateGenesisProposalsAndVotes(seenIDs); err != nil {
		return err
	}

	return nil
}
