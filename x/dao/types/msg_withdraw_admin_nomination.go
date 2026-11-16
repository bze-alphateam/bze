package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgWithdrawAdminNomination{}

// NewMsgWithdrawAdminNomination builds a MsgWithdrawAdminNomination.
func NewMsgWithdrawAdminNomination(authority string, daoId uint64) *MsgWithdrawAdminNomination {
	return &MsgWithdrawAdminNomination{Authority: authority, DaoId: daoId}
}

// ValidateBasic runs the checks that need no state. Whether the signer is the
// current admin, and whether a nomination is pending, is checked by the msg
// server.
func (msg *MsgWithdrawAdminNomination) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	if msg.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao id must be positive")
	}

	return nil
}
