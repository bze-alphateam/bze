package guardian_test

import (
	"testing"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	guardian "github.com/bze-alphateam/bze/x/guardian/module"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

// A paused automation survives a genesis export and import: only governance
// resumes it.
func TestGenesis_AutomationPausedRoundTrip(t *testing.T) {
	k, ctx := keepertest.GuardianKeeper(t, nil)

	genesis := types.GenesisState{
		Params:           types.NewParams(sdk.AccAddress("guardian_address____").String()),
		AutomationPaused: true,
	}
	guardian.InitGenesis(ctx, &k, genesis)
	require.True(t, k.IsAutomationPaused(ctx))

	exported := guardian.ExportGenesis(ctx, &k)
	require.Equal(t, genesis, *exported)

	other, otherCtx := keepertest.GuardianKeeper(t, nil)
	guardian.InitGenesis(otherCtx, &other, *exported)
	require.True(t, other.IsAutomationPaused(otherCtx))
}

func TestGenesis_DefaultIsNotPaused(t *testing.T) {
	require.False(t, types.DefaultGenesis().AutomationPaused)

	k, ctx := keepertest.GuardianKeeper(t, nil)
	guardian.InitGenesis(ctx, &k, *types.DefaultGenesis())
	require.False(t, k.IsAutomationPaused(ctx))
	require.False(t, guardian.ExportGenesis(ctx, &k).AutomationPaused)
}

func TestRegisterServices_RoutesAutomationMsgsAndQuery(t *testing.T) {
	k, _ := keepertest.GuardianKeeper(t, nil)
	cdc := keepertest.GuardianCodec()

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(cdc.InterfaceRegistry())
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(cdc.InterfaceRegistry())

	mm := module.NewManager(guardian.NewAppModule(cdc, &k))
	require.NoError(t, mm.RegisterServices(module.NewConfigurator(cdc, msr, qr)))

	require.NotNil(t, msr.Handler(&types.MsgPauseAutomation{}))
	require.NotNil(t, msr.Handler(&types.MsgResumeAutomation{}))
	require.NotNil(t, qr.Route("/bze.guardian.v1.Query/AutomationPaused"))
}

func TestAutoCLIOptions_Automation(t *testing.T) {
	opts := guardian.AppModule{}.AutoCLIOptions()

	query := map[string]string{}
	for _, m := range opts.Query.RpcCommandOptions {
		query[m.RpcMethod] = m.Use
	}
	require.Equal(t, "automation-paused", query["AutomationPaused"])

	tx := map[string]*autocliv1.RpcCommandOptions{}
	for _, m := range opts.Tx.RpcCommandOptions {
		tx[m.RpcMethod] = m
	}

	pause, ok := tx["PauseAutomation"]
	require.True(t, ok, "pause-automation command")
	require.Equal(t, "pause-automation", pause.Use)
	require.False(t, pause.GovProposal, "the guardian signs it directly")

	resume, ok := tx["ResumeAutomation"]
	require.True(t, ok, "resume-automation command")
	require.Equal(t, "resume-automation", resume.Use)
	require.True(t, resume.GovProposal, "governance only")
}
