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

var minPrice = math.LegacyNewDecWithPrec(1, 10)

var _ sdk.Msg = &MsgCreateOrder{}

func NewMsgCreateOrder(creator string, orderType string, amount math.Int, price math.LegacyDec, marketId string) *MsgCreateOrder {
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

	if msg.MarketId == "" {
		return errorsmod.Wrapf(ErrInvalidOrderMarketId, "no market id provided")
	}

	if msg.OrderType != OrderTypeSell && msg.OrderType != OrderTypeBuy {
		return errorsmod.Wrapf(ErrInvalidOrderType, "invalid order type")
	}

	if !msg.Amount.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidOrderAmount, "invalid order amount")
	}

	if msg.Price.LTE(minPrice) {
		return errorsmod.Wrapf(ErrInvalidOrderPrice, "price should be higher than %s", minPrice.String())
	}

	return nil
}

func TheOtherOrderType(orderType string) string {
	if orderType == OrderTypeBuy {
		return OrderTypeSell
	}

	return OrderTypeBuy
}
