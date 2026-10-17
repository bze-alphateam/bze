package types

import (
	"fmt"
	"slices"

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

	seen := make(map[uint64]Dao, len(gs.Daos))
	for _, dao := range gs.Daos {
		if _, ok := seen[dao.Id]; ok {
			return fmt.Errorf("duplicate dao id %d", dao.Id)
		}
		seen[dao.Id] = dao

		// ids are handed out by the counter and never reused
		if dao.Id == 0 || dao.Id > gs.DaoCount {
			return fmt.Errorf("I5: dao id %d is outside [1, dao_count %d]", dao.Id, gs.DaoCount)
		}

		if err := validateGenesisDao(dao); err != nil {
			return errorsmod.Wrapf(err, "dao %d", dao.Id)
		}
	}

	proposals := make(map[uint64]Proposal, len(gs.Proposals))
	for _, proposal := range gs.Proposals {
		if _, ok := proposals[proposal.Id]; ok {
			return fmt.Errorf("duplicate proposal id %d", proposal.Id)
		}
		proposals[proposal.Id] = proposal

		if proposal.Id == 0 || proposal.Id > gs.ProposalCount {
			return fmt.Errorf("I5: proposal id %d is outside [1, proposal_count %d]", proposal.Id, gs.ProposalCount)
		}

		// DAOs are never deleted, so every proposal's DAO is in the genesis
		if _, ok := seen[proposal.DaoId]; !ok {
			return fmt.Errorf("proposal %d belongs to unknown dao %d", proposal.Id, proposal.DaoId)
		}

		if err := validateGenesisProposal(proposal); err != nil {
			return errorsmod.Wrapf(err, "proposal %d", proposal.Id)
		}
	}

	ballots, err := gs.validateBallots(seen, proposals)
	if err != nil {
		return err
	}

	if err := gs.validateVoteIndex(proposals, ballots); err != nil {
		return err
	}

	if err := gs.validateJoinRecords(); err != nil {
		return err
	}

	return gs.validatePins()
}

// ballotKey identifies a ballot: its proposal and voter.
type ballotKey struct {
	proposalId uint64
	voter      string
}

// validateBallots checks every ballot (I3) and the tallies of the proposals
// in voting against them (I2), and returns the ballots by key.
func (gs GenesisState) validateBallots(daos map[uint64]Dao, proposals map[uint64]Proposal) (map[ballotKey]Ballot, error) {
	ballots := make(map[ballotKey]Ballot, len(gs.Ballots))
	tallies := make(map[uint64]map[VoteOption]math.Int)
	for _, ballot := range gs.Ballots {
		key := ballotKey{proposalId: ballot.ProposalId, voter: ballot.Voter}
		if _, ok := ballots[key]; ok {
			return nil, fmt.Errorf("duplicate ballot of %s on proposal %d", ballot.Voter, ballot.ProposalId)
		}
		ballots[key] = ballot

		// a cleanup deletes every ballot before the proposal
		proposal, ok := proposals[ballot.ProposalId]
		if !ok {
			return nil, fmt.Errorf("ballot of %s on unknown proposal %d", ballot.Voter, ballot.ProposalId)
		}

		if err := validateGenesisBallot(ballot, daos[proposal.DaoId].Type, proposal.PinnedElectorate); err != nil {
			return nil, errorsmod.Wrapf(err, "ballot of %s on proposal %d", ballot.Voter, ballot.ProposalId)
		}

		if tallies[ballot.ProposalId] == nil {
			tallies[ballot.ProposalId] = map[VoteOption]math.Int{VoteOption_YES: math.ZeroInt(), VoteOption_NO: math.ZeroInt(), VoteOption_ABSTAIN: math.ZeroInt()}
		}
		tallies[ballot.ProposalId][ballot.Option] = tallies[ballot.ProposalId][ballot.Option].Add(ballot.Weight)
	}

	// I2: a finished proposal's tally is its final result, kept as it was
	for _, proposal := range gs.Proposals {
		if proposal.Status != ProposalStatus_VOTING {
			continue
		}

		tally := tallies[proposal.Id]
		for _, option := range []struct {
			option VoteOption
			value  math.Int
		}{{VoteOption_YES, proposal.Yes}, {VoteOption_NO, proposal.No}, {VoteOption_ABSTAIN, proposal.Abstain}} {
			counted := math.ZeroInt()
			if tally != nil {
				counted = tally[option.option]
			}

			if !option.value.Equal(counted) {
				return nil, fmt.Errorf("I2: proposal %d has %s %s, its ballots add up to %s", proposal.Id, option.option, option.value, counted)
			}
		}
	}

	return ballots, nil
}

