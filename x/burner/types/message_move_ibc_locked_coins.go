package types

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/bze-alphateam/bze/bzeutils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgMoveIbcLockedCoins{}

func NewMsgMoveIbcLockedCoins(creator string, denom string) *MsgMoveIbcLockedCoins {
	return &MsgMoveIbcLockedCoins{
		Creator: creator,
		Denom:   denom,
	}
}

func (msg *MsgMoveIbcLockedCoins) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "missing denom")
	}

	//LP Shares CANNOT be moved
	//we add this here in case we're ever thinking about removing the check if it's IBC denom
	if bzeutils.IsLpTokenDenom(msg.Denom) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "cannot move LP shares")
	}

	//for now we allow only ibc coins to be moved from lock account
	if !bzeutils.IsIBCDenom(msg.Denom) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "only ibc denoms are allowed")
	}

	return nil
}
