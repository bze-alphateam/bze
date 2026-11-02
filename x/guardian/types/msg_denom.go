package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgBlockDenom{}
	_ sdk.Msg = &MsgUnblockDenom{}
)

// ValidateBasic does a sanity check on the provided data. The keeper runs it
// too: a message executed by x/gov or x/authz does not pass through the
// transaction decoder.
func (m *MsgBlockDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errorsmod.Wrap(err, "invalid signer address")
	}

	return validateDenom(m.Denom)
}

// ValidateBasic does a sanity check on the provided data, see MsgBlockDenom.
func (m *MsgUnblockDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return validateDenom(m.Denom)
}

func validateDenom(denom string) error {
	if err := sdk.ValidateDenom(denom); err != nil {
		return errorsmod.Wrap(ErrInvalidDenom, err.Error())
	}

	return nil
}
