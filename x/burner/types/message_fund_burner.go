package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgFundBurner{}

func NewMsgFundBurner(creator string, amount string) *MsgFundBurner {
	return &MsgFundBurner{
		Creator: creator,
		Amount:  amount,
	}
}

func (msg *MsgFundBurner) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	amount, err := sdk.ParseCoinsNormalized(msg.Amount)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid amount (%s)", err)
	}

	if !amount.IsAllPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "amounts must be positive")
	}

	return nil
}