// validateGenesisBallot checks one ballot: a static-list ballot weighs 1, or 0
// once its voter left the list, and has no shares; a stake ballot weighs the
// sum of its shares, one per source of the proposal's pinned electorate (I3).
func validateGenesisBallot(ballot Ballot, daoType DaoType, pinned *StakeElectorate) error {
	if _, err := sdk.AccAddressFromBech32(ballot.Voter); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid voter address (%s)", err)
	}

	if _, ok := VoteOption_name[int32(ballot.Option)]; !ok {
		return fmt.Errorf("unknown option %d", ballot.Option)
	}

	if ballot.Weight.IsNil() || ballot.Weight.IsNegative() {
		return fmt.Errorf("weight must be a non-negative integer")
	}

	if daoType == DaoType_DAO_TYPE_STATIC_LIST {
		if len(ballot.Shares) > 0 {
			return fmt.Errorf("I3: a static-list ballot has no shares")
		}

		if ballot.Weight.GT(math.OneInt()) {
			return fmt.Errorf("I3: a static-list ballot weighs at most 1, got %s", ballot.Weight)
		}

		return nil
	}

	pinnedSources := make(map[string]struct{})
	for _, source := range pinned.Sources() {
		pinnedSources[string(source)] = struct{}{}
	}

	sum := math.ZeroInt()
	seen := make(map[string]struct{}, len(ballot.Shares))
	for _, share := range ballot.Shares {
		if _, ok := pinnedSources[share.Source]; !ok {
			return fmt.Errorf("I3: share of %s, a source the proposal did not pin", share.Source)
		}

		if _, ok := seen[share.Source]; ok {
			return fmt.Errorf("I3: two shares of %s", share.Source)
		}
		seen[share.Source] = struct{}{}

		if share.Amount.IsNil() || !share.Amount.IsPositive() {
			return fmt.Errorf("I3: share of %s must be positive", share.Source)
		}

		sum = sum.Add(share.Amount)
	}

	if !sum.Equal(ballot.Weight) {
		return fmt.Errorf("I3: weight %s is not the sum of its shares %s", ballot.Weight, sum)
	}

	return nil
}

// validateVoteIndex checks I6 on the vote index: every share of a ballot on a
// proposal in voting has its entry, so an Exit finds it (D2), and every entry
// of a proposal in voting names such a share. Entries of a proposal that left
// voting, or is cleaned up, are leftovers the next Exit of the address prunes.
func (gs GenesisState) validateVoteIndex(proposals map[uint64]Proposal, ballots map[ballotKey]Ballot) error {
	type entryKey struct {
		source     string
		address    string
		proposalId uint64
	}

	entries := make(map[entryKey]struct{}, len(gs.VoteIndex))
	for _, entry := range gs.VoteIndex {
		key := entryKey{source: entry.Source, address: entry.Address, proposalId: entry.ProposalId}
		if _, ok := entries[key]; ok {
			return fmt.Errorf("duplicate vote-index entry %s/%s/%d", entry.Source, entry.Address, entry.ProposalId)
		}
		entries[key] = struct{}{}

		if err := ValidateSource(entry.Source); err != nil {
			return errorsmod.Wrapf(err, "vote-index entry of %s on proposal %d", entry.Address, entry.ProposalId)
		}

		if _, err := sdk.AccAddressFromBech32(entry.Address); err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "vote-index entry of proposal %d: invalid address (%s)", entry.ProposalId, err)
		}

		if entry.ProposalId == 0 || entry.ProposalId > gs.ProposalCount {
			return fmt.Errorf("I5: vote-index entry of proposal %d, outside [1, proposal_count %d]", entry.ProposalId, gs.ProposalCount)
		}

		if proposal, ok := proposals[entry.ProposalId]; !ok || proposal.Status != ProposalStatus_VOTING {
			continue
		}

		ballot, ok := ballots[ballotKey{proposalId: entry.ProposalId, voter: entry.Address}]
		if !ok || !slices.ContainsFunc(ballot.Shares, func(s BallotShare) bool { return s.Source == entry.Source }) {
			return fmt.Errorf("I6: vote-index entry %s/%s/%d names no share of a ballot", entry.Source, entry.Address, entry.ProposalId)
		}
	}

	for _, ballot := range gs.Ballots {
		if proposals[ballot.ProposalId].Status != ProposalStatus_VOTING {
			continue
		}

		for _, share := range ballot.Shares {
			if _, ok := entries[entryKey{source: share.Source, address: ballot.Voter, proposalId: ballot.ProposalId}]; !ok {
				return fmt.Errorf("I6: the %s share of %s on proposal %d has no vote-index entry", share.Source, ballot.Voter, ballot.ProposalId)
			}
		}
	}

	return nil
}

