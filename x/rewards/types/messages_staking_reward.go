package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
)

const (
	TypeMsgCreateStakingReward = "create_staking_reward"
	TypeMsgUpdateStakingReward = "update_staking_reward"

	tenYearsInDays = 365 * 10

	defaultStakedAmount     = "0"
	defaultDistributedStake = "0"
)

var _ sdk.Msg = &MsgCreateStakingReward{}

func NewMsgCreateStakingReward(
	creator string,
	prizeAmount string,
	prizeDenom string,
	stakingDenom string,
	duration string,
	minStake string,
	lock string,

) *MsgCreateStakingReward {
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

func (msg *MsgCreateStakingReward) Route() string {
	return RouterKey
}

func (msg *MsgCreateStakingReward) Type() string {
	return TypeMsgCreateStakingReward
}

func (msg *MsgCreateStakingReward) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgCreateStakingReward) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreateStakingReward) ToStakingReward() (StakingReward, error) {
	sr := StakingReward{}

	amtInt, ok := sdk.NewIntFromString(msg.PrizeAmount)
	if !ok {
		return sr, sdkerrors.Wrapf(ErrInvalidAmount, "could not convert order amount")
	}
	if !amtInt.IsPositive() {
		return sr, sdkerrors.Wrapf(ErrInvalidAmount, "amount should be greater than 0")
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

	minStake, ok := sdk.NewIntFromString(msg.MinStake)
	if !ok || minStake.IsNegative() {
		return sr, ErrInvalidMinStake
	}
	sr.MinStake = minStake.Uint64()

	durationInt, err := strconv.Atoi(msg.Duration)
	if err != nil {
		return sr, sdkerrors.Wrapf(ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}
	if durationInt <= 0 || durationInt > tenYearsInDays {
		return sr, ErrInvalidDuration
	}
	sr.Duration = uint32(durationInt)

	lockInt, err := strconv.Atoi(msg.Lock)
	if err != nil {
		return sr, sdkerrors.Wrapf(ErrInvalidLockingTime, "could not convert string to int: %s", err.Error())
	}
	if lockInt < 0 || lockInt > tenYearsInDays {
		return sr, ErrInvalidLockingTime
	}
	sr.Lock = uint32(lockInt)

	sr.StakedAmount = defaultStakedAmount
	sr.DistributedStake = defaultDistributedStake

	return sr, nil
}

func (msg *MsgCreateStakingReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	_, err = msg.ToStakingReward()
	if err != nil {
		return err
	}

	return nil
}

var _ sdk.Msg = &MsgUpdateStakingReward{}

func NewMsgUpdateStakingReward(
	creator string,
	rewardId string,
	duration string,

) *MsgUpdateStakingReward {
	return &MsgUpdateStakingReward{
		Creator:  creator,
		RewardId: rewardId,
		Duration: duration,
	}
}

func (msg *MsgUpdateStakingReward) Route() string {
	return RouterKey
}

func (msg *MsgUpdateStakingReward) Type() string {
	return TypeMsgUpdateStakingReward
}

func (msg *MsgUpdateStakingReward) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgUpdateStakingReward) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUpdateStakingReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	durationInt, err := strconv.ParseUint(msg.Duration, 10, 32)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidDuration, "could not convert duration to int: %s", err.Error())
	}

	if durationInt <= 0 {
		return ErrInvalidDuration
	}

	return nil
}
