package keeper_test

import (
	"testing"
	"time"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/epochs/types"
	"github.com/stretchr/testify/require"
)

func TestSafeGetEpochCountByIdentifier_NonExistent(t *testing.T) {
	k, ctx := keepertest.EpochKeeper(t)

	count, err := k.SafeGetEpochCountByIdentifier(ctx, "nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
	require.Equal(t, int64(0), count)
}

func TestSafeGetEpochCountByIdentifier_NotStarted(t *testing.T) {
	k, ctx := keepertest.EpochKeeper(t)

	// Add epoch that hasn't started counting yet
	epoch := types.NewGenesisEpochInfo("hour", time.Hour)
	epoch.StartTime = ctx.BlockTime().Add(time.Hour) // starts in the future
	err := k.AddEpochInfo(ctx, epoch)
	require.NoError(t, err)

	count, err := k.SafeGetEpochCountByIdentifier(ctx, "hour")
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

func TestSafeGetEpochCountByIdentifier_CaughtUp(t *testing.T) {
	k, ctx := keepertest.EpochKeeper(t)

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(now)

	// Add epoch and simulate it being started and current
	epoch := types.EpochInfo{
		Identifier:              "hour",
		StartTime:               now.Add(-2 * time.Hour),
		Duration:                time.Hour,
		CurrentEpoch:            10,
		CurrentEpochStartTime:   now.Add(-30 * time.Minute), // started 30 min ago
		EpochCountingStarted:    true,
		CurrentEpochStartHeight: 100,
	}
	err := k.AddEpochInfo(ctx, epoch)
	require.NoError(t, err)

	count, err := k.SafeGetEpochCountByIdentifier(ctx, "hour")
	require.NoError(t, err)
	require.Equal(t, int64(10), count)
}

func TestSafeGetEpochCountByIdentifier_Stale(t *testing.T) {
	k, ctx := keepertest.EpochKeeper(t)

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(now)

	// Add epoch where block time is past the epoch end time (stale/catching up)
	epoch := types.EpochInfo{
		Identifier:              "hour",
		StartTime:               now.Add(-5 * time.Hour),
		Duration:                time.Hour,
		CurrentEpoch:            3,
		CurrentEpochStartTime:   now.Add(-3 * time.Hour), // epoch ended 2 hours ago
		EpochCountingStarted:    true,
		CurrentEpochStartHeight: 50,
	}
	err := k.AddEpochInfo(ctx, epoch)
	require.NoError(t, err)

	count, err := k.SafeGetEpochCountByIdentifier(ctx, "hour")
	require.Error(t, err)
	require.Contains(t, err.Error(), "catching up")
	require.Equal(t, int64(0), count)
}
