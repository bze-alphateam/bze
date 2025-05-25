package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgAddLiquidity{}

func NewMsgAddLiquidity(creator string, poolId string, baseAmount string, quoteAmount string, minLpTokens string) *MsgAddLiquidity {
  return &MsgAddLiquidity{
		Creator: creator,
    PoolId: poolId,
    BaseAmount: baseAmount,
    QuoteAmount: quoteAmount,
    MinLpTokens: minLpTokens,
	}
}

func (msg *MsgAddLiquidity) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Creator)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
  	}
  return nil
}

