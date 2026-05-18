package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgRenounceAdmin{}

// ValidateBasic performs stateless validation of MsgRenounceAdmin.
//
// Rules:
//   - authority is valid bech32.
//   - dao_id is non-zero.
//
// Stateful checks deferred to the keeper:
//   - authority equals the DAO's current admin.
//   - DAO is not already self-administered.
func (m *MsgRenounceAdmin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "authority: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	return nil
}
