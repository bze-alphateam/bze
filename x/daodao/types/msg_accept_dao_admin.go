package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgAcceptDaoAdmin{}

// ValidateBasic performs stateless validation of MsgAcceptDaoAdmin.
func (m *MsgAcceptDaoAdmin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.NewAdmin); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "new_admin: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	return nil
}
