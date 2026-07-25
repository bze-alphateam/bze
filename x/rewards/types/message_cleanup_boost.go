package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCleanupBoost{}

func NewMsgCleanupBoost(creator, rewardId string, limit uint32) *MsgCleanupBoost {
	return &MsgCleanupBoost{
		Creator:  creator,
		RewardId: rewardId,
		Limit:    limit,
	}
}

func (msg *MsgCleanupBoost) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.RewardId == "" {
		return errorsmod.Wrap(ErrInvalidRewardId, "reward_id should not be empty")
	}

	// limit is clamped server-side (1..200) in the cleanup handler; a zero or
	// oversized value is a valid message here.

	return nil
}