// validateJoinRecords checks the join records. A record may outlive its
// source's pins: unpinning keeps the records, and an old join height is below
// the submit height of any later proposal, so it never makes stake ineligible.
func (gs GenesisState) validateJoinRecords() error {
	type joinKey struct {
		source  string
		address string
	}

	seen := make(map[joinKey]struct{}, len(gs.JoinRecords))
	for _, record := range gs.JoinRecords {
		key := joinKey{source: record.Source, address: record.Address}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate join record of %s on %s", record.Address, record.Source)
		}
		seen[key] = struct{}{}

		if err := ValidateSource(record.Source); err != nil {
			return errorsmod.Wrapf(err, "join record of %s", record.Address)
		}

		if _, err := sdk.AccAddressFromBech32(record.Address); err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "join record on %s: invalid address (%s)", record.Source, err)
		}

		if record.Height < 0 {
			return fmt.Errorf("join record of %s on %s has a negative height", record.Address, record.Source)
		}
	}

	return nil
}

// validatePins checks I4: each source's pin count is the number of DAOs whose
// electorate holds it plus the number of proposals in voting that pinned it.
func (gs GenesisState) validatePins() error {
	want := make(map[string]uint64)
	for _, dao := range gs.Daos {
		for _, source := range dao.Electorate.Sources() {
			want[string(source)]++
		}
	}

	for _, proposal := range gs.Proposals {
		if proposal.Status != ProposalStatus_VOTING {
			continue
		}

		for _, source := range proposal.PinnedElectorate.Sources() {
			want[string(source)]++
		}
	}

	seen := make(map[string]struct{}, len(gs.Pins))
	for _, pin := range gs.Pins {
		if _, ok := seen[pin.Source]; ok {
			return fmt.Errorf("duplicate pin count of %s", pin.Source)
		}
		seen[pin.Source] = struct{}{}

		if pin.Count != want[pin.Source] {
			return fmt.Errorf("I4: %s has %d pins, the daos and proposals in voting use it %d times", pin.Source, pin.Count, want[pin.Source])
		}
	}

	for source, count := range want {
		if _, ok := seen[source]; !ok {
			return fmt.Errorf("I4: %s has no pins, the daos and proposals in voting use it %d times", source, count)
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
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "I7: admin is not set")
	}

	admin, err := sdk.AccAddressFromBech32(dao.Admin)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "I7: invalid admin address (%s)", err)
	}

	// a nomination names someone other than the admin and the DAO itself
	if dao.PendingAdmin != "" {
		pending, err := sdk.AccAddressFromBech32(dao.PendingAdmin)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "I7: invalid pending_admin address (%s)", err)
		}

		if pending.Equals(admin) {
			return errorsmod.Wrap(ErrInvalidNominee, "I7: pending_admin is the admin")
		}

		if pending.String() == dao.Address {
			return errorsmod.Wrap(ErrInvalidNominee, "I7: pending_admin is the dao itself")
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
