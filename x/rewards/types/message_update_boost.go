package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdateBoost{}

func NewMsgUpdateBoost(creator, rewardId, boostId, days string) *MsgUpdateBoost {
	return &MsgUpdateBoost{
		Creator:  creator,
		RewardId: rewardId,
		BoostId:  boostId,
		Days:     days,
	}
}

func (msg *MsgUpdateBoost) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.RewardId == "" {
		return errorsmod.Wrap(ErrInvalidRewardId, "reward_id should not be empty")
	}

	if msg.BoostId == "" {
		return errorsmod.Wrap(ErrInvalidBoostId, "boost_id should not be empty")
	}

	return validateBoostDays(msg.Days)
}
