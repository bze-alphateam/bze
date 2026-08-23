package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgClaimDenomRewards{}

func NewMsgClaimDenomRewards(creator string, denom string) *MsgClaimDenomRewards {
	return &MsgClaimDenomRewards{
		Creator: creator,
		Denom:   denom,
	}
}

func (msg *MsgClaimDenomRewards) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrap(ErrInvalidStakingDenom, "denom cannot be empty")
	}

	if err := sdk.ValidateDenom(msg.Denom); err != nil {
		return errorsmod.Wrap(ErrInvalidStakingDenom, err.Error())
	}

	return nil
}
