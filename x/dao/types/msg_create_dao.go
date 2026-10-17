package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateDao{}

// ValidateBasic runs the checks that need no state. The member and staking
// reward caps, the voting period bounds and the electorate sources depend on
// params or state and are checked by the msg server.
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
		return validateStakeElectorate(msg.Members, msg.Electorate)
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

// validateStakeElectorate requires no member list (D29), unique non-empty
// staking reward ids (D14) and a valid Denom Reward denom when one is set. The
// electorate may be empty: a DAO with an external admin can be set up first.
func validateStakeElectorate(members []string, electorate *StakeElectorate) error {
	if len(members) > 0 {
		return errorsmod.Wrap(ErrInvalidElectorate, "a stake dao cannot have a member list")
	}

	if electorate == nil {
		return nil
	}

	seen := make(map[string]struct{}, len(electorate.SrIds))
	for _, id := range electorate.SrIds {
		if id == "" {
			return errorsmod.Wrap(ErrInvalidElectorate, "empty staking reward id")
		}

		if _, ok := seen[id]; ok {
			return errorsmod.Wrapf(ErrInvalidElectorate, "duplicate staking reward %s", id)
		}
		seen[id] = struct{}{}
	}

	if electorate.DrDenom != "" {
		if err := sdk.ValidateDenom(electorate.DrDenom); err != nil {
			return errorsmod.Wrapf(ErrInvalidElectorate, "invalid denom reward denom %s (%s)", electorate.DrDenom, err)
		}
	}

	return nil
}
