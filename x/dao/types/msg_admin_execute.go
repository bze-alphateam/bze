package types

import (
	errorsmod "cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
)

var (
	_ sdk.Msg                            = &MsgAdminExecute{}
	_ codectypes.UnpackInterfacesMessage = &MsgAdminExecute{}
)

// NewMsgAdminExecute packs msgs into a MsgAdminExecute.
func NewMsgAdminExecute(admin string, daoId uint64, msgs []sdk.Msg) (*MsgAdminExecute, error) {
	anys, err := sdktx.SetMsgs(msgs)
	if err != nil {
		return nil, err
	}

	return &MsgAdminExecute{Admin: admin, DaoId: daoId, Msgs: anys}, nil
}

// UnpackMsgs returns the messages to run as the DAO.
func (msg *MsgAdminExecute) UnpackMsgs() ([]sdk.Msg, error) {
	return sdktx.GetMsgs(msg.Msgs, "dao admin execute")
}

// UnpackInterfaces implements UnpackInterfacesMessage, so the Anys are
// resolved when the transaction is decoded.
func (msg MsgAdminExecute) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return sdktx.UnpackInterfaces(unpacker, msg.Msgs)
}

// ValidateBasic runs the checks that need no state. The message count and size
// caps are params and are checked by the msg server, as are the signers of the
// inner messages.
func (msg *MsgAdminExecute) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Admin); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}

	if msg.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao id must be positive")
	}

	if len(msg.Msgs) == 0 {
		return errorsmod.Wrap(ErrInvalidMsgs, "at least one message is required")
	}

	if _, err := msg.UnpackMsgs(); err != nil {
		return errorsmod.Wrap(ErrInvalidMsgs, err.Error())
	}

	return nil
}
