package dao_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/keeper"
	dao "github.com/bze-alphateam/bze/x/dao/module"
	"github.com/bze-alphateam/bze/x/dao/types"
)

// The module's services go through the real SDK plumbing: MsgUpdateParams is
// routed by the MsgServiceRouter and the Params query by the GRPCQueryRouter.
func TestRegisterServices_RoutesMsgAndQuery(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), log.NewNopLogger(), authority, nil, nil)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	msr := baseapp.NewMsgServiceRouter()
	msr.SetInterfaceRegistry(registry)
	qr := baseapp.NewGRPCQueryRouter()
	qr.SetInterfaceRegistry(registry)
	mm := module.NewManager(dao.NewAppModule(cdc, &k, nil, nil))
	require.NoError(t, mm.RegisterServices(module.NewConfigurator(cdc, msr, qr)))

	msg := &types.MsgUpdateParams{Authority: authority, Params: types.DefaultParams()}
	handler := msr.Handler(msg)
	require.NotNil(t, handler)
	_, err := handler(ctx, msg)
	require.NoError(t, err)

	bad := &types.MsgUpdateParams{Authority: sdk.AccAddress("not_the_gov_account_").String()}
	_, err = handler(ctx, bad)
	require.ErrorIs(t, err, types.ErrInvalidSigner)

	require.NotNil(t, qr.Route("/bze.dao.Query/Params"))
}

// The CLI exposes `bzed q dao params` and `bzed tx dao update-params`; the
// latter is authority gated, so it is submitted as a governance proposal.
func TestAutoCLIOptions(t *testing.T) {
	opts := dao.AppModule{}.AutoCLIOptions()

	require.Equal(t, "bze.dao.Query", opts.Query.Service)
	require.Len(t, opts.Query.RpcCommandOptions, 1)
	require.Equal(t, "Params", opts.Query.RpcCommandOptions[0].RpcMethod)
	require.Equal(t, "params", opts.Query.RpcCommandOptions[0].Use)

	require.Equal(t, "bze.dao.Msg", opts.Tx.Service)
	require.Len(t, opts.Tx.RpcCommandOptions, 1)
	updateParams := opts.Tx.RpcCommandOptions[0]
	require.Equal(t, "UpdateParams", updateParams.RpcMethod)
	require.Equal(t, "update-params [params]", updateParams.Use)
	require.False(t, updateParams.Skip)
	require.True(t, updateParams.GovProposal)
}
