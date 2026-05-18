package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgUpdateDepositConfig{}

// ValidateBasic performs stateless validation of MsgUpdateDepositConfig.
//
// Rules:
//   - authority is valid bech32.
//   - dao_id is non-zero.
//   - deposit passes the stateless brick-prevention caps
//     (ValidateDepositConfigStateless).
//
// Stateful checks deferred to the keeper:
//   - authority equals the DAO's admin.
//   - deposit_period <= Params.max_deposit_period.
func (m *MsgUpdateDepositConfig) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "authority: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	return ValidateDepositConfigStateless(m.Deposit)
}
