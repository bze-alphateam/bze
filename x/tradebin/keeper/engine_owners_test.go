package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ownerPriceEntries counts the owner-price index entries at the market, side and price, read
// independently of the engine: every entry, matched to the level through the order it points at.
func (suite *IntegrationTestSuite) ownerPriceEntries(orderType, price string) uint64 {
	entries := uint64(0)
	for _, ref := range suite.k.GetAllOwnerPriceOrder(suite.ctx) {
		order, found := suite.k.GetOrder(suite.ctx, ref.MarketId, ref.OrderType, ref.Id)
		suite.Require().True(found, "index entry %s points at a missing order", ref.Id)
		// the aggregate key normalizes the price ("1" and "1.0" are one level)
		if string(types.AggOrderKey(order.MarketId, order.OrderType, order.Price)) == string(types.AggOrderKey(getMarketId(), orderType, price)) {
			entries++
		}
	}

	return entries
}

// distinctRestingOwners counts the distinct owners of the orders resting at the market, side and price
func (suite *IntegrationTestSuite) distinctRestingOwners(orderType, price string) uint64 {
	var owners []string
	suite.k.IteratePriceOrders(suite.ctx, getMarketId(), orderType, price, func(ref types.OrderReference) bool {
		order, found := suite.k.GetOrder(suite.ctx, ref.MarketId, ref.OrderType, ref.Id)
		suite.Require().True(found)
		for _, o := range owners {
			if o == order.Owner {
				return false
			}
		}
		owners = append(owners, order.Owner)

		return false
	})

	return uint64(len(owners))
}

// requireOwners checks the owner count of the level and its invariants: it equals the index entries at
// the level and never exceeds the distinct owners resting there. want 0 means the aggregate is gone.
func (suite *IntegrationTestSuite) requireOwners(orderType, price string, want uint64) {
	agg, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), orderType, price)
	if want == 0 {
		suite.Require().False(found, "aggregate still there with %d owners", agg.Owners)
		suite.Require().Zero(suite.ownerPriceEntries(orderType, price))
		suite.Require().Zero(suite.distinctRestingOwners(orderType, price))

		return
	}

	suite.Require().True(found)
	suite.Require().Equal(want, agg.Owners)
	suite.Require().Equal(suite.ownerPriceEntries(orderType, price), agg.Owners)
	suite.Require().LessOrEqual(agg.Owners, suite.distinctRestingOwners(orderType, price))
}

// fillOrder runs an immediate-or-cancel order on the suite context
func (suite *IntegrationTestSuite) fillOrder(orderType, amount, price string, owner sdk.AccAddress) {
	suite.Require().NoError(suite.newEngine().FillOrder(suite.ctx, limitOrder(orderType, amount, price, owner)))
}

func (suite *IntegrationTestSuite) cancelOrder(id string) {
	order, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, id)
	suite.Require().True(found)
	suite.Require().NoError(suite.newEngine().CancelOrder(suite.ctx, order))
}

// The owner count follows place (+1), merge (unchanged), partial fill (unchanged), full fill (-1) and
// cancel (-1); the aggregate goes away with the last owner.
func (suite *IntegrationTestSuite) TestEngineOwners_PlaceMergeFillCancel() {
	suite.k.SetMarket(suite.ctx, market)
	suite.allowBank()
	engine := suite.newEngine()
	a, b, c := makerAddr(0), makerAddr(1), makerAddr(2)
	taker := sdk.AccAddress("addr2_______________")

	suite.placeOrders(engine, limitOrder(types.OrderTypeSell, "10", "1", a))
	suite.requireOwners(types.OrderTypeSell, "1", 1)
	suite.placeOrders(engine, limitOrder(types.OrderTypeSell, "10", "1", b), limitOrder(types.OrderTypeSell, "10", "1", c))
	suite.requireOwners(types.OrderTypeSell, "1", 3)

	// a's top-up merges into its resting order
	suite.placeOrders(engine, limitOrder(types.OrderTypeSell, "5", "1.0", a))
	suite.Require().Len(suite.k.GetAllOrder(suite.ctx), 3)
	suite.requireOwners(types.OrderTypeSell, "1", 3)

	// a (15, first in line) is partly filled, then filled in full
	suite.fillOrder(types.OrderTypeBuy, "6", "1", taker)
	suite.requireOwners(types.OrderTypeSell, "1", 3)
	suite.fillOrder(types.OrderTypeBuy, "9", "1", taker)
	suite.requireOwners(types.OrderTypeSell, "1", 2)
	suite.requireNoOwnerPriceOrder(a, types.OrderTypeSell, "1")

	suite.cancelOrder(orderId(2))
	suite.requireOwners(types.OrderTypeSell, "1", 1)
	suite.cancelOrder(orderId(1))
	suite.requireOwners(types.OrderTypeSell, "1", 0)
}

