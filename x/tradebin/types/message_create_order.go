package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
)

const TypeMsgCreateOrder = "create_order"
const (
	OrderTypeBuy  = "buy"
	OrderTypeSell = "sell"
)

var _ sdk.Msg = &MsgCreateOrder{}

func NewMsgCreateOrder(creator string, orderType string, amount int64, price string, marketId string) *MsgCreateOrder {
	return &MsgCreateOrder{
		Creator:   creator,
		OrderType: orderType,
		Amount:    amount,
		Price:     price,
		MarketId:  marketId,
	}
}

func (msg *MsgCreateOrder) Route() string {
	return RouterKey
}

func (msg *MsgCreateOrder) Type() string {
	return TypeMsgCreateOrder
}

func (msg *MsgCreateOrder) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgCreateOrder) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreateOrder) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.OrderType != OrderTypeSell && msg.OrderType != OrderTypeBuy {
		return sdkerrors.Wrapf(ErrInvalidOrderType, "invalid order type")
	}

	if msg.Amount <= 0 {
		return sdkerrors.Wrapf(ErrInvalidOrderAmount, "invalid order amount")
	}

	priceFloat, err := strconv.ParseFloat(msg.Price, 64)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidOrderPrice, "order price is not float: %v", err)
	}

	if priceFloat <= 0 {
		return sdkerrors.Wrapf(ErrInvalidOrderPrice, "price should be higher than 0")
	}

	return nil
}
