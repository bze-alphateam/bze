package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name.
	ModuleName = "daodao"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey is the message route for daodao.
	RouterKey = ModuleName

	// QuerierRoute defines the module's querier route.
	QuerierRoute = ModuleName
)

// Key prefixes. Each entry documents its key shape and the stored value type.
var (
	// ParamsKey stores the module's Params.
	//   value: Params
	ParamsKey = []byte{0x00}

	// DaoIDCounterKey holds the next DAO id to allocate.
	//   value: uvarint(uint64) — next id (assigned then incremented)
	DaoIDCounterKey = []byte{0x01}

	// DaoKeyPrefix indexes Dao records by id.
	//   key:   0x02 | uvarint(id)
	//   value: Dao
	DaoKeyPrefix = []byte{0x02}

	// DaoByAddressKeyPrefix indexes DAO ids by their derived account address.
	//   key:   0x03 | addr.Bytes()
	//   value: uvarint(id)
	DaoByAddressKeyPrefix = []byte{0x03}

	// SubDaoKeyPrefix indexes child ids per parent (set semantics).
	//   key:   0x04 | uvarint(parent_id) | uvarint(child_id)
	//   value: []byte{}
	SubDaoKeyPrefix = []byte{0x04}

	// DaoByCreatorKeyPrefix indexes DAO ids per creator address (set semantics).
	//   key:   0x05 | addr.Bytes() | uvarint(id)
	//   value: []byte{}
	DaoByCreatorKeyPrefix = []byte{0x05}

	// -------- Epic 2: voting backend storage --------

	// MemberKeyPrefix holds (address → weight) entries for STATIC DAOs.
	//   key:   0x10 | uvarint(dao_id) | addr.Bytes()
	//   value: uvarint(weight)
	MemberKeyPrefix = []byte{0x10}

	// StaticTotalPowerKeyPrefix caches the sum of weights for a STATIC DAO,
	// kept in sync on every member upsert/delete. Reads avoid a full member
	// scan; an invariant verifies the value matches the running sum.
	//   key:   0x11 | uvarint(dao_id)
	//   value: uvarint(total)
	StaticTotalPowerKeyPrefix = []byte{0x11}

	// SnapshotPowerKeyPrefix holds per-(address) voting power captured at a
	// proposal's snapshot id. Written by Epic 3's MsgCreateProposal via
	// VotingBackend.SnapshotAll; read at vote/tally time.
	//   key:   0x12 | uvarint(dao_id) | uvarint(snapshot_id) | addr.Bytes()
	//   value: uvarint(power)
	SnapshotPowerKeyPrefix = []byte{0x12}

	// SnapshotTotalKeyPrefix caches the total voting power captured at a
	// snapshot. Pair with SnapshotPowerKeyPrefix.
	//   key:   0x13 | uvarint(dao_id) | uvarint(snapshot_id)
	//   value: uvarint(total)
	SnapshotTotalKeyPrefix = []byte{0x13}

	// SnapshotIDCounterKeyPrefix holds the next snapshot id to allocate for
	// a given DAO. Used by Epic 3's MsgCreateProposal.
	//   key:   0x14 | uvarint(dao_id)
	//   value: uvarint(next_snapshot_id)
	SnapshotIDCounterKeyPrefix = []byte{0x14}

	// -------- Epic 3: proposals & voting --------

	// ProposalIDCounterKeyPrefix holds the next proposal id to allocate for
	// a given DAO. Counter is per-DAO and strictly monotonic.
	//   key:   0x50 | uvarint(dao_id)
	//   value: uvarint(next_proposal_id)
	ProposalIDCounterKeyPrefix = []byte{0x50}

	// ProposalKeyPrefix indexes proposals by (dao_id, proposal_id).
	//   key:   0x51 | uvarint(dao_id) | uvarint(proposal_id)
	//   value: Proposal
	ProposalKeyPrefix = []byte{0x51}

	// ProposalByStatusKeyPrefix is a (dao_id, status, proposal_id) index used
	// for status-filtered proposal listings. Updated atomically with the
	// Proposal record on every status transition (delete old key, write new).
	//   key:   0x52 | uvarint(dao_id) | uvarint(status) | uvarint(proposal_id)
	//   value: []byte{}
	ProposalByStatusKeyPrefix = []byte{0x52}

	// VoteKeyPrefix indexes votes by (dao_id, proposal_id, voter). Replaced
	// in-place on revote (when governance_snapshot.allow_revote is true).
	//   key:   0x60 | uvarint(dao_id) | uvarint(proposal_id) | voter.Bytes()
	//   value: Vote
	VoteKeyPrefix = []byte{0x60}

	// ExpiringProposalKeyPrefix is the end-blocker finalization queue,
	// keyed by the proposal's currently-relevant deadline (in nanos,
	// big-endian) so range scans `[0, blockTimeNs]` produce all proposals
	// due to be finalized.
	//
	//   key:   0x70 | uvarint(unix_nanos) | uvarint(dao_id) | uvarint(proposal_id)
	//   value: []byte{}
	//
	// (dao_id, proposal_id) suffix keeps the key unique when two proposals
	// share the same deadline (block-rounded times collide easily).
	//
	// Epic 4 introduces a dual-phase lifecycle:
	//   DEPOSIT_PERIOD → unix_nanos = Proposal.deposit_deadline
	//   VOTING         → unix_nanos = Proposal.voting_end
	// On transition (deposit met → voting), the keeper removes the old
	// key and writes a new one.
	ExpiringProposalKeyPrefix = []byte{0x70}

	// -------- Epic 4: deposit period --------

	// DepositRecordKeyPrefix indexes per-(proposal, depositor) deposit
	// rows. Multiple deposits from the same address aggregate into one row.
	//   key:   0x80 | uvarint(dao_id) | uvarint(proposal_id) | depositor.Bytes()
	//   value: DepositRecord
	DepositRecordKeyPrefix = []byte{0x80}

	// -------- Epic 6: polls --------

	// PollIDCounterKeyPrefix holds the next poll id to allocate for a DAO.
	// Per-DAO and strictly monotonic; independent of the proposal counter.
	//   key:   0x90 | uvarint(dao_id)
	//   value: uvarint(next_poll_id)
	PollIDCounterKeyPrefix = []byte{0x90}

	// PollKeyPrefix indexes polls by (dao_id, poll_id).
	//   key:   0x91 | uvarint(dao_id) | uvarint(poll_id)
	//   value: Poll
	PollKeyPrefix = []byte{0x91}

	// PollByStatusKeyPrefix is a (dao_id, status, poll_id) index used
	// for status-filtered poll listings. Maintained atomically with the
	// Poll record on every status transition.
	//   key:   0x92 | uvarint(dao_id) | uvarint(status) | uvarint(poll_id)
	//   value: []byte{}
	PollByStatusKeyPrefix = []byte{0x92}

	// PollVoteKeyPrefix indexes per-(poll, voter) selections. Replaced
	// in-place on revote (when allow_revote_snapshot is true).
	//   key:   0xA0 | uvarint(dao_id) | uvarint(poll_id) | voter.Bytes()
	//   value: PollVote
	PollVoteKeyPrefix = []byte{0xA0}

	// PollDepositRecordKeyPrefix indexes per-(poll, depositor) deposit
	// rows. Same DepositRecord proto shape as proposals; separate key
	// prefix so refund/forfeit iterators iterate the right family.
	//   key:   0xB0 | uvarint(dao_id) | uvarint(poll_id) | depositor.Bytes()
	//   value: DepositRecord
	PollDepositRecordKeyPrefix = []byte{0xB0}

	// ExpiringPollKeyPrefix is the end-blocker finalization queue for
	// polls, parallel to ExpiringProposalKey. Status-aware: keyed by
	// deposit_deadline for DEPOSIT_PERIOD polls; voting_end for VOTING
	// polls.
	//   key:   0xC0 | uvarint(unix_nanos) | uvarint(dao_id) | uvarint(poll_id)
	//   value: []byte{}
	ExpiringPollKeyPrefix = []byte{0xC0}
)

