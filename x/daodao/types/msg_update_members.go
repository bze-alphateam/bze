package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgUpdateMembers{}

// ValidateBasic performs stateless validation of MsgUpdateMembers.
//
// Rules:
//   - authority is valid bech32
//   - dao_id is non-zero (state existence verified later in the keeper)
//   - every `add` entry: address valid bech32, weight > 0, no duplicates
//   - every `remove` entry: address valid bech32, no duplicates
//   - `add` and `remove` are disjoint
//   - at least one of `add` or `remove` is non-empty (a no-op message is
//     pointless and we reject it rather than waste a block)
//
// "STATIC backend only" and "post-update ≥ 1 member" are stateful checks
// enforced in the keeper, not here.
func (m *MsgUpdateMembers) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "authority: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	if len(m.Add) == 0 && len(m.Remove) == 0 {
		return errorsmod.Wrap(ErrInvalidStaticMembers, "msg has no add or remove entries")
	}

	// Validate `add` — note ValidateStaticMembers requires len ≥ 1, so we
	// only invoke it when there's something to add.
	if len(m.Add) > 0 {
		if err := ValidateStaticMembers(m.Add); err != nil {
			return err
		}
	}

	// Validate `remove`: bech32, no duplicates.
	removed := make(map[string]struct{}, len(m.Remove))
	for i, a := range m.Remove {
		addr, err := sdk.AccAddressFromBech32(a)
		if err != nil {
			return errorsmod.Wrapf(ErrInvalidAddress, "remove[%d]: %s", i, err.Error())
		}
		key := addr.String()
		if _, dup := removed[key]; dup {
			return errorsmod.Wrapf(ErrInvalidStaticMembers, "duplicate remove address %s", key)
		}
		removed[key] = struct{}{}
	}

	// Disjointness: nothing in `add` may also be in `remove`.
	for _, mem := range m.Add {
		addr, _ := sdk.AccAddressFromBech32(mem.Address) // already validated
		if _, dup := removed[addr.String()]; dup {
			return errorsmod.Wrapf(ErrInvalidStaticMembers, "address %s appears in both add and remove", addr.String())
		}
	}

	return nil
}
