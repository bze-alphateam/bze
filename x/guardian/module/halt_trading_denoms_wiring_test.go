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

// MsgHaltTradingDenoms is routed by the real MsgServiceRouter.
func TestRegisterServices_RoutesHaltTradingDenoms(t *testing.T) {
	k, _ := keepertest.GuardianKeeper(t, nil)
	cdc := keepertest.GuardianCodec()

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(cdc.InterfaceRegistry())
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(cdc.InterfaceRegistry())

	mm := module.NewManager(guardian.NewAppModule(cdc, &k))
	require.NoError(t, mm.RegisterServices(module.NewConfigurator(cdc, msr, qr)))

	require.NotNil(t, msr.Handler(&types.MsgHaltTradingDenoms{}))
}

// halt-trading-denoms is a plain tx (signed by the guardian, not a gov
// proposal) taking the denoms as variadic positional args.
func TestAutoCLIOptions_HaltTradingDenoms(t *testing.T) {
	opts := guardian.AppModule{}.AutoCLIOptions()

	for _, m := range opts.Tx.RpcCommandOptions {
		if m.RpcMethod != "HaltTradingDenoms" {
			continue
		}
		require.Equal(t, "halt-trading-denoms [denom]...", m.Use)
		require.False(t, m.GovProposal)
		require.Len(t, m.PositionalArgs, 1)
		require.Equal(t, "denoms", m.PositionalArgs[0].ProtoField)
		require.True(t, m.PositionalArgs[0].Varargs)
		return
	}
	t.Fatal("no autocli entry for HaltTradingDenoms")
}
