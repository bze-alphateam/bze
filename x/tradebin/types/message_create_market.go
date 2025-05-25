package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateMarket{}

func NewMsgCreateMarket(creator string, base string, quote string) *MsgCreateMarket {
  return &MsgCreateMarket{
		Creator: creator,
    Base: base,
    Quote: quote,
	}
}

func (msg *MsgCreateMarket) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Creator)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
  	}
  return nil
}

