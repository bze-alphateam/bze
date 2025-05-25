package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgFillOrders{}

func NewMsgFillOrders(creator string, marketId string, orderType string, orders string) *MsgFillOrders {
  return &MsgFillOrders{
		Creator: creator,
    MarketId: marketId,
    OrderType: orderType,
    Orders: orders,
	}
}

func (msg *MsgFillOrders) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Creator)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
  	}
  return nil
}

