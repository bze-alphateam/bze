package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	ratelimit "github.com/cosmos/ibc-apps/modules/rate-limiting/v10"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
)

const (
	rlChannel = "channel-0"
	rlDenom   = "ubze"
)

// newRateLimitApp builds the real app with an open transfer channel and some
// ubze supply, so a rate limit can be added for (channel-0, ubze).
func newRateLimitApp(t *testing.T) (*app.App, sdk.Context) {
	t.Helper()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()
	bApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)
	ctx := bApp.NewUncachedContext(false, cmtproto.Header{Time: time.Date(2026, 11, 2, 13, 30, 0, 0, time.UTC)})

	require.NoError(t, bApp.BankKeeper.SetParams(ctx, banktypes.DefaultParams()))
	supply := sdk.NewCoins(sdk.NewInt64Coin(rlDenom, 1_000_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, supply))
	bApp.IBCKeeper.ChannelKeeper.SetChannel(ctx, ibctransfertypes.PortID, rlChannel, channeltypes.Channel{
		State:    channeltypes.OPEN,
		Ordering: channeltypes.UNORDERED,
		Counterparty: channeltypes.Counterparty{
			PortId:    ibctransfertypes.PortID,
			ChannelId: "channel-9",
		},
		ConnectionHops: []string{"connection-0"},
		Version:        ibctransfertypes.V1,
	})

	return bApp, ctx
}

func govAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

func handle(bApp *app.App, ctx sdk.Context, msg sdk.Msg) error {
	_, err := bApp.MsgServiceRouter().Handler(msg)(ctx, msg)
	return err
}

// The transfer route is the rate-limiting middleware, outermost.
func TestRateLimit_TransferRouteIsRateLimitMiddleware(t *testing.T) {
	bApp, _ := newRateLimitApp(t)

	route, found := bApp.IBCKeeper.PortKeeper.Router.Route(ibctransfertypes.ModuleName)
	require.True(t, found)
	require.IsType(t, ratelimit.IBCMiddleware{}, route)
}

func TestRateLimit_GovAuthorityManagesRateLimits(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)
	authority := govAuthority()

	add := &ratelimittypes.MsgAddRateLimit{
		Authority:         authority,
		Denom:             rlDenom,
		ChannelOrClientId: rlChannel,
		MaxPercentSend:    sdkmath.NewInt(2),
		MaxPercentRecv:    sdkmath.NewInt(2),
		DurationHours:     24,
	}
	require.NoError(t, handle(bApp, ctx, add))

	stored, found := bApp.RatelimitKeeper.GetRateLimit(ctx, rlDenom, rlChannel)
	require.True(t, found)
	require.Equal(t, sdkmath.NewInt(1_000_000), stored.Flow.ChannelValue)
	require.Equal(t, sdkmath.NewInt(2), stored.Quota.MaxPercentSend)

	require.NoError(t, handle(bApp, ctx, &ratelimittypes.MsgUpdateRateLimit{
		Authority:         authority,
		Denom:             rlDenom,
		ChannelOrClientId: rlChannel,
		MaxPercentSend:    sdkmath.NewInt(5),
		MaxPercentRecv:    sdkmath.NewInt(0),
		DurationHours:     12,
	}))
	stored, _ = bApp.RatelimitKeeper.GetRateLimit(ctx, rlDenom, rlChannel)
	require.Equal(t, sdkmath.NewInt(5), stored.Quota.MaxPercentSend)
	require.True(t, stored.Quota.MaxPercentRecv.IsZero())
	require.Equal(t, uint64(12), stored.Quota.DurationHours)

	require.NoError(t, handle(bApp, ctx, &ratelimittypes.MsgResetRateLimit{
		Authority: authority, Denom: rlDenom, ChannelOrClientId: rlChannel,
	}))

	require.NoError(t, handle(bApp, ctx, &ratelimittypes.MsgRemoveRateLimit{
		Authority: authority, Denom: rlDenom, ChannelOrClientId: rlChannel,
	}))
	_, found = bApp.RatelimitKeeper.GetRateLimit(ctx, rlDenom, rlChannel)
	require.False(t, found)
}

func TestRateLimit_NonAuthorityIsRefused(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)
	stranger := sdk.AccAddress("rl_not_the_authority").String()

	err := handle(bApp, ctx, &ratelimittypes.MsgAddRateLimit{
		Authority:         stranger,
		Denom:             rlDenom,
		ChannelOrClientId: rlChannel,
		MaxPercentSend:    sdkmath.NewInt(2),
		MaxPercentRecv:    sdkmath.NewInt(2),
		DurationHours:     24,
	})
	require.ErrorIs(t, err, govtypes.ErrInvalidSigner)
	require.Empty(t, bApp.RatelimitKeeper.GetAllRateLimits(ctx))
}

// A pair without a channel or supply cannot be limited, which is what lets the
// initial-limits step skip missing paths later.
func TestRateLimit_AddForUnknownDenomIsRefused(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)

	err := handle(bApp, ctx, &ratelimittypes.MsgAddRateLimit{
		Authority:         govAuthority(),
		Denom:             "uunknown",
		ChannelOrClientId: rlChannel,
		MaxPercentSend:    sdkmath.NewInt(2),
		MaxPercentRecv:    sdkmath.NewInt(2),
		DurationHours:     24,
	})
	require.ErrorIs(t, err, ratelimittypes.ErrZeroChannelValue)
}

// The ratelimit module has no VersionMap entry before the upgrade: the handler
// initialises it from its default genesis (hour epoch from the block time, no
// rate limits, empty blacklist and whitelist).
func TestRateLimit_UpgradeHandlerInitialisesModule(t *testing.T) {
	bApp, ctx := newRateLimitApp(t)

	vm := bApp.ModuleManager.GetVersionMap()
	delete(vm, ratelimittypes.ModuleName)

	newVM, err := v830.CreateUpgradeHandler(bApp.Configurator(), bApp.ModuleManager, bApp.GuardianKeeper)(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, vm,
	)
	require.NoError(t, err)
	require.Equal(t, uint64(2), newVM[ratelimittypes.ModuleName])

	epoch := bApp.RatelimitKeeper.GetHourEpoch(ctx)
	require.Equal(t, uint64(13), epoch.EpochNumber)
	require.Equal(t, time.Date(2026, 11, 2, 13, 0, 0, 0, time.UTC), epoch.EpochStartTime)
	require.Equal(t, time.Hour, epoch.Duration)
	require.Empty(t, bApp.RatelimitKeeper.GetAllRateLimits(ctx))
	require.Empty(t, bApp.RatelimitKeeper.GetAllBlacklistedDenoms(ctx))
	require.Empty(t, bApp.RatelimitKeeper.GetAllWhitelistedAddressPairs(ctx))
}
