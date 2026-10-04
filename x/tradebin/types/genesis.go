package types

import (
	"fmt"

	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params: v2types.DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate

	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// one store key per halted denom: valid denoms, never the native denom, no duplicates, no cap
	if err := ValidateHaltedDenoms(gs.HaltedDenoms, 0, gs.Params.NativeDenom); err != nil {
		return fmt.Errorf("invalid halted_denoms: %w", err)
	}

	return nil
}
