package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCleanupProposal{}

// NewMsgCleanupProposal builds a MsgCleanupProposal.
func NewMsgCleanupProposal(sender string, proposalId, limit uint64) *MsgCleanupProposal {
	return &MsgCleanupProposal{Sender: sender, ProposalId: proposalId, Limit: limit}
}

// ValidateBasic runs the checks that need no state. Anyone can clean up; any
// limit is valid (0 and values above the cleanup_batch_size param mean that
// param); whether the proposal is finished is checked by the msg server.
func (msg *MsgCleanupProposal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}

	if msg.ProposalId == 0 {
		return errorsmod.Wrap(ErrProposalNotFound, "proposal id must be positive")
	}

	return nil
}
