package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgRemoveLiquidity{}

func NewMsgRemoveLiquidity(creator string, poolId string, lpTokens string, minBase string, minQuote string) *MsgRemoveLiquidity {
  return &MsgRemoveLiquidity{
		Creator: creator,
    PoolId: poolId,
    LpTokens: lpTokens,
    MinBase: minBase,
    MinQuote: minQuote,
	}
}

func (msg *MsgRemoveLiquidity) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Creator)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
  	}
  return nil
}

