package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgPayPublisherRespect{}

const (
	titleMaxLength = 320
	titleMinLength = 10
	urlMaxLength   = 2048
)

func NewMsgPayPublisherRespect(creator string, address string, amount string) *MsgPayPublisherRespect {
	return &MsgPayPublisherRespect{
		Creator: creator,
		Address: address,
		Amount:  amount,
	}
}

func (msg *MsgPayPublisherRespect) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid publisher address (%s)", err)
	}

	amount, err := sdk.ParseCoinNormalized(msg.Amount)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "invalid amount (%s)", err)
	}

	if !amount.IsPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, "invalid amount: amount should be positive")
	}

	return nil
}
