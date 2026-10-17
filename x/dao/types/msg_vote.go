package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgVote{}

// NewMsgVote builds a MsgVote.
func NewMsgVote(voter string, proposalId uint64, option VoteOption) *MsgVote {
	return &MsgVote{Voter: voter, ProposalId: proposalId, Option: option}
}

// ValidateBasic runs the checks that need no state. Whether the proposal is in
// voting and the voter has weight on it is checked by the msg server.
func (msg *MsgVote) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Voter); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid voter address (%s)", err)
	}

	if msg.ProposalId == 0 {
		return errorsmod.Wrap(ErrProposalNotFound, "proposal id must be positive")
	}

	// an omitted option decodes to UNSPECIFIED: never count it as a vote
	if msg.Option == VoteOption_VOTE_OPTION_UNSPECIFIED {
		return errorsmod.Wrap(ErrInvalidVote, "option is not set")
	}

	if _, ok := VoteOption_name[int32(msg.Option)]; !ok {
		return errorsmod.Wrapf(ErrInvalidVote, "unknown option %d", msg.Option)
	}

	return nil
}
