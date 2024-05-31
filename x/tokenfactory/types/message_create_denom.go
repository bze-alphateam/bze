package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgCreateDenom = "create_denom"

var _ sdk.Msg = &MsgCreateDenom{}

func NewMsgCreateDenom(creator string, subdenom string) *MsgCreateDenom {
	return &MsgCreateDenom{
		Creator:  creator,
		Subdenom: subdenom,
	}
}

func (msg *MsgCreateDenom) Route() string {
	return RouterKey
}

func (msg *MsgCreateDenom) Type() string {
	return TypeMsgCreateDenom
}

func (msg *MsgCreateDenom) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgCreateDenom) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreateDenom) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
