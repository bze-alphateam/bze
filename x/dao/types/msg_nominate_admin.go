package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgNominateAdmin{}

// NewMsgNominateAdmin builds a MsgNominateAdmin; an empty nominee hands the
// DAO to its voters.
func NewMsgNominateAdmin(authority string, daoId uint64, nominee string) *MsgNominateAdmin {
	return &MsgNominateAdmin{Authority: authority, DaoId: daoId, Nominee: nominee}
}

// ValidateBasic runs the checks that need no state. Who the current admin is,
// and whether the nominee differs from it and from the DAO, is checked by the
// msg server.
func (msg *MsgNominateAdmin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	if msg.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao id must be positive")
	}

	if msg.Nominee != "" {
		if _, err := sdk.AccAddressFromBech32(msg.Nominee); err != nil {
			return errorsmod.Wrapf(ErrInvalidNominee, "invalid nominee address (%s)", err)
		}
	}

	return nil
}
