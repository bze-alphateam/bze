package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgCreateMarket = "create_market"

var _ sdk.Msg = &MsgCreateMarket{}

func NewMsgCreateMarket(creator string, asset1 string, asset2 string) *MsgCreateMarket {
	return &MsgCreateMarket{
		Creator: creator,
		Asset1:  asset1,
		Asset2:  asset2,
	}
}

func (msg *MsgCreateMarket) Route() string {
	return RouterKey
}

func (msg *MsgCreateMarket) Type() string {
	return TypeMsgCreateMarket
}

func (msg *MsgCreateMarket) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgCreateMarket) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreateMarket) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
