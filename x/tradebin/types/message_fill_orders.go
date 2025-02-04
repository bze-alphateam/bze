package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgFillOrders{}

func NewMsgFillOrders(creator string, marketId string, orderType string, orders []*FillOrderItem) *MsgFillOrders {
	return &MsgFillOrders{
		Creator:   creator,
		MarketId:  marketId,
		OrderType: orderType,
		Orders:    orders,
	}
}

func (msg *MsgFillOrders) Route() string {
	return RouterKey
}

func (msg *MsgFillOrders) Type() string {
	return TypeMsgCancelOrder
}

func (msg *MsgFillOrders) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{creator}
}

func (msg *MsgFillOrders) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)

	return sdk.MustSortJSON(bz)
}

func (msg *MsgFillOrders) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.MarketId == "" {
		return sdkerrors.Wrapf(ErrInvalidOrderMarketId, "empty market_id")
	}

	if msg.OrderType != OrderTypeSell && msg.OrderType != OrderTypeBuy {
		return sdkerrors.Wrapf(ErrInvalidOrderType, "invalid order type")
	}

	if len(msg.Orders) == 0 {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "no orders to fill provided")
	}

	return nil
}
