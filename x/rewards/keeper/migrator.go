package keeper

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/rewards/exported"
	v3 "github.com/bze-alphateam/bze/x/rewards/migrations/v3"
	v4 "github.com/bze-alphateam/bze/x/rewards/migrations/v4"
	v5 "github.com/bze-alphateam/bze/x/rewards/migrations/v5"
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

// Migrate2to3 migrates the x/rewards module state from the consensus version 2 to
// version 3. Specifically, it takes the parameters that are currently stored
// and managed by the x/params modules and stores them directly into the x/rewards
// module state.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	adapter := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte{})

	m.keeper.Logger().Info("migrating x/rewards state from consensus version 2 to version 3")

	return v3.Migrate(ctx, store, m.legacySubspace, m.keeper.cdc)
}

// Migrate3to4 migrates the x/rewards module state from consensus version 3 to
// version 4. It adds the new ExtraGasForExitStake parameter with default value.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	adapter := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte{})

	m.keeper.Logger().Info("migrating x/rewards state from consensus version 3 to version 4")

	return v4.Migrate(ctx, store, m.keeper.cdc)
}

// Migrate4to5 migrates the x/rewards module state from consensus version 4 to
// version 5. It backfills the boost participant reverse index for every existing
// participant (so pre-upgrade participants are reachable by the boost
// finalization sweep — Security A5) and sets the new boost params
// (create_boost_fee, max_boosts_per_reward) to their defaults.
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	adapter := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte{})

	m.keeper.Logger().Info("migrating x/rewards state from consensus version 4 to version 5")

	return v5.Migrate(ctx, store, m.keeper.cdc)
}