// A taker that empties a level of several owners removes the aggregate with its count at 0, and the
// remainder resting on the other side starts that level with its owner.
func (suite *IntegrationTestSuite) TestEngineOwners_TakerEmptiesLevelAndRests() {
	suite.k.SetMarket(suite.ctx, market)
	suite.allowBank()
	suite.restSells(3, "10", "1")
	suite.requireOwners(types.OrderTypeSell, "1", 3)

	taker := sdk.AccAddress("addr2_______________")
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeBuy, "40", "1", taker))

	suite.requireOwners(types.OrderTypeSell, "1", 0)
	suite.requireOwners(types.OrderTypeBuy, "1", 1)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, "1", "10")
}

// seedDuplicates rests the book a pre-v8.3.0 chain could leave at the upgrade: owner x with two orders
// at one price (ids 5 and 7, the index points at 5) and owner y with one (id 6), with the count the
// migration writes.
func (suite *IntegrationTestSuite) seedDuplicates() (x, y sdk.AccAddress) {
	suite.k.SetMarket(suite.ctx, market)
	suite.k.SetOrderCounter(suite.ctx, 10)
	x, y = makerAddr(0), makerAddr(1)
	for _, o := range []types.Order{
		withId(limitOrder(types.OrderTypeSell, "10", "1", x), orderId(5)),
		withId(limitOrder(types.OrderTypeSell, "10", "1", y), orderId(6)),
		withId(limitOrder(types.OrderTypeSell, "10", "1", x), orderId(7)),
	} {
		suite.k.SaveOrder(suite.ctx, o)
	}
	suite.k.SetAggregatedOrder(suite.ctx, types.AggregatedOrder{MarketId: getMarketId(), OrderType: types.OrderTypeSell, Amount: "30", Price: "1"})
	suite.k.RecountPriceLevelOwners(suite.ctx)
	suite.requireOwnerPriceOrder(x, types.OrderTypeSell, "1", orderId(5))
	suite.requireOwners(types.OrderTypeSell, "1", 2)

	return x, y
}

// The indexed order of an owner is removed while a pre-upgrade duplicate of theirs still rests, the
// owner places again at that price, then the duplicate is removed. The count follows the index entries
// at every step: it may undercount the resting owners, never overcount them, and never goes below 0.
func (suite *IntegrationTestSuite) TestEngineOwners_PreUpgradeDuplicateDuplicateRemovedLast() {
	x, _ := suite.seedDuplicates()
	suite.allowBank()

	suite.cancelOrder(orderId(5))
	// x still rests order 7, but has no index entry any more
	suite.requireOwners(types.OrderTypeSell, "1", 1)
	suite.Require().Equal(uint64(2), suite.distinctRestingOwners(types.OrderTypeSell, "1"))

	// x has no entry, so the new order gets a new id and is counted again
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeSell, "10", "1", x))
	suite.requireOwnerPriceOrder(x, types.OrderTypeSell, "1", orderId(10))
	suite.requireOwners(types.OrderTypeSell, "1", 2)

	// the entry points at x's new order: removing the duplicate leaves the count as it is
	suite.cancelOrder(orderId(7))
	suite.requireOwners(types.OrderTypeSell, "1", 2)

	suite.cancelOrder(orderId(10))
	suite.requireOwners(types.OrderTypeSell, "1", 1)
	suite.cancelOrder(orderId(6))
	suite.requireOwners(types.OrderTypeSell, "1", 0)
}

