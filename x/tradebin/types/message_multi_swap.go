package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgMultiSwap = "multi_swap"
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

func (msg *MsgMultiSwap) Route() string {
	return RouterKey
}

func (msg *MsgMultiSwap) Type() string {
	return TypeMsgMultiSwap
}

func (msg *MsgMultiSwap) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgMultiSwap) GetCreatorAcc() sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil
	}

	return creator
}

func (msg *MsgMultiSwap) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic - validates basic fields of the multi swap msg
// DO NOT CHANGE input coin and min output coin validation
func (msg *MsgMultiSwap) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.GetRoutes()) <= 0 || len(msg.GetRoutes()) > MaxRoutes {
		return errors.Wrapf(ErrInvalidRoutes, "routes length must be between 0 and %d", MaxRoutes)
	}

	if !msg.Input.IsValid() && !msg.Input.IsPositive() {
		return errors.Wrapf(ErrInvalidOrderAmount, "input amount (%s) is not valid", msg.GetInput().String())
	}

	if !msg.MinOutput.IsValid() || !msg.MinOutput.IsPositive() {
		return errors.Wrapf(ErrInvalidOrderAmount, "minimum output (%s) is not valid", msg.GetMinOutput().String())
	}

	//make sure to validate the input coin and output min coin
	ic := msg.GetInput()
	if !ic.IsPositive() {
		return errors.Wrapf(ErrInvalidOrderAmount, "input is not positive (%s)", ic.String())
	}

	moc := msg.GetMinOutput()
	if !moc.IsPositive() {
		return errors.Wrapf(ErrInvalidOrderAmount, "minimum output is not positive (%s)", moc.String())
	}

	return nil
}
