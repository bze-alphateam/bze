package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const MaxRoutes = 5

var _ sdk.Msg = &MsgMultiSwap{}

func NewMsgMultiSwap(creator string, routes []string, input, minOutput sdk.Coin) *MsgMultiSwap {
	return &MsgMultiSwap{
		Creator:   creator,
		Routes:    routes,
		Input:     input,
		MinOutput: minOutput,
	}
}

func (msg *MsgMultiSwap) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.GetRoutes()) <= 0 || len(msg.GetRoutes()) > MaxRoutes {
		return errorsmod.Wrapf(ErrInvalidRoutes, "routes length must be between 0 and %d", MaxRoutes)
	}

	tempMap := make(map[string]struct{})
	for _, route := range msg.GetRoutes() {
		if _, ok := tempMap[route]; ok {
			return errorsmod.Wrapf(ErrInvalidRoutes, "duplicate route %s", route)
		}
		tempMap[route] = struct{}{}
	}

	if !msg.Input.IsValid() || !msg.Input.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidOrderAmount, "input amount (%s) is not valid", msg.GetInput().String())
	}

	if !msg.MinOutput.IsValid() || !msg.MinOutput.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidOrderAmount, "minimum output (%s) is not valid", msg.GetMinOutput().String())
	}

	return nil
}
