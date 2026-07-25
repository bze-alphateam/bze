package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSetDenomBranding{}

func NewMsgSetDenomBranding(creator, denom string, branding *DenomBranding) *MsgSetDenomBranding {
	return &MsgSetDenomBranding{
		Creator:  creator,
		Denom:    denom,
		Branding: branding,
	}
}

func (msg *MsgSetDenomBranding) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Denom == "" {
		return ErrInvalidDenom.Wrap("denom cannot be empty")
	}

	// an empty branding is a valid "clear branding" request
	if msg.Branding.IsEmpty() {
		return nil
	}

	return msg.Branding.Validate()
}
