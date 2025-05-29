package types

import (
	"strings"

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

var _ sdk.Msg = &MsgStartRaffle{}

func NewMsgStartRaffle(creator, pot, duration, chances, ratio, ticketPrice, denom string) *MsgStartRaffle {
	return &MsgStartRaffle{
		Creator:     creator,
		Pot:         pot,
		Duration:    duration,
		Chances:     chances,
		Ratio:       ratio,
		TicketPrice: ticketPrice,
		Denom:       denom,
	}
}

func (msg *MsgStartRaffle) ValidateBasic() error {
	_, err := msg.ToStorageRaffle()
	if err != nil {
		return err
	}

	return nil
}

func (msg *MsgStartRaffle) ToStorageRaffle() (raffle Raffle, err error) {
	_, err = sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	potCoin, ok := math.NewIntFromString(msg.Pot)
	if !ok {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid pot provided (could not convert to string)")
	}

	if !potCoin.IsPositive() {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidCoins, "provided pot is not positive")
	}
	raffle.Pot = msg.Pot

	duration, ok := math.NewIntFromString(msg.Duration)
	if !ok {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid duration (%s)", msg.Duration)
	}

	if !duration.IsPositive() {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidRequest, "duration should be positive (%s)", msg.Duration)
	}

	if duration.GT(math.NewInt(DurationMax)) || duration.LT(math.NewInt(DurationMin)) {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidRequest, "duration have a value between %d and %d", DurationMin, DurationMax)
	}
	raffle.Duration = duration.Uint64()

	ratio, err := math.LegacyNewDecFromStr(msg.Ratio)
	if err != nil {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid ratio provided (%s)", err)
	}

	if !ratio.IsPositive() {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidRequest, "ratio is not positive (%s)", ratio.String())
	}

	if ratio.LT(math.LegacyMustNewDecFromStr(RatioMin)) || ratio.GT(math.LegacyMustNewDecFromStr(RatioMax)) {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidRequest, "ratio must have a value between %s and %s", RatioMin, RatioMax)
	}

	raffle.Ratio = msg.Ratio

	chances, ok := math.NewIntFromString(msg.Chances)
	if !ok {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid chances provided (%s)", err)
	}

	if chances.LT(math.NewInt(ChancesMin)) || chances.GT(math.NewInt(ChancesMax)) {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidRequest, "chances should have a value between %d and %d", ChancesMin, ChancesMax)
	}
	raffle.Chances = chances.Uint64()

	ticketCoin, ok := math.NewIntFromString(msg.TicketPrice)
	if !ok {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid ticket price provided (could not convert)")
	}

	if !ticketCoin.IsPositive() {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidCoins, "provided ticket price is not positive")
	}

	raffle.TicketPrice = msg.TicketPrice

	if !msg.isAllowedDenomForRaffle(msg.Denom) {
		return raffle, errors.Wrapf(sdkerrors.ErrInvalidCoins, "coin not allowed in raffles")
	}

	raffle.Denom = msg.Denom

	return raffle, nil
}

func (msg *MsgStartRaffle) isAllowedDenomForRaffle(denom string) bool {
	return !strings.HasPrefix(denom, "ibc/")
}
