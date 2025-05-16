package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgActivateTradingReward{}

func NewMsgActivateTradingReward(creator string, rewardId string) *MsgActivateTradingReward {
  return &MsgActivateTradingReward{
		Creator: creator,
    RewardId: rewardId,
	}
}

func (msg *MsgActivateTradingReward) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Creator)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
  	}
  return nil
}

