package types_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestDefaultGenesis_IsValid(t *testing.T) {
	gs := types.DefaultGenesis()

	require.Equal(t, types.DefaultParams(), gs.Params)
	require.NoError(t, gs.Validate())
}

func genesisConfig() types.DaoConfig {
	return types.DaoConfig{
		MinQuorum:         math.LegacyMustNewDecFromStr("0.4"),
		Threshold:         math.LegacyMustNewDecFromStr("0.5"),
		VotingPeriod:      72 * time.Hour,
		ExpeditedPeriod:   24 * time.Hour,
		ExpediteQuorum:    math.LegacyMustNewDecFromStr("0.6"),
		ExpediteThreshold: math.LegacyMustNewDecFromStr("0.66"),
	}
}

// validGenesis holds a static DAO 1 with an external admin and a pending
// nomination, and a stake DAO 2; proposal 3 of DAO 1 is in voting with one
// yes ballot.
func validGenesis() types.GenesisState {
	voting := validProposal(3, 1, types.ProposalStatus_VOTING)
	voting.Yes = math.OneInt()

	return types.GenesisState{
		Params: types.DefaultParams(),
		Daos: []types.Dao{
			{
				Id:           1,
				Address:      types.DaoAddress(1).String(),
				Admin:        sdk.AccAddress("dao_admin___________").String(),
				PendingAdmin: sdk.AccAddress("dao_nominee_________").String(),
				Type:         types.DaoType_DAO_TYPE_STATIC_LIST,
				Name:         "Static",
				Config:       genesisConfig(),
				Members:      []string{sdk.AccAddress("dao_member_a________").String()},
			},
			{
				Id:         2,
				Address:    types.DaoAddress(2).String(),
				Admin:      types.DaoAddress(2).String(),
				Type:       types.DaoType_DAO_TYPE_STAKE,
				Name:       "Stake",
				Config:     genesisConfig(),
				Electorate: &types.StakeElectorate{SrIds: []string{"1"}, DrDenom: "ubze"},
			},
		},
		DaoCount:      2,
		Proposals: []types.Proposal{
			voting,
			validProposal(9, 2, types.ProposalStatus_CANCELLED),
		},
		ProposalCount: 9,
		Ballots: []types.Ballot{
			{ProposalId: 3, Voter: sdk.AccAddress("dao_member_a________").String(), Option: types.VoteOption_YES, Weight: math.OneInt()},
		},
		Pins: []types.PinCount{{Source: "dr/ubze", Count: 1}, {Source: "sr/1", Count: 1}},
	}
}

func validProposal(id, daoId uint64, status types.ProposalStatus) types.Proposal {
	deposit := sdk.NewInt64Coin("ubze", 1_000)

	return types.Proposal{
		Id:                id,
		DaoId:             daoId,
		Proposer:          sdk.AccAddress("dao_member_a________").String(),
		Title:             "Fund tooling",
		Status:            status,
		PinnedConfig:      genesisConfig(),
		PinnedDenominator: math.NewInt(10),
		PinnedDeposit:     &deposit,
		Yes:               math.ZeroInt(),
		No:                math.ZeroInt(),
		Abstain:           math.ZeroInt(),
	}
}

