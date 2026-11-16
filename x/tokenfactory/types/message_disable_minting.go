package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgDisableMinting{}

func NewMsgDisableMinting(creator, denom string) *MsgDisableMinting {
	return &MsgDisableMinting{
		Creator: creator,
		Denom:   denom,
	}
}

func (msg *MsgDisableMinting) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	_, _, err = DeconstructDenom(msg.Denom)
	if err != nil {
		return err
	}

	return nil
}
