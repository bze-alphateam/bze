package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgVote{}

// ValidateBasic performs stateless validation of MsgVote.
//
// Rules:
//   - voter is valid bech32.
//   - dao_id and proposal_id are non-zero.
//   - option is one of YES / NO / ABSTAIN (UNSPECIFIED rejected).
//
// Stateful checks deferred to the keeper:
//   - Proposal exists and is in PROPOSAL_STATUS_VOTING.
//   - Voter has non-zero SnapshotPower at the proposal's snapshot_id.
//   - Revote permission check (governance_snapshot.allow_revote).
func (m *MsgVote) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Voter); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "voter: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	if m.ProposalId == 0 {
		return errorsmod.Wrap(ErrProposalNotFound, "proposal_id must be non-zero")
	}
	switch m.Option {
	case VoteOption_VOTE_OPTION_YES,
		VoteOption_VOTE_OPTION_NO,
		VoteOption_VOTE_OPTION_ABSTAIN:
		return nil
	default:
		return errorsmod.Wrapf(ErrInvalidVoteOption,
			"option must be YES/NO/ABSTAIN, got %v", m.Option)
	}
}
