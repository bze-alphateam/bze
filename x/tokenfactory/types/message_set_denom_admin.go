package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSetDenomAdmin{}

func NewMsgSetDenomAdmin(authority, denom, newAdmin string) *MsgSetDenomAdmin {
	return &MsgSetDenomAdmin{
		Authority: authority,
		Denom:     denom,
		NewAdmin:  newAdmin,
	}
}

func (msg *MsgSetDenomAdmin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrap(ErrInvalidDenom, "denom must not be empty")
	}

	if err := sdk.ValidateDenom(msg.Denom); err != nil {
		return errorsmod.Wrapf(ErrInvalidDenom, "invalid denom %s (%s)", msg.Denom, err)
	}

	// governance gives a denom to someone, it never locks one: renouncing stays the admin's decision
	if msg.NewAdmin == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "new admin must not be empty")
	}

	if _, err := sdk.AccAddressFromBech32(msg.NewAdmin); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid new admin address (%s)", err)
	}

	return nil
}
