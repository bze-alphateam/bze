package v5

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// OwnerPriceKeeper reads the resting orders and writes the owner-price index.
type OwnerPriceKeeper interface {
	GetAllOrder(ctx sdk.Context) []types.Order
	IndexOwnerPriceOrder(ctx sdk.Context, order types.Order)
}

// MigrateOwnerPriceIndex builds the owner-price index (Tb/op/) from every resting order. From consensus
// version 5 a new order of an owner at a market, side and price where that owner already rests an
// order is added to that order.
//
// Before version 5 one owner could rest several orders at one price. They stay separate orders and the
// index points at the lowest id of them (the first in FIFO order), so later orders of that owner at
// that price are added to the earliest one.
//
// The orders are read first, the index is written after their iterator is closed.
func MigrateOwnerPriceIndex(ctx sdk.Context, k OwnerPriceKeeper) {
	for _, order := range k.GetAllOrder(ctx) {
		k.IndexOwnerPriceOrder(ctx, order)
	}
}
