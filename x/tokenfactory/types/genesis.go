package types

import errorsmod "cosmossdk.io/errors"

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params:        DefaultParams(),
		FactoryDenoms: nil,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate
	for _, genDenom := range gs.FactoryDenoms {
		if genDenom.DenomConfig == nil {
			continue
		}
		if err := genDenom.DenomConfig.Validate(); err != nil {
			return errorsmod.Wrapf(err, "denom %s", genDenom.Denom)
		}
	}

	seenBrandingDenoms := make(map[string]struct{})
	for _, record := range gs.DenomBrandings {
		if record.Denom == "" {
			return ErrInvalidDenom.Wrap("branding record with empty denom")
		}
		if _, seen := seenBrandingDenoms[record.Denom]; seen {
			return ErrInvalidBranding.Wrapf("duplicate branding record for denom %s", record.Denom)
		}
		seenBrandingDenoms[record.Denom] = struct{}{}
		if err := record.Branding.Validate(); err != nil {
			return err
		}
	}

	return gs.Params.Validate()
}
