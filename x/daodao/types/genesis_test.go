package types_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// validGovernance returns a permissive GovernanceConfig that passes
// ValidateGovernanceConfigStateless. Used by genesis fixtures so we
// exercise non-governance behavior without governance noise.
func validGovernance() types.GovernanceConfig {
	return types.GovernanceConfig{
		ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
		ThresholdBps: 5_000,
		QuorumBps:    0,
		VotingPeriod: 24 * time.Hour,
		AllowRevote:  true,
	}
}

// validDeposit is the DepositConfig threaded into "valid" genesis fixtures.
func validDeposit() types.DepositConfig {
	return types.DepositConfig{
		MinDeposit:         sdk.NewInt64Coin("ubze", 1),
		DepositPeriod:      7 * 24 * time.Hour,
		ForfeitDestination: types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
		VotingRefundPolicy: types.RefundPolicy_REFUND_POLICY_ON_PASS,
	}
}

// validDao returns a DAO with realistic bech32 addresses and a STATIC
// voting backend (the simplest valid shape). The same `creator` is used for
// `admin` so the record is self-consistent. Governance / deposit are set
// to permissive defaults.
func validDao(id uint64, creator string) types.Dao {
	return types.Dao{
		Id:             id,
		Metadata:       types.DaoMetadata{Name: "dao"},
		Creator:        creator,
		AccountAddress: types.DaoAccountAddress(id).String(),
		Admin:          creator,
		VotingBackend:  types.VotingBackendType_VOTING_BACKEND_STATIC,
		Governance:     validGovernance(),
		Deposit:        validDeposit(),
	}
}

