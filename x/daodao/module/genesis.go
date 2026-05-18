package daodao

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/keeper"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// InitGenesis loads module state from a validated GenesisState.
//
// Order matters:
//  1. Params first (so any read of params during DAO restoration is correct).
//  2. DAOs: for each, ensure the BaseAccount exists in x/auth, write the
//     Dao record, write the secondary indices.
//  3. DAO counter — after every DAO is written, set the counter to the
//     genesis value (which Validate has confirmed is > max id).
//  4. Snapshot tables (totals + powers) and per-DAO snapshot id counters.
//     Snapshots are loaded BEFORE proposals because votes inserted in
//     step 6 read SnapshotPower for their snapshot id; we don't want a
//     race where a vote insertion sees a missing snapshot row.
//  5. Proposals: persist each record AND its derived state — the
//     ProposalByStatusKey index plus, for VOTING proposals, the
//     ExpiringProposalKey timer entry. Per-DAO proposal id counters are
//     set after the proposal write loop.
//  6. Votes: write each Vote record. We do NOT recompute tallies here —
//     the tally is part of the Proposal record that we already restored.
//
// Validation has already been run by ValidateGenesis at the module level;
// this function panics on any unexpected state, treating it as data
// corruption.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	// --- DAOs ---
	for i := range genState.Daos {
		dao := genState.Daos[i]
		addr, err := sdk.AccAddressFromBech32(dao.AccountAddress)
		if err != nil {
			panic(fmt.Errorf("genesis: dao %d: invalid account_address: %w", dao.Id, err))
		}
		if !k.HasAccount(ctx, addr) {
			acc := k.NewAccountWithAddress(ctx, addr)
			k.SetAccount(ctx, acc)
		}
		k.SetDao(ctx, dao)
		if err := k.SetDaoIndices(ctx, dao); err != nil {
			panic(fmt.Errorf("genesis: dao %d: set indices: %w", dao.Id, err))
		}
	}
	k.SetDaoIDCounter(ctx, genState.DaoIdCounter)

	// --- Snapshot tables (before proposals — see step ordering above) ---
	for _, st := range genState.SnapshotTotals {
		k.SetSnapshotTotal(ctx, st.DaoId, st.SnapshotId, st.Total)
	}
	for _, sp := range genState.SnapshotPowers {
		addr, err := sdk.AccAddressFromBech32(sp.Address)
		if err != nil {
			// Validate has run; this is data corruption.
			panic(fmt.Errorf("genesis: snapshot_power (dao=%d, snap=%d, addr=%q): %w",
				sp.DaoId, sp.SnapshotId, sp.Address, err))
		}
		k.SetSnapshotPower(ctx, sp.DaoId, sp.SnapshotId, addr, sp.Power)
	}
	for _, c := range genState.SnapshotIdCounters {
		k.SetSnapshotIDCounter(ctx, c.DaoId, c.Value)
	}

	// --- Proposals (primary record + status index + expiring queue) ---
	for i := range genState.Proposals {
		p := genState.Proposals[i]
		k.SetProposalNew(ctx, p)
		// Re-enqueue BOTH DEPOSIT_PERIOD and VOTING proposals so the
		// end-blocker can finalize them. DEPOSIT_PERIOD entries fire at
		// deposit_deadline (forfeit path); VOTING entries fire at
		// voting_end (tally path). EnqueueExpiringProposal picks the
		// right deadline via proposalDeadlineNs based on status.
		// Terminal proposals (PASSED/REJECTED/EXECUTED/...) don't belong
		// on the queue.
		if p.Status == types.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD ||
			p.Status == types.ProposalStatus_PROPOSAL_STATUS_VOTING {
			k.EnqueueExpiringProposal(ctx, p)
		}
	}
	for _, c := range genState.ProposalIdCounters {
		k.SetProposalIDCounter(ctx, c.DaoId, c.Value)
	}

	// --- Votes ---
	for _, v := range genState.Votes {
		if err := k.SetVote(ctx, v); err != nil {
			panic(fmt.Errorf("genesis: vote (dao=%d, proposal=%d, voter=%q): %w",
				v.DaoId, v.ProposalId, v.Voter, err))
		}
	}

	// --- Deposit records (Epic 4) ---
	for _, r := range genState.DepositRecords {
		if err := k.SetDepositRecord(ctx, r); err != nil {
			panic(fmt.Errorf("genesis: deposit_record (dao=%d, proposal=%d, depositor=%q): %w",
				r.DaoId, r.ProposalId, r.Depositor, err))
		}
	}

	// --- STATIC members (post-review Finding 2 fix) ---
	// Group entries by dao_id, then restore each DAO's member set in one
	// call so the cached StaticTotalPowerKey is set to the full per-DAO
	// sum. Validate has already confirmed each entry's dao_id maps to a
	// STATIC DAO in `daos`.
	staticMembersByDao := make(map[uint64][]types.StaticMember)
	for _, m := range genState.StaticMembers {
		staticMembersByDao[m.DaoId] = append(staticMembersByDao[m.DaoId], types.StaticMember{
			Address: m.Address,
			Weight:  m.Weight,
		})
	}
	for daoID, members := range staticMembersByDao {
		if err := k.InitStaticMembers(ctx, daoID, members); err != nil {
			panic(fmt.Errorf("genesis: static_members for dao %d: %w", daoID, err))
		}
	}

	// --- Polls (Epic 6) ---
	// Order parallel to proposals: persist record + status index, then
	// re-enqueue open polls on the end-blocker timer.
	for i := range genState.Polls {
		p := genState.Polls[i]
		k.SetPollNew(ctx, p)
		if p.Status == types.PollStatus_POLL_STATUS_DEPOSIT_PERIOD ||
			p.Status == types.PollStatus_POLL_STATUS_VOTING {
			k.EnqueueExpiringPoll(ctx, p)
		}
	}
	for _, c := range genState.PollIdCounters {
		k.SetPollIDCounter(ctx, c.DaoId, c.Value)
	}
	for _, v := range genState.PollVotes {
		if err := k.SetPollVote(ctx, v); err != nil {
			panic(fmt.Errorf("genesis: poll_vote (dao=%d, poll=%d, voter=%q): %w",
				v.DaoId, v.PollId, v.Voter, err))
		}
	}
	for _, r := range genState.PollDepositRecords {
		if err := k.SetPollDepositRecord(ctx, r); err != nil {
			panic(fmt.Errorf("genesis: poll_deposit_record (dao=%d, poll=%d, depositor=%q): %w",
				r.DaoId, r.ProposalId, r.Depositor, err))
		}
	}
}

