package types_test

import (
	"testing"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	"github.com/stretchr/testify/require"
)

func genesisWithHaltedDenoms(denoms ...string) *types.GenesisState {
	return &types.GenesisState{Params: v2types.DefaultParams(), HaltedDenoms: denoms}
}

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc:     "halted denoms: valid list",
			genState: genesisWithHaltedDenoms("uusdc", "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"),
			valid:    true,
		},
		{
			desc:     "halted denoms: the native denom cannot be halted",
			genState: genesisWithHaltedDenoms("uusdc", v2types.DefaultNativeDenom),
			valid:    false,
		},
		{
			desc:     "halted denoms: duplicate",
			genState: genesisWithHaltedDenoms("uusdc", "uusdc"),
			valid:    false,
		},
		{
			desc:     "halted denoms: malformed denom",
			genState: genesisWithHaltedDenoms("1bad"),
			valid:    false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
