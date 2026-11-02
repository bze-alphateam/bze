package v830_test

import (
	"errors"
	"testing"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	v830testutil "github.com/bze-alphateam/bze/app/upgrades/v830/testutil"
)

var ubzeChannels = []string{
	"channel-0", "channel-2", "channel-3", "channel-6",
	"channel-10", "channel-11", "channel-12", "channel-13",
}

func TestInitialRateLimits_Policy(t *testing.T) {
	limits := v830.InitialRateLimits()
	require.Len(t, limits, 9)

	gov := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	seen := map[string]bool{}
	var ubzeSeen []string
	for _, l := range limits {
		require.Equal(t, uint64(24), l.DurationHours)
		require.NoError(t, sdk.ValidateDenom(l.Denom))
		require.NoError(t, host.ChannelIdentifierValidator(l.ChannelOrClientId))

		key := l.Denom + "|" + l.ChannelOrClientId
		require.False(t, seen[key], "duplicate %s", key)
		seen[key] = true

		l.Authority = gov
		require.NoError(t, l.ValidateBasic())

		if l.Denom == "ubze" {
			ubzeSeen = append(ubzeSeen, l.ChannelOrClientId)
			require.Equal(t, int64(2), l.MaxPercentSend.Int64())
			require.Equal(t, int64(2), l.MaxPercentRecv.Int64())
		}
	}
	require.ElementsMatch(t, ubzeChannels, ubzeSeen)

	var usdc []ratelimittypes.MsgAddRateLimit
	for _, l := range limits {
		if l.Denom == "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4" {
			usdc = append(usdc, l)
		}
	}
	require.Len(t, usdc, 1)
	require.Equal(t, "channel-3", usdc[0].ChannelOrClientId)
	require.Equal(t, int64(100), usdc[0].MaxPercentSend.Int64())
	require.True(t, usdc[0].MaxPercentRecv.IsZero())
}

func runHandler(t *testing.T, keeper v830.RateLimitKeeper) error {
	t.Helper()
	mm := module.NewManager()
	cfg := module.NewConfigurator(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		baseapp.NewMsgServiceRouter(),
		baseapp.NewGRPCQueryRouter(),
	)
	ctx := testutil.DefaultContext(storetypes.NewKVStoreKey("v830"), storetypes.NewTransientStoreKey("v830_t"))
	_, err := v830.CreateUpgradeHandler(cfg, mm, keeper)(ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{})
	return err
}

func TestUpgradeHandler_AddsEveryRateLimitWithGovAuthority(t *testing.T) {
	k := v830testutil.NewMockRateLimitKeeper(gomock.NewController(t))
	gov := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	var added []string
	k.EXPECT().AddRateLimit(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ sdk.Context, msg *ratelimittypes.MsgAddRateLimit) error {
			require.Equal(t, gov, msg.Authority)
			added = append(added, msg.Denom+"|"+msg.ChannelOrClientId)
			return nil
		}).Times(9)

	require.NoError(t, runHandler(t, k))
	require.Len(t, added, 9)
}

func TestUpgradeHandler_SkipsMissingPathsAndContinues(t *testing.T) {
	for _, skip := range []error{
		ratelimittypes.ErrChannelNotFound,
		ratelimittypes.ErrZeroChannelValue,
		ratelimittypes.ErrRateLimitAlreadyExists,
	} {
		k := v830testutil.NewMockRateLimitKeeper(gomock.NewController(t))
		calls := 0
		k.EXPECT().AddRateLimit(gomock.Any(), gomock.Any()).DoAndReturn(
			func(sdk.Context, *ratelimittypes.MsgAddRateLimit) error {
				calls++
				if calls%2 == 1 {
					return skip
				}
				return nil
			}).Times(9)

		require.NoError(t, runHandler(t, k), skip.Error())
		require.Equal(t, 9, calls)
	}
}

func TestUpgradeHandler_AbortsOnOtherRateLimitErrors(t *testing.T) {
	boom := errors.New("store failure")
	k := v830testutil.NewMockRateLimitKeeper(gomock.NewController(t))
	gomock.InOrder(
		k.EXPECT().AddRateLimit(gomock.Any(), gomock.Any()).Return(nil),
		k.EXPECT().AddRateLimit(gomock.Any(), gomock.Any()).Return(boom),
	)

	require.ErrorIs(t, runHandler(t, k), boom)
}
