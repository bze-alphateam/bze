package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgFundBurner = "fund_burner"

var _ sdk.Msg = &MsgFundBurner{}

func NewMsgFundBurner(creator string, amount string) *MsgFundBurner {
	return &MsgFundBurner{
		Creator: creator,
		Amount:  amount,
	}
}

func (msg *MsgFundBurner) Route() string {
	return RouterKey
}

func (msg *MsgFundBurner) Type() string {
	return TypeMsgFundBurner
}

func (msg *MsgFundBurner) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgFundBurner) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgFundBurner) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
