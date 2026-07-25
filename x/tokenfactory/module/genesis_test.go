package tokenfactory_test

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/testutil"
	"go.uber.org/mock/gomock"
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	"github.com/bze-alphateam/bze/testutil/sample"
	tokenfactory "github.com/bze-alphateam/bze/x/tokenfactory/module"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)

	k, ctx := keepertest.TokenfactoryKeeper(t, nil, acc, nil)
	tokenfactory.InitGenesis(ctx, k, genesisState)
	got := tokenfactory.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}

func TestGenesis_BrandingRoundTrip(t *testing.T) {
	creator := sample.AccAddress()
	denom := "factory/" + creator + "/branded"
	branding := types.DenomBranding{
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

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		FactoryDenoms: []types.GenesisDenom{
			{Denom: denom, DenomAuthority: types.DenomAuthority{Admin: creator}},
		},
		DenomBrandings: []types.DenomBrandingRecord{
			{Denom: denom, Branding: &branding},
		},
	}
	require.NoError(t, genesisState.Validate())

	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)
	bank := testutil.NewMockBankKeeper(ctrl)
	bank.EXPECT().GetDenomMetaData(gomock.Any(), denom).Return(banktypes.Metadata{}, false).Times(1)
	bank.EXPECT().SetDenomMetaData(gomock.Any(), gomock.Any()).Times(1)

	k, ctx := keepertest.TokenfactoryKeeper(t, bank, acc, nil)
	tokenfactory.InitGenesis(ctx, k, genesisState)

	stored, found := k.GetDenomBranding(ctx, denom)
	require.True(t, found)
	require.True(t, branding.Equal(stored))

	got := tokenfactory.ExportGenesis(ctx, k)
	require.NotNil(t, got)
	require.Equal(t, genesisState.FactoryDenoms, got.FactoryDenoms)
	require.Equal(t, genesisState.DenomBrandings, got.DenomBrandings)
}

func TestGenesis_BrandingWithoutAuthorityPanics(t *testing.T) {
	creator := sample.AccAddress()
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		DenomBrandings: []types.DenomBrandingRecord{
			{
				Denom: "factory/" + creator + "/orphan",
				Branding: &types.DenomBranding{
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
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)

	k, ctx := keepertest.TokenfactoryKeeper(t, nil, acc, nil)
	require.Panics(t, func() {
		tokenfactory.InitGenesis(ctx, k, genesisState)
	})
}
