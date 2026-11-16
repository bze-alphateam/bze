package keeper

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/burner/exported"
	"github.com/bze-alphateam/bze/x/burner/migrations/v4"
	v5 "github.com/bze-alphateam/bze/x/burner/migrations/v5"
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

// Migrate3to4 migrates the x/burner module state from the consensus version 1 to
// version 2. Specifically, it takes the parameters that are currently stored
// and managed by the x/params modules and stores them directly into the x/burner
// module state.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	adapter := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte{})

	m.keeper.Logger().Info("migrating x/burner state from consensus version 3 to version 4")

	return v4.Migrate(store, m.keeper.cdc)
}

// Migrate4to5 migrates the x/burner module state from consensus version 4 to version 5: it gives the stored params the
// default limits of MsgMoveIbcLockedCoins.
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	m.keeper.Logger().Info("migrating x/burner state from consensus version 4 to version 5")

	return v5.MigrateParams(ctx, m.keeper)
}
