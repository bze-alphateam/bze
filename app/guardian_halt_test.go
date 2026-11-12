package app_test

import (
	"context"
	"testing"

	"cosmossdk.io/core/header"
	"cosmossdk.io/x/upgrade"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	guardiantypes "github.com/bze-alphateam/bze/x/guardian/types"
)

const haltAt = 500

// atHeight puts the context at height h, as baseapp does: the block height and
// the header info x/upgrade reads.
func atHeight(ctx sdk.Context, h int64) sdk.Context {
	return ctx.WithBlockHeight(h).WithHeaderInfo(header.Info{Height: h, Time: ctx.BlockTime()})
}

func newHaltApp(t *testing.T) (*app.App, sdk.Context) {
	t.Helper()
	bApp, ctx := newRateLimitApp(t)
	require.NoError(t, bApp.GuardianKeeper.SetParams(ctx, guardiantypes.NewParams(guardianSigner)))

	return bApp, atHeight(ctx, haltAt)
}

// The guardian's MsgHaltChain goes through the real router to the real
// x/upgrade keeper: the plan is scheduled at the next height, replacing the
// plan that was there.
func TestGuardian_HaltChainReplacesThePlan(t *testing.T) {
	bApp, ctx := newHaltApp(t)
	require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: "v9.0.0", Height: 10_000}))

	require.NoError(t, handle(bApp, ctx, &guardiantypes.MsgHaltChain{Signer: guardianSigner, Reason: "incident"}))

	plan, err := bApp.UpgradeKeeper.GetUpgradePlan(ctx)
	require.NoError(t, err)
	require.Equal(t, upgradetypes.Plan{Name: "emergency-halt-501", Height: haltAt + 1, Info: "incident"}, plan)
}

// A stranger is refused through the router and the plan is left as it was.
func TestGuardian_HaltChainStrangerRefused(t *testing.T) {
	bApp, ctx := newHaltApp(t)

	err := handle(bApp, ctx, &guardiantypes.MsgHaltChain{Signer: sdk.AccAddress("stranger_address____").String(), Reason: "x"})
	require.ErrorIs(t, err, guardiantypes.ErrUnauthorized)

	_, err = bApp.UpgradeKeeper.GetUpgradePlan(ctx)
	require.ErrorIs(t, err, upgradetypes.ErrNoUpgradePlanFound)
}

// The next block stops: the binary has no handler for the halt plan, so the
// x/upgrade pre-blocker refuses the block with the usual "UPGRADE NEEDED".
// The block before it still runs.
func TestGuardian_HaltChainStopsTheNextBlock(t *testing.T) {
	bApp, ctx := newHaltApp(t)
	require.NoError(t, handle(bApp, ctx, &guardiantypes.MsgHaltChain{Signer: govAuthority(), Reason: "incident"}))

	_, err := upgrade.PreBlocker(ctx, bApp.UpgradeKeeper)
	require.NoError(t, err, "the block that scheduled it is past its pre-blocker already")

	_, err = upgrade.PreBlocker(atHeight(ctx, haltAt+1), bApp.UpgradeKeeper)
	require.ErrorContains(t, err, `UPGRADE "emergency-halt-501" NEEDED`)
}

// A binary that registers a handler for the plan name restarts the chain at
// that height and the plan is cleared.
func TestGuardian_HaltChainHandlerRestarts(t *testing.T) {
	bApp, ctx := newHaltApp(t)
	require.NoError(t, handle(bApp, ctx, &guardiantypes.MsgHaltChain{Signer: guardianSigner, Reason: "incident"}))

	bApp.UpgradeKeeper.SetUpgradeHandler(guardiantypes.HaltPlanName(haltAt+1), func(_ context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return vm, nil
	})
	next := atHeight(ctx, haltAt+1)
	_, err := upgrade.PreBlocker(next, bApp.UpgradeKeeper)
	require.NoError(t, err)

	_, err = bApp.UpgradeKeeper.GetUpgradePlan(next)
	require.ErrorIs(t, err, upgradetypes.ErrNoUpgradePlanFound)
}
