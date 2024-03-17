package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgJoinStaking = "join_staking"

var _ sdk.Msg = &MsgJoinStaking{}

func NewMsgJoinStaking(creator string, rewardId string, amount string) *MsgJoinStaking {
	return &MsgJoinStaking{
		Creator:  creator,
		RewardId: rewardId,
		Amount:   amount,
	}
}

func (msg *MsgJoinStaking) Route() string {
	return RouterKey
}

func (msg *MsgJoinStaking) Type() string {
	return TypeMsgJoinStaking
}

func (msg *MsgJoinStaking) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgJoinStaking) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgJoinStaking) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.RewardId == "" {
		return sdkerrors.Wrapf(ErrInvalidRewardId, "empty reward id")
	}

	if msg.Amount == "" {
		return sdkerrors.Wrapf(ErrInvalidAmount, "empty amount provided")
	}

	amtInt, ok := sdk.NewIntFromString(msg.Amount)
	if !ok {
		return sdkerrors.Wrapf(ErrInvalidAmount, "could not convert amount")
	}
	if !amtInt.IsPositive() {
		return sdkerrors.Wrapf(ErrInvalidAmount, "amount should be greater than 0")
	}

	return nil
}
