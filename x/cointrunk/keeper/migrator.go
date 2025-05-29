package keeper

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/cointrunk/exported"
	v2 "github.com/bze-alphateam/bze/x/cointrunk/migrations/v2"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Migrator struct {
	keeper         Keeper
	legacySubspace exported.Subspace
}

func NewMigrator(k Keeper, ss exported.Subspace) Migrator {
	return Migrator{
		keeper:         k,
		legacySubspace: ss,
	}
}

func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	adapter := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte{})

	m.keeper.Logger().Info("migrating x/cointrunk state from consensus version 1 to version 2")

	err := v2.MigrateParams(ctx, store, m.legacySubspace, m.keeper.cdc)
	if err != nil {
		return err
	}
	m.keeper.Logger().Info("x/cointrunk params migrated from consensus version 1 to version 2")

	pStore := m.keeper.getPrefixedStore(ctx, types.KeyPrefix(types.PublisherKeyPrefix))

	err = v2.MigratePublishers(pStore, m.keeper.cdc)
	if err != nil {
		return err
	}

	m.keeper.Logger().Info("x/cointrunk publishers migrated from consensus version 1 to version 2")

	return nil
}
