package dao_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	dao "github.com/bze-alphateam/bze/x/dao/module"
	"github.com/bze-alphateam/bze/x/dao/types"
)

var (
	memberA  = sdk.AccAddress("dao_member_a________")
	memberB  = sdk.AccAddress("dao_member_b________")
	admin    = sdk.AccAddress("dao_admin___________")
	nominee  = sdk.AccAddress("dao_nominee_________")
	outsider = sdk.AccAddress("dao_outsider________")
)

func genesisConfig() types.DaoConfig {
	return types.DaoConfig{
		MinQuorum:         math.LegacyMustNewDecFromStr("0.4"),
		Threshold:         math.LegacyMustNewDecFromStr("0.5"),
		VotingPeriod:      72 * time.Hour,
		AllowExpedite:     true,
		ExpeditedPeriod:   24 * time.Hour,
		ExpediteQuorum:    math.LegacyMustNewDecFromStr("0.6"),
		ExpediteThreshold: math.LegacyMustNewDecFromStr("0.66"),
		ProposalDeposit:   &sdk.Coin{Denom: "ubze", Amount: math.NewInt(1_000)},
	}
}

// genesisDaos returns three DAOs: 1 a self-owned static list, 2 a stake DAO
// with an external admin and a pending nomination, 3 a stake DAO owned by DAO 1.
func genesisDaos() []types.Dao {
	return []types.Dao{
		{
			Id:          1,
			Address:     types.DaoAddress(1).String(),
			Admin:       types.DaoAddress(1).String(),
			Type:        types.DaoType_DAO_TYPE_STATIC_LIST,
			Name:        "BZE builders",
			Description: "funds BZE tooling",
			Config:      genesisConfig(),
			Members:     []string{memberA.String(), memberB.String()},
			CreatedAt:   1_700_000_000,
		},
		{
			Id:           2,
			Address:      types.DaoAddress(2).String(),
			Admin:        admin.String(),
			PendingAdmin: nominee.String(),
			Type:         types.DaoType_DAO_TYPE_STAKE,
			Name:         "Stakers",
			Config:       genesisConfig(),
			Electorate:   &types.StakeElectorate{SrIds: []string{"3", "7"}, DrDenom: "ubze"},
			CreatedAt:    1_700_000_100,
		},
		{
			Id:         3,
			Address:    types.DaoAddress(3).String(),
			Admin:      types.DaoAddress(1).String(),
			Type:       types.DaoType_DAO_TYPE_STAKE,
			Name:       "Sub DAO",
			Config:     genesisConfig(),
			Electorate: &types.StakeElectorate{SrIds: []string{"3"}},
			CreatedAt:  1_700_000_200,
		},
	}
}

func TestGenesis_DefaultRoundTrip(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := keepertest.DaoKeeper(t, keepertest.DaoCodec(), nil, nil, nil, nil)
	dao.InitGenesis(ctx, &k, genesisState)
	got := dao.ExportGenesis(ctx, &k)

	require.NotNil(t, got)
	require.Equal(t, genesisState, *got)
}

// Export, import into a fresh keeper and export again give the same state; the
// address and admin indexes and the DAO pins are rebuilt on import.
func TestGenesis_DaosRoundTrip(t *testing.T) {
	genesisState := types.GenesisState{
		Params:        types.DefaultParams(),
		Daos:          genesisDaos(),
		DaoCount:      3,
		ProposalCount: 5,
	}
	require.NoError(t, genesisState.Validate())

	k, ctx := keepertest.DaoKeeper(t, keepertest.DaoCodec(), nil, nil, nil, nil)
	dao.InitGenesis(ctx, &k, genesisState)
	exported := dao.ExportGenesis(ctx, &k)
	require.Equal(t, genesisState, *exported)

	fresh, freshCtx := keepertest.DaoKeeper(t, keepertest.DaoCodec(), nil, nil, nil, nil)
	dao.InitGenesis(freshCtx, &fresh, *exported)
	require.Equal(t, *exported, *dao.ExportGenesis(freshCtx, &fresh))

	require.Equal(t, uint64(3), fresh.GetDaoCounter(freshCtx))
	require.Equal(t, uint64(5), fresh.GetProposalCounter(freshCtx))

	for _, d := range genesisDaos() {
		res, err := fresh.DaoByAddress(freshCtx, &types.QueryDaoByAddressRequest{Address: d.Address})
		require.NoError(t, err)
		require.Equal(t, d, res.Dao)

		// I7: the admin index holds exactly the admin, never the pending admin
		require.True(t, fresh.HasDaoAdminIndex(freshCtx, sdk.MustAccAddressFromBech32(d.Admin), d.Id))
		require.False(t, fresh.HasDaoAdminIndex(freshCtx, nominee, d.Id))
		require.False(t, fresh.HasDaoAdminIndex(freshCtx, outsider, d.Id))
	}
	require.False(t, fresh.HasDaoAdminIndex(freshCtx, admin, 1))
	require.False(t, fresh.HasDaoAdminIndex(freshCtx, admin, 3))

	// the pins count the DAOs using each source
	require.Equal(t, uint64(2), fresh.GetSourcePins(freshCtx, types.StakingRewardSource("3")))
	require.Equal(t, uint64(1), fresh.GetSourcePins(freshCtx, types.StakingRewardSource("7")))
	require.Equal(t, uint64(1), fresh.GetSourcePins(freshCtx, types.DenomRewardSource("ubze")))
	require.Zero(t, fresh.GetSourcePins(freshCtx, types.StakingRewardSource("1")))
}
