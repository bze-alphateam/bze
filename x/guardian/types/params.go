package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultParams returns the default parameters: no guardian, so only the
// governance authority can sign the guardian messages.
func DefaultParams() Params {
	return Params{}
}

// NewParams creates a new Params instance
func NewParams(guardian string) Params {
	return Params{Guardian: guardian}
}

// Validate validates the set of params: the guardian is empty or an address.
func (p Params) Validate() error {
	if p.Guardian == "" {
		return nil
	}

	if _, err := sdk.AccAddressFromBech32(p.Guardian); err != nil {
		return errorsmod.Wrapf(err, "invalid guardian address (%s)", p.Guardian)
	}

	return nil
}
