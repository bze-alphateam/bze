package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgCleanupBoost = "cleanup_boost"
)

var _ sdk.Msg = &MsgCleanupBoost{}

func NewMsgCleanupBoost(creator, rewardId, boostId string, limit uint32) *MsgCleanupBoost {
	return &MsgCleanupBoost{
		Creator:  creator,
		RewardId: rewardId,
		BoostId:  boostId,
		Limit:    limit,
	}
}

// ValidateBasic accepts any limit value: 0 means "use the cleanup_batch_size
// param" and larger values are clamped server-side by that same param.
func (msg *MsgCleanupBoost) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.RewardId == "" {
		return errorsmod.Wrap(ErrInvalidRewardId, "reward_id should not be empty")
	}

	if msg.BoostId == "" {
		return errorsmod.Wrap(ErrInvalidBoostId, "boost_id should not be empty")
	}

	return nil
}
