package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateLiquidityPool{}

func NewMsgCreateLiquidityPool(creator string, base string, quote string, fee string, feeDest string, stable bool, initialBase string, initialQuote string) *MsgCreateLiquidityPool {
  return &MsgCreateLiquidityPool{
		Creator: creator,
    Base: base,
    Quote: quote,
    Fee: fee,
    FeeDest: feeDest,
    Stable: stable,
    InitialBase: initialBase,
    InitialQuote: initialQuote,
	}
}

func (msg *MsgCreateLiquidityPool) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Creator)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
  	}
  return nil
}

