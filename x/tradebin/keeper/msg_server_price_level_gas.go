package keeper

import (
	stdmath "math"
	"math/bits"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const priceLevelGasDescriptor = "price level spam protection"

// consumePriceLevelGas charges a new owner joining a crowded price level (BZE-250). When the level on
// the order's side already holds more than priceLevelOwnersWindow distinct owners and the creator has
// no order resting there, it consumes (owners - window) * priceLevelExtraGas gas on the transaction's
// gas meter. Topping up an order the creator already rests at that price costs nothing extra, and no
// order is ever refused.
//
// The count is read before the order is matched: an order that crosses the book is charged for the
// level its remainder may rest on, as it stands when the transaction starts. Matching never changes
// the owners on the order's own side, so this is also the count the remainder joins.
func (k Keeper) consumePriceLevelGas(ctx sdk.Context, msg *types.MsgCreateOrder, params v2types.Params) {
	agg, found := k.GetAggregatedOrder(ctx, msg.MarketId, msg.OrderType, msg.Price)
	if !found || agg.Owners <= params.PriceLevelOwnersWindow {
		return
	}

	if _, resting := k.GetOwnerPriceOrder(ctx, msg.Creator, msg.MarketId, msg.OrderType, msg.Price); resting {
		return
	}

	ctx.GasMeter().ConsumeGas(priceLevelExtraGas(agg.Owners-params.PriceLevelOwnersWindow, params.PriceLevelExtraGas), priceLevelGasDescriptor)
}

// priceLevelExtraGas multiplies without wrapping: a product above the uint64 range is the maximum gas,
// which no transaction can pay.
func priceLevelExtraGas(owners, gasPerOwner uint64) uint64 {
	hi, lo := bits.Mul64(owners, gasPerOwner)
	if hi != 0 {
		return stdmath.MaxUint64
	}

	return lo
}
