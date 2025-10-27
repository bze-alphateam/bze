package keeper

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/tradebin/exported"
	"github.com/bze-alphateam/bze/x/tradebin/migrations/v3"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Migrator struct {
	keeper         *Keeper
	legacySubspace exported.Subspace
}

func NewMigrator(k *Keeper, ss exported.Subspace) Migrator {
	return Migrator{
		keeper:         k,
		legacySubspace: ss,
	}
}

// Migrate2to3 migrates the x/tradebin module state from the consensus version 1 to
// version 2. Specifically, it takes the parameters that are currently stored
// and managed by the x/params modules and stores them directly into the x/tradebin
// module state.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	adapter := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte{})

	m.keeper.Logger().Info("migrating x/tradebin state from consensus version 2 to version 3")

	return v3.Migrate(ctx, store, m.legacySubspace, m.keeper.cdc)
}
