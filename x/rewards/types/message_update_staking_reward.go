package types

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdateStakingReward{}

func NewMsgUpdateStakingReward(creator string, rewardId string, duration string) *MsgUpdateStakingReward {
	return &MsgUpdateStakingReward{
		Creator:  creator,
		RewardId: rewardId,
		Duration: duration,
	}
}

func (msg *MsgUpdateStakingReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	durationInt, err := strconv.ParseInt(msg.Duration, 10, 32)
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}

	if durationInt <= 0 || durationInt > int64(HundredYearsInDays) {
		return ErrInvalidDuration
	}

	return nil
}
