package tokenfactory_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/tokenfactory/keeper"
	tokenfactory "github.com/bze-alphateam/bze/x/tokenfactory/module"
	"github.com/bze-alphateam/bze/x/tokenfactory/testutil"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// registerServices does what appBuilder.Build does: module manager RegisterServices through the real
// SDK configurator. The returned msg service router is what BaseApp dispatches transactions through.
func registerServices(t *testing.T) (keeper.Keeper, sdk.Context, *baseapp.MsgServiceRouter) {
	t.Helper()
	ctrl := gomock.NewController(t)
	bank := testutil.NewMockBankKeeper(ctrl)
	acc := testutil.NewMockAccountKeeper(ctrl)
	k, ctx := keepertest.TokenfactoryKeeper(t, bank, acc, nil)

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(registry)
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(registry)
	cfg := module.NewConfigurator(cdc, msr, qr)

	mm := module.NewManager(tokenfactory.NewAppModule(cdc, k, acc, bank, nil, nil))
	require.NoError(t, mm.RegisterServices(cfg))

	return k, ctx, msr
}

// A passed proposal executes MsgSetDenomAdmin through the msg service router and the admin changes.
func TestSetDenomAdmin_GovAuthorityThroughRouter(t *testing.T) {
	k, ctx, msr := registerServices(t)
	creator := sdk.AccAddress("wiring-creator").String()
	newAdmin := sdk.AccAddress("wiring-dao").String()
	denom := "factory/" + creator + "/token"
	require.NoError(t, k.SetDenomAuthority(ctx, denom, types.DenomAuthority{Admin: ""}))

	msg := types.NewMsgSetDenomAdmin(authtypes.NewModuleAddress(govtypes.ModuleName).String(), denom, newAdmin)
	handler := msr.Handler(msg)
	require.NotNil(t, handler, "MsgSetDenomAdmin must be routable after RegisterServices")
	_, err := handler(ctx, msg)
	require.NoError(t, err)

	dAuth, err := k.GetDenomAuthority(ctx, denom)
	require.NoError(t, err)
	require.Equal(t, newAdmin, dAuth.Admin)
}

// Any other signer is refused on the router path too and changes nothing.
func TestSetDenomAdmin_NonAuthorityRejectedThroughRouter(t *testing.T) {
	k, ctx, msr := registerServices(t)
	creator := sdk.AccAddress("wiring-creator").String()
	denom := "factory/" + creator + "/token"
	require.NoError(t, k.SetDenomAuthority(ctx, denom, types.DenomAuthority{Admin: creator}))

	msg := types.NewMsgSetDenomAdmin(sdk.AccAddress("not-the-gov-account_").String(), denom, sdk.AccAddress("wiring-dao").String())
	_, err := msr.Handler(msg)(ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidSigner)

	dAuth, err := k.GetDenomAuthority(ctx, denom)
	require.NoError(t, err)
	require.Equal(t, creator, dAuth.Admin)
}
