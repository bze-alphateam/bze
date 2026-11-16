package tokenfactory_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	tokenfactory "github.com/bze-alphateam/bze/x/tokenfactory/module"
	"github.com/bze-alphateam/bze/x/tokenfactory/testutil"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// TestRunMigrations_TokenfactoryV3ToV4_ThroughModuleManager proves the 3->4 migration is registered
// under the module's name and version: a module manager holding {tokenfactory: 3} - the VersionMap
// the v8.3.0 upgrade handler receives - lands on ConsensusVersion 4 with the mintable flag set from
// the admin (BZE-217).
func TestRunMigrations_TokenfactoryV3ToV4_ThroughModuleManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	bank := testutil.NewMockBankKeeper(ctrl)
	acc := testutil.NewMockAccountKeeper(ctrl)
	k, ctx := keepertest.TokenfactoryKeeper(t, bank, acc, nil)

	admin := sdk.AccAddress("migration-admin").String()
	held := "factory/" + admin + "/held"
	renounced := "factory/" + admin + "/renounced"
	// consensus-version-3 records carry no flag
	require.NoError(t, k.SetDenomAuthority(ctx, held, types.DenomAuthority{Admin: admin}))
	require.NoError(t, k.SetDenomAuthority(ctx, renounced, types.DenomAuthority{Admin: ""}))

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(registry)
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(registry)
	cfg := module.NewConfigurator(cdc, msr, qr)

	am := tokenfactory.NewAppModule(cdc, k, acc, bank, nil, nil)
	mm := module.NewManager(am)
	require.NoError(t, mm.RegisterServices(cfg))
	require.Equal(t, uint64(4), am.ConsensusVersion())

	newVM, err := mm.RunMigrations(ctx, cfg, module.VersionMap{types.ModuleName: 3})
	require.NoError(t, err)
	require.Equal(t, uint64(4), newVM[types.ModuleName])

	dAuth, err := k.GetDenomAuthority(ctx, held)
	require.NoError(t, err)
	require.Equal(t, types.DenomAuthority{Admin: admin, Mintable: true}, dAuth)

	dAuth, err = k.GetDenomAuthority(ctx, renounced)
	require.NoError(t, err)
	require.Equal(t, types.DenomAuthority{Admin: "", Mintable: false}, dAuth)
}

// MsgDisableMinting is routable once the module registers its services.
func TestDisableMinting_ThroughRouter(t *testing.T) {
	k, ctx, msr := registerServices(t)
	admin := sdk.AccAddress("wiring-admin").String()
	denom := "factory/" + admin + "/token"
	require.NoError(t, k.SetDenomAuthority(ctx, denom, types.DenomAuthority{Admin: admin, Mintable: true}))

	msg := types.NewMsgDisableMinting(admin, denom)
	handler := msr.Handler(msg)
	require.NotNil(t, handler, "MsgDisableMinting must be routable after RegisterServices")
	_, err := handler(ctx, msg)
	require.NoError(t, err)

	dAuth, err := k.GetDenomAuthority(ctx, denom)
	require.NoError(t, err)
	require.False(t, dAuth.Mintable)
}

// MsgSetDenomConfig is routable once the module registers its services.
func TestSetDenomConfig_ThroughRouter(t *testing.T) {
	k, ctx, msr := registerServices(t)
	admin := sdk.AccAddress("wiring-admin").String()
	denom := "factory/" + admin + "/token"
	require.NoError(t, k.SetDenomAuthority(ctx, denom, types.DenomAuthority{Admin: admin, Mintable: true}))

	msg := types.NewMsgSetDenomConfig(admin, denom, "", math.LegacyMustNewDecFromStr("0.01"), math.LegacyOneDec())
	handler := msr.Handler(msg)
	require.NotNil(t, handler, "MsgSetDenomConfig must be routable after RegisterServices")
	_, err := handler(ctx, msg)
	require.NoError(t, err)

	transferFee, gasSurcharge := k.GetDenomFeeRates(ctx, denom)
	require.Equal(t, "0.010000000000000000", transferFee.String())
	require.Equal(t, "1.000000000000000000", gasSurcharge.String())
}
