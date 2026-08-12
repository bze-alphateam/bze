package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgJoinDenomReward{}

func NewMsgJoinDenomReward(creator string, denom string, amount string) *MsgJoinDenomReward {
	return &MsgJoinDenomReward{
		Creator: creator,
		Denom:   denom,
		Amount:  amount,
	}
}

func (msg *MsgJoinDenomReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrap(ErrInvalidStakingDenom, "denom cannot be empty")
	}

	if msg.Amount == "" {
		return errorsmod.Wrap(ErrInvalidAmount, "empty amount provided")
	}

	amtInt, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return errorsmod.Wrap(ErrInvalidAmount, "could not convert amount")
	}
	if !amtInt.IsPositive() {
		return errorsmod.Wrap(ErrInvalidAmount, "amount should be greater than 0")
	}

	return nil
}
