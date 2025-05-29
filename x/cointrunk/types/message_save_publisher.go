package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSavePublisher{}

const (
	nameMinLen = 3
	nameMaxLen = 256
)

func NewMsgSavePublisher(creator string, name string, address string, active bool) *MsgSavePublisher {
	return &MsgSavePublisher{
		Authority: creator,
		Name:      name,
		Address:   address,
		Active:    active,
	}
}

func (msg *MsgSavePublisher) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority (%s)", err)
	}

	if len(msg.Name) < nameMinLen {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid name: expecting at least %d characters", nameMinLen)
	}

	if len(msg.Name) > nameMaxLen {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid name: expecting not more than %d characters", nameMaxLen)
	}

	_, err = sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid publisher address (%s)", err)
	}

	return nil
}
