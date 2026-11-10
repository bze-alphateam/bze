package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// priceLevelFill is one resting order the message fills: the order with its amount already reduced,
// the executed amount and what is left of the order.
type priceLevelFill struct {
	order     types.Order
	amount    math.Int
	remaining math.Int
}

type priceLevelMatch struct {
	fills           []priceLevelFill
	maxFillsReached bool
}

// matchPriceLevel walks the resting orders at the message's price on the opposite side in FIFO order
// and decides how much of each one the message fills, without writing anything: the caller settles
// the fills after the iterator is closed. msgAmount and aggAmount are reduced by every fill.
//
// It stops, in this order of priority, when the message amount is spent, when it is below minAmount
// or when maxFills resting orders were examined. Every examined order counts, filled or skipped (an
// order the message can not fill is skipped and the walk goes on to the next one). maxFillsReached is
// set only when the last condition stopped it while another resting order was still there.
func (pe *ProcessingEngine) matchPriceLevel(ctx sdk.Context, message *types.Order, msgAmount, aggAmount *math.Int, minAmount math.Int) (priceLevelMatch, error) {
	var res priceLevelMatch
	var err error
	examined := uint64(0)

	pe.k.IteratePriceOrders(ctx, message.MarketId, types.TheOtherOrderType(message.OrderType), message.Price, func(orderRef types.OrderReference) bool {
		//stop when all message amount was spent
		if !msgAmount.IsPositive() {
			pe.logger.Debug("msg amount is not positive anymore. exiting oppositeOrderRefs iteration")
			return true
		}

		// msgAmount changes after each fill. In case it goes too low to be executed we can stop trying.
		// The caller of this function will refund the msg owner's coins.
		if msgAmount.LT(minAmount) {
			pe.logger.Debug("msg amount is too low. exiting oppositeOrderRefs iteration", "amount", msgAmount.String(), "minAmount", minAmount.String())
			return true
		}

		if examined >= pe.maxFills {
			pe.logger.Debug("max fills reached. exiting oppositeOrderRefs iteration", "max_fills", pe.maxFills)
			res.maxFillsReached = true
			return true
		}
		examined++

		orderToFill, _ := pe.k.GetOrder(ctx, orderRef.MarketId, orderRef.OrderType, orderRef.Id)
		pe.logger.Debug("preparing to fill message with order", "orderToFill", orderToFill)
		// stored amounts may predate canonical validation; keep this parse as is
		orderAmountInt, ok := math.NewIntFromString(orderToFill.Amount)
		if !ok {
			err = fmt.Errorf("could not convert order to fill amount")
			return true
		}

		//find how much to send to the found order's owner
		amountToExecute := pe.getExecutedAmount(*msgAmount, orderAmountInt, minAmount)
		if amountToExecute.IsZero() {
			return false
		}

		*msgAmount = msgAmount.Sub(amountToExecute)
		*aggAmount = aggAmount.Sub(amountToExecute)

		orderAmountInt = orderAmountInt.Sub(amountToExecute)
		orderToFill.Amount = orderAmountInt.String()

		res.fills = append(res.fills, priceLevelFill{
			order:     orderToFill,
			amount:    amountToExecute,
			remaining: orderAmountInt,
		})

		return false
	})

	if err != nil {
		return priceLevelMatch{}, err
	}

	return res, nil
}
