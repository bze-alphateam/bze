package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgExitDenomReward{}

func NewMsgExitDenomReward(creator string, denom string) *MsgExitDenomReward {
	return &MsgExitDenomReward{
		Creator: creator,
		Denom:   denom,
	}
}

func (msg *MsgExitDenomReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrap(ErrInvalidStakingDenom, "denom cannot be empty")
	}

	return nil
}
