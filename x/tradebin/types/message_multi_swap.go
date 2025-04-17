package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgMultiSwap = "multi_swap"
const MaxRoutes = 5

var _ sdk.Msg = &MsgMultiSwap{}

func NewMsgMultiSwap(creator string, routes []string, input string, minOutput string) *MsgMultiSwap {
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

	if len(msg.GetInput()) <= 0 {
		return errors.Wrapf(ErrInvalidOrderAmount, "invalid input amount (%s)", msg.GetInput())
	}

	if len(msg.GetMinOutput()) <= 0 {
		return errors.Wrapf(ErrInvalidOrderAmount, "invalid minimum output (%s)", msg.GetMinOutput())
	}

	//make sure to validate the input coin and output min coin
	ic, err := msg.GetInputCoin()
	if err != nil {
		return errors.Wrapf(ErrInvalidOrderAmount, "invalid input (%s)", err)
	}

	if !ic.IsPositive() {
		return errors.Wrapf(ErrInvalidOrderAmount, "input is not positive (%s)", ic.String())
	}

	moc, err := msg.GetMinOutputCoin()
	if err != nil {
		return errors.Wrapf(ErrInvalidOrderAmount, "invalid minimum output (%s)", err)
	}

	if !moc.IsPositive() {
		return errors.Wrapf(ErrInvalidOrderAmount, "minimum output is not positive (%s)", moc.String())
	}

	return nil
}

func (msg *MsgMultiSwap) GetInputCoin() (c sdk.Coin, e error) {
	c, e = sdk.ParseCoinNormalized(msg.GetInput())

	return c, e
}

func (msg *MsgMultiSwap) GetMinOutputCoin() (c sdk.Coin, e error) {
	c, e = sdk.ParseCoinNormalized(msg.GetMinOutput())

	return c, e
}
