package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateDao{}

// ValidateBasic runs the checks that need no state. The member cap and the
// voting period bounds depend on the params and are checked by the msg server.
func (msg *MsgCreateDao) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Admin != "" {
		if _, err := sdk.AccAddressFromBech32(msg.Admin); err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
		}
	}

	if err := ValidateMetadata(msg.Name, msg.Description); err != nil {
		return err
	}

	if err := msg.Config.Validate(); err != nil {
		return err
	}

	switch msg.Type {
	case DaoType_DAO_TYPE_STATIC_LIST:
		return validateStaticElectorate(msg.Members, msg.Electorate)
	case DaoType_DAO_TYPE_STAKE:
		return errorsmod.Wrap(ErrInvalidDaoType, "stake DAOs not supported yet")
	default:
		return errorsmod.Wrapf(ErrInvalidDaoType, "unknown dao type %d", msg.Type)
	}
}

// validateStaticElectorate requires a non-empty list of unique valid addresses
// and no stake electorate: the two kinds are never mixed (D29).
func validateStaticElectorate(members []string, electorate *StakeElectorate) error {
	if !electorate.IsEmpty() {
		return errorsmod.Wrap(ErrInvalidElectorate, "a static-list dao cannot have a stake electorate")
	}

	if len(members) == 0 {
		return errorsmod.Wrap(ErrInvalidMembers, "a static-list dao needs at least one member")
	}

	seen := make(map[string]struct{}, len(members))
	for _, member := range members {
		addr, err := sdk.AccAddressFromBech32(member)
		if err != nil {
			return errorsmod.Wrapf(ErrInvalidMembers, "invalid member address %s (%s)", member, err)
		}

		// compare canonical forms: the same address may be written in another case
		key := addr.String()
		if _, ok := seen[key]; ok {
			return errorsmod.Wrapf(ErrInvalidMembers, "duplicate member %s", member)
		}
		seen[key] = struct{}{}
	}

	return nil
}
