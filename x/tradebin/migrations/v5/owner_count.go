package v5

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PriceLevelOwnersKeeper recounts the distinct owners of every price level.
type PriceLevelOwnersKeeper interface {
	RecountPriceLevelOwners(ctx sdk.Context)
}

// MigratePriceLevelOwners writes the distinct-owner count into every aggregate. It runs after
// MigrateOwnerPriceIndex and counts the index entries of each level, not its orders: the duplicate
// orders one owner may rest at one price from before version 5 count once.
func MigratePriceLevelOwners(ctx sdk.Context, k PriceLevelOwnersKeeper) {
	k.RecountPriceLevelOwners(ctx)
}
