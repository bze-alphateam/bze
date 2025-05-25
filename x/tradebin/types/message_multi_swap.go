package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgMultiSwap{}

func NewMsgMultiSwap(creator string, routes string, input string, minOutput string) *MsgMultiSwap {
  return &MsgMultiSwap{
		Creator: creator,
    Routes: routes,
    Input: input,
    MinOutput: minOutput,
	}
}

func (msg *MsgMultiSwap) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Creator)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
  	}
  return nil
}

