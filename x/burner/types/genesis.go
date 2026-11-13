package types

import "fmt"

// this line is used by starport scaffolding # genesis/types/import

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
	seen := make(map[string]struct{}, len(gs.IbcMoveLocks))
	for _, l := range gs.IbcMoveLocks {
		if l.Denom == "" {
			return fmt.Errorf("ibc move lock with empty denom")
		}

		if l.LastMoveHeight < 0 {
			return fmt.Errorf("ibc move lock of %s has a negative height: %d", l.Denom, l.LastMoveHeight)
		}

		if _, ok := seen[l.Denom]; ok {
			return fmt.Errorf("duplicate ibc move lock for denom %s", l.Denom)
		}
		seen[l.Denom] = struct{}{}
	}

	return gs.Params.Validate()
}
