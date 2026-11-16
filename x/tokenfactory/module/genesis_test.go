package tokenfactory_test

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/testutil"
	"go.uber.org/mock/gomock"
	"testing"

	"cosmossdk.io/math"
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

// The mintable flag travels through genesis export and import, so a non-mintable denom never comes
// back mintable after a genesis restart (BZE-217).
func TestGenesis_MintableRoundTrip(t *testing.T) {
	creator := sample.AccAddress()
	mintableDenom := "factory/" + creator + "/open"
	fixedDenom := "factory/" + creator + "/fixed"
	renouncedDenom := "factory/" + creator + "/renounced"

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		FactoryDenoms: []types.GenesisDenom{
			{Denom: fixedDenom, DenomAuthority: types.DenomAuthority{Admin: creator, Mintable: false}},
			{Denom: mintableDenom, DenomAuthority: types.DenomAuthority{Admin: creator, Mintable: true}},
			{Denom: renouncedDenom, DenomAuthority: types.DenomAuthority{Admin: "", Mintable: false}},
		},
	}
	require.NoError(t, genesisState.Validate())

	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)
	bank := testutil.NewMockBankKeeper(ctrl)
	bank.EXPECT().GetDenomMetaData(gomock.Any(), gomock.Any()).Return(banktypes.Metadata{}, false).Times(3)
	bank.EXPECT().SetDenomMetaData(gomock.Any(), gomock.Any()).Times(3)

	k, ctx := keepertest.TokenfactoryKeeper(t, bank, acc, nil)
	tokenfactory.InitGenesis(ctx, k, genesisState)

	for _, gd := range genesisState.FactoryDenoms {
		dAuth, err := k.GetDenomAuthority(ctx, gd.Denom)
		require.NoError(t, err)
		require.Equal(t, gd.DenomAuthority, dAuth, gd.Denom)
	}

	got := tokenfactory.ExportGenesis(ctx, k)
	require.ElementsMatch(t, genesisState.FactoryDenoms, got.FactoryDenoms)
}

// Stored denom configs travel through genesis; a denom without one exports no config.
func TestGenesis_DenomConfigRoundTrip(t *testing.T) {
	creator := sample.AccAddress()
	cp := sample.AccAddress()
	configured := "factory/" + creator + "/configured"
	plain := "factory/" + creator + "/plain"
	cfg := types.DenomConfig{CommunityPool: cp, TransferFee: math.LegacyMustNewDecFromStr("0.001"), GasSurcharge: math.LegacyMustNewDecFromStr("1.5")}

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		FactoryDenoms: []types.GenesisDenom{
			{Denom: configured, DenomAuthority: types.DenomAuthority{Admin: creator, Mintable: true}, DenomConfig: &cfg},
			{Denom: plain, DenomAuthority: types.DenomAuthority{Admin: creator, Mintable: true}},
		},
	}
	require.NoError(t, genesisState.Validate())

	ctrl := gomock.NewController(t)
	acc := testutil.NewMockAccountKeeper(ctrl)
	bank := testutil.NewMockBankKeeper(ctrl)
	bank.EXPECT().GetDenomMetaData(gomock.Any(), gomock.Any()).Return(banktypes.Metadata{}, false).Times(2)
	bank.EXPECT().SetDenomMetaData(gomock.Any(), gomock.Any()).Times(2)

	k, ctx := keepertest.TokenfactoryKeeper(t, bank, acc, nil)
	tokenfactory.InitGenesis(ctx, k, genesisState)

	require.Equal(t, cfg, k.GetDenomConfig(ctx, configured))
	require.Equal(t, types.DefaultDenomConfig(), k.GetDenomConfig(ctx, plain))

	got := tokenfactory.ExportGenesis(ctx, k)
	require.ElementsMatch(t, genesisState.FactoryDenoms, got.FactoryDenoms)
}

// A genesis config holding the defaults stores nothing and exports as no config.
func TestGenesis_DefaultDenomConfigNotStored(t *testing.T) {
	creator := sample.AccAddress()
	denom := "factory/" + creator + "/token"
	defaults := types.DefaultDenomConfig()
	genesisState := types.GenesisState{
		Params:        types.DefaultParams(),
		FactoryDenoms: []types.GenesisDenom{{Denom: denom, DenomAuthority: types.DenomAuthority{Admin: creator, Mintable: true}, DenomConfig: &defaults}},
	}

	ctrl := gomock.NewController(t)
	bank := testutil.NewMockBankKeeper(ctrl)
	bank.EXPECT().GetDenomMetaData(gomock.Any(), gomock.Any()).Return(banktypes.Metadata{}, false).Times(1)
	bank.EXPECT().SetDenomMetaData(gomock.Any(), gomock.Any()).Times(1)

	k, ctx := keepertest.TokenfactoryKeeper(t, bank, testutil.NewMockAccountKeeper(ctrl), nil)
	tokenfactory.InitGenesis(ctx, k, genesisState)

	require.Nil(t, k.GetDenomPrefixStore(ctx, denom).Get([]byte(types.DenomConfigKey)))
	got := tokenfactory.ExportGenesis(ctx, k)
	require.Len(t, got.FactoryDenoms, 1)
	require.Nil(t, got.FactoryDenoms[0].DenomConfig)
}
