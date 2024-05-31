package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgExitStaking = "exit_staking"

var _ sdk.Msg = &MsgExitStaking{}

func NewMsgExitStaking(creator string, rewardId string) *MsgExitStaking {
	return &MsgExitStaking{
		Creator:  creator,
		RewardId: rewardId,
	}
}

func (msg *MsgExitStaking) Route() string {
	return RouterKey
}

func (msg *MsgExitStaking) Type() string {
	return TypeMsgExitStaking
}

func (msg *MsgExitStaking) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgExitStaking) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgExitStaking) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.RewardId == "" {
		return sdkerrors.Wrapf(ErrInvalidRewardId, "empty reward id")
	}

	return nil
}
