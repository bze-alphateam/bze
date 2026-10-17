package dao_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/dao/keeper"
	dao "github.com/bze-alphateam/bze/x/dao/module"
	"github.com/bze-alphateam/bze/x/dao/testutil"
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

// genesisKeeper returns a keeper whose bank reports escrow as the dao module
// account balance.
func genesisKeeper(t *testing.T, escrow sdk.Coins) (keeper.Keeper, sdk.Context) {
	bank := testutil.NewMockBankKeeper(gomock.NewController(t))
	bank.EXPECT().GetAllBalances(gomock.Any(), authtypes.NewModuleAddress(types.ModuleName)).Return(escrow).AnyTimes()

	return keepertest.DaoKeeper(t, keepertest.DaoCodec(), nil, nil, bank, nil)
}

func TestGenesis_DefaultRoundTrip(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := genesisKeeper(t, sdk.NewCoins())
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

	k, ctx := genesisKeeper(t, sdk.NewCoins())
	dao.InitGenesis(ctx, &k, genesisState)
	exported := dao.ExportGenesis(ctx, &k)
	require.Equal(t, genesisState, *exported)

	fresh, freshCtx := genesisKeeper(t, sdk.NewCoins())
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

func ubze(amount int64) *sdk.Coin {
	coin := sdk.NewInt64Coin("ubze", amount)
	return &coin
}

// genesisProposal is a proposal of the given DAO and status, tallied 3/1/1.
func genesisProposal(id, daoId uint64, status types.ProposalStatus) types.Proposal {
	return types.Proposal{
		Id:                id,
		DaoId:             daoId,
		Proposer:          memberA.String(),
		Title:             "Fund tooling",
		Status:            status,
		SubmitHeight:      100 + int64(id),
		SubmitTime:        1_700_001_000,
		VotingEnd:         1_700_260_200,
		PinnedConfig:      genesisConfig(),
		PinnedDenominator: math.NewInt(10),
		Yes:               math.NewInt(3),
		No:                math.NewInt(1),
		Abstain:           math.NewInt(1),
	}
}

// genesisProposals returns a proposal in every status, ids 1..8 with 6 missing
// (cleaned up): 1 and 2 are VOTING with a deposit each (2 on stake DAO 2, over
// SR 3 and the ubze DR), 3 is expedited and VOTING without a deposit, the
// others are finished or failed, some with a released deposit and a pinned
// electorate that must not count.
func genesisProposals(t *testing.T) []types.Proposal {
	send, err := codectypes.NewAnyWithValue(&banktypes.MsgSend{
		FromAddress: types.DaoAddress(1).String(),
		ToAddress:   memberB.String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("ubze", 30)),
	})
	require.NoError(t, err)

	voting := genesisProposal(1, 1, types.ProposalStatus_VOTING)
	voting.Msgs = []*codectypes.Any{send}
	voting.PinnedDeposit = ubze(1_000)

	stakeVoting := genesisProposal(2, 2, types.ProposalStatus_VOTING)
	stakeVoting.PinnedElectorate = &types.StakeElectorate{SrIds: []string{"3"}, DrDenom: "ubze"}
	stakeVoting.PinnedDeposit = ubze(500)

	expedited := genesisProposal(3, 1, types.ProposalStatus_VOTING)
	expedited.Expedited = true

	failed := genesisProposal(5, 2, types.ProposalStatus_EXECUTION_FAILED)
	failed.PinnedElectorate = &types.StakeElectorate{SrIds: []string{"7"}}
	failed.PinnedDeposit = ubze(1_000)

	rejected := genesisProposal(4, 3, types.ProposalStatus_REJECTED)
	rejected.PinnedElectorate = &types.StakeElectorate{SrIds: []string{"3"}}
	rejected.CleanupCursor = "cursor"

	return []types.Proposal{
		voting,
		stakeVoting,
		expedited,
		rejected,
		failed,
		genesisProposal(7, 1, types.ProposalStatus_EXECUTED),
		genesisProposal(8, 1, types.ProposalStatus_EXPIRED),
		genesisProposal(9, 2, types.ProposalStatus_CANCELLED),
	}
}

func proposalsGenesis(t *testing.T) types.GenesisState {
	gs := types.GenesisState{
		Params:        types.DefaultParams(),
		Daos:          genesisDaos(),
		DaoCount:      3,
		Proposals:     genesisProposals(t),
		ProposalCount: 10,
	}
	require.NoError(t, gs.Validate())

	return gs
}

// the deposits of proposals 1 and 2, the only ones in voting with one
var votingEscrow = sdk.NewCoins(sdk.NewInt64Coin("ubze", 1_500))

