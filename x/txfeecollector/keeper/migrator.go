package keeper

import (
	"cosmossdk.io/store/prefix"
	v2 "github.com/bze-alphateam/bze/x/txfeecollector/migrations/v2"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(k Keeper) Migrator {
	return Migrator{
		keeper: k,
	}
}

// Migrate1to2 migrates the x/txfeecollector module state from consensus version 1 to
// version 2. It sets default parameters for the module.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	adapter := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte{})

	m.keeper.Logger().Info("migrating x/txfeecollector state from consensus version 1 to version 2")

	return v2.Migrate(ctx, store, m.keeper.cdc)
}
