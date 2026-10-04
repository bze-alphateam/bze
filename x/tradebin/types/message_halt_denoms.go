package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgHaltDenoms{}
	_ sdk.Msg = &MsgUnhaltDenoms{}
)

func NewMsgHaltDenoms(authority string, denoms []string) *MsgHaltDenoms {
	return &MsgHaltDenoms{Authority: authority, Denoms: denoms}
}

func NewMsgUnhaltDenoms(authority string, denoms []string) *MsgUnhaltDenoms {
	return &MsgUnhaltDenoms{Authority: authority, Denoms: denoms}
}

// ValidateBasic does a sanity check on the provided data.
func (m *MsgHaltDenoms) ValidateBasic() error {
	return m.Validate()
}

// Validate validates the MsgHaltDenoms request.
func (m *MsgHaltDenoms) Validate() error {
	return validateHaltMessage(m.Authority, m.Denoms)
}

// ValidateBasic does a sanity check on the provided data.
func (m *MsgUnhaltDenoms) ValidateBasic() error {
	return m.Validate()
}

// Validate validates the MsgUnhaltDenoms request.
func (m *MsgUnhaltDenoms) Validate() error {
	return validateHaltMessage(m.Authority, m.Denoms)
}

// validateHaltMessage is the stateless part shared by both messages: a bech32 authority and a non-empty,
// duplicate-free list of valid denoms, at most MaxDenomsPerHaltMsg long. The native denom check needs
// the params and lives in the msg server.
func validateHaltMessage(authority string, denoms []string) error {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if len(denoms) == 0 {
		return errorsmod.Wrap(ErrInvalidDenom, "no denoms provided")
	}

	if err := ValidateHaltedDenoms(denoms, MaxDenomsPerHaltMsg, ""); err != nil {
		return errorsmod.Wrap(ErrInvalidDenom, err.Error())
	}

	return nil
}
