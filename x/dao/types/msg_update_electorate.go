package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdateElectorate{}

// NewMsgUpdateElectorate builds a MsgUpdateElectorate; authority is the DAO's
// own address.
func NewMsgUpdateElectorate(authority string, daoId uint64, electorate *StakeElectorate) *MsgUpdateElectorate {
	return &MsgUpdateElectorate{Authority: authority, DaoId: daoId, Electorate: electorate}
}

// ValidateBasic runs the checks that need no state: unique non-empty staking
// reward ids (D14) and a valid Denom Reward denom. The DAO type, the authority
// and the sources themselves are checked by the msg server.
func (msg *MsgUpdateElectorate) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	if msg.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao id must be positive")
	}

	return validateStakeElectorate(nil, msg.Electorate)
}
