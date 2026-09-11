package tradebin_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	tradebin "github.com/bze-alphateam/bze/x/tradebin/module"
	"github.com/bze-alphateam/bze/x/tradebin/testutil"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// wiringFixture is a real tradebin keeper (in-memory store, default params) held by POINTER, the
// way depinject hands app.TradebinKeeper to app.go and to the module (ModuleOutputs{TradebinKeeper: &k}).
type wiringFixture struct {
	k    *keeper.Keeper
	ctx  sdk.Context
	bank *testutil.MockBankKeeper
	acc  *testutil.MockAccountKeeper
}

func newWiringFixture(t *testing.T) wiringFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	bank := testutil.NewMockBankKeeper(ctrl)
	acc := testutil.NewMockAccountKeeper(ctrl)
	k, ctx := keepertest.TradebinKeeper(t, bank, acc)

	return wiringFixture{k: &k, ctx: ctx, bank: bank, acc: acc}
}

// registerServices does what appBuilder.Build does: module manager RegisterServices through the real
// SDK configurator, which is where tradebin's RegisterServices builds the msg server. The returned
// msg service router is what BaseApp dispatches transactions through.
func (f wiringFixture) registerServices(t *testing.T) *baseapp.MsgServiceRouter {
	t.Helper()
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(registry)
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(registry)
	cfg := module.NewConfigurator(cdc, msr, qr)

	mm := module.NewManager(tradebin.NewAppModule(cdc, f.k, f.acc, f.bank, nil))
	require.NoError(t, mm.RegisterServices(cfg))

	return msr
}

// swapThroughRouter seeds a pool and dispatches a MsgMultiSwap through the msg service router (the
// path BaseApp uses for a tx), then checks the swap really executed. Returns the swapper.
func (f wiringFixture) swapThroughRouter(t *testing.T, msr *baseapp.MsgServiceRouter) sdk.AccAddress {
	t.Helper()
	creator := sdk.AccAddress("hooks-wiring-swapper")
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         "ubze",
		Quote:        "stake",
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1_000_000),
		ReserveQuote: math.NewInt(2_000_000),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(3, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(4, 1),
		},
		Creator: creator.String(),
	}
	f.k.SetLiquidityPool(f.ctx, pool)

	f.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), creator, types.ModuleName, gomock.Any()).Return(nil).AnyTimes()
	f.bank.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	f.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, creator, gomock.Any()).Return(nil).AnyTimes()
	f.acc.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(authtypes.NewEmptyModuleAccount(types.ModuleName)).AnyTimes()

	msg := &types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     sdk.NewCoin("ubze", math.NewInt(1000)),
		MinOutput: sdk.NewCoin("stake", math.NewInt(1900)),
	}
	handler := msr.Handler(msg)
	require.NotNil(t, handler, "MsgMultiSwap must be routable after RegisterServices")
	_, err := handler(f.ctx, msg)
	require.NoError(t, err)

	updated, found := f.k.GetLiquidityPool(f.ctx, "pool1")
	require.True(t, found)
	require.Equal(t, math.NewInt(1_001_000), updated.ReserveBase, "the swap itself must have executed")

	return creator
}

func recordingHook(calls *[]string) types.OnMarketOrderFill {
	return func(_ sdk.Context, marketId, amountTraded, userAddress string) {
		*calls = append(*calls, marketId+" "+amountTraded+" "+userAddress)
	}
}

// TestOnOrderFillHooks_RegisteredAfterRegisterServices_FireOnAmmSwap reproduces the app's wiring
// order since v8.0.0: appBuilder.Build (RegisterServices) constructs the msg server first, and only
// afterwards does app.setTradebinHooks() register the order-fill hooks. A msg server holding a
// by-value keeper copy froze the empty hook slice at registration time, so AMM swaps never invoked
// the hooks (the orderbook path was unaffected: EndBlock builds its ProcessingEngine from the keeper
// pointer every block). With the pointer shared, a swap dispatched through the real msg service
// router fires the hook registered later, with the pool id and the base amount traded.
func TestOnOrderFillHooks_RegisteredAfterRegisterServices_FireOnAmmSwap(t *testing.T) {
	f := newWiringFixture(t)

	// 1. RegisterServices — the msg server is built here, before any hook exists
	msr := f.registerServices(t)

	// 2. hooks are registered afterwards, exactly like app.go does after appBuilder.Build
	var calls []string
	f.k.SetOnOrderFillHooks([]types.OnMarketOrderFill{recordingHook(&calls)})

	// 3. the swap goes through the router the way BaseApp delivers a tx
	creator := f.swapThroughRouter(t, msr)

	require.Equal(t, []string{"pool1 1000 " + creator.String()}, calls, "the hook registered after RegisterServices must fire on an AMM swap")
}

// The pre-v8 wiring order (hooks first, RegisterServices later) keeps working too: the order of
// registration must not matter.
func TestOnOrderFillHooks_RegisteredBeforeRegisterServices_FireOnAmmSwap(t *testing.T) {
	f := newWiringFixture(t)

	var calls []string
	f.k.SetOnOrderFillHooks([]types.OnMarketOrderFill{recordingHook(&calls)})
	msr := f.registerServices(t)

	creator := f.swapThroughRouter(t, msr)

	require.Equal(t, []string{"pool1 1000 " + creator.String()}, calls)
}

// The mechanism, isolated: the msg server must read the hooks from the keeper it was built with,
// not from a snapshot. NewMsgServerImpl(f.k) is the expression RegisterServices uses; EndBlock's
// ProcessingEngine reads the same pointer. Both views must agree after a later SetOnOrderFillHooks.
func TestOnOrderFillHooks_MsgServerSharesTheKeeper(t *testing.T) {
	f := newWiringFixture(t)
	srv := keeper.NewMsgServerImpl(f.k) // built before hooks exist, as in RegisterServices

	f.k.SetOnOrderFillHooks([]types.OnMarketOrderFill{recordingHook(new([]string))})

	// msgServer embeds the keeper, so its promoted GetOnOrderFillHooks is reachable through an interface
	view, ok := srv.(interface {
		GetOnOrderFillHooks() []types.OnMarketOrderFill
	})
	require.True(t, ok)
	require.Len(t, f.k.GetOnOrderFillHooks(), 1)
	require.Len(t, view.GetOnOrderFillHooks(), 1, "the msg server must see hooks registered after it was built")
}
