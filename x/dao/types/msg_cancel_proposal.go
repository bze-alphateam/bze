package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCancelProposal{}

// NewMsgCancelProposal builds a MsgCancelProposal.
func NewMsgCancelProposal(proposer string, proposalId uint64) *MsgCancelProposal {
	return &MsgCancelProposal{Proposer: proposer, ProposalId: proposalId}
}

// ValidateBasic runs the checks that need no state. Whether the proposal is in
// voting and the signer proposed it is checked by the msg server.
func (msg *MsgCancelProposal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid proposer address (%s)", err)
	}

	if msg.ProposalId == 0 {
		return errorsmod.Wrap(ErrProposalNotFound, "proposal id must be positive")
	}

	return nil
}
