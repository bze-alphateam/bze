package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgAcceptAdminNomination{}

// NewMsgAcceptAdminNomination builds a MsgAcceptAdminNomination.
func NewMsgAcceptAdminNomination(nominee string, daoId uint64) *MsgAcceptAdminNomination {
	return &MsgAcceptAdminNomination{Nominee: nominee, DaoId: daoId}
}

// ValidateBasic runs the checks that need no state. Whether the signer is the
// pending admin is checked by the msg server.
func (msg *MsgAcceptAdminNomination) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Nominee); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid nominee address (%s)", err)
	}

	if msg.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao id must be positive")
	}

	return nil
}
