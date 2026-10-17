package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ codectypes.UnpackInterfacesMessage = GenesisState{}

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

	seenProposals := make(map[uint64]struct{}, len(gs.Proposals))
	for _, proposal := range gs.Proposals {
		if _, ok := seenProposals[proposal.Id]; ok {
			return fmt.Errorf("duplicate proposal id %d", proposal.Id)
		}
		seenProposals[proposal.Id] = struct{}{}

		// I5
		if proposal.Id == 0 || proposal.Id > gs.ProposalCount {
			return fmt.Errorf("proposal id %d is outside [1, proposal_count %d]", proposal.Id, gs.ProposalCount)
		}

		// DAOs are never deleted, so every proposal's DAO is in the genesis
		if _, ok := seen[proposal.DaoId]; !ok {
			return fmt.Errorf("proposal %d belongs to unknown dao %d", proposal.Id, proposal.DaoId)
		}

		if err := validateGenesisProposal(proposal); err != nil {
			return errorsmod.Wrapf(err, "proposal %d", proposal.Id)
		}
	}

	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage, so the messages of the
// imported proposals are resolved.
func (gs GenesisState) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	for _, proposal := range gs.Proposals {
		if err := proposal.UnpackInterfaces(unpacker); err != nil {
			return err
		}
	}

	return nil
}

// validateGenesisProposal checks the stateless rules of an imported proposal
// record. The I1 escrow balance is checked by InitGenesis, against x/bank.
func validateGenesisProposal(proposal Proposal) error {
	if _, ok := ProposalStatus_name[int32(proposal.Status)]; !ok {
		return fmt.Errorf("unknown status %d", proposal.Status)
	}

	// a refund sends the deposit back to the proposer
	if _, err := sdk.AccAddressFromBech32(proposal.Proposer); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid proposer address (%s)", err)
	}

	tallies := []struct {
		name  string
		value math.Int
	}{
		{"yes", proposal.Yes},
		{"no", proposal.No},
		{"abstain", proposal.Abstain},
		{"pinned_denominator", proposal.PinnedDenominator},
	}
	for _, tally := range tallies {
		if tally.value.IsNil() || tally.value.IsNegative() {
			return fmt.Errorf("%s must be a non-negative integer", tally.name)
		}
	}

	if proposal.PinnedDeposit != nil {
		if err := proposal.PinnedDeposit.Validate(); err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "invalid pinned_deposit (%s)", err)
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
