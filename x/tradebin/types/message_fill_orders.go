package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	maxFillOrders = 50
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

func (msg *MsgFillOrders) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.MarketId == "" {
		return errorsmod.Wrapf(ErrInvalidOrderMarketId, "empty market_id")
	}

	if msg.OrderType != OrderTypeSell && msg.OrderType != OrderTypeBuy {
		return errorsmod.Wrapf(ErrInvalidOrderType, "invalid order type")
	}

	if len(msg.Orders) == 0 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "no orders to fill provided")
	}

	if len(msg.Orders) > maxFillOrders {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "too many orders to fill, max %d orders allowed", maxFillOrders)
	}

	pricesMap := make(map[string]struct{})
	for _, fo := range msg.Orders {
		decPrice, err := math.LegacyNewDecFromStr(fo.Price)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid price %s: %s", fo.Price, err.Error())
		}

		if !decPrice.IsPositive() {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "price should be positive: %s", fo.Price)
		}

		if decPrice.LTE(minPrice) {
			return errorsmod.Wrapf(ErrInvalidOrderPrice, "price should be higher than %s", minPrice.String())
		}

		if _, ok := pricesMap[fo.Price]; ok {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "duplicate price %s found in items", fo.Price)
		}
		pricesMap[fo.Price] = struct{}{}
	}

	return nil
}
