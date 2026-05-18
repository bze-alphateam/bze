package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/tx/signing"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/daodao/keeper"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// DaodaoKeeper builds an in-memory daodao keeper for tests.
//
// Callers pass mocked or real implementations of the expected keepers. The
// pattern mirrors testutil/keeper/rewards.go and testutil/keeper/burner.go.
func DaodaoKeeper(
	t testing.TB,
	acc types.AccountKeeper,
	bank types.BankKeeper,
	distr types.DistrKeeper,
	rewards types.RewardsKeeper,
) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	// Epic 5: the codec's signing context needs an AddressCodec to extract
	// signers from registered sdk.Msgs. The default NewInterfaceRegistry()
	// builds without one, which trips GetMsgV1Signers ("InterfaceRegistry
	// requires a proper address codec ...") the first time we validate a
	// proposal's msgs[]. Use the same bech32 codec the chain runs with.
	registry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: gogoproto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
			ValidatorAddressCodec: address.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		},
	})
	require.NoError(t, err)
	// Register daodao's own sdk.Msg interfaces so the codec's signing
	// context can resolve them when tests put daodao messages into a
	// proposal's msgs[] bundle. Tests that include non-daodao messages
	// (e.g. bank.MsgSend) should call types.RegisterInterfaces from those
	// modules on the same registry — exposed via the suite if needed.
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		acc,
		bank,
		distr,
		rewards,
	)

	// Initialize BlockTime to a fixed non-zero value. The default sdk.Context
	// has time.Time{} (year 1) whose UnixNano() is negative and overflows
	// when cast to uint64 — which breaks the Epic 3 expiring-proposal queue
	// (keyed by uint64(BlockTime.UnixNano())). 2024-01-01 is far enough in
	// the past not to interfere with realistic future-dated voting_end
	// arithmetic and far enough after Unix epoch that all relevant
	// timestamps fit comfortably in int64.
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := sdk.NewContext(
		stateStore,
		cmtproto.Header{Time: startTime},
		false,
		log.NewNopLogger(),
	)

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ctx
}
