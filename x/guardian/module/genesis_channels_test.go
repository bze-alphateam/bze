package guardian_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	guardian "github.com/bze-alphateam/bze/x/guardian/module"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

// Closed channels survive an export and import: an exported chain must not reopen them.
func TestGenesis_ClosedChannelsRoundTrip(t *testing.T) {
	k, ctx := keepertest.GuardianKeeper(t, nil)

	genesis := types.GenesisState{Params: types.DefaultParams(), ClosedChannels: []string{"channel-0", "channel-12"}}
	guardian.InitGenesis(ctx, &k, genesis)

	require.True(t, k.IsChannelClosed(ctx, "channel-0"))
	require.True(t, k.IsChannelClosed(ctx, "channel-12"))
	require.False(t, k.IsChannelClosed(ctx, "channel-1"))

	exported := guardian.ExportGenesis(ctx, &k)
	require.ElementsMatch(t, genesis.ClosedChannels, exported.ClosedChannels)

	other, otherCtx := keepertest.GuardianKeeper(t, nil)
	guardian.InitGenesis(otherCtx, &other, *exported)
	require.Equal(t, exported, guardian.ExportGenesis(otherCtx, &other))
}

func TestGenesis_InvalidClosedChannelPanics(t *testing.T) {
	k, ctx := keepertest.GuardianKeeper(t, nil)

	require.Panics(t, func() {
		guardian.InitGenesis(ctx, &k, types.GenesisState{Params: types.DefaultParams(), ClosedChannels: []string{"nope"}})
	})
}

// Both stop switches with state survive an export and import together.
func TestGenesis_PauseAndClosedChannelsRoundTrip(t *testing.T) {
	k, ctx := keepertest.GuardianKeeper(t, nil)

	genesis := types.GenesisState{Params: types.DefaultParams(), AutomationPaused: true, ClosedChannels: []string{"channel-3"}}
	guardian.InitGenesis(ctx, &k, genesis)
	require.True(t, k.IsAutomationPaused(ctx))
	require.True(t, k.IsChannelClosed(ctx, "channel-3"))

	exported := guardian.ExportGenesis(ctx, &k)
	require.Equal(t, genesis, *exported)

	other, otherCtx := keepertest.GuardianKeeper(t, nil)
	guardian.InitGenesis(otherCtx, &other, *exported)
	require.Equal(t, exported, guardian.ExportGenesis(otherCtx, &other))
}
