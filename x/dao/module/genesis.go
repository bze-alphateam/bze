package dao

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/keeper"
	"github.com/bze-alphateam/bze/x/dao/types"
)

// InitGenesis initializes the module's state from a provided genesis state. It
// checks the invariants I2 to I7 first and I1 once the state is written, and
// panics, naming the invariant, when one is broken.
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState types.GenesisState) {
	if err := genState.Validate(); err != nil {
		panic(err)
	}

	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	k.SetAllDaos(ctx, genState.Daos)
	k.SetAllProposals(ctx, genState.Proposals)
	k.SetDaoCounter(ctx, genState.DaoCount)
	k.SetProposalCounter(ctx, genState.ProposalCount)
	k.SetAllBallots(ctx, genState.Ballots)
	k.SetAllVoteIndex(ctx, genState.VoteIndex)
	k.SetAllJoinRecords(ctx, genState.JoinRecords)
	k.SetAllPins(ctx, genState.Pins)
	k.SetAllMemberRecords(ctx, genState.MemberRecords)

	// I1: x/bank imports first, so the escrow balance is known here
	if err := k.CheckDepositEscrow(ctx); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.Daos = k.GetAllDaos(ctx)
	genesis.DaoCount = k.GetDaoCounter(ctx)
	genesis.ProposalCount = k.GetProposalCounter(ctx)
	genesis.Proposals = k.GetAllProposals(ctx)
	genesis.Ballots = k.GetAllBallots(ctx)
	genesis.VoteIndex = k.GetAllVoteIndex(ctx)
	genesis.JoinRecords = k.GetAllJoinRecords(ctx)
	genesis.Pins = k.GetAllPins(ctx)
	genesis.MemberRecords = k.GetAllMemberRecords(ctx)

	return genesis
}
