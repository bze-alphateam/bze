package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgRemoveStakingReward{}

func NewMsgRemoveStakingReward(creator string, rewardId string) *MsgRemoveStakingReward {
	return &MsgRemoveStakingReward{
		Creator:  creator,
		RewardId: rewardId,
	}
}

func (msg *MsgRemoveStakingReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.RewardId == "" {
		return errorsmod.Wrap(ErrInvalidRewardId, "reward_id cannot be empty")
	}

	return nil
}
