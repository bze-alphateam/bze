package guardian_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	guardian "github.com/bze-alphateam/bze/x/guardian/module"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

// The channel switch messages and query go through the real SDK routers.
func TestRegisterServices_RoutesChannelSwitch(t *testing.T) {
	k, _ := keepertest.GuardianKeeper(t, nil)
	cdc := keepertest.GuardianCodec()

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(cdc.InterfaceRegistry())
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(cdc.InterfaceRegistry())

	mm := module.NewManager(guardian.NewAppModule(cdc, &k))
	require.NoError(t, mm.RegisterServices(module.NewConfigurator(cdc, msr, qr)))

	require.NotNil(t, msr.Handler(&types.MsgCloseChannel{}))
	require.NotNil(t, msr.Handler(&types.MsgReopenChannel{}))
	require.NotNil(t, qr.Route("/bze.guardian.v1.Query/ClosedChannels"))
}

func TestAutoCLIOptions_ChannelSwitch(t *testing.T) {
	opts := guardian.AppModule{}.AutoCLIOptions()

	query := map[string]string{}
	for _, m := range opts.Query.RpcCommandOptions {
		query[m.RpcMethod] = m.Use
	}
	require.Equal(t, "closed-channels", query["ClosedChannels"])

	gov := map[string]bool{}
	uses := map[string]string{}
	for _, m := range opts.Tx.RpcCommandOptions {
		uses[m.RpcMethod] = m.Use
		gov[m.RpcMethod] = m.GovProposal
	}
	require.Equal(t, "close-channel [channel-id]", uses["CloseChannel"])
	require.False(t, gov["CloseChannel"])
	require.Equal(t, "reopen-channel [channel-id]", uses["ReopenChannel"])
	require.True(t, gov["ReopenChannel"])
}