// DaoKey returns the storage key for a DAO with the given id.
func DaoKey(id uint64) []byte {
	return append(append([]byte{}, DaoKeyPrefix...), sdk.Uint64ToBigEndian(id)...)
}

// DaoByAddressKey returns the storage key for the address → id index.
func DaoByAddressKey(addr sdk.AccAddress) []byte {
	return append(append([]byte{}, DaoByAddressKeyPrefix...), addr.Bytes()...)
}

// SubDaoIterationPrefix returns the prefix that iterates all sub-DAO entries
// of a given parent. Used as the prefix.NewStore base for ranging.
func SubDaoIterationPrefix(parentID uint64) []byte {
	return append(append([]byte{}, SubDaoKeyPrefix...), sdk.Uint64ToBigEndian(parentID)...)
}

// SubDaoKey returns the storage key for the (parent, child) set membership.
func SubDaoKey(parentID, childID uint64) []byte {
	return append(SubDaoIterationPrefix(parentID), sdk.Uint64ToBigEndian(childID)...)
}

// DaoByCreatorIterationPrefix returns the prefix that iterates all DAO ids
// for a given creator address.
func DaoByCreatorIterationPrefix(creator sdk.AccAddress) []byte {
	return append(append([]byte{}, DaoByCreatorKeyPrefix...), creator.Bytes()...)
}

