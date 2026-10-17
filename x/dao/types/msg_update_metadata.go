package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdateMetadata{}

// NewMsgUpdateMetadata builds a MsgUpdateMetadata; authority is the DAO's own
// address.
func NewMsgUpdateMetadata(authority string, daoId uint64, name, description string) *MsgUpdateMetadata {
	return &MsgUpdateMetadata{Authority: authority, DaoId: daoId, Name: name, Description: description}
}

// ValidateBasic runs the checks that need no state. Whether the authority is
// the DAO's address is checked by the msg server.
func (msg *MsgUpdateMetadata) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	if msg.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao id must be positive")
	}

	return ValidateMetadata(msg.Name, msg.Description)
}
