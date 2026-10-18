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
// address and admin indexes are rebuilt on import, the pins imported as they are.
func TestGenesis_DaosRoundTrip(t *testing.T) {
	genesisState := types.GenesisState{
		Params:        types.DefaultParams(),
		Daos:          genesisDaos(),
		DaoCount:      3,
		ProposalCount: 5,
		Pins:          []types.PinCount{{Source: "dr/ubze", Count: 1}, {Source: "sr/3", Count: 2}, {Source: "sr/7", Count: 1}},
		MemberRecords: []types.MemberRecord{{DaoId: 1, Address: memberB.String(), Height: 42}},
	}
	require.NoError(t, genesisState.Validate())

	k, ctx := genesisKeeper(t, sdk.NewCoins())
	dao.InitGenesis(ctx, &k, genesisState)
	exported := dao.ExportGenesis(ctx, &k)
	require.Equal(t, genesisState, *exported)

	record, found := k.GetMemberRecord(ctx, 1, memberB)
	require.True(t, found)
	require.Equal(t, int64(42), record.Height)

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

// genesisProposals returns a proposal in every status, ids 1..9 with 6 missing
// (cleaned up): 1 and 2 are VOTING with a deposit each (2 on stake DAO 2, over
// SR 3 and the ubze DR), 3 is expedited and VOTING without a deposit, the
// others are finished or failed, some with a released deposit and a pinned
// electorate that must not count. The tallies of the proposals in voting are
// those of genesisBallots.
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
	voting.Yes, voting.No, voting.Abstain = math.OneInt(), math.OneInt(), math.ZeroInt()

	stakeVoting := genesisProposal(2, 2, types.ProposalStatus_VOTING)
	stakeVoting.PinnedElectorate = &types.StakeElectorate{SrIds: []string{"3"}, DrDenom: "ubze"}
	stakeVoting.PinnedDeposit = ubze(500)
	stakeVoting.Yes, stakeVoting.No, stakeVoting.Abstain = math.NewInt(10), math.ZeroInt(), math.ZeroInt()

	expedited := genesisProposal(3, 1, types.ProposalStatus_VOTING)
	expedited.Expedited = true
	expedited.Yes, expedited.No, expedited.Abstain = math.ZeroInt(), math.ZeroInt(), math.OneInt()

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

// genesisBallots: on 1 a yes and a no member vote; on 2 memberA's yes counts
// shares of SR 3 and the ubze DR, memberB's no was reduced to 0 by exits; on
// 3 one abstain; on finished 5 a ballot whose tally is final.
func genesisBallots() []types.Ballot {
	return []types.Ballot{
		{ProposalId: 1, Voter: memberA.String(), Option: types.VoteOption_YES, Weight: math.OneInt()},
		{ProposalId: 1, Voter: memberB.String(), Option: types.VoteOption_NO, Weight: math.OneInt()},
		{ProposalId: 2, Voter: memberA.String(), Option: types.VoteOption_YES, Weight: math.NewInt(10), Shares: []types.BallotShare{
			{Source: "sr/3", Amount: math.NewInt(4)},
			{Source: "dr/ubze", Amount: math.NewInt(6)},
		}},
		{ProposalId: 2, Voter: memberB.String(), Option: types.VoteOption_NO, Weight: math.ZeroInt()},
		{ProposalId: 3, Voter: memberA.String(), Option: types.VoteOption_ABSTAIN, Weight: math.OneInt()},
		{ProposalId: 5, Voter: memberB.String(), Option: types.VoteOption_YES, Weight: math.NewInt(3), Shares: []types.BallotShare{
			{Source: "sr/7", Amount: math.NewInt(3)},
		}},
	}
}

// proposalsGenesis is the full state: the DAOs, proposals and ballots above,
// the vote index of memberA's shares on 2 plus leftovers of finished 5 and
// cleaned-up 6, join records (one on a source no longer pinned) and the pins
// of the DAOs and of proposal 2. Every list is in export order.
func proposalsGenesis(t *testing.T) types.GenesisState {
	gs := types.GenesisState{
		Params:        types.DefaultParams(),
		Daos:          genesisDaos(),
		DaoCount:      3,
		Proposals:     genesisProposals(t),
		ProposalCount: 10,
		Ballots:       genesisBallots(),
		VoteIndex: []types.VoteIndexEntry{
			{Source: "sr/3", Address: memberA.String(), ProposalId: 2},
			{Source: "sr/3", Address: memberA.String(), ProposalId: 6},
			{Source: "sr/7", Address: memberB.String(), ProposalId: 5},
			{Source: "dr/ubze", Address: memberA.String(), ProposalId: 2},
		},
		JoinRecords: []types.JoinRecord{
			{Source: "sr/1", Address: outsider.String(), Height: 7},
			{Source: "sr/3", Address: memberB.String(), Height: 150},
		},
		Pins: []types.PinCount{{Source: "dr/ubze", Count: 2}, {Source: "sr/3", Count: 3}, {Source: "sr/7", Count: 1}},
	}
	require.NoError(t, gs.Validate())

	return gs
}

// the deposits of proposals 1 and 2, the only ones in voting with one
var votingEscrow = sdk.NewCoins(sdk.NewInt64Coin("ubze", 1_500))

// The full state survives export and import; the open markers and the DAO
// proposal index are rebuilt, the ballots, vote index, join records and pins
// are imported as they are.
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

	// the imported ballots, index and join records are the ones the keeper reads
	for _, b := range genesisBallots() {
		stored, found := fresh.GetBallot(freshCtx, b.ProposalId, sdk.MustAccAddressFromBech32(b.Voter))
		require.True(t, found)
		require.Equal(t, b, stored)
	}
	require.True(t, fresh.HasVoteIndex(freshCtx, types.DenomRewardSource("ubze"), memberA, 2))
	require.True(t, fresh.HasVoteIndex(freshCtx, types.StakingRewardSource("7"), memberB, 5))
	require.False(t, fresh.HasVoteIndex(freshCtx, types.StakingRewardSource("3"), memberB, 2))
	record, found := fresh.GetJoinRecord(freshCtx, types.StakingRewardSource("3"), memberB)
	require.True(t, found)
	require.Equal(t, int64(150), record.Height)
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
				"I1: dao module account holds "+tc.balance.String()+", less than the deposits of the proposals in voting ("+votingEscrow.String()+")",
				initGenesis)
		})
	}
}

