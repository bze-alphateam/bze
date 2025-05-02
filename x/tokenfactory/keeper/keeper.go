package keeper

import (
	"cosmossdk.io/store/prefix"
	"fmt"
	"github.com/cosmos/cosmos-sdk/runtime"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		bankKeeper    types.BankKeeper
		distrkeeper   types.DistrKeeper
		accountKeeper types.AccountKeeper

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	bankKeeper types.BankKeeper,
	distrKeeper types.DistrKeeper,
	accountKeeper types.AccountKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		authority:     authority,
		logger:        logger,
		bankKeeper:    bankKeeper,
		distrkeeper:   distrKeeper,
		accountKeeper: accountKeeper,
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

// GetDenomPrefixStore returns the substore for a specific denom
func (k Keeper) GetDenomPrefixStore(ctx sdk.Context, denom string) prefix.Store {
	return k.getPrefixedStore(ctx, types.GetDenomPrefixStore(denom))
}

// GetCreatorPrefixStore returns the substore for a specific creator address
func (k Keeper) GetCreatorPrefixStore(ctx sdk.Context, creator string) prefix.Store {
	return k.getPrefixedStore(ctx, types.GetCreatorPrefix(creator))
}

// GetCreatorsPrefixStore returns the substore that contains a list of creators
func (k Keeper) GetCreatorsPrefixStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.GetCreatorsPrefix())
}

// InitGenesis initializes module accounts
func (k Keeper) InitGenesis(ctx sdk.Context) {
	_ = k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
}
