package keeper

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/tradebin/exported"
	v3 "github.com/bze-alphateam/bze/x/tradebin/migrations/v3"
	v4 "github.com/bze-alphateam/bze/x/tradebin/migrations/v4"
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

// Migrate2to3 migrates the x/tradebin module state from the consensus version 2 to
// version 3. Specifically, it takes the parameters that are currently stored
// and managed by the x/params modules and stores them directly into the x/tradebin
// module state.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	adapter := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte{})

	m.keeper.Logger().Info("migrating x/tradebin state from consensus version 2 to version 3")

	return v3.Migrate(ctx, store, m.legacySubspace, m.keeper.cdc)
}

// Migrate3to4 migrates the x/tradebin module state from consensus version 3 to
// version 4. It adds new gas and liquidity parameters with default values.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	adapter := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte{})

	m.keeper.Logger().Info("migrating x/tradebin state from consensus version 3 to version 4")

	return v4.Migrate(ctx, store, m.keeper.cdc)
}
