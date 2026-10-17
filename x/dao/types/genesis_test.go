package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestDefaultGenesis_IsValid(t *testing.T) {
	gs := types.DefaultGenesis()

	require.Equal(t, types.DefaultParams(), gs.Params)
	require.NoError(t, gs.Validate())
}

func TestGenesisState_Validate(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		genState types.GenesisState
		wantErr  bool
	}{
		{desc: "default", genState: *types.DefaultGenesis()},
		{desc: "zero value has invalid params", genState: types.GenesisState{}, wantErr: true},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