// DaoByCreatorKey returns the storage key for the (creator, id) set entry.
func DaoByCreatorKey(creator sdk.AccAddress, id uint64) []byte {
	return append(DaoByCreatorIterationPrefix(creator), sdk.Uint64ToBigEndian(id)...)
}

// -------- Epic 2 key helpers --------

// MembersIterationPrefix returns the prefix to iterate every member entry
// of a given STATIC DAO.
func MembersIterationPrefix(daoID uint64) []byte {
	return append(append([]byte{}, MemberKeyPrefix...), sdk.Uint64ToBigEndian(daoID)...)
}

// MemberKey returns the (dao_id, address) → weight storage key.
func MemberKey(daoID uint64, addr sdk.AccAddress) []byte {
	return append(MembersIterationPrefix(daoID), addr.Bytes()...)
}

// StaticTotalPowerKey returns the cached-total storage key for a STATIC DAO.
func StaticTotalPowerKey(daoID uint64) []byte {
	return append(append([]byte{}, StaticTotalPowerKeyPrefix...), sdk.Uint64ToBigEndian(daoID)...)
}

// SnapshotPowerIterationPrefix returns the prefix to iterate every captured
// (address → power) entry for a given snapshot.
func SnapshotPowerIterationPrefix(daoID, snapshotID uint64) []byte {
	out := append([]byte{}, SnapshotPowerKeyPrefix...)
	out = append(out, sdk.Uint64ToBigEndian(daoID)...)
	out = append(out, sdk.Uint64ToBigEndian(snapshotID)...)
	return out
}

// SnapshotPowerKey returns the (dao, snap, address) → power storage key.
func SnapshotPowerKey(daoID, snapshotID uint64, addr sdk.AccAddress) []byte {
	return append(SnapshotPowerIterationPrefix(daoID, snapshotID), addr.Bytes()...)
}

// SnapshotTotalKey returns the (dao, snap) → total storage key.
func SnapshotTotalKey(daoID, snapshotID uint64) []byte {
	out := append([]byte{}, SnapshotTotalKeyPrefix...)
	out = append(out, sdk.Uint64ToBigEndian(daoID)...)
	out = append(out, sdk.Uint64ToBigEndian(snapshotID)...)
	return out
}

// SnapshotIDCounterKey returns the next-snapshot-id storage key for a DAO.
func SnapshotIDCounterKey(daoID uint64) []byte {
	return append(append([]byte{}, SnapshotIDCounterKeyPrefix...), sdk.Uint64ToBigEndian(daoID)...)
}

// -------- Epic 3 key helpers --------

// ProposalIDCounterKey returns the next-proposal-id storage key for a DAO.
func ProposalIDCounterKey(daoID uint64) []byte {
	return append(append([]byte{}, ProposalIDCounterKeyPrefix...), sdk.Uint64ToBigEndian(daoID)...)
}

