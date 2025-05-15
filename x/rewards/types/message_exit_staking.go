package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgExitStaking{}

func NewMsgExitStaking(creator string, rewardId string) *MsgExitStaking {
  return &MsgExitStaking{
		Creator: creator,
    RewardId: rewardId,
	}
}

func (msg *MsgExitStaking) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Creator)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
  	}
  return nil
}

