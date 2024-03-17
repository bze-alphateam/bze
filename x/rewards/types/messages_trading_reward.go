package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
)

const (
	TypeMsgCreateTradingReward = "create_trading_reward"

	maxAllowedSlots = 100
)

var _ sdk.Msg = &MsgCreateTradingReward{}

func NewMsgCreateTradingReward(
	creator string,
	prizeAmount string,
	prizeDenom string,
	duration string,
	marketId string,
	slots string,

) *MsgCreateTradingReward {
	return &MsgCreateTradingReward{
		Creator:     creator,
		PrizeAmount: prizeAmount,
		PrizeDenom:  prizeDenom,
		Duration:    duration,
		MarketId:    marketId,
		Slots:       slots,
	}
}

func (msg *MsgCreateTradingReward) Route() string {
	return RouterKey
}

func (msg *MsgCreateTradingReward) Type() string {
	return TypeMsgCreateTradingReward
}

func (msg *MsgCreateTradingReward) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgCreateTradingReward) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreateTradingReward) ToTradingReward() (TradingReward, error) {
	tr := TradingReward{}

	amtInt, ok := sdk.NewIntFromString(msg.PrizeAmount)
	if !ok {
		return tr, sdkerrors.Wrapf(ErrInvalidAmount, "could not convert order amount")
	}
	if !amtInt.IsPositive() {
		return tr, sdkerrors.Wrapf(ErrInvalidAmount, "amount should be greater than 0")
	}
	tr.PrizeAmount = amtInt.String()

	if msg.PrizeDenom == "" {
		return tr, ErrInvalidPrizeDenom
	}
	tr.PrizeDenom = msg.PrizeDenom

	if msg.MarketId == "" {
		return tr, ErrInvalidMarketId
	}
	tr.MarketId = msg.MarketId

	durationInt, err := strconv.Atoi(msg.Duration)
	if err != nil {
		return tr, sdkerrors.Wrapf(ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}
	if durationInt <= 0 || durationInt > tenYearsInDays {
		return tr, ErrInvalidDuration
	}
	tr.Duration = uint32(durationInt)

	slotsInt, err := strconv.Atoi(msg.Slots)
	if err != nil {
		return tr, sdkerrors.Wrapf(ErrInvalidSlots, "could not convert slots to int: %s", err.Error())
	}
	if slotsInt <= 0 || slotsInt > maxAllowedSlots {
		return tr, ErrInvalidSlots
	}
	tr.Slots = uint32(slotsInt)

	return tr, nil
}

func (msg *MsgCreateTradingReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	_, err = msg.ToTradingReward()
	if err != nil {
		return err
	}

	return nil
}
