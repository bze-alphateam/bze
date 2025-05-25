package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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

func (msg *MsgCancelOrder) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.MarketId == "" {
		return errorsmod.Wrapf(ErrInvalidOrderMarketId, "empty market_id")
	}

	if msg.OrderId == "" {
		return errorsmod.Wrapf(ErrInvalidOrderId, "empty order_id")
	}

	if msg.OrderType != OrderTypeSell && msg.OrderType != OrderTypeBuy {
		return errorsmod.Wrapf(ErrInvalidOrderType, "invalid order type")
	}

	return nil
}
