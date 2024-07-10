package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgDistributeStakingRewards = "distribute_staking_rewards"

var _ sdk.Msg = &MsgDistributeStakingRewards{}

func NewMsgDistributeStakingRewards(creator string, rewardId string, amount string) *MsgDistributeStakingRewards {
	return &MsgDistributeStakingRewards{
		Creator:  creator,
		RewardId: rewardId,
		Amount:   amount,
	}
}

func (msg *MsgDistributeStakingRewards) Route() string {
	return RouterKey
}

func (msg *MsgDistributeStakingRewards) Type() string {
	return TypeMsgDistributeStakingRewards
}

func (msg *MsgDistributeStakingRewards) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgDistributeStakingRewards) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgDistributeStakingRewards) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	amtInt, ok := sdk.NewIntFromString(msg.Amount)
	if !ok {
		return sdkerrors.Wrapf(ErrInvalidAmount, "could not convert order amount")
	}

	if !amtInt.IsPositive() {
		return sdkerrors.Wrapf(ErrInvalidAmount, "amount should be greater than 0")
	}

	return nil
}
