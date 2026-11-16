package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSettleProposal{}

// NewMsgSettleProposal builds a MsgSettleProposal.
func NewMsgSettleProposal(sender string, proposalId uint64) *MsgSettleProposal {
	return &MsgSettleProposal{Sender: sender, ProposalId: proposalId}
}

// ValidateBasic runs the checks that need no state. Anyone can settle; whether
// the proposal can be settled now is checked by the msg server.
func (msg *MsgSettleProposal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address (%s)", err)
	}

	if msg.ProposalId == 0 {
		return errorsmod.Wrap(ErrProposalNotFound, "proposal id must be positive")
	}

	return nil
}
