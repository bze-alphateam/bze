package v2types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	maxFillOrders = 50
)

func (msg *MsgFillOrders) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.MarketId == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty market_id")
	}

	if msg.OrderType != OrderTypeSell && msg.OrderType != OrderTypeBuy {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "invalid order type")
	}

	if len(msg.Orders) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "no orders to fill provided")
	}

	if len(msg.Orders) > maxFillOrders {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "too many orders to fill, max %d orders allowed", maxFillOrders)
	}

	pricesMap := make(map[string]struct{})
	for _, fo := range msg.Orders {
		if !fo.Price.IsPositive() {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "price should be positive: %s", fo.Price.String())
		}

		if fo.Price.LTE(minPrice) {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "price should be higher than %s", minPrice.String())
		}

		priceStr := fo.Price.String()
		if _, ok := pricesMap[priceStr]; ok {
			return errors.Wrapf(sdkerrors.ErrInvalidRequest, "duplicate price %s found in items", priceStr)
		}
		pricesMap[priceStr] = struct{}{}
	}

	return nil
}
