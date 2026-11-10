package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdateMembers{}

// NewMsgUpdateMembers builds a MsgUpdateMembers; authority is the DAO's own
// address.
func NewMsgUpdateMembers(authority string, daoId uint64, add, remove []string) *MsgUpdateMembers {
	return &MsgUpdateMembers{Authority: authority, DaoId: daoId, Add: add, Remove: remove}
}

// ValidateBasic runs the checks that need no state: at least one change, valid
// addresses, no duplicates within a list and no address in both. Membership,
// the DAO type and the authority are checked by the msg server.
func (msg *MsgUpdateMembers) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	if msg.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao id must be positive")
	}

	if len(msg.Add) == 0 && len(msg.Remove) == 0 {
		return errorsmod.Wrap(ErrInvalidMembers, "nothing to add or remove")
	}

	_, added, err := uniqueAddresses("add", msg.Add)
	if err != nil {
		return err
	}

	removed, _, err := uniqueAddresses("remove", msg.Remove)
	if err != nil {
		return err
	}

	// walk the list, not the set, so the first overlap in Remove is reported
	for _, addr := range removed {
		if _, ok := added[addr]; ok {
			return errorsmod.Wrapf(ErrInvalidMembers, "%s is both added and removed", addr)
		}
	}

	return nil
}

// uniqueAddresses returns the canonical form of each address, in input order
// and as a set, rejecting invalid ones and duplicates (an address written in
// another case is a duplicate).
func uniqueAddresses(list string, addrs []string) ([]string, map[string]struct{}, error) {
	keys := make([]string, 0, len(addrs))
	seen := make(map[string]struct{}, len(addrs))
	for _, a := range addrs {
		addr, err := sdk.AccAddressFromBech32(a)
		if err != nil {
			return nil, nil, errorsmod.Wrapf(ErrInvalidMembers, "invalid %s address %s (%s)", list, a, err)
		}

		key := addr.String()
		if _, ok := seen[key]; ok {
			return nil, nil, errorsmod.Wrapf(ErrInvalidMembers, "duplicate %s address %s", list, a)
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	return keys, seen, nil
}