// ProposalsIterationPrefix returns the prefix to iterate every proposal of
// a given DAO. Used as the prefix.NewStore base for paginated listings.
func ProposalsIterationPrefix(daoID uint64) []byte {
	return append(append([]byte{}, ProposalKeyPrefix...), sdk.Uint64ToBigEndian(daoID)...)
}

// ProposalKey returns the (dao_id, proposal_id) → Proposal storage key.
func ProposalKey(daoID, proposalID uint64) []byte {
	return append(ProposalsIterationPrefix(daoID), sdk.Uint64ToBigEndian(proposalID)...)
}

// ProposalsByStatusIterationPrefix returns the prefix to iterate every
// proposal of `dao_id` currently in `status`.
func ProposalsByStatusIterationPrefix(daoID uint64, status ProposalStatus) []byte {
	out := append([]byte{}, ProposalByStatusKeyPrefix...)
	out = append(out, sdk.Uint64ToBigEndian(daoID)...)
	out = append(out, sdk.Uint64ToBigEndian(uint64(status))...)
	return out
}

// ProposalByStatusKey returns the (dao_id, status, proposal_id) index key.
func ProposalByStatusKey(daoID uint64, status ProposalStatus, proposalID uint64) []byte {
	return append(ProposalsByStatusIterationPrefix(daoID, status), sdk.Uint64ToBigEndian(proposalID)...)
}

// VotesIterationPrefix returns the prefix to iterate every vote on a given
// proposal. Used as the prefix.NewStore base for paginated listings.
func VotesIterationPrefix(daoID, proposalID uint64) []byte {
	out := append([]byte{}, VoteKeyPrefix...)
	out = append(out, sdk.Uint64ToBigEndian(daoID)...)
	out = append(out, sdk.Uint64ToBigEndian(proposalID)...)
	return out
}

// VoteKey returns the (dao_id, proposal_id, voter) → Vote storage key.
func VoteKey(daoID, proposalID uint64, voter sdk.AccAddress) []byte {
	return append(VotesIterationPrefix(daoID, proposalID), voter.Bytes()...)
}

// ExpiringProposalKey returns the storage key for the end-blocker queue
// entry. unixNs is the voting_end timestamp's UnixNano() value (must be
// non-negative — voting_end is always in the future).
func ExpiringProposalKey(unixNs uint64, daoID, proposalID uint64) []byte {
	out := append([]byte{}, ExpiringProposalKeyPrefix...)
	out = append(out, sdk.Uint64ToBigEndian(unixNs)...)
	out = append(out, sdk.Uint64ToBigEndian(daoID)...)
	out = append(out, sdk.Uint64ToBigEndian(proposalID)...)
	return out
}

// ExpiringProposalUntilPrefix returns an end-exclusive upper bound for
// scanning the expiring-proposal queue up to and including `unixNs`.
// Callers iterate `[ExpiringProposalKeyPrefix, ExpiringProposalUntilPrefix(unixNs)]`.
//
// Implementation: nanosecond+1 because the iterator range is exclusive on
// the upper bound. Using `unixNs + 1` matches Cosmos SDK gov's pattern.
func ExpiringProposalUntilPrefix(unixNs uint64) []byte {
	return append(append([]byte{}, ExpiringProposalKeyPrefix...), sdk.Uint64ToBigEndian(unixNs+1)...)
}

// -------- Epic 4 key helpers --------

// DepositRecordsIterationPrefix returns the prefix that iterates every
// DepositRecord row for a given proposal. Used as the prefix.NewStore
// base for the paginated Deposits query and for refund/forfeit fan-out.
func DepositRecordsIterationPrefix(daoID, proposalID uint64) []byte {
	out := append([]byte{}, DepositRecordKeyPrefix...)
	out = append(out, sdk.Uint64ToBigEndian(daoID)...)
	out = append(out, sdk.Uint64ToBigEndian(proposalID)...)
	return out
}

// DepositRecordKey returns the (dao_id, proposal_id, depositor) → DepositRecord
// storage key.
func DepositRecordKey(daoID, proposalID uint64, depositor sdk.AccAddress) []byte {
	return append(DepositRecordsIterationPrefix(daoID, proposalID), depositor.Bytes()...)
}

