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

// The module's services go through the real SDK plumbing: both messages are
// routed by the MsgServiceRouter and Params by the GRPCQueryRouter.
func TestRegisterServices_RoutesMsgsAndQuery(t *testing.T) {
	k, _ := keepertest.GuardianKeeper(t, nil)
	cdc := keepertest.GuardianCodec()

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(cdc.InterfaceRegistry())
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(cdc.InterfaceRegistry())

	mm := module.NewManager(guardian.NewAppModule(cdc, &k))
	require.NoError(t, mm.RegisterServices(module.NewConfigurator(cdc, msr, qr)))

	require.NotNil(t, msr.Handler(&types.MsgUpdateParams{}))
	require.NotNil(t, msr.Handler(&types.MsgTightenRateLimit{}))
	require.NotNil(t, qr.Route("/bze.guardian.v1.Query/Params"))
}

func TestAutoCLIOptions(t *testing.T) {
	opts := guardian.AppModule{}.AutoCLIOptions()

	require.Equal(t, "bze.guardian.v1.Query", opts.Query.Service)
	require.Equal(t, "bze.guardian.v1.Msg", opts.Tx.Service)

	uses := map[string]string{}
	for _, m := range opts.Query.RpcCommandOptions {
		uses[m.RpcMethod] = m.Use
	}
	for _, m := range opts.Tx.RpcCommandOptions {
		uses[m.RpcMethod] = m.Use
	}

	require.Equal(t, "params", uses["Params"])
	require.Contains(t, uses["TightenRateLimit"], "tighten-rate-limit [denom] [channel-id] [max-percent-send] [max-percent-recv]")
	require.Contains(t, uses["UpdateParams"], "update-params")
}

func TestConsensusVersion(t *testing.T) {
	require.Equal(t, uint64(1), guardian.AppModule{}.ConsensusVersion())
}