func TestGenesisState_Validate(t *testing.T) {
	addr := sample.AccAddress()

	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "single dao, no parent",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				// STATIC DAOs require at least one member row in genesis
				// (post-review Finding 2). Every validDao() here uses
				// `addr` as both creator and the implicit lone member.
				StaticMembers: []types.StaticMemberEntry{
					{DaoId: 1, Address: addr, Weight: 1},
				},
			},
			valid: true,
		},
		{
			desc: "two daos with valid parent linkage",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 3,
				Daos: []types.Dao{
					validDao(1, addr),
					func() types.Dao { d := validDao(2, addr); d.ParentDaoId = 1; return d }(),
				},
				StaticMembers: []types.StaticMemberEntry{
					{DaoId: 1, Address: addr, Weight: 1},
					{DaoId: 2, Address: addr, Weight: 1},
				},
			},
			valid: true,
		},
		{
			desc:     "empty GenesisState fails (no fee destination set)",
			genState: &types.GenesisState{},
			valid:    false,
		},
		{
			desc: "duplicate dao id",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr), validDao(1, addr)},
			},
			valid: false,
		},
		{
			desc: "dao id zero",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 1,
				Daos:         []types.Dao{validDao(0, addr)},
			},
			valid: false,
		},
		{
			desc: "account_address mismatch",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos: []types.Dao{
					func() types.Dao {
						d := validDao(1, addr)
						d.AccountAddress = sample.AccAddress() // arbitrary, NOT derived from id
						return d
					}(),
				},
			},
			valid: false,
		},
		{
			desc: "parent_dao_id missing from genesis",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos: []types.Dao{
					func() types.Dao { d := validDao(1, addr); d.ParentDaoId = 99; return d }(),
				},
			},
			valid: false,
		},
		{
			desc: "dao_id_counter not greater than max id",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 1,
				Daos:         []types.Dao{validDao(1, addr)},
			},
			valid: false,
		},
		{
			desc: "empty daos with dao_id_counter = 0 rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 0,
			},
			valid: false,
		},
		{
			desc: "indirect parent cycle rejected (1 -> 2 -> 1)",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 3,
				Daos: []types.Dao{
					func() types.Dao { d := validDao(1, addr); d.ParentDaoId = 2; return d }(),
					func() types.Dao { d := validDao(2, addr); d.ParentDaoId = 1; return d }(),
				},
			},
			valid: false,
		},
		{
			desc: "deeper parent cycle rejected (1 -> 2 -> 3 -> 1)",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 4,
				Daos: []types.Dao{
					func() types.Dao { d := validDao(1, addr); d.ParentDaoId = 2; return d }(),
					func() types.Dao { d := validDao(2, addr); d.ParentDaoId = 3; return d }(),
					func() types.Dao { d := validDao(3, addr); d.ParentDaoId = 1; return d }(),
				},
			},
			valid: false,
		},
		{
			desc: "voting_backend unspecified rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos: []types.Dao{
					func() types.Dao {
						d := validDao(1, addr)
						d.VotingBackend = types.VotingBackendType_VOTING_BACKEND_UNSPECIFIED
						return d
					}(),
				},
			},
			valid: false,
		},
		{
			// Epic 2 rejects REWARD_STAKED in genesis entirely (asymmetric with
			// MsgCreateDao, which also rejects it). Both the empty-reward_id
			// and populated-reward_id cases are invalid until Epic 3 + Epic 5
			// land. See the comment in types/genesis.go.
			desc: "REWARD_STAKED rejected in genesis (empty reward_id)",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos: []types.Dao{
					func() types.Dao {
						d := validDao(1, addr)
						d.VotingBackend = types.VotingBackendType_VOTING_BACKEND_REWARD_STAKED
						d.RewardId = ""
						return d
					}(),
				},
			},
			valid: false,
		},
		{
			desc: "STATIC with reward_id set rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos: []types.Dao{
					func() types.Dao {
						d := validDao(1, addr)
						d.RewardId = "stray-reward-id"
						return d
					}(),
				},
			},
			valid: false,
		},
		{
			// Flipped from valid → invalid in Epic 2: REWARD_STAKED is not an
			// accepted genesis variant yet, even with a populated reward_id.
			// Once Epic 3 (snapshot iterator) and Epic 5 (backend swap) land,
			// genesis should accept REWARD_STAKED with non-empty reward_id and
			// this case flips back to valid.
			desc: "REWARD_STAKED with reward_id still rejected in Epic 2",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos: []types.Dao{
					func() types.Dao {
						d := validDao(1, addr)
						d.VotingBackend = types.VotingBackendType_VOTING_BACKEND_REWARD_STAKED
						d.RewardId = "00000000-0000-0000-0000-000000000001"
						return d
					}(),
				},
			},
			valid: false,
		},
		{
			// Epic 3 follow-up: genesis enforces Params.MaxVotingPeriod the
			// same way MsgCreateDao / MsgUpdateGovernanceConfig do.
			// DefaultMaxVotingPeriod is 30 days; 365d must be rejected.
			desc: "voting_period above Params.MaxVotingPeriod rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos: []types.Dao{
					func() types.Dao {
						d := validDao(1, addr)
						d.Governance.VotingPeriod = 365 * 24 * time.Hour
						return d
					}(),
				},
			},
			valid: false,
		},
		{
			// Epic 3 genesis: a proposal whose dao_id isn't in `daos` is
			// referential corruption.
			desc: "proposal references missing DAO",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              99, // not in `daos`
					ProposalId:         1,
					Proposer:           addr,
					Title:              "stray",
					SnapshotId:         1,
					Status:             types.ProposalStatus_PROPOSAL_STATUS_VOTING,
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
				}},
			},
			valid: false,
		},
		{
			// Epic 3 genesis: a proposal whose snapshot_id has no matching
			// snapshot_total row.
			desc: "proposal missing snapshot_total",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "no-snap",
					SnapshotId:         1,
					Status:             types.ProposalStatus_PROPOSAL_STATUS_VOTING,
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				// no SnapshotTotals — orphan
			},
			valid: false,
		},
		{
			// Epic 3 genesis: a vote whose (dao_id, proposal_id) doesn't
			// resolve to a known proposal.
			desc: "vote references missing proposal",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Votes: []types.Vote{{
					DaoId:      1,
					ProposalId: 99, // no such proposal
					Voter:      addr,
					Option:     types.VoteOption_VOTE_OPTION_YES,
					Power:      1,
				}},
			},
			valid: false,
		},
		{
			// Epic 3 genesis: a proposal_id_counter is set but the DAO has
			// proposals with higher ids.
			desc: "proposal_id_counter not greater than max proposal_id",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         5,
					Proposer:           addr,
					Title:              "high-id",
					SnapshotId:         1,
					Status:             types.ProposalStatus_PROPOSAL_STATUS_PASSED,
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 3}}, // ≤ max id 5
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 1}},
			},
			valid: false,
		},
		{
			// Epic 3 genesis: a DAO has proposals but no proposal_id_counter
			// entry — would allow id reuse after import.
			desc: "proposals present without proposal_id_counter",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "no-counter",
					SnapshotId:         1,
					Status:             types.ProposalStatus_PROPOSAL_STATUS_PASSED,
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
				}},
				// no ProposalIdCounters
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 1}},
			},
			valid: false,
		},
		{
			// Genesis review #1 (P1): VOTING proposal with zero VotingEnd
			// would silently never expire (uint64 cast of negative UnixNano
			// wraps to ~6.4e18). Must be rejected.
			desc: "VOTING proposal with zero voting_end rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "no-end",
					SnapshotId:         1,
					// VotingEnd intentionally left as time.Time{}.
					Status:             types.ProposalStatus_PROPOSAL_STATUS_VOTING,
					Tally:              types.Tally{TotalPower: 1},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 1}},
			},
			valid: false,
		},
		{
			// Genesis review #2 (P2): a proposal whose frozen
			// governance_snapshot violates the brick-prevention caps must
			// be rejected — EndBlock feeds this directly into computeOutcome.
			desc: "proposal governance_snapshot with UNSPECIFIED rule rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "bad-gov",
					SnapshotId:         1,
					VotingEnd:          time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
					Status:             types.ProposalStatus_PROPOSAL_STATUS_VOTING,
					Tally:              types.Tally{TotalPower: 1},
					GovernanceSnapshot: types.GovernanceConfig{
						// ApprovalRule UNSPECIFIED → ValidateGovernanceConfigStateless rejects
						ThresholdBps: 5_000,
						VotingPeriod: time.Hour,
					},
					DepositSnapshot: validDeposit(),
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 1}},
			},
			valid: false,
		},
		{
			// Genesis review #3 (P2): tally with voted-power > total_power.
			desc: "proposal tally voted-power exceeds total_power rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "over-total",
					SnapshotId:         1,
					Status:             types.ProposalStatus_PROPOSAL_STATUS_PASSED,
					Tally:              types.Tally{YesPower: 10, TotalPower: 5},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 5}},
			},
			valid: false,
		},
		{
			// Genesis review #3: Tally.TotalPower diverges from snapshot_total.
			desc: "proposal tally total_power mismatches snapshot_total rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "total-mismatch",
					SnapshotId:         1,
					Status:             types.ProposalStatus_PROPOSAL_STATUS_PASSED,
					Tally:              types.Tally{TotalPower: 99},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 5}},
			},
			valid: false,
		},
		{
			// Genesis review #3: Vote.Power does not match the snapshot row.
			desc: "vote power diverges from snapshot row rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "vote-mismatch",
					SnapshotId:         1,
					Status:             types.ProposalStatus_PROPOSAL_STATUS_PASSED,
					Tally:              types.Tally{YesPower: 5, TotalPower: 5},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
				}},
				Votes: []types.Vote{{
					DaoId:      1,
					ProposalId: 1,
					Voter:      addr,
					Option:     types.VoteOption_VOTE_OPTION_YES,
					Power:      5, // claim 5 but snapshot row says 3
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotPowers:     []types.SnapshotPowerEntry{{DaoId: 1, SnapshotId: 1, Address: addr, Power: 3}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 5}},
			},
			valid: false,
		},
		{
			// Genesis review #3: votes' sum does not match the proposal's
			// stored Tally — runtime invariant violated.
			desc: "tally does not reconcile with sum of votes",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "tally-vs-votes",
					SnapshotId:         1,
					Status:             types.ProposalStatus_PROPOSAL_STATUS_PASSED,
					// Tally claims yes=5 but only one vote of power 3 exists.
					Tally:              types.Tally{YesPower: 5, TotalPower: 5},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
				}},
				Votes: []types.Vote{{
					DaoId:      1,
					ProposalId: 1,
					Voter:      addr,
					Option:     types.VoteOption_VOTE_OPTION_YES,
					Power:      3,
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotPowers:     []types.SnapshotPowerEntry{{DaoId: 1, SnapshotId: 1, Address: addr, Power: 3}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 5}},
			},
			valid: false,
		},
		{
			// Genesis review #3: a voter not in the snapshot must have
			// power=0 on their Vote row.
			desc: "vote from non-snapshot voter with non-zero power rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "ghost-voter",
					SnapshotId:         1,
					Status:             types.ProposalStatus_PROPOSAL_STATUS_PASSED,
					Tally:              types.Tally{YesPower: 7, TotalPower: 5},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
				}},
				Votes: []types.Vote{{
					DaoId:      1,
					ProposalId: 1,
					Voter:      sample.AccAddress(), // not in snapshot
					Option:     types.VoteOption_VOTE_OPTION_YES,
					Power:      7,
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 5}},
			},
			valid: false,
		},
		{
			// Reviewer Finding 4: DEPOSIT_PERIOD proposal with zero
			// deposit_deadline would silently never expire (uint64 cast
			// of a zero time wraps to a huge queue key).
			desc: "DEPOSIT_PERIOD proposal with zero deposit_deadline rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "no-deposit-deadline",
					SnapshotId:         1,
					// DepositDeadline intentionally zero-value.
					Status:             types.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD,
					Tally:              types.Tally{TotalPower: 1},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
					DepositCollected:   sdk.NewInt64Coin("ubze", 0),
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 1}},
			},
			valid: false,
		},
		{
			// Reviewer Finding 6: VOTING with collected < min_deposit
			// bypasses the deposit gate. Reject.
			desc: "VOTING proposal with collected below min_deposit rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "underfunded-voting",
					SnapshotId:         1,
					VotingEnd:          time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
					Status:             types.ProposalStatus_PROPOSAL_STATUS_VOTING,
					Tally:              types.Tally{TotalPower: 1},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(), // min = 1ubze
					DepositCollected:   sdk.NewInt64Coin("ubze", 0),
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 1}},
			},
			valid: false,
		},
		{
			// Reviewer Finding 6: DEPOSIT_PERIOD with collected >= min
			// should have auto-transitioned to VOTING.
			desc: "DEPOSIT_PERIOD proposal with collected >= min_deposit rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "stuck-in-deposit",
					SnapshotId:         1,
					DepositDeadline:    time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
					Status:             types.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD,
					Tally:              types.Tally{TotalPower: 1},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
					DepositCollected:   sdk.NewInt64Coin("ubze", 1), // == min; should be VOTING
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 1}},
				DepositRecords: []types.DepositRecord{
					{DaoId: 1, ProposalId: 1, Depositor: addr, Amount: sdk.NewInt64Coin("ubze", 1)},
				},
			},
			valid: false,
		},
		{
			// Reviewer Question 3: open proposal whose deposit_collected
			// denom doesn't match min_deposit denom would panic Coin.Add
			// on a top-up.
			desc: "open proposal with deposit_collected denom mismatch rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "wrong-denom",
					SnapshotId:         1,
					DepositDeadline:    time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
					Status:             types.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD,
					Tally:              types.Tally{TotalPower: 1},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(), // denom = "ubze"
					// Wrong denom — would explode on MsgDeposit's Coin.Add.
					DepositCollected: sdk.NewInt64Coin("ibc/atom", 0),
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 1}},
			},
			valid: false,
		},
		{
			// Reviewer Round-2 Finding 1: a PASSED proposal whose tally
			// does not satisfy ComputeOutcome's pass condition is
			// rejected. Without this, a crafted genesis can mark a
			// proposal PASSED with zero votes; MsgExecuteProposal would
			// then trust the status and dispatch the bundle.
			desc: "PASSED proposal with non-passing tally rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				StaticMembers: []types.StaticMemberEntry{
					{DaoId: 1, Address: addr, Weight: 1},
				},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "free-pass",
					SnapshotId:         1,
					Status:             types.ProposalStatus_PROPOSAL_STATUS_PASSED,
					Tally:              types.Tally{TotalPower: 1}, // zero votes
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
					DepositCollected:   sdk.NewInt64Coin("ubze", 1),
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 1}},
			},
			valid: false,
		},
		{
			// Reviewer Round-2 Finding 2: a vote attached to a
			// DEPOSIT_PERIOD proposal is rejected. Without this, an
			// imported tally + votes would go live the moment MsgDeposit
			// promoted the proposal to VOTING.
			desc: "vote on DEPOSIT_PERIOD proposal rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				StaticMembers: []types.StaticMemberEntry{
					{DaoId: 1, Address: addr, Weight: 1},
				},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "preload",
					SnapshotId:         1,
					DepositDeadline:    time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
					Status:             types.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD,
					Tally:              types.Tally{TotalPower: 1},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
					DepositCollected:   sdk.NewInt64Coin("ubze", 0),
				}},
				Votes: []types.Vote{{
					DaoId: 1, ProposalId: 1, Voter: addr,
					Option: types.VoteOption_VOTE_OPTION_YES, Power: 1,
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotPowers:     []types.SnapshotPowerEntry{{DaoId: 1, SnapshotId: 1, Address: addr, Power: 1}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 1}},
			},
			valid: false,
		},
		{
			// Reviewer Finding 2 (Round 1): a STATIC DAO with no
			// static_member entries is rejected (runtime invariant).
			desc: "STATIC DAO with no static_members rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				// StaticMembers omitted.
			},
			valid: false,
		},
		{
			// Reviewer Finding 2: a static_member whose dao_id is not in
			// `daos` is dangling.
			desc: "static_member references missing DAO",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				StaticMembers: []types.StaticMemberEntry{
					{DaoId: 1, Address: addr, Weight: 1},
					{DaoId: 99, Address: addr, Weight: 1}, // dangling
				},
			},
			valid: false,
		},
		{
			// Reviewer Finding 2: weight=0 violates the runtime invariant.
			desc: "static_member with zero weight rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				StaticMembers: []types.StaticMemberEntry{
					{DaoId: 1, Address: addr, Weight: 0},
				},
			},
			valid: false,
		},
		{
			// Reviewer Finding 2: (dao, addr) must be unique.
			desc: "duplicate static_member (dao, addr) pair rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				StaticMembers: []types.StaticMemberEntry{
					{DaoId: 1, Address: addr, Weight: 1},
					{DaoId: 1, Address: addr, Weight: 5}, // dup
				},
			},
			valid: false,
		},
		{
			// Reviewer Round-2 Finding 5: per-DAO weight sum that overflows
			// uint64 is rejected at Validate so InitGenesis never panics
			// inside applyStaticMembersInit's overflow check.
			desc: "static_members per-DAO weight sum overflow rejected",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				StaticMembers: []types.StaticMemberEntry{
					// Two near-MaxUint64 weights sum > MaxUint64. Use
					// distinct addresses (uniqueness check passes).
					{DaoId: 1, Address: addr, Weight: 18_000_000_000_000_000_000},
					{DaoId: 1, Address: sample.AccAddress(), Weight: 18_000_000_000_000_000_000},
				},
			},
			valid: false,
		},
		{
			// Epic 3 genesis: full happy path.
			desc: "valid genesis with proposal + vote + snapshots + counters",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				DaoIdCounter: 2,
				Daos:         []types.Dao{validDao(1, addr)},
				Proposals: []types.Proposal{{
					DaoId:              1,
					ProposalId:         1,
					Proposer:           addr,
					Title:              "happy",
					SnapshotId:         1,
					Status:             types.ProposalStatus_PROPOSAL_STATUS_PASSED,
					Tally:              types.Tally{YesPower: 1, TotalPower: 1},
					GovernanceSnapshot: validGovernance(),
					DepositSnapshot:    validDeposit(),
				}},
				Votes: []types.Vote{{
					DaoId:      1,
					ProposalId: 1,
					Voter:      addr,
					Option:     types.VoteOption_VOTE_OPTION_YES,
					Power:      1,
				}},
				ProposalIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotIdCounters: []types.PerDaoUint64{{DaoId: 1, Value: 2}},
				SnapshotPowers:     []types.SnapshotPowerEntry{{DaoId: 1, SnapshotId: 1, Address: addr, Power: 1}},
				SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: 1, SnapshotId: 1, Total: 1}},
				StaticMembers: []types.StaticMemberEntry{
					{DaoId: 1, Address: addr, Weight: 1},
				},
			},
			valid: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
