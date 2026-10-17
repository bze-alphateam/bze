package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdateDaoConfig{}

// NewMsgUpdateDaoConfig builds a MsgUpdateDaoConfig; authority is the DAO's
// own address.
func NewMsgUpdateDaoConfig(authority string, daoId uint64, config DaoConfig) *MsgUpdateDaoConfig {
	return &MsgUpdateDaoConfig{Authority: authority, DaoId: daoId, Config: config}
}

// ValidateBasic runs the checks that need no state. Whether the authority is
// the DAO's address and the periods are within the params is checked by the
// msg server.
func (msg *MsgUpdateDaoConfig) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	if msg.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao id must be positive")
	}

	return msg.Config.Validate()
}
