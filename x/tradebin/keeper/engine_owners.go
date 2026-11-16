package keeper

import (
	"fmt"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The aggregate of a price level counts the distinct owners resting there (BZE-250). The count is the
// number of owner-price index entries at that market, side and price: the engine adds one right before
// a SaveOrder that creates an entry and removes one right before a RemoveOrder that deletes one. The
// checks below mirror IndexOwnerPriceOrder and unindexOwnerPriceOrder: SaveOrder creates the entry
// only when the owner has none at the level (pointing it at a lower id changes nothing) and RemoveOrder
// deletes it only when it points at the removed order.

// createsOwnerPriceEntry reports whether saving the order creates its owner's index entry at the level.
func (pe *ProcessingEngine) createsOwnerPriceEntry(ctx sdk.Context, order types.Order) bool {
	_, found := pe.k.GetOwnerPriceOrder(ctx, order.Owner, order.MarketId, order.OrderType, order.Price)

	return !found
}

// removesOwnerPriceEntry reports whether removing the order deletes its owner's index entry at the level.
func (pe *ProcessingEngine) removesOwnerPriceEntry(ctx sdk.Context, order types.Order) bool {
	ref, found := pe.k.GetOwnerPriceOrder(ctx, order.Owner, order.MarketId, order.OrderType, order.Price)

	return found && ref.Id == order.Id
}

// removeOwner takes one owner off the aggregate. A count already at 0 means the aggregate and the index
// disagree: it fails instead of wrapping around.
func removeOwner(agg *types.AggregatedOrder) error {
	if agg.Owners == 0 {
		return fmt.Errorf("price level %s %s %s has no owner to remove", agg.MarketId, agg.OrderType, agg.Price)
	}
	agg.Owners--

	return nil
}

// checkEmptyLevelOwners fails when an aggregate whose amount reached zero still counts owners.
func checkEmptyLevelOwners(agg *types.AggregatedOrder) error {
	if agg.Owners != 0 {
		return fmt.Errorf("emptied price level %s %s %s still counts %d owners", agg.MarketId, agg.OrderType, agg.Price, agg.Owners)
	}

	return nil
}