// -------- Epic 6 key helpers --------

// PollIDCounterKey returns the next-poll-id storage key for a DAO.
func PollIDCounterKey(daoID uint64) []byte {
	return append(append([]byte{}, PollIDCounterKeyPrefix...), sdk.Uint64ToBigEndian(daoID)...)
}

// PollsIterationPrefix returns the prefix that iterates every poll of a
// given DAO. Used as the prefix.NewStore base for paginated listings.
func PollsIterationPrefix(daoID uint64) []byte {
	return append(append([]byte{}, PollKeyPrefix...), sdk.Uint64ToBigEndian(daoID)...)
}

// PollKey returns the (dao_id, poll_id) → Poll storage key.
func PollKey(daoID, pollID uint64) []byte {
	return append(PollsIterationPrefix(daoID), sdk.Uint64ToBigEndian(pollID)...)
}

// PollsByStatusIterationPrefix returns the prefix to iterate every poll
// of `dao_id` currently in `status`.
func PollsByStatusIterationPrefix(daoID uint64, status PollStatus) []byte {
	out := append([]byte{}, PollByStatusKeyPrefix...)
	out = append(out, sdk.Uint64ToBigEndian(daoID)...)
	out = append(out, sdk.Uint64ToBigEndian(uint64(status))...)
	return out
}

// PollByStatusKey returns the (dao_id, status, poll_id) index key.
func PollByStatusKey(daoID uint64, status PollStatus, pollID uint64) []byte {
	return append(PollsByStatusIterationPrefix(daoID, status), sdk.Uint64ToBigEndian(pollID)...)
}

// PollVotesIterationPrefix returns the prefix to iterate every vote on
// a given poll.
func PollVotesIterationPrefix(daoID, pollID uint64) []byte {
	out := append([]byte{}, PollVoteKeyPrefix...)
	out = append(out, sdk.Uint64ToBigEndian(daoID)...)
	out = append(out, sdk.Uint64ToBigEndian(pollID)...)
	return out
}

// PollVoteKey returns the (dao_id, poll_id, voter) → PollVote storage key.
func PollVoteKey(daoID, pollID uint64, voter sdk.AccAddress) []byte {
	return append(PollVotesIterationPrefix(daoID, pollID), voter.Bytes()...)
}

// PollDepositRecordsIterationPrefix returns the prefix that iterates
// every DepositRecord for a single poll.
func PollDepositRecordsIterationPrefix(daoID, pollID uint64) []byte {
	out := append([]byte{}, PollDepositRecordKeyPrefix...)
	out = append(out, sdk.Uint64ToBigEndian(daoID)...)
	out = append(out, sdk.Uint64ToBigEndian(pollID)...)
	return out
}

// PollDepositRecordKey returns the (dao_id, poll_id, depositor) →
// DepositRecord storage key.
func PollDepositRecordKey(daoID, pollID uint64, depositor sdk.AccAddress) []byte {
	return append(PollDepositRecordsIterationPrefix(daoID, pollID), depositor.Bytes()...)
}

// ExpiringPollKey returns the end-blocker queue key for a poll. unixNs
// is the relevant deadline (deposit_deadline for DEPOSIT_PERIOD polls,
// voting_end for VOTING polls). Status-aware enqueue/dequeue lives in
// the keeper layer.
func ExpiringPollKey(unixNs, daoID, pollID uint64) []byte {
	out := append([]byte{}, ExpiringPollKeyPrefix...)
	out = append(out, sdk.Uint64ToBigEndian(unixNs)...)
	out = append(out, sdk.Uint64ToBigEndian(daoID)...)
	out = append(out, sdk.Uint64ToBigEndian(pollID)...)
	return out
}

// ExpiringPollUntilPrefix returns an end-exclusive upper bound for
// scanning the expiring-poll queue up to and including `unixNs`. Mirrors
// ExpiringProposalUntilPrefix.
func ExpiringPollUntilPrefix(unixNs uint64) []byte {
	return append(append([]byte{}, ExpiringPollKeyPrefix...), sdk.Uint64ToBigEndian(unixNs+1)...)
}
