package keeper

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/tx/signing"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/keeper"
	"github.com/bze-alphateam/bze/x/dao/types"
)

// DaoCodec returns a codec that knows the x/dao messages and the bank and
// distribution messages the executor tests run as a DAO.
func DaoCodec() codec.Codec {
	cfg := sdk.GetConfig()
	registry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          addresscodec.NewBech32Codec(cfg.GetBech32AccountAddrPrefix()),
			ValidatorAddressCodec: addresscodec.NewBech32Codec(cfg.GetBech32ValidatorAddrPrefix()),
		},
	})
	if err != nil {
		panic(err)
	}
	types.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	distrtypes.RegisterInterfaces(registry)

	return codec.NewProtoCodec(registry)
}

// DaoKeeper returns an x/dao keeper on an in-memory store. The router may be
// nil for tests that execute no messages as a DAO.
func DaoKeeper(t testing.TB, cdc codec.Codec, router baseapp.MessageRouter, acc types.AccountKeeper, bank types.BankKeeper, rewards types.RewardsKeeper) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		acc,
		bank,
		rewards,
		router,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ctx
}