func TestGenesisState_Validate(t *testing.T) {
	member := sdk.AccAddress("dao_member_a________").String()

	for _, tc := range []struct {
		desc    string
		mutate  func(gs *types.GenesisState)
		wantErr string
	}{
		{desc: "default", mutate: func(gs *types.GenesisState) { *gs = *types.DefaultGenesis() }},
		{desc: "valid daos", mutate: func(*types.GenesisState) {}},
		{desc: "counter above the highest id", mutate: func(gs *types.GenesisState) { gs.DaoCount = 7 }},
		{desc: "self-owned stake dao with no electorate", mutate: func(gs *types.GenesisState) {
			gs.Daos[1].Electorate = nil
			gs.Pins = nil
		}},
		{desc: "zero value has invalid params", mutate: func(gs *types.GenesisState) { gs.Params = types.Params{} }, wantErr: "must be"},
		{desc: "duplicate id", mutate: func(gs *types.GenesisState) { gs.Daos[1] = gs.Daos[0] }, wantErr: "duplicate dao id 1"},
		{desc: "id above the counter", mutate: func(gs *types.GenesisState) { gs.DaoCount = 1 }, wantErr: "outside [1, dao_count 1]"},
		{desc: "id zero", mutate: func(gs *types.GenesisState) {
			gs.Daos[0].Id = 0
			gs.Daos[0].Address = types.DaoAddress(0).String()
		}, wantErr: "dao id 0 is outside"},
		{desc: "address not derived from the id", mutate: func(gs *types.GenesisState) { gs.Daos[0].Address = types.DaoAddress(2).String() }, wantErr: "not the derived address"},
		{desc: "invalid address", mutate: func(gs *types.GenesisState) { gs.Daos[0].Address = "bad" }, wantErr: "not the derived address"},
		{desc: "missing admin", mutate: func(gs *types.GenesisState) { gs.Daos[0].Admin = "" }, wantErr: "admin is not set"},
		{desc: "invalid admin", mutate: func(gs *types.GenesisState) { gs.Daos[0].Admin = "bad" }, wantErr: "invalid admin address"},
		{desc: "invalid pending admin", mutate: func(gs *types.GenesisState) { gs.Daos[0].PendingAdmin = "bad" }, wantErr: "invalid pending_admin address"},
		{desc: "pending admin is the admin", mutate: func(gs *types.GenesisState) { gs.Daos[0].PendingAdmin = gs.Daos[0].Admin }, wantErr: "pending_admin is the admin"},
		{desc: "pending admin is the dao", mutate: func(gs *types.GenesisState) { gs.Daos[0].PendingAdmin = gs.Daos[0].Address }, wantErr: "pending_admin is the dao itself"},
		{desc: "invalid name", mutate: func(gs *types.GenesisState) { gs.Daos[0].Name = "" }, wantErr: "name must have"},
		{desc: "invalid config", mutate: func(gs *types.GenesisState) { gs.Daos[0].Config.Threshold = math.LegacyZeroDec() }, wantErr: "threshold must be"},
		{desc: "static dao without members", mutate: func(gs *types.GenesisState) { gs.Daos[0].Members = nil }, wantErr: "at least one member"},
		{desc: "static dao with duplicate members", mutate: func(gs *types.GenesisState) { gs.Daos[0].Members = []string{member, member} }, wantErr: "duplicate member"},
		{desc: "static dao with a stake electorate", mutate: func(gs *types.GenesisState) {
			gs.Daos[0].Electorate = &types.StakeElectorate{SrIds: []string{"1"}}
		}, wantErr: "cannot have a stake electorate"},
		{desc: "stake dao with members", mutate: func(gs *types.GenesisState) { gs.Daos[1].Members = []string{member} }, wantErr: "cannot have a member list"},
		{desc: "stake dao with duplicate rewards", mutate: func(gs *types.GenesisState) {
			gs.Daos[1].Electorate.SrIds = []string{"1", "1"}
		}, wantErr: "duplicate staking reward"},
		{desc: "unknown type", mutate: func(gs *types.GenesisState) { gs.Daos[1].Type = 9 }, wantErr: "unknown dao type"},
		{desc: "no proposals", mutate: func(gs *types.GenesisState) {
			gs.Proposals = nil
			gs.Ballots = nil
		}},
		{desc: "proposal without a deposit", mutate: func(gs *types.GenesisState) { gs.Proposals[0].PinnedDeposit = nil }},
		{desc: "duplicate proposal id", mutate: func(gs *types.GenesisState) { gs.Proposals[1].Id = 3 }, wantErr: "duplicate proposal id 3"},
		{desc: "proposal id above the counter", mutate: func(gs *types.GenesisState) { gs.ProposalCount = 8 }, wantErr: "proposal id 9 is outside [1, proposal_count 8]"},
		{desc: "proposal id zero", mutate: func(gs *types.GenesisState) { gs.Proposals[0].Id = 0 }, wantErr: "proposal id 0 is outside"},
		{desc: "proposal of an unknown dao", mutate: func(gs *types.GenesisState) { gs.Proposals[1].DaoId = 3 }, wantErr: "proposal 9 belongs to unknown dao 3"},
		{desc: "unknown proposal status", mutate: func(gs *types.GenesisState) { gs.Proposals[0].Status = 6 }, wantErr: "unknown status 6"},
		{desc: "invalid proposer", mutate: func(gs *types.GenesisState) { gs.Proposals[0].Proposer = "bad" }, wantErr: "invalid proposer address"},
		{desc: "negative yes", mutate: func(gs *types.GenesisState) { gs.Proposals[0].Yes = math.NewInt(-1) }, wantErr: "yes must be a non-negative integer"},
		{desc: "negative no", mutate: func(gs *types.GenesisState) { gs.Proposals[0].No = math.NewInt(-1) }, wantErr: "no must be a non-negative integer"},
		{desc: "missing abstain", mutate: func(gs *types.GenesisState) { gs.Proposals[0].Abstain = math.Int{} }, wantErr: "abstain must be a non-negative integer"},
		{desc: "negative denominator", mutate: func(gs *types.GenesisState) { gs.Proposals[0].PinnedDenominator = math.NewInt(-1) }, wantErr: "pinned_denominator must be a non-negative integer"},
		{desc: "invalid deposit denom", mutate: func(gs *types.GenesisState) { gs.Proposals[0].PinnedDeposit.Denom = "!" }, wantErr: "invalid pinned_deposit"},
		{desc: "negative deposit", mutate: func(gs *types.GenesisState) { gs.Proposals[0].PinnedDeposit.Amount = math.NewInt(-5) }, wantErr: "invalid pinned_deposit"},
		{desc: "duplicate ballot", mutate: func(gs *types.GenesisState) { gs.Ballots = append(gs.Ballots, gs.Ballots[0]) }, wantErr: "duplicate ballot of " + member + " on proposal 3"},
		{desc: "ballot of an unknown proposal", mutate: func(gs *types.GenesisState) { gs.Ballots[0].ProposalId = 4 }, wantErr: "ballot of " + member + " on unknown proposal 4"},
		{desc: "invalid voter", mutate: func(gs *types.GenesisState) { gs.Ballots[0].Voter = "bad" }, wantErr: "invalid voter address"},
		{desc: "unknown option", mutate: func(gs *types.GenesisState) { gs.Ballots[0].Option = 3 }, wantErr: "unknown option 3"},
		{desc: "negative weight", mutate: func(gs *types.GenesisState) { gs.Ballots[0].Weight = math.NewInt(-1) }, wantErr: "weight must be a non-negative integer"},
		{desc: "vote-index leftover of a finished proposal", mutate: func(gs *types.GenesisState) {
			gs.VoteIndex = []types.VoteIndexEntry{{Source: "sr/1", Address: member, ProposalId: 9}}
		}},
		{desc: "vote-index leftover of a cleaned-up proposal", mutate: func(gs *types.GenesisState) {
			gs.VoteIndex = []types.VoteIndexEntry{{Source: "sr/1", Address: member, ProposalId: 5}}
		}},
		{desc: "duplicate vote-index entry", mutate: func(gs *types.GenesisState) {
			gs.VoteIndex = []types.VoteIndexEntry{{Source: "sr/1", Address: member, ProposalId: 9}, {Source: "sr/1", Address: member, ProposalId: 9}}
		}, wantErr: "duplicate vote-index entry sr/1/" + member + "/9"},
		{desc: "vote-index entry with an invalid source", mutate: func(gs *types.GenesisState) {
			gs.VoteIndex = []types.VoteIndexEntry{{Source: "xx/1", Address: member, ProposalId: 9}}
		}, wantErr: `invalid source "xx/1"`},
		{desc: "vote-index entry with an invalid address", mutate: func(gs *types.GenesisState) {
			gs.VoteIndex = []types.VoteIndexEntry{{Source: "sr/1", Address: "bad", ProposalId: 9}}
		}, wantErr: "invalid address"},
		{desc: "join record on an unpinned source", mutate: func(gs *types.GenesisState) {
			gs.JoinRecords = []types.JoinRecord{{Source: "sr/8", Address: member, Height: 5}}
		}},
		{desc: "duplicate join record", mutate: func(gs *types.GenesisState) {
			gs.JoinRecords = []types.JoinRecord{{Source: "sr/1", Address: member, Height: 5}, {Source: "sr/1", Address: member, Height: 6}}
		}, wantErr: "duplicate join record of " + member + " on sr/1"},
		{desc: "join record with an invalid source", mutate: func(gs *types.GenesisState) {
			gs.JoinRecords = []types.JoinRecord{{Source: "dr/", Address: member, Height: 5}}
		}, wantErr: `invalid source "dr/"`},
		{desc: "join record with an invalid address", mutate: func(gs *types.GenesisState) {
			gs.JoinRecords = []types.JoinRecord{{Source: "sr/1", Address: "bad", Height: 5}}
		}, wantErr: "invalid address"},
		{desc: "join record with a negative height", mutate: func(gs *types.GenesisState) {
			gs.JoinRecords = []types.JoinRecord{{Source: "sr/1", Address: member, Height: -1}}
		}, wantErr: "negative height"},
		{desc: "duplicate pin count", mutate: func(gs *types.GenesisState) { gs.Pins = append(gs.Pins, gs.Pins[0]) }, wantErr: "duplicate pin count of dr/ubze"},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			gs := validGenesis()
			tc.mutate(&gs)

			err := gs.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
