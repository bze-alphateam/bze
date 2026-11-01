package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// assignPlaceOrderId gives a limit order its id (BZE-249): the id of the order its owner already rests
// at the same market, side and price, or a new one from the order counter.
func (pe *ProcessingEngine) assignPlaceOrderId(ctx sdk.Context, order types.Order) (types.Order, error) {
	ref, found := pe.k.GetOwnerPriceOrder(ctx, order.Owner, order.MarketId, order.OrderType, order.Price)
	if !found {
		return pe.k.AssignOrderId(ctx, order), nil
	}

	if _, found = pe.k.GetOrder(ctx, ref.MarketId, ref.OrderType, ref.Id); !found {
		return order, fmt.Errorf("order %s of the owner-price index not found", ref.Id)
	}
	order.Id = ref.Id

	return order, nil
}

// restOrder rests the amount of the message on the book. When the message carries the id of an order
// that rests already (its owner's order at that price), the amount is added to that order, which keeps
// its id, creation time and place in the book. Otherwise the message is saved as a new order.
// Either way the amount is added to the aggregate of the price level.
func (pe *ProcessingEngine) restOrder(ctx sdk.Context, message types.Order) error {
	existing, found := pe.k.GetOrder(ctx, message.MarketId, message.OrderType, message.Id)
	if !found {
		order := pe.saveOrder(ctx, message, message.Amount)
		pe.logger.Info("order saved on the book", "order", *order)

		return pe.addOrderToAggregate(ctx, order)
	}

	existingAmount, ok := math.NewIntFromString(existing.Amount)
	if !ok {
		return fmt.Errorf("could not convert order amount")
	}

	addedAmount, ok := math.NewIntFromString(message.Amount)
	if !ok {
		return fmt.Errorf("could not convert order amount")
	}

	existing.Amount = existingAmount.Add(addedAmount).String()
	pe.k.SaveOrder(ctx, existing)
	pe.emitOrderAmendedEvent(ctx, &existing, addedAmount)
	pe.logger.Info("order amount added to the resting order", "order", existing)

	return pe.addOrderToAggregate(ctx, &message)
}

func (pe *ProcessingEngine) emitOrderAmendedEvent(ctx sdk.Context, order *types.Order, addedAmount math.Int) {
	err := ctx.EventManager().EmitTypedEvent(
		&types.OrderAmendedEvent{
			Id:          order.Id,
			MarketId:    order.MarketId,
			OrderType:   order.OrderType,
			Price:       order.Price,
			Owner:       order.Owner,
			AddedAmount: addedAmount.String(),
			Amount:      order.Amount,
		},
	)

	if err != nil {
		pe.logger.Error(err.Error())
	}
}
