package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgDistributeDenomRewards{}

func NewMsgDistributeDenomRewards(creator string, denom string, prizeDenom string, amount math.Int) *MsgDistributeDenomRewards {
	return &MsgDistributeDenomRewards{
		Creator:    creator,
		Denom:      denom,
		PrizeDenom: prizeDenom,
		Amount:     amount,
	}
}

func (msg *MsgDistributeDenomRewards) ValidateBasic() error {
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

	if msg.PrizeDenom == "" {
		return errorsmod.Wrap(ErrInvalidPrizeDenom, "prize_denom cannot be empty")
	}

	if err := sdk.ValidateDenom(msg.PrizeDenom); err != nil {
		return errorsmod.Wrap(ErrInvalidPrizeDenom, err.Error())
	}

	if msg.Amount.IsNil() || !msg.Amount.IsPositive() {
		return errorsmod.Wrap(ErrInvalidAmount, "amount should be greater than 0")
	}

	return nil
}
