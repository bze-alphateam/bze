package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	OrderTypeBuy  = "buy"
	OrderTypeSell = "sell"

	MessageTypeCancel   = "cancel"
	MessageTypeFillBuy  = "fill_buy"
	MessageTypeFillSell = "fill_sell"
)

var _ sdk.Msg = &MsgCreateOrder{}

func NewMsgCreateOrder(creator string, orderType string, amount string, price string, marketId string) *MsgCreateOrder {
	return &MsgCreateOrder{
		Creator:   creator,
		OrderType: orderType,
		Amount:    amount,
		Price:     price,
		MarketId:  marketId,
	}
}

func (msg *MsgCreateOrder) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.OrderType != OrderTypeSell && msg.OrderType != OrderTypeBuy {
		return errorsmod.Wrapf(ErrInvalidOrderType, "invalid order type")
	}

	amtInt, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return errorsmod.Wrapf(ErrInvalidOrderAmount, "could not convert order amount")
	}
	if !amtInt.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidOrderAmount, "invalid order amount")
	}

	priceDec, err := math.LegacyNewDecFromStr(msg.Price)
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidOrderPrice, "invalid price provided")
	}

	if priceDec.LTE(math.LegacyZeroDec()) {
		return errorsmod.Wrapf(ErrInvalidOrderPrice, "price should be higher than 0")
	}

	return nil
}

func TheOtherOrderType(orderType string) string {
	if orderType == OrderTypeBuy {
		return OrderTypeSell
	}

	return OrderTypeBuy
}
