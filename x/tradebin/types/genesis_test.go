package types_test

import (
	"testing"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	for _, tc := range []struct {
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
			desc: "valid genesis state",
			genState: &types.GenesisState{

				MarketList: []types.Market{
					{
						Asset1: "0",
						Asset2: "0",
					},
					{
						Asset1: "1",
						Asset2: "1",
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated market",
			genState: &types.GenesisState{
				MarketList: []types.Market{
					{
						Asset1: "0",
						Asset2: "0",
					},
					{
						Asset1: "0",
						Asset2: "0",
					},
				},
			},
			valid: false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	} {
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
