package types

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgCreateBoost = "create_boost"
	TypeMsgUpdateBoost = "update_boost"
)

var _ sdk.Msg = &MsgCreateBoost{}

func NewMsgCreateBoost(creator, rewardId, denom, dailyAmount, days string) *MsgCreateBoost {
	return &MsgCreateBoost{
		Creator:     creator,
		RewardId:    rewardId,
		Denom:       denom,
		DailyAmount: dailyAmount,
		Days:        days,
	}
}

func (msg *MsgCreateBoost) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.RewardId == "" {
		return errorsmod.Wrap(ErrInvalidRewardId, "reward_id should not be empty")
	}

	if msg.Denom == "" {
		return errorsmod.Wrap(ErrInvalidBoostDenom, "denom should not be empty")
	}

	dailyAmount, ok := math.NewIntFromString(msg.DailyAmount)
	if !ok {
		return errorsmod.Wrap(ErrInvalidAmount, "could not convert daily_amount to int")
	}
	if !dailyAmount.IsPositive() {
		return errorsmod.Wrap(ErrInvalidAmount, "daily_amount should be greater than 0")
	}

	return validateBoostDays(msg.Days)
}

func validateBoostDays(days string) error {
	daysInt, err := strconv.Atoi(days)
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidBoostDays, "could not convert days to int: %s", err.Error())
	}
	if daysInt < 1 || daysInt > HundredYearsInDays {
		return errorsmod.Wrapf(ErrInvalidBoostDays, "days should be between 1 and %d", HundredYearsInDays)
	}

	return nil
}
