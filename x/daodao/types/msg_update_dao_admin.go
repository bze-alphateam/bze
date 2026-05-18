package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgUpdateDaoAdmin{}

// ValidateBasic performs stateless validation of MsgUpdateDaoAdmin (nominate).
func (m *MsgUpdateDaoAdmin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "authority: %s", err.Error())
	}
	if _, err := sdk.AccAddressFromBech32(m.NewAdmin); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "new_admin: %s", err.Error())
	}
	if m.Authority == m.NewAdmin {
		return errorsmod.Wrap(ErrInvalidAddress, "new_admin must differ from current authority")
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	return nil
}
