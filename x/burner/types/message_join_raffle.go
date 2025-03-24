package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgJoinRaffle = "join_raffle"
const maxAllowedTickets = 50

var _ sdk.Msg = &MsgJoinRaffle{}

func NewMsgJoinRaffle(creator string, denom string) *MsgJoinRaffle {
	return &MsgJoinRaffle{
		Creator: creator,
		Denom:   denom,
	}
}

func (msg *MsgJoinRaffle) Route() string {
	return RouterKey
}

func (msg *MsgJoinRaffle) Type() string {
	return TypeMsgJoinRaffle
}

func (msg *MsgJoinRaffle) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgJoinRaffle) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgJoinRaffle) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Denom == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "missing denom")
	}

	if msg.GetTickets() < 1 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "missing tickets")
	}

	if msg.GetTickets() > maxAllowedTickets {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "can not buy more than %d tickets", maxAllowedTickets)
	}

	return nil
}
