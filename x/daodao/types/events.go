package types

// Untyped sdk.Event constants. These are emitted by the daodao keeper at
// every state-mutating message. They're useful for indexers and UI feeds.
//
// Epic 1 ships with untyped events; if downstream consumers want typed
// events (proto messages emitted via ctx.EventManager().EmitTypedEvent),
// that's an additive change in a later epic.
const (
	EventTypeCreateDao         = "create_dao"
	EventTypeUpdateDaoMetadata = "update_dao_metadata"
	EventTypeUpdateDaoAdmin    = "update_dao_admin"
	EventTypeAcceptDaoAdmin    = "accept_dao_admin"
	EventTypeUpdateMembers     = "update_members"

	// Epic 3 event types.
	EventTypeCreateProposal         = "create_proposal"
	EventTypeVote                   = "vote"
	EventTypeUpdateGovernanceConfig = "update_governance_config"
	EventTypeProposalFinalized      = "proposal_finalized"
	EventTypeProposalEarlyClosed    = "proposal_early_closed"

	// Common attribute keys
	AttributeKeyDaoID          = "dao_id"
	AttributeKeyAccountAddress = "account_address"
	AttributeKeyCreator        = "creator"
	AttributeKeyAdmin          = "admin"
	AttributeKeyNewAdmin       = "new_admin"
	AttributeKeyParentDaoID    = "parent_dao_id"
	AttributeKeyFeeAmount      = "fee_amount"
	AttributeKeyFeeDestination = "fee_destination"
	AttributeKeyVotingBackend  = "voting_backend"
	AttributeKeyAddedCount     = "added_count"
	AttributeKeyRemovedCount   = "removed_count"
	AttributeKeyTotalPower     = "total_power"

	// Epic 3 attribute keys.
	AttributeKeyProposalID    = "proposal_id"
	AttributeKeyProposer      = "proposer"
	AttributeKeyVoter         = "voter"
	AttributeKeyVoteOption    = "vote_option"
	AttributeKeyVotePower     = "vote_power"
	AttributeKeySnapshotID    = "snapshot_id"
	AttributeKeyVotingEnd     = "voting_end"
	AttributeKeyOutcome       = "outcome"
	AttributeKeyYesPower      = "yes_power"
	AttributeKeyNoPower       = "no_power"
	AttributeKeyAbstainPower  = "abstain_power"
	AttributeKeyApprovalRule  = "approval_rule"
	AttributeKeyThresholdBps  = "threshold_bps"
	AttributeKeyQuorumBps     = "quorum_bps"
	AttributeKeyVotingPeriod  = "voting_period"
	AttributeKeyAllowRevote   = "allow_revote"

	// Epic 4 event types.
	EventTypeDeposit              = "deposit"
	EventTypeUpdateDepositConfig  = "update_deposit_config"
	EventTypeDepositPeriodExpired = "deposit_period_expired"
	EventTypeDepositForfeit       = "deposit_forfeit"
	EventTypeDepositRefund        = "deposit_refund"

	// Epic 4 attribute keys.
	AttributeKeyDepositor        = "depositor"
	AttributeKeyDepositAmount    = "deposit_amount"
	AttributeKeyDepositCollected = "deposit_collected"
	AttributeKeyDepositDeadline  = "deposit_deadline"
	AttributeKeyForfeitDest      = "forfeit_destination"
	AttributeKeyRefundPolicy     = "refund_policy"
	AttributeKeyMinDeposit       = "min_deposit"
	AttributeKeyDepositPeriod    = "deposit_period"

	// Epic 5 event types.
	EventTypeExecuteProposal     = "execute_proposal"
	EventTypeExecutionFailed     = "execution_failed"
	EventTypeRenounceAdmin       = "renounce_admin"
	EventTypeUpdateVotingBackend = "update_voting_backend"

	// Epic 5 attribute keys.
	AttributeKeyExecutor    = "executor"
	AttributeKeyMsgIndex    = "msg_index"
	AttributeKeyMsgTypeURL  = "msg_type_url"
	AttributeKeyFailureMsg  = "failure_msg"
	AttributeKeyRewardID    = "reward_id"

	// Epic 6 event types.
	EventTypeCreatePoll        = "create_poll"
	EventTypeVoteOnPoll        = "vote_on_poll"
	EventTypeDepositOnPoll     = "deposit_on_poll"
	EventTypePollFinalized     = "poll_finalized"
	EventTypePollDepositExpired = "poll_deposit_expired"

	// Epic 6 attribute keys.
	AttributeKeyPollID             = "poll_id"
	AttributeKeyChoiceIndices      = "choice_indices"
	AttributeKeyWinningChoiceIndex = "winning_choice_index"
)
