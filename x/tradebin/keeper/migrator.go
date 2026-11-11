package keeper

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/tradebin/exported"
	v3 "github.com/bze-alphateam/bze/x/tradebin/migrations/v3"
	v4 "github.com/bze-alphateam/bze/x/tradebin/migrations/v4"
	v5 "github.com/bze-alphateam/bze/x/tradebin/migrations/v5"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Migrator struct {
	keeper         *Keeper
	legacySubspace exported.Subspace
	bank           v5.BankKeeper
}

func NewMigrator(k *Keeper, ss exported.Subspace, bank v5.BankKeeper) Migrator {
	return Migrator{
		keeper:         k,
		legacySubspace: ss,
		bank:           bank,
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

// Migrate4to5 migrates the x/tradebin module state from consensus version 4 to
// version 5. Orders execute inside their transaction from version 5: every message
// still queued is refunded to its owner and the queue store is removed. Param field 11
// becomes maxFillsPerOrder and is set to its default if it is 0; fields 7 and 8 become
// priceLevelOwnersWindow (set to 500) and priceLevelExtraGas (kept). The owner-price index is
// built from the resting orders, then every aggregate gets its distinct-owner count. Last, the pool price
// average params are set to their defaults and every pool gets its price average record.
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	adapter := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte{})

	m.keeper.Logger().Info("migrating x/tradebin state from consensus version 4 to version 5")

	err := v5.Migrate(ctx, store, m.keeper, m.bank)
	if err != nil {
		return err
	}

	err = v5.MigrateParams(ctx, m.keeper)
	if err != nil {
		return err
	}

	v5.MigrateOwnerPriceIndex(ctx, m.keeper)
	v5.MigratePriceLevelOwners(ctx, m.keeper)

	return v5.MigratePoolPriceAverage(ctx, m.keeper)
}
