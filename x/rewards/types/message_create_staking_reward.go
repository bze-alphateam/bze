package types

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgCreateStakingReward = "create_staking_reward"
	TypeMsgUpdateStakingReward = "update_staking_reward"

	tenYearsInDays     = 365 * 10
	HundredYearsInDays = 365 * 100

	defaultStakedAmount     = "0"
	defaultDistributedStake = "0"
)

var _ sdk.Msg = &MsgCreateStakingReward{}

func NewMsgCreateStakingReward(creator string, prizeAmount string, prizeDenom string, stakingDenom string, duration string, minStake string, lock string) *MsgCreateStakingReward {
	return &MsgCreateStakingReward{
		Creator:      creator,
		PrizeAmount:  prizeAmount,
		PrizeDenom:   prizeDenom,
		StakingDenom: stakingDenom,
		Duration:     duration,
		MinStake:     minStake,
		Lock:         lock,
	}
}

func (msg *MsgCreateStakingReward) ToStakingReward() (StakingReward, error) {
	sr := StakingReward{}

	amtInt, ok := math.NewIntFromString(msg.PrizeAmount)
	if !ok {
		return sr, errorsmod.Wrapf(ErrInvalidAmount, "could not convert order amount")
	}
	if !amtInt.IsPositive() {
		return sr, errorsmod.Wrapf(ErrInvalidAmount, "amount should be greater than 0")
	}
	sr.PrizeAmount = msg.PrizeAmount

	if msg.PrizeDenom == "" {
		return sr, ErrInvalidPrizeDenom
	}
	sr.PrizeDenom = msg.PrizeDenom

	if msg.StakingDenom == "" {
		return sr, ErrInvalidStakingDenom
	}
	sr.StakingDenom = msg.StakingDenom

	minStake, ok := math.NewIntFromString(msg.MinStake)
	if !ok || minStake.IsNegative() {
		return sr, ErrInvalidMinStake
	}
	sr.MinStake = minStake.Uint64()

	durationInt, err := strconv.Atoi(msg.Duration)
	if err != nil {
		return sr, errorsmod.Wrapf(ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}
	if durationInt <= 0 || durationInt > HundredYearsInDays {
		return sr, errorsmod.Wrapf(ErrInvalidDuration, "duration should be between 1 and %d days", HundredYearsInDays)
	}
	sr.Duration = uint32(durationInt)

	lockInt, err := strconv.Atoi(msg.Lock)
	if err != nil {
		return sr, errorsmod.Wrapf(ErrInvalidLockingTime, "could not convert string to int: %s", err.Error())
	}
	if lockInt < 0 || lockInt > tenYearsInDays {
		return sr, errorsmod.Wrapf(ErrInvalidLockingTime, "locking time should be between 0 and %d days", tenYearsInDays)
	}
	sr.Lock = uint32(lockInt)

	sr.StakedAmount = defaultStakedAmount
	sr.DistributedStake = defaultDistributedStake

	return sr, nil
}

func (msg *MsgCreateStakingReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	_, err = msg.ToStakingReward()
	if err != nil {
		return err
	}

	return nil
}