// ExportGenesis dumps the module's state.
//
// The export walks each table in deterministic key order (the underlying
// KVStore iteration is byte-sorted by key, so successive exports produce
// identical bytes given identical state). Derived state (status index,
// expiring queue) is NOT exported — InitGenesis rebuilds it from
// `proposals`.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.DaoIdCounter = k.GetDaoIDCounter(ctx)

	k.IterateDaos(ctx, func(d types.Dao) (stop bool) {
		genesis.Daos = append(genesis.Daos, d)
		return false
	})

	k.IterateAllProposals(ctx, func(p types.Proposal) (stop bool) {
		genesis.Proposals = append(genesis.Proposals, p)
		return false
	})

	k.IterateAllVotes(ctx, func(v types.Vote) (stop bool) {
		genesis.Votes = append(genesis.Votes, v)
		return false
	})

	k.IterateProposalIDCounters(ctx, func(daoID, counter uint64) (stop bool) {
		genesis.ProposalIdCounters = append(genesis.ProposalIdCounters,
			types.PerDaoUint64{DaoId: daoID, Value: counter})
		return false
	})

	k.IterateSnapshotIDCounters(ctx, func(daoID, counter uint64) (stop bool) {
		genesis.SnapshotIdCounters = append(genesis.SnapshotIdCounters,
			types.PerDaoUint64{DaoId: daoID, Value: counter})
		return false
	})

	k.IterateAllSnapshotPowers(ctx, func(daoID, snapID uint64, addr sdk.AccAddress, power uint64) (stop bool) {
		genesis.SnapshotPowers = append(genesis.SnapshotPowers, types.SnapshotPowerEntry{
			DaoId:      daoID,
			SnapshotId: snapID,
			Address:    addr.String(),
			Power:      power,
		})
		return false
	})

	k.IterateAllSnapshotTotals(ctx, func(daoID, snapID, total uint64) (stop bool) {
		genesis.SnapshotTotals = append(genesis.SnapshotTotals, types.SnapshotTotalEntry{
			DaoId:      daoID,
			SnapshotId: snapID,
			Total:      total,
		})
		return false
	})

	k.IterateAllDepositRecords(ctx, func(r types.DepositRecord) (stop bool) {
		genesis.DepositRecords = append(genesis.DepositRecords, r)
		return false
	})

	// --- STATIC members (post-review Finding 2 fix) ---
	k.IterateAllStaticMembers(ctx, func(daoID uint64, addr sdk.AccAddress, weight uint64) (stop bool) {
		genesis.StaticMembers = append(genesis.StaticMembers, types.StaticMemberEntry{
			DaoId:   daoID,
			Address: addr.String(),
			Weight:  weight,
		})
		return false
	})

	// --- Polls (Epic 6) ---
	k.IterateAllPolls(ctx, func(p types.Poll) (stop bool) {
		genesis.Polls = append(genesis.Polls, p)
		return false
	})
	k.IterateAllPollVotes(ctx, func(v types.PollVote) (stop bool) {
		genesis.PollVotes = append(genesis.PollVotes, v)
		return false
	})
	k.IteratePollIDCounters(ctx, func(daoID, counter uint64) (stop bool) {
		genesis.PollIdCounters = append(genesis.PollIdCounters,
			types.PerDaoUint64{DaoId: daoID, Value: counter})
		return false
	})
	k.IterateAllPollDepositRecords(ctx, func(r types.DepositRecord) (stop bool) {
		genesis.PollDepositRecords = append(genesis.PollDepositRecords, r)
		return false
	})

	return genesis
}
