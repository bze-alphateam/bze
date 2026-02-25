package v2types

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	DurationMin = 1
	DurationMax = 180
	RatioMin    = "0.01"
	RatioMax    = "1.00"
	ChancesMin  = 1
	ChancesMax  = 1_000_000
)

func (msg *MsgStartRaffle) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if !msg.Pot.IsPositive() {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "provided pot is not positive")
	}

	if msg.Duration < DurationMin || msg.Duration > DurationMax {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "duration must have a value between %d and %d", DurationMin, DurationMax)
	}

	if !msg.Ratio.IsPositive() {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "ratio is not positive (%s)", msg.Ratio.String())
	}

	if msg.Ratio.LT(math.LegacyMustNewDecFromStr(RatioMin)) || msg.Ratio.GT(math.LegacyMustNewDecFromStr(RatioMax)) {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "ratio must have a value between %s and %s", RatioMin, RatioMax)
	}

	if msg.Chances < ChancesMin || msg.Chances > ChancesMax {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "chances should have a value between %d and %d", ChancesMin, ChancesMax)
	}

	if !msg.TicketPrice.IsPositive() {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "provided ticket price is not positive")
	}

	if msg.Denom == "" {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "no denom provided")
	}

	return nil
}
