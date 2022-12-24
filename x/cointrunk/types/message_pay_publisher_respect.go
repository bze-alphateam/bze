package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgPayPublisherRespect = "pay_publisher_respect"

var _ sdk.Msg = &MsgPayPublisherRespect{}

func NewMsgPayPublisherRespect(creator string, address string, amount string) *MsgPayPublisherRespect {
	return &MsgPayPublisherRespect{
		Creator: creator,
		Address: address,
		Amount:  amount,
	}
}

func (msg *MsgPayPublisherRespect) Route() string {
	return RouterKey
}

func (msg *MsgPayPublisherRespect) Type() string {
	return TypeMsgPayPublisherRespect
}

func (msg *MsgPayPublisherRespect) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPayPublisherRespect) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPayPublisherRespect) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid publisher address (%s)", err)
	}

	amount, err := sdk.ParseCoinNormalized(msg.Amount)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid amount (%s)", err)
	}

	if !amount.IsPositive() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, "invalid amount: amount should be positive")
	}

	return nil
}
