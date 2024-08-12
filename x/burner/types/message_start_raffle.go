package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strings"
)

const (
	TypeMsgStartRaffle = "start_raffle"
	DurationMin        = 1
	DurationMax        = 180
	RatioMin           = "0.01"
	RatioMax           = "1.00"
	ChancesMin         = 1
	ChancesMax         = 1_000_000
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

func (msg *MsgStartRaffle) Route() string {
	return RouterKey
}

func (msg *MsgStartRaffle) Type() string {
	return TypeMsgStartRaffle
}

func (msg *MsgStartRaffle) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStartRaffle) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
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
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	potCoin, ok := sdk.NewIntFromString(msg.Pot)
	if !ok {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid pot provided (could not convert to string)")
	}

	if !potCoin.IsPositive() {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "provided pot is not positive")
	}
	raffle.Pot = msg.Pot

	duration, ok := sdk.NewIntFromString(msg.Duration)
	if !ok {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid duration (%s)", duration.String())
	}

	if !duration.IsPositive() {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "duration should be positive (%s)", duration.String())
	}

	if duration.GT(sdk.NewInt(DurationMax)) || duration.LT(sdk.NewInt(DurationMin)) {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "duration have a value between %d and %d", DurationMin, DurationMax)
	}
	raffle.Duration = duration.Uint64()

	ratio, err := sdk.NewDecFromStr(msg.Ratio)
	if err != nil {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid ratio provided (%s)", err)
	}

	if !ratio.IsPositive() {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "ratio is not positive (%s)", ratio.String())
	}

	if ratio.LT(sdk.MustNewDecFromStr(RatioMin)) || ratio.GT(sdk.MustNewDecFromStr(RatioMax)) {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "ratio must have a value between %s and %s", RatioMin, RatioMax)
	}

	raffle.Ratio = msg.Ratio

	chances, ok := sdk.NewIntFromString(msg.Chances)
	if !ok {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid chances provided (%s)", err)
	}

	if chances.LT(sdk.NewInt(ChancesMin)) || chances.GT(sdk.NewInt(ChancesMax)) {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "chances have a value greater between %d and %d", ChancesMin, ChancesMax)
	}
	raffle.Chances = chances.Uint64()

	ticketCoin, ok := sdk.NewIntFromString(msg.TicketPrice)
	if !ok {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid ticket price provided (could not convert)")
	}

	if !ticketCoin.IsPositive() {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "provided ticket price is not positive")
	}

	raffle.TicketPrice = msg.TicketPrice

	if !msg.isAllowedDenomForRaffle(msg.Denom) {
		return raffle, sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "ibc coins are not allowed in raffles")
	}

	raffle.Denom = msg.Denom

	return raffle, nil
}

func (msg *MsgStartRaffle) isAllowedDenomForRaffle(denom string) bool {
	return !strings.HasPrefix(denom, "ibc/")
}
