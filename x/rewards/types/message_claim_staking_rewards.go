package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgClaimStakingRewards = "claim_staking_rewards"

var _ sdk.Msg = &MsgClaimStakingRewards{}

func NewMsgClaimStakingRewards(creator string, rewardId string) *MsgClaimStakingRewards {
	return &MsgClaimStakingRewards{
		Creator:  creator,
		RewardId: rewardId,
	}
}

func (msg *MsgClaimStakingRewards) Route() string {
	return RouterKey
}

func (msg *MsgClaimStakingRewards) Type() string {
	return TypeMsgClaimStakingRewards
}

func (msg *MsgClaimStakingRewards) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgClaimStakingRewards) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgClaimStakingRewards) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	
	if msg.RewardId == "" {
		return sdkerrors.Wrapf(ErrInvalidRewardId, "empty reward id")
	}

	return nil
}
