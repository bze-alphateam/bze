package keeper_test

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func getRandomOrder(amt int64) types.Order {
	return types.Order{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    fmt.Sprintf("%d", amt),
		Price:     "1233001",
		Owner:     "test1_____",
	}
}

func getRandomOrderCollection(len int) (coll []types.Order) {
	for i := 1; i <= len; i++ {
		coll = append(coll, getRandomOrder(int64(2*i)))
	}

	return
}

func (suite *IntegrationTestSuite) getUserOrderRef(order types.Order) (types.OrderReference, bool) {
	userOrders := suite.k.GetAllUserOrder(suite.ctx)

	found := false
	var ordRef2 types.OrderReference
	for _, ref := range userOrders {
		if ref.Id == order.Id {
			ordRef2 = ref
			found = true
			break
		}
	}

	return ordRef2, found
}

func (suite *IntegrationTestSuite) getPriceOrderRef(order types.Order) (types.OrderReference, bool) {
	prices := suite.k.GetPriceOrderByPrice(suite.ctx, order.MarketId, order.OrderType, order.Price)

	found := false
	var ordRef types.OrderReference
	for _, ref := range prices {
		if ref.Id == order.Id {
			ordRef = ref
			found = true
			break
		}
	}

	return ordRef, found
}

// newOrder gives the order an id and saves it on the book the way the engine rests a maker order
func (suite *IntegrationTestSuite) newOrder(ctx sdk.Context, order types.Order) types.Order {
	return suite.k.SaveOrder(ctx, suite.k.AssignOrderId(ctx, order))
}

func (suite *IntegrationTestSuite) TestStore_AssignOrderIdAndSaveOrder() {
	cases := map[string]struct {
		MarketId  string
		OrderType string
		Amount    string
		Price     string
		Owner     string
	}{
		"buy order": {
			MarketId:  getMarketId(),
			OrderType: types.OrderTypeBuy,
			Amount:    "10",
			Price:     "100",
			Owner:     "123",
		},
		"sell order": {
			MarketId:  getMarketId(),
			OrderType: types.OrderTypeSell,
			Amount:    "100",
			Price:     "10",
			Owner:     "1234444555666",
		},
	}

	blockTime := time.Unix(1_700_000_000, 0)
	for name, c := range cases {
		order := types.Order{
			MarketId:  c.MarketId,
			OrderType: c.OrderType,
			Amount:    c.Amount,
			Price:     c.Price,
			Owner:     c.Owner,
		}
		ctx := suite.ctx.WithBlockTime(blockTime)

		// AssignOrderId: id from the counter, created at from the block time, counter + 1, nothing stored
		beforeCounter := suite.k.GetOrderCounter(suite.ctx)
		withId := suite.k.AssignOrderId(ctx, order)
		afterCounter := suite.k.GetOrderCounter(suite.ctx)
		suite.Require().Equal(beforeCounter+1, afterCounter, name)
		suite.Require().Equal(fmt.Sprintf("%024d", beforeCounter), withId.Id, name)
		suite.Require().Equal(blockTime.Unix(), withId.CreatedAt, name)
		suite.Require().Equal(order.MarketId, withId.MarketId, name)
		suite.Require().Equal(order.OrderType, withId.OrderType, name)
		suite.Require().Equal(order.Amount, withId.Amount, name)
		suite.Require().Equal(order.Price, withId.Price, name)
		suite.Require().Equal(order.Owner, withId.Owner, name)
		_, found := suite.k.GetOrder(suite.ctx, withId.MarketId, withId.OrderType, withId.Id)
		suite.Require().False(found, name)
		_, ok := suite.getPriceOrderRef(withId)
		suite.Require().False(ok, name)
		_, ok = suite.getUserOrderRef(withId)
		suite.Require().False(ok, name)

		// SaveOrder: stores the order and its references, leaves the counter alone
		savedOrder := suite.k.SaveOrder(ctx, withId)
		suite.Require().Equal(withId, savedOrder, name)
		suite.Require().Equal(afterCounter, suite.k.GetOrderCounter(suite.ctx), name)
		stored, found := suite.k.GetOrder(suite.ctx, withId.MarketId, withId.OrderType, withId.Id)
		suite.Require().True(found, name)
		suite.Require().Equal(withId, stored, name)

		//check order reference is present in price index
		ordRef, ok := suite.getPriceOrderRef(savedOrder)
		suite.Require().True(ok, name)
		suite.Require().Equal(savedOrder.Id, ordRef.Id)
		suite.Require().Equal(savedOrder.MarketId, ordRef.MarketId)
		suite.Require().Equal(savedOrder.OrderType, ordRef.OrderType)

		//check order reference is present in user index
		ordRef2, ok := suite.getUserOrderRef(savedOrder)
		suite.Require().True(ok, name)
		suite.Require().Equal(ordRef, ordRef2)
	}
}

