package keeper

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

type (
	Keeper struct {
		cdc           codec.BinaryCodec
		storeService  store.KVStoreService
		logger        log.Logger
		accountKeeper types.AccountKeeper
		bankKeeper    types.BankKeeper
		distrKeeper   types.DistrKeeper

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		onOrderFillHooks []types.OnMarketOrderFill
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	distrKeeper types.DistrKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		authority:     authority,
		logger:        logger,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
		distrKeeper:   distrKeeper,
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) getPrefixedStore(ctx sdk.Context, p []byte) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	return prefix.NewStore(storeAdapter, p)
}

func (k Keeper) largeZeroFillId(id uint64) string {
	return fmt.Sprintf("%024d", id)
}

func (k Keeper) smallZeroFillId(id uint64) string {
	return fmt.Sprintf("%012d", id)
}

func (k *Keeper) SetOnOrderFillHooks(hooks []types.OnMarketOrderFill) {
	k.onOrderFillHooks = hooks
}

func (k Keeper) GetOnOrderFillHooks() []types.OnMarketOrderFill {
	return k.onOrderFillHooks
}
