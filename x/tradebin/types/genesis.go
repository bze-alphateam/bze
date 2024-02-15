package types

import (
	"fmt"
)

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		MarketList: []Market{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in market
	marketIndexMap := make(map[string]struct{})

	for _, elem := range gs.MarketList {
		index := string(MarketKey(elem.Asset1, elem.Asset2))
		if _, ok := marketIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for market")
		}
		marketIndexMap[index] = struct{}{}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