func (suite *IntegrationTestSuite) TestStore_GetOrderSdkCoin_Error() {
	_, _, err := suite.k.GetOrderSdkCoin("NOT_A_TYPE", "100", math.NewInt(2), &market)
	suite.Require().NotNil(err)

	_, _, err = suite.k.GetOrderSdkCoin(types.OrderTypeBuy, "0.", math.NewInt(1), &market)
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestStore_GetAllOrders() {
	initial := suite.k.GetAllOrder(suite.ctx)
	addedOrders := 7
	randCol := getRandomOrderCollection(addedOrders)
	for _, ord := range randCol {
		suite.newOrder(suite.ctx, ord)
	}

	res := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(res), len(initial)+addedOrders)
}

func (suite *IntegrationTestSuite) TestStore_RemoveOrder() {
	order := getRandomOrder(int64(2))
	savedOrder := suite.newOrder(suite.ctx, order)
	suite.Require().NotEmpty(savedOrder.Id)
	suite.k.RemoveOrder(suite.ctx, savedOrder)

	_, ok := suite.k.GetOrder(suite.ctx, order.MarketId, order.OrderType, savedOrder.Id)
	suite.Require().False(ok)

	_, ok = suite.getUserOrderRef(savedOrder)
	suite.Require().False(ok)

	_, ok = suite.getPriceOrderRef(savedOrder)
	suite.Require().False(ok)
}

func (suite *IntegrationTestSuite) TestStore_GetAllPriceOrder() {
	initial := suite.k.GetAllOrder(suite.ctx)
	addedOrders := 7
	randCol := getRandomOrderCollection(addedOrders)
	for _, ord := range randCol {
		suite.newOrder(suite.ctx, ord)
	}

	res := suite.k.GetAllPriceOrder(suite.ctx)
	suite.Require().Equal(len(res), len(initial)+addedOrders)
}

func (suite *IntegrationTestSuite) TestStore_AggregatedOrder() {
	agg := types.AggregatedOrder{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    "10",
		Price:     "1",
	}

	suite.k.SetAggregatedOrder(suite.ctx, agg)
	found, ok := suite.k.GetAggregatedOrder(suite.ctx, agg.MarketId, agg.OrderType, agg.Price)
	suite.Require().True(ok)
	suite.Equal(found.MarketId, agg.MarketId)
	suite.Equal(found.OrderType, agg.OrderType)
	suite.Equal(found.Amount, agg.Amount)
	suite.Equal(found.Price, agg.Price)

	list := suite.k.GetAllAggregatedOrder(suite.ctx)
	suite.Require().NotEmpty(list)
	suite.Require().Equal(len(list), 1)

	suite.k.RemoveAggregatedOrder(suite.ctx, agg)
	list = suite.k.GetAllAggregatedOrder(suite.ctx)
	suite.Require().Empty(list)

	_, ok = suite.k.GetAggregatedOrder(suite.ctx, agg.MarketId, agg.OrderType, agg.Price)
	suite.Require().False(ok)
}
