package types_test

import (
	"testing"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func validGenesisBranding() *types.DenomBranding {
	return &types.DenomBranding{
		Font: "inter",
		Light: &types.BrandingColors{
			Background: "#FFFFFF",
			Text:       "#111111",
			Primary:    "#1a2b3c",
			Secondary:  "#a1b2c3",
		},
		Dark: &types.BrandingColors{
			Background: "#000000",
			Text:       "#eeeeee",
			Primary:    "#c0ffee",
			Secondary:  "#abcdef",
		},
	}
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
			desc: "valid branding records",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DenomBrandings: []types.DenomBrandingRecord{
					{Denom: "factory/creator/one", Branding: validGenesisBranding()},
					{Denom: "factory/creator/two", Branding: validGenesisBranding()},
				},
			},
			valid: true,
		},
		{
			desc: "duplicate branding denom",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DenomBrandings: []types.DenomBrandingRecord{
					{Denom: "factory/creator/one", Branding: validGenesisBranding()},
					{Denom: "factory/creator/one", Branding: validGenesisBranding()},
				},
			},
			valid: false,
		},
		{
			desc: "branding record with empty denom",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DenomBrandings: []types.DenomBrandingRecord{
					{Denom: "", Branding: validGenesisBranding()},
				},
			},
			valid: false,
		},
		{
			desc: "branding record with nil branding",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DenomBrandings: []types.DenomBrandingRecord{
					{Denom: "factory/creator/one"},
				},
			},
			valid: false,
		},
		{
			desc: "branding record with invalid package",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DenomBrandings: []types.DenomBrandingRecord{
					{Denom: "factory/creator/one", Branding: &types.DenomBranding{Font: "inter"}},
				},
			},
			valid: false,
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