// The same sequence, removing x's new order before the duplicate.
func (suite *IntegrationTestSuite) TestEngineOwners_PreUpgradeDuplicateNewOrderRemovedFirst() {
	x, _ := suite.seedDuplicates()
	suite.allowBank()

	suite.cancelOrder(orderId(5))
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeSell, "10", "1", x))
	suite.requireOwners(types.OrderTypeSell, "1", 2)

	suite.cancelOrder(orderId(10))
	suite.requireOwners(types.OrderTypeSell, "1", 1)
	suite.Require().Equal(uint64(2), suite.distinctRestingOwners(types.OrderTypeSell, "1"))

	// the duplicate has no entry: removing it changes nothing and does not go below the remaining owner
	suite.cancelOrder(orderId(7))
	suite.requireOwners(types.OrderTypeSell, "1", 1)
	suite.cancelOrder(orderId(6))
	suite.requireOwners(types.OrderTypeSell, "1", 0)
}

// A partly filled duplicate without an index entry gets one when it is saved, and is counted again.
func (suite *IntegrationTestSuite) TestEngineOwners_PartlyFilledUnindexedDuplicateIsCounted() {
	x, _ := suite.seedDuplicates()
	suite.allowBank()
	taker := sdk.AccAddress("addr2_______________")

	suite.cancelOrder(orderId(5))
	suite.requireOwners(types.OrderTypeSell, "1", 1)

	// y (6) is filled in full, x's duplicate (7) in part
	suite.fillOrder(types.OrderTypeBuy, "15", "1", taker)
	suite.requireOwnerPriceOrder(x, types.OrderTypeSell, "1", orderId(7))
	suite.requireOwners(types.OrderTypeSell, "1", 1)

	suite.fillOrder(types.OrderTypeBuy, "5", "1", taker)
	suite.requireOwners(types.OrderTypeSell, "1", 0)
}

// An aggregate whose count disagrees with the index makes the engine fail instead of wrapping the
// count below 0 or dropping a level that still counts owners.
func (suite *IntegrationTestSuite) TestEngineOwners_InconsistentCountFails() {
	suite.k.SetMarket(suite.ctx, market)
	suite.allowBank()
	owner := makerAddr(0)
	order := withId(limitOrder(types.OrderTypeSell, "10", "1", owner), orderId(0))
	suite.k.SaveOrder(suite.ctx, order)
	agg := types.AggregatedOrder{MarketId: getMarketId(), OrderType: types.OrderTypeSell, Amount: "20", Price: "1"}

	suite.k.SetAggregatedOrder(suite.ctx, agg)
	err := suite.newEngine().CancelOrder(suite.ctx, order)
	suite.Require().ErrorContains(err, "has no owner to remove")

	// cancelling the last order of a level that still counts another owner (the failed cancel ran on
	// no cache context, so the order is saved again first)
	suite.k.SaveOrder(suite.ctx, order)
	agg.Amount = "10"
	agg.Owners = 2
	suite.k.SetAggregatedOrder(suite.ctx, agg)
	err = suite.newEngine().CancelOrder(suite.ctx, order)
	suite.Require().ErrorContains(err, "still counts 1 owners")

	// a taker filling an order its level does not count
	suite.k.SaveOrder(suite.ctx, order)
	agg.Owners = 0
	suite.k.SetAggregatedOrder(suite.ctx, agg)
	err = suite.newEngine().FillOrder(suite.ctx, limitOrder(types.OrderTypeBuy, "10", "1", sdk.AccAddress("addr2_______________")))
	suite.Require().ErrorContains(err, "has no owner to remove")
}