// Proposals in every status survive export and import; the open markers and the
// DAO proposal index are rebuilt, and the sources are pinned by the DAOs and
// the proposals in voting only (I4).
func TestGenesis_ProposalsRoundTrip(t *testing.T) {
	genesisState := proposalsGenesis(t)

	k, ctx := genesisKeeper(t, votingEscrow)
	dao.InitGenesis(ctx, &k, genesisState)
	exported := dao.ExportGenesis(ctx, &k)
	require.Equal(t, genesisState, *exported)

	// through JSON, as a genesis file carries it: the messages survive (JSON
	// decodes an empty list as empty, not nil, and marshalling caches a JSON form
	// in each Any, so from here the bytes are compared)
	cdc := keepertest.DaoCodec()
	exportedJSON := cdc.MustMarshalJSON(exported)
	var decoded types.GenesisState
	cdc.MustUnmarshalJSON(exportedJSON, &decoded)
	require.Equal(t, exportedJSON, cdc.MustMarshalJSON(&decoded))

	fresh, freshCtx := genesisKeeper(t, votingEscrow)
	dao.InitGenesis(freshCtx, &fresh, decoded)
	require.Equal(t, exportedJSON, cdc.MustMarshalJSON(dao.ExportGenesis(freshCtx, &fresh)))
	require.Equal(t, uint64(10), fresh.GetProposalCounter(freshCtx))

	for _, p := range genesisState.Proposals {
		stored, found := fresh.GetProposal(freshCtx, p.Id)
		require.True(t, found)
		require.Equal(t, p, stored)
		require.True(t, fresh.HasDaoProposalIndex(freshCtx, p.DaoId, p.Id), "proposal %d", p.Id)
		require.Equal(t, p.Status == types.ProposalStatus_VOTING, fresh.IsOpenProposal(freshCtx, p.DaoId, p.Id), "proposal %d", p.Id)
	}
	_, found := fresh.GetProposal(freshCtx, 6)
	require.False(t, found)
	require.False(t, fresh.HasDaoProposalIndex(freshCtx, 1, 6))

	// the imported messages are resolved
	msgs, err := genesisState.Proposals[0].UnpackMsgs()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	// DAOs: SR 3 twice, SR 7 and the ubze DR once; proposal 2 adds SR 3 and
	// the ubze DR; proposals 4 and 5 left voting and pin nothing
	require.Equal(t, uint64(3), fresh.GetSourcePins(freshCtx, types.StakingRewardSource("3")))
	require.Equal(t, uint64(1), fresh.GetSourcePins(freshCtx, types.StakingRewardSource("7")))
	require.Equal(t, uint64(2), fresh.GetSourcePins(freshCtx, types.DenomRewardSource("ubze")))
}

// I1: the import fails unless the dao module account holds at least the
// deposits of the proposals in voting; a surplus (coins sent to the address
// before the module existed) is accepted.
func TestGenesis_DepositEscrow(t *testing.T) {
	for _, tc := range []struct {
		desc    string
		balance sdk.Coins
		ok      bool
	}{
		{"exact", votingEscrow, true},
		{"surplus", sdk.NewCoins(sdk.NewInt64Coin("ubze", 1_501)), true},
		{"other denom besides", votingEscrow.Add(sdk.NewInt64Coin("uvdl", 1)), true},
		{"one short", sdk.NewCoins(sdk.NewInt64Coin("ubze", 1_499)), false},
		{"only another denom", sdk.NewCoins(sdk.NewInt64Coin("uvdl", 1_500)), false},
		{"empty", sdk.NewCoins(), false},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			k, ctx := genesisKeeper(t, tc.balance)
			initGenesis := func() { dao.InitGenesis(ctx, &k, proposalsGenesis(t)) }
			if tc.ok {
				require.NotPanics(t, initGenesis)
				return
			}

			require.PanicsWithError(t,
				"dao module account holds "+tc.balance.String()+", less than the deposits of the proposals in voting ("+votingEscrow.String()+")",
				initGenesis)
		})
	}
}

// Without deposits in escrow the balance is not read: the upgrade that adds
// the module imports its default genesis whatever the address already holds.
func TestGenesis_NoEscrowSkipsBalance(t *testing.T) {
	genesisState := proposalsGenesis(t)
	genesisState.Proposals[0].PinnedDeposit = nil
	genesisState.Proposals[1].Status = types.ProposalStatus_CANCELLED

	k, ctx := keepertest.DaoKeeper(t, keepertest.DaoCodec(), nil, nil, testutil.NewMockBankKeeper(gomock.NewController(t)), nil)
	dao.InitGenesis(ctx, &k, genesisState)
	require.Len(t, k.GetAllProposals(ctx), len(genesisState.Proposals))
}
