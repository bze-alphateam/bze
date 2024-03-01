package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgCancelOrder = "cancel_order"

	OrderTypeCancel = "cancel"
)

var _ sdk.Msg = &MsgCancelOrder{}

func NewMsgCancelOrder(creator string, marketId string, orderId string, orderType string) *MsgCancelOrder {
	return &MsgCancelOrder{
		Creator:   creator,
		MarketId:  marketId,
		OrderId:   orderId,
		OrderType: orderType,
	}
}

func (msg *MsgCancelOrder) Route() string {
	return RouterKey
}

func (msg *MsgCancelOrder) Type() string {
	return TypeMsgCancelOrder
}

func (msg *MsgCancelOrder) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgCancelOrder) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCancelOrder) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.MarketId == "" {
		return sdkerrors.Wrapf(ErrInvalidOrderMarketId, "empty market_id")
	}

	if msg.OrderId == "" {
		return sdkerrors.Wrapf(ErrInvalidOrderId, "empty order_id")
	}

	if msg.OrderType != OrderTypeSell && msg.OrderType != OrderTypeBuy {
		return sdkerrors.Wrapf(ErrInvalidOrderType, "invalid order type")
	}

	return nil
}
