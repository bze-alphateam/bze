package types

import (
	errorsmod "cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
)

var (
	_ sdk.Msg                            = &MsgSubmitProposal{}
	_ codectypes.UnpackInterfacesMessage = &MsgSubmitProposal{}
)

// NewMsgSubmitProposal packs msgs into a MsgSubmitProposal.
func NewMsgSubmitProposal(proposer string, daoId uint64, title, summary string, msgs []sdk.Msg, expedited bool) (*MsgSubmitProposal, error) {
	anys, err := sdktx.SetMsgs(msgs)
	if err != nil {
		return nil, err
	}

	return &MsgSubmitProposal{
		Proposer:  proposer,
		DaoId:     daoId,
		Title:     title,
		Summary:   summary,
		Msgs:      anys,
		Expedited: expedited,
	}, nil
}

// UnpackMsgs returns the messages the proposal runs if it passes.
func (msg *MsgSubmitProposal) UnpackMsgs() ([]sdk.Msg, error) {
	return sdktx.GetMsgs(msg.Msgs, "dao proposal")
}

// UnpackInterfaces implements UnpackInterfacesMessage, so the Anys are
// resolved when the transaction is decoded.
func (msg MsgSubmitProposal) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return sdktx.UnpackInterfaces(unpacker, msg.Msgs)
}

// ValidateBasic runs the checks that need no state. The title, summary,
// message count and size caps are params and are checked by the msg server,
// as are the signers of the inner messages. A proposal may carry no message
// at all (a signalling proposal).
func (msg *MsgSubmitProposal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid proposer address (%s)", err)
	}

	if msg.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao id must be positive")
	}

	if msg.Title == "" {
		return errorsmod.Wrap(ErrInvalidProposal, "a title is required")
	}

	if _, err := msg.UnpackMsgs(); err != nil {
		return errorsmod.Wrap(ErrInvalidMsgs, err.Error())
	}

	return nil
}
