package scavenge_test

import (
	"testing"

	keepertest "github.com/cosmonaut/bzedgev5/testutil/keeper"
	"github.com/cosmonaut/bzedgev5/x/scavenge"
	"github.com/cosmonaut/bzedgev5/x/scavenge/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		ScavengeList: []types.Scavenge{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		CommitList: []types.Commit{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ScavengeKeeper(t)
	scavenge.InitGenesis(ctx, *k, genesisState)
	got := scavenge.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	require.Len(t, got.ScavengeList, len(genesisState.ScavengeList))
	require.Subset(t, genesisState.ScavengeList, got.ScavengeList)
	require.Len(t, got.CommitList, len(genesisState.CommitList))
	require.Subset(t, genesisState.CommitList, got.CommitList)
	// this line is used by starport scaffolding # genesis/test/assert
}
