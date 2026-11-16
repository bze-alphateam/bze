package types

import "fmt"

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

	seen := make(map[string]struct{}, len(gs.ClosedChannels))
	for _, channelID := range gs.ClosedChannels {
		if err := ValidateChannelID(channelID); err != nil {
			return err
		}
		if _, dup := seen[channelID]; dup {
			return fmt.Errorf("duplicate closed channel %s", channelID)
		}
		seen[channelID] = struct{}{}
	}

	return nil
}
