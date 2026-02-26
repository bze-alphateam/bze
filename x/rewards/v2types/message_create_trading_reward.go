package v2types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	maxAllowedSlots    = 10
	maxAllowedDuration = 365
)

func (msg *MsgCreateTradingReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if !msg.PrizeAmount.IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "amount should be greater than 0")
	}

	if msg.PrizeDenom == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "prize denom cannot be empty")
	}

	if msg.MarketId == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "market id cannot be empty")
	}

	if msg.Duration == 0 || msg.Duration > maxAllowedDuration {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "duration should be between 1 and %d", maxAllowedDuration)
	}

	if msg.Slots == 0 || msg.Slots > maxAllowedSlots {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "slots should be between 1 and %d", maxAllowedSlots)
	}

	return nil
}
