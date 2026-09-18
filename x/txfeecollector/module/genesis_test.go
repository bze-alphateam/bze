package txfeecollector_test

import (
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	txfeecollector "github.com/bze-alphateam/bze/x/txfeecollector/module"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.TxfeecollectorKeeper(t)
	txfeecollector.InitGenesis(ctx, k, genesisState)
	got := txfeecollector.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}

// Blocked inbound transfers are part of the module params, so a chain export carries
// them and an import restores them: a restarted or forked chain keeps the block.
func TestGenesis_BlockedIbcInboundRoundTrip(t *testing.T) {
	blocked := []types.BlockedIbcTransfer{
		{ChannelId: "channel-3", BaseDenom: "uusdc"},
	}

	params := types.DefaultParams()
	params.BlockedIbcInbound = blocked
	genesisState := types.GenesisState{Params: params}
	require.NoError(t, genesisState.Validate())

	k, ctx := keepertest.TxfeecollectorKeeper(t)
	txfeecollector.InitGenesis(ctx, k, genesisState)
	require.True(t, k.GetParams(ctx).IsInboundBlocked("channel-3", "uusdc"))

	got := txfeecollector.ExportGenesis(ctx, k)
	require.NotNil(t, got)
	require.Equal(t, blocked, got.Params.BlockedIbcInbound)
}

// A genesis carrying a malformed entry is rejected up front instead of imported.
func TestGenesis_InvalidBlockedIbcInboundIsRejected(t *testing.T) {
	params := types.DefaultParams()
	params.BlockedIbcInbound = []types.BlockedIbcTransfer{
		{ChannelId: "not a channel", BaseDenom: "uusdc"},
	}

	err := types.GenesisState{Params: params}.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blocked ibc inbound channel id")
}
