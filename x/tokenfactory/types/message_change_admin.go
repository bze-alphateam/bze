package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgChangeAdmin{}

func NewMsgChangeAdmin(creator, denom, newAdmin string) *MsgChangeAdmin {
	return &MsgChangeAdmin{
		Creator:  creator,
		Denom:    denom,
		NewAdmin: newAdmin,
	}
}

func (msg *MsgChangeAdmin) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.NewAdmin)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid new admin address (%s)", err)
	}

	if msg.Creator == msg.NewAdmin {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "new admin must be different from creator")
	}

	if msg.Denom == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "denom must not be empty")
	}

	return nil
}
