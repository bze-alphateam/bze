package types

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateDenomRewardSchedule{}

func NewMsgCreateDenomRewardSchedule(creator string, denom string, prizeDenom string, dailyAmount math.Int, duration string) *MsgCreateDenomRewardSchedule {
	return &MsgCreateDenomRewardSchedule{
		Creator:     creator,
		Denom:       denom,
		PrizeDenom:  prizeDenom,
		DailyAmount: dailyAmount,
		Duration:    duration,
	}
}

func (msg *MsgCreateDenomRewardSchedule) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrap(ErrInvalidStakingDenom, "denom cannot be empty")
	}

	if msg.PrizeDenom == "" {
		return errorsmod.Wrap(ErrInvalidPrizeDenom, "prize_denom cannot be empty")
	}

	if msg.DailyAmount.IsNil() || !msg.DailyAmount.IsPositive() {
		return errorsmod.Wrap(ErrInvalidAmount, "daily_amount should be greater than 0")
	}

	durationInt, err := strconv.Atoi(msg.Duration)
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}
	if durationInt <= 0 || durationInt > HundredYearsInDays {
		return errorsmod.Wrapf(ErrInvalidDuration, "duration should be between 1 and %d days", HundredYearsInDays)
	}

	return nil
}
