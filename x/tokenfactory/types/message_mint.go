package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgMint{}

func NewMsgMint(creator string, coins sdk.Coin) *MsgMint {
	return &MsgMint{
		Creator: creator,
		Coins:   coins,
	}
}

func (msg *MsgMint) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if !msg.Coins.IsValid() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "invalid coins (%s)", msg.Coins)
	}

	if !msg.Coins.IsPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, "cannot mint non positive coins")
	}

	return nil
}
