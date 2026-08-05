package types

import (
	"fmt"
	"strconv"
	// this line is used by starport scaffolding # genesis/types/import
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate
	if err := gs.validateBoosts(); err != nil {
		return err
	}

	return gs.Params.Validate()
}

// validateBoosts checks the boost genesis lists: ids are globally unique and
// never reused, so a duplicate id, a participant entry pointing at a missing
// boost, or a counter behind the highest id would all corrupt the
// absent-entry rule after import.
func (gs GenesisState) validateBoosts() error {
	seenIds := make(map[string]struct{}, len(gs.BoostList))
	boostKeys := make(map[string]struct{}, len(gs.BoostList))
	var maxId uint64

	for _, boost := range gs.BoostList {
		if _, found := seenIds[boost.Id]; found {
			return fmt.Errorf("duplicate boost id %s", boost.Id)
		}
		seenIds[boost.Id] = struct{}{}
		boostKeys[boost.RewardId+"/"+boost.Id] = struct{}{}

		id, err := strconv.ParseUint(boost.Id, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid boost id %s: %w", boost.Id, err)
		}

		if id > maxId {
			maxId = id
		}
	}

	if gs.BoostCounter < maxId {
		return fmt.Errorf("boost counter %d is behind the highest boost id %d", gs.BoostCounter, maxId)
	}

	for _, participant := range gs.BoostParticipantList {
		if _, found := boostKeys[participant.RewardId+"/"+participant.BoostId]; !found {
			return fmt.Errorf(
				"boost participant %s references missing boost %s on reward %s",
				participant.Address, participant.BoostId, participant.RewardId,
			)
		}
	}

	return nil
}
