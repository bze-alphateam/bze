package types

import (
	"errors"
	"time"
)

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	epochs := []EpochInfo{
		NewGenesisEpochInfo("day", time.Hour*24), // alphabetical order
		NewGenesisEpochInfo("hour", time.Hour),
		NewGenesisEpochInfo("week", time.Hour*24*7),
	}

	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
		Epochs: epochs,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate

	epochIdentifiers := map[string]bool{}
	for _, epoch := range gs.Epochs {
		if err := epoch.Validate(); err != nil {
			return err
		}

		if epochIdentifiers[epoch.Identifier] {
			return errors.New("epoch identifier should be unique")
		}
		
		epochIdentifiers[epoch.Identifier] = true
	}

	return gs.Params.Validate()
}
