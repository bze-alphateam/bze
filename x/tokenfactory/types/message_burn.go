package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgBurn{}

func NewMsgBurn(creator string, coins string) *MsgBurn {
	return &MsgBurn{
		Creator: creator,
		Coins:   coins,
	}
}

func (msg *MsgBurn) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	c, err := sdk.ParseCoinNormalized(msg.Coins)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "invalid coins (%s)", err)
	}

	if !c.IsPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, "cannot burn non positive coins")
	}

	return nil
}
