package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// assignPlaceOrderId gives a limit order a new id from the order counter. When its owner already rests
// an order at the same market, side and price (BZE-249), that order must exist: a resting remainder is
// merged into it under the new id (see restOrder).
func (pe *ProcessingEngine) assignPlaceOrderId(ctx sdk.Context, order types.Order) (types.Order, error) {
	ref, found := pe.k.GetOwnerPriceOrder(ctx, order.Owner, order.MarketId, order.OrderType, order.Price)
	if found {
		if _, found = pe.k.GetOrder(ctx, ref.MarketId, ref.OrderType, ref.Id); !found {
			return order, fmt.Errorf("order %s of the owner-price index not found", ref.Id)
		}
	}

	return pe.k.AssignOrderId(ctx, order), nil
}

// restOrder rests the amount of the message on the book. When its owner already rests an order at that
// market, side and price, the two are merged into one order under the id of the message, at the back of
// the price level (see replaceRestingOrder). Otherwise the message is saved as a new order.
// Either way the amount is added to the aggregate of the price level, and the owner is counted there
// when the order created its owner-price index entry.
func (pe *ProcessingEngine) restOrder(ctx sdk.Context, message types.Order) error {
	ref, found := pe.k.GetOwnerPriceOrder(ctx, message.Owner, message.MarketId, message.OrderType, message.Price)
	if !found {
		newOwner := pe.createsOwnerPriceEntry(ctx, message)
		order := pe.saveOrder(ctx, message, message.Amount)
		pe.logger.Info("order saved on the book", "order", *order)

		return pe.addOrderToAggregate(ctx, order, newOwner)
	}

	existing, found := pe.k.GetOrder(ctx, ref.MarketId, ref.OrderType, ref.Id)
	if !found {
		return fmt.Errorf("order %s of the owner-price index not found", ref.Id)
	}

	return pe.replaceRestingOrder(ctx, existing, message)
}

// replaceRestingOrder merges the message into the order its owner rests at the same price (BZE-267):
// the merged order (existing amount + message amount) is saved under the id and creation time of the
// message, so it goes to the back of the price level, and the existing order is removed. A top-up does
// not keep the place of the resting order.
//
// RemoveOrder deletes the owner-price index entry (it points at the existing order) and SaveOrder
// creates it again for the new id, so the net effect on the index is the entry pointing at the new id:
// the owner stays counted at the level once and the count is not touched. The coins of the existing
// order stay escrowed in the module, the aggregate grows by the message amount only.
func (pe *ProcessingEngine) replaceRestingOrder(ctx sdk.Context, existing, message types.Order) error {
	// stored amounts may predate canonical validation; keep this parse as is
	existingAmount, ok := math.NewIntFromString(existing.Amount)
	if !ok {
		return fmt.Errorf("could not convert order amount")
	}

	addedAmount, ok := math.NewIntFromString(message.Amount)
	if !ok {
		return fmt.Errorf("could not convert order amount")
	}

	merged := existing
	merged.Id = message.Id
	merged.CreatedAt = message.CreatedAt
	merged.Amount = existingAmount.Add(addedAmount).String()

	pe.k.RemoveOrder(ctx, existing)
	pe.k.SaveOrder(ctx, merged)
	pe.emitOrderAmendedEvent(ctx, &merged, addedAmount, existing.Id)
	pe.logger.Info("order amount added to the resting order under a new id", "order", merged, "previous_id", existing.Id)

	return pe.addOrderToAggregate(ctx, &message, false)
}

func (pe *ProcessingEngine) emitOrderAmendedEvent(ctx sdk.Context, order *types.Order, addedAmount math.Int, previousId string) {
	err := ctx.EventManager().EmitTypedEvent(
		&types.OrderAmendedEvent{
			Id:          order.Id,
			MarketId:    order.MarketId,
			OrderType:   order.OrderType,
			Price:       order.Price,
			Owner:       order.Owner,
			AddedAmount: addedAmount.String(),
			Amount:      order.Amount,
			PreviousId:  previousId,
		},
	)

	if err != nil {
		pe.logger.Error(err.Error())
	}
}