// Without deposits in escrow the balance is not read: the upgrade that adds
// the module imports its default genesis whatever the address already holds.
func TestGenesis_NoEscrowSkipsBalance(t *testing.T) {
	genesisState := proposalsGenesis(t)
	genesisState.Proposals[0].PinnedDeposit = nil
	genesisState.Proposals[1].PinnedDeposit = nil

	k, ctx := keepertest.DaoKeeper(t, keepertest.DaoCodec(), nil, nil, testutil.NewMockBankKeeper(gomock.NewController(t)), nil)
	dao.InitGenesis(ctx, &k, genesisState)
	require.Len(t, k.GetAllProposals(ctx), len(genesisState.Proposals))
}

// Each broken invariant fails the import with an error naming it.
func TestGenesis_BrokenInvariants(t *testing.T) {
	for _, tc := range []struct {
		invariant string
		balance   sdk.Coins
		mutate    func(gs *types.GenesisState)
		wantErr   string
	}{
		{"I1", sdk.NewCoins(sdk.NewInt64Coin("ubze", 1_499)), func(*types.GenesisState) {}, "I1: dao module account holds 1499ubze"},
		{"I2", votingEscrow, func(gs *types.GenesisState) { gs.Proposals[0].Yes = math.NewInt(2) }, "I2: proposal 1 has YES 2, its ballots add up to 1"},
		{"I2 removed member", votingEscrow, func(gs *types.GenesisState) { gs.Ballots[1].Weight = math.ZeroInt() }, "I2: proposal 1 has NO 1, its ballots add up to 0"},
		{"I3", votingEscrow, func(gs *types.GenesisState) { gs.Ballots[2].Shares[0].Amount = math.NewInt(5) }, "I3: weight 10 is not the sum of its shares 11"},
		{"I3 unpinned source", votingEscrow, func(gs *types.GenesisState) { gs.Ballots[2].Shares[0].Source = "sr/7" }, "I3: share of sr/7, a source the proposal did not pin"},
		{"I3 static shares", votingEscrow, func(gs *types.GenesisState) {
			gs.Ballots[0].Shares = []types.BallotShare{{Source: "sr/3", Amount: math.OneInt()}}
		}, "I3: a static-list ballot has no shares"},
		{"I3 static weight", votingEscrow, func(gs *types.GenesisState) {
			gs.Ballots[0].Weight = math.NewInt(2)
			gs.Proposals[0].Yes = math.NewInt(2)
		}, "I3: a static-list ballot weighs at most 1, got 2"},
		{"I4", votingEscrow, func(gs *types.GenesisState) { gs.Pins[1].Count = 2 }, "I4: sr/3 has 2 pins, the daos and proposals in voting use it 3 times"},
		{"I4 missing", votingEscrow, func(gs *types.GenesisState) { gs.Pins = gs.Pins[1:] }, "I4: dr/ubze has no pins, the daos and proposals in voting use it 2 times"},
		{"I4 stray", votingEscrow, func(gs *types.GenesisState) {
			gs.Pins = append(gs.Pins, types.PinCount{Source: "sr/9", Count: 1})
		}, "I4: sr/9 has 1 pins, the daos and proposals in voting use it 0 times"},
		{"I5 dao", votingEscrow, func(gs *types.GenesisState) { gs.DaoCount = 2 }, "I5: dao id 3 is outside [1, dao_count 2]"},
		{"I5 proposal", votingEscrow, func(gs *types.GenesisState) { gs.ProposalCount = 8 }, "I5: proposal id 9 is outside [1, proposal_count 8]"},
		{"I5 vote index", votingEscrow, func(gs *types.GenesisState) { gs.VoteIndex[1].ProposalId = 11 }, "I5: vote-index entry of proposal 11, outside [1, proposal_count 10]"},
		{"I6 missing entry", votingEscrow, func(gs *types.GenesisState) { gs.VoteIndex = gs.VoteIndex[1:] }, "I6: the sr/3 share of " + memberA.String() + " on proposal 2 has no vote-index entry"},
		{"I6 stray entry", votingEscrow, func(gs *types.GenesisState) {
			gs.VoteIndex = append(gs.VoteIndex, types.VoteIndexEntry{Source: "sr/3", Address: memberB.String(), ProposalId: 2})
		}, "I6: vote-index entry sr/3/" + memberB.String() + "/2 names no share of a ballot"},
		{"I7", votingEscrow, func(gs *types.GenesisState) { gs.Daos[1].PendingAdmin = admin.String() }, "I7: pending_admin is the admin"},
		{"I7 no admin", votingEscrow, func(gs *types.GenesisState) { gs.Daos[0].Admin = "" }, "I7: admin is not set"},
	} {
		t.Run(tc.invariant, func(t *testing.T) {
			gs := proposalsGenesis(t)
			tc.mutate(&gs)

			k, ctx := genesisKeeper(t, tc.balance)
			defer func() {
				r := recover()
				require.NotNil(t, r, "the import must fail")
				err, ok := r.(error)
				require.True(t, ok, "panic with %v", r)
				require.ErrorContains(t, err, tc.wantErr)
			}()
			dao.InitGenesis(ctx, &k, gs)
		})
	}
}
