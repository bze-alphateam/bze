package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/daodao module sentinel errors.
var (
	// ErrInvalidSigner is returned when a MsgUpdateParams signer is not the
	// chain governance authority.
	ErrInvalidSigner = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")

	// ErrInvalidAddress is returned when an address fails bech32 parsing.
	ErrInvalidAddress = sdkerrors.Register(ModuleName, 1101, "invalid address")

	// ErrInvalidMetadata wraps validation failures on DaoMetadata fields.
	ErrInvalidMetadata = sdkerrors.Register(ModuleName, 1102, "invalid DAO metadata")

	// ErrDaoNotFound is returned when a DAO id has no record.
	ErrDaoNotFound = sdkerrors.Register(ModuleName, 1103, "DAO not found")

	// ErrParentCycle is returned when a proposed parent_dao_id would create
	// a cycle in the DAO hierarchy.
	ErrParentCycle = sdkerrors.Register(ModuleName, 1104, "parent DAO chain forms a cycle")

	// ErrParentNotFound is returned when parent_dao_id references a missing DAO.
	ErrParentNotFound = sdkerrors.Register(ModuleName, 1105, "parent DAO not found")

	// ErrUnauthorized is returned when an admin-gated message is signed by
	// an address that is not the current admin.
	ErrUnauthorized = sdkerrors.Register(ModuleName, 1106, "unauthorized: signer is not the DAO admin")

	// ErrPendingAdminMismatch is returned when MsgAcceptDaoAdmin is signed
	// by an address other than the recorded pending_admin.
	ErrPendingAdminMismatch = sdkerrors.Register(ModuleName, 1107, "signer does not match pending admin")

	// ErrNoPendingAdmin is returned when MsgAcceptDaoAdmin is called but no
	// nomination is in flight.
	ErrNoPendingAdmin = sdkerrors.Register(ModuleName, 1108, "no pending admin to accept")

	// (Reserved: 1109 was ErrAddressAlreadyExists, removed when CreateDao
	// learned to gracefully reuse pre-funded BaseAccounts at the deterministic
	// DAO address. Do not reuse this code for unrelated errors.)

	// ErrInsufficientCreationFee is returned when the creator cannot pay the
	// configured creation fee.
	ErrInsufficientCreationFee = sdkerrors.Register(ModuleName, 1110, "insufficient balance to pay DAO creation fee")

	// ErrInvalidParams wraps Params validation failures.
	ErrInvalidParams = sdkerrors.Register(ModuleName, 1111, "invalid daodao params")

	// -------- Epic 2: voting backend errors --------

	// ErrMissingVotingConfig is returned when MsgCreateDao has no
	// voting_config oneof variant set.
	ErrMissingVotingConfig = sdkerrors.Register(ModuleName, 1200, "voting_config is required")

	// ErrVotingConfigNotAllowed is returned when a voting_config variant is
	// rejected at creation time (Epic 2 rejects REWARD_STAKED — it becomes
	// reachable in Epic 5 via MsgUpdateVotingBackend).
	ErrVotingConfigNotAllowed = sdkerrors.Register(ModuleName, 1201, "this voting backend cannot be set at DAO creation")

	// ErrInvalidStaticMembers wraps STATIC member list validation failures
	// (empty list, zero weight, duplicate address, bad bech32).
	ErrInvalidStaticMembers = sdkerrors.Register(ModuleName, 1202, "invalid static member list")

	// ErrNotStaticBackend is returned when a STATIC-only operation
	// (MsgUpdateMembers, Members query) targets a DAO whose voting_backend
	// is not STATIC.
	ErrNotStaticBackend = sdkerrors.Register(ModuleName, 1203, "DAO is not a STATIC-backed DAO")

	// ErrMemberNotFound is returned when MsgUpdateMembers asks to remove an
	// address that isn't a member.
	ErrMemberNotFound = sdkerrors.Register(ModuleName, 1204, "address is not a DAO member")

	// ErrEmptyMembership is returned when an update would leave a STATIC
	// DAO with zero members. STATIC DAOs MUST have ≥ 1 member at all times.
	ErrEmptyMembership = sdkerrors.Register(ModuleName, 1205, "STATIC DAO must have at least one member")

	// ErrAmountOverflow is returned when a uint64 cannot represent a value
	// from rewards' string-typed amount fields.
	ErrAmountOverflow = sdkerrors.Register(ModuleName, 1206, "amount does not fit in uint64")

	// -------- Epic 3: proposals & voting --------

	// ErrInvalidGovernanceConfig wraps GovernanceConfig brick-prevention
	// failures (out-of-range threshold/quorum/voting_period, missing
	// approval_rule, etc.).
	ErrInvalidGovernanceConfig = sdkerrors.Register(ModuleName, 1300, "invalid governance config")

	// ErrInvalidProposalContent wraps MsgCreateProposal title/description
	// validation failures and msgs-cardinality / decode failures.
	ErrInvalidProposalContent = sdkerrors.Register(ModuleName, 1301, "invalid proposal content")

	// ErrProposalNotFound is returned when (dao_id, proposal_id) does not
	// resolve to a stored Proposal.
	ErrProposalNotFound = sdkerrors.Register(ModuleName, 1302, "proposal not found")

	// ErrProposalNotVoting is returned when a vote / early-close path runs
	// on a proposal that is not in PROPOSAL_STATUS_VOTING.
	ErrProposalNotVoting = sdkerrors.Register(ModuleName, 1303, "proposal is not in voting status")

	// ErrInvalidVoteOption is returned when MsgVote.option is UNSPECIFIED or
	// otherwise not in the allowed VoteOption set.
	ErrInvalidVoteOption = sdkerrors.Register(ModuleName, 1304, "invalid vote option")

	// ErrNoVotingPower is returned when the proposer (at creation) or voter
	// (at MsgVote) has zero voting power in the relevant view (current vs.
	// snapshotted).
	ErrNoVotingPower = sdkerrors.Register(ModuleName, 1305, "address has no voting power in this DAO")

	// ErrRevoteNotAllowed is returned when an already-voted address tries
	// to vote again on a proposal whose governance_snapshot.allow_revote
	// is false.
	ErrRevoteNotAllowed = sdkerrors.Register(ModuleName, 1306, "revoting is disabled for this proposal")

	// ErrFlashVoteLockTooShort is returned when a REWARD_STAKED DAO's
	// governance.voting_period exceeds the program's `lock`, allowing a
	// voter to flash-stake-vote-unstake within a single proposal window.
	ErrFlashVoteLockTooShort = sdkerrors.Register(ModuleName, 1307, "reward program lock is shorter than DAO voting period")

	// -------- Epic 4: deposit period --------

	// ErrInvalidDepositConfig wraps brick-prevention failures on a
	// DepositConfig (zero min_deposit, bad denom, out-of-range
	// deposit_period, unknown destination / refund-policy variant).
	ErrInvalidDepositConfig = sdkerrors.Register(ModuleName, 1400, "invalid deposit config")

	// ErrInvalidDepositAmount is returned when MsgDeposit / initial_deposit
	// has the wrong denom, zero amount where non-zero required, or for a
	// non-member proposer who attached less than min_deposit.
	ErrInvalidDepositAmount = sdkerrors.Register(ModuleName, 1401, "invalid deposit amount")

	// ErrProposalNotInDepositPeriod is returned by MsgDeposit on a proposal
	// whose status is not DEPOSIT_PERIOD.
	ErrProposalNotInDepositPeriod = sdkerrors.Register(ModuleName, 1402, "proposal is not in deposit-period status")

	// -------- Epic 5: execution dispatcher & admin renouncement --------

	// ErrInvalidProposalSigners wraps signer validation failures on a
	// proposal's msgs[] bundle: zero signers, multi-signer messages, or a
	// signer that's not the DAO's account address.
	ErrInvalidProposalSigners = sdkerrors.Register(ModuleName, 1500, "invalid proposal message signers")

	// ErrProposalNotPassed is returned when MsgExecuteProposal targets a
	// proposal whose status is not PASSED.
	ErrProposalNotPassed = sdkerrors.Register(ModuleName, 1501, "proposal is not in passed status")

	// ErrUnknownMsgType is returned when a dispatched msg has no registered
	// MsgServer handler (typically a typo in proto registration).
	ErrUnknownMsgType = sdkerrors.Register(ModuleName, 1502, "unknown message type in proposal")

	// ErrAlreadySelfAdmin is returned when MsgRenounceAdmin is submitted on
	// a DAO whose admin is already its own account.
	ErrAlreadySelfAdmin = sdkerrors.Register(ModuleName, 1503, "DAO is already self-administered")

	// ErrBackendTypeMismatch is returned when MsgUpdateVotingBackend tries
	// to change the backend TYPE (STATIC ↔ REWARD_STAKED). v1 supports
	// same-type reconfiguration only.
	ErrBackendTypeMismatch = sdkerrors.Register(ModuleName, 1504, "cross-type voting-backend migration is not supported in v1")

	// (Reserved: 1505 was ErrRewardOwnershipMismatch, removed when
	// MsgUpdateVotingBackend's reward-creator check was deferred. The
	// `x/rewards.StakingReward` proto doesn't carry a Creator field, so
	// the check would require a cross-module schema change. Do not reuse
	// this code for unrelated errors — restore the original meaning when
	// the rewards-side change lands.)

	// -------- Epic 6: polls --------

	// ErrInvalidPollContent wraps stateless poll-creation failures: bad
	// title/description, malformed choice list, max_selections out of
	// range, quorum_bps over cap.
	ErrInvalidPollContent = sdkerrors.Register(ModuleName, 1600, "invalid poll content")

	// ErrPollNotFound is returned when (dao_id, poll_id) does not resolve
	// to a stored Poll.
	ErrPollNotFound = sdkerrors.Register(ModuleName, 1601, "poll not found")

	// ErrPollNotVoting is returned when MsgVoteOnPoll targets a poll
	// whose status is not POLL_STATUS_VOTING.
	ErrPollNotVoting = sdkerrors.Register(ModuleName, 1602, "poll is not in voting status")

	// ErrPollNotInDepositPeriod is returned by MsgDepositOnPoll on a poll
	// whose status is not POLL_STATUS_DEPOSIT_PERIOD.
	ErrPollNotInDepositPeriod = sdkerrors.Register(ModuleName, 1603, "poll is not in deposit-period status")

	// ErrInvalidPollSelection wraps MsgVoteOnPoll selection-set failures:
	// empty list, too many entries, out-of-range index, duplicate index,
	// NOTA-mixed-with-others (exclusivity).
	ErrInvalidPollSelection = sdkerrors.Register(ModuleName, 1604, "invalid poll selection")

	// ErrBundleMsgTypeNotAllowed is returned when a proposal's msgs[]
	// bundle includes a message type that cannot safely run inside the
	// execution dispatcher (e.g. MsgExecuteProposal, which would re-enter
	// dispatch and recurse).
	ErrBundleMsgTypeNotAllowed = sdkerrors.Register(ModuleName, 1605, "message type not allowed in proposal bundle")
)
