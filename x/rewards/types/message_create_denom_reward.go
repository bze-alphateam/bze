package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateDenomReward{}

func NewMsgCreateDenomReward(creator string, denom string) *MsgCreateDenomReward {
	return &MsgCreateDenomReward{
		Creator: creator,
		Denom:   denom,
	}
}

func (msg *MsgCreateDenomReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrap(ErrInvalidStakingDenom, "denom cannot be empty")
	}

	return nil
}
