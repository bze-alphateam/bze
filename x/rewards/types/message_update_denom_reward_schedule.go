package types

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdateDenomRewardSchedule{}

func NewMsgUpdateDenomRewardSchedule(creator string, denom string, scheduleId string, duration string) *MsgUpdateDenomRewardSchedule {
	return &MsgUpdateDenomRewardSchedule{
		Creator:    creator,
		Denom:      denom,
		ScheduleId: scheduleId,
		Duration:   duration,
	}
}

func (msg *MsgUpdateDenomRewardSchedule) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrap(ErrInvalidStakingDenom, "denom cannot be empty")
	}

	if err := sdk.ValidateDenom(msg.Denom); err != nil {
		return errorsmod.Wrap(ErrInvalidStakingDenom, err.Error())
	}

	if msg.ScheduleId == "" {
		return errorsmod.Wrap(ErrInvalidScheduleId, "schedule_id cannot be empty")
	}

	// duration here is the number of extra days to add, mirrors MsgUpdateStakingReward.
	durationInt, err := strconv.ParseInt(msg.Duration, 10, 32)
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}
	if durationInt <= 0 || durationInt > int64(HundredYearsInDays) {
		return errorsmod.Wrapf(ErrInvalidDuration, "duration should be between 1 and %d days", HundredYearsInDays)
	}

	return nil
}
