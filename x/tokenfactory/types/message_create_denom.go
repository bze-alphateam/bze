package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strings"
)

var _ sdk.Msg = &MsgCreateDenom{}

func NewMsgCreateDenom(creator string, subdenom string) *MsgCreateDenom {
	return &MsgCreateDenom{
		Creator:  creator,
		Subdenom: subdenom,
	}
}

func (msg *MsgCreateDenom) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Subdenom == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "subdenom must not be empty")
	}

	if strings.Contains(msg.Subdenom, "_") {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "subdenom should not contain _")
	}

	return nil
}
