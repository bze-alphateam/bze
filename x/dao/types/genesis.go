package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	seen := make(map[uint64]struct{}, len(gs.Daos))
	for _, dao := range gs.Daos {
		if _, ok := seen[dao.Id]; ok {
			return fmt.Errorf("duplicate dao id %d", dao.Id)
		}
		seen[dao.Id] = struct{}{}

		// I5: ids are handed out by the counter and never reused
		if dao.Id == 0 || dao.Id > gs.DaoCount {
			return fmt.Errorf("dao id %d is outside [1, dao_count %d]", dao.Id, gs.DaoCount)
		}

		if err := validateGenesisDao(dao); err != nil {
			return errorsmod.Wrapf(err, "dao %d", dao.Id)
		}
	}

	return nil
}

// validateGenesisDao checks the stateless rules of an imported DAO record. The
// param bounds (voting periods, member and staking reward caps) are not
// checked: params can change after a DAO was created, and an existing DAO keeps
// its settings.
func validateGenesisDao(dao Dao) error {
	if dao.Address != DaoAddress(dao.Id).String() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "address %s is not the derived address of the dao", dao.Address)
	}

	if dao.Admin == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "admin is not set")
	}

	admin, err := sdk.AccAddressFromBech32(dao.Admin)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}

	// I7: a nomination names someone other than the admin and the DAO itself
	if dao.PendingAdmin != "" {
		pending, err := sdk.AccAddressFromBech32(dao.PendingAdmin)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid pending_admin address (%s)", err)
		}

		if pending.Equals(admin) {
			return errorsmod.Wrap(ErrInvalidNominee, "pending_admin is the admin")
		}

		if pending.String() == dao.Address {
			return errorsmod.Wrap(ErrInvalidNominee, "pending_admin is the dao itself")
		}
	}

	if err := ValidateMetadata(dao.Name, dao.Description); err != nil {
		return err
	}

	if err := dao.Config.Validate(); err != nil {
		return err
	}

	switch dao.Type {
	case DaoType_DAO_TYPE_STATIC_LIST:
		return validateStaticElectorate(dao.Members, dao.Electorate)
	case DaoType_DAO_TYPE_STAKE:
		return validateStakeElectorate(dao.Members, dao.Electorate)
	default:
		return errorsmod.Wrapf(ErrInvalidDaoType, "unknown dao type %d", dao.Type)
	}
}
