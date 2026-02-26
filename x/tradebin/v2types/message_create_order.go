package v2types

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	OrderTypeBuy  = "buy"
	OrderTypeSell = "sell"
)

var minPrice = math.LegacyNewDecWithPrec(1, 10)

func (msg *MsgCreateOrder) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.MarketId == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "no market id provided")
	}

	if msg.OrderType != OrderTypeSell && msg.OrderType != OrderTypeBuy {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "invalid order type")
	}

	if !msg.Amount.IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "invalid order amount")
	}

	if msg.Price.LTE(minPrice) {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "price should be higher than %s", minPrice.String())
	}

	return nil
}
