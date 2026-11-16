package txfeecollector_test

import (
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/txfeecollector/keeper"
	"github.com/bze-alphateam/bze/x/txfeecollector/testutil"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

func newKeeperWithGuardian(guardian types.GuardianKeeper) keeper.Keeper {
	return keeper.NewKeeper(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(types.StoreKey)),
		log.NewNopLogger(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil, nil, nil, nil, nil, guardian,
	)
}

// The keeper answers the closed-channel check of the IBC middleware and send gate
// from the guardian, channel by channel.
func TestKeeper_IsChannelClosedAsksTheGuardian(t *testing.T) {
	guardian := testutil.NewMockGuardianKeeper(gomock.NewController(t))
	guardian.EXPECT().IsChannelClosed(gomock.Any(), "channel-1").Return(true)
	guardian.EXPECT().IsChannelClosed(gomock.Any(), "channel-2").Return(false)

	k := newKeeperWithGuardian(guardian)

	require.True(t, k.IsChannelClosed(sdk.Context{}, "channel-1"))
	require.False(t, k.IsChannelClosed(sdk.Context{}, "channel-2"))
}

// A keeper built without a guardian (nil) has no channel closed.
func TestKeeper_NilGuardianClosesNoChannel(t *testing.T) {
	require.False(t, newKeeperWithGuardian(nil).IsChannelClosed(sdk.Context{}, "channel-1"))
}
