package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
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

func (suite *IntegrationTestSuite) TestNewOrder() {
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

	for _, c := range cases {
		order := types.Order{
			MarketId:  c.MarketId,
			OrderType: c.OrderType,
			Amount:    c.Amount,
			Price:     c.Price,
			Owner:     c.Owner,
		}
		beforeCounter := suite.k.GetOrderCounter(suite.ctx)
		savedOrder := suite.k.NewOrder(suite.ctx, order)
		afterCounter := suite.k.GetOrderCounter(suite.ctx)
		suite.Require().Equal(beforeCounter, afterCounter-1)
		suite.Require().Equal(order.MarketId, savedOrder.MarketId)
		suite.Require().Equal(order.OrderType, savedOrder.OrderType)
		suite.Require().Equal(order.Amount, savedOrder.Amount)
		suite.Require().Equal(order.Price, savedOrder.Price)
		suite.Require().Equal(order.Owner, savedOrder.Owner)
		suite.Require().NotEmpty(savedOrder.Id)
		suite.Require().Greater(savedOrder.CreatedAt, int64(0))

		//check order reference is present in price index
		ordRef, ok := suite.getPriceOrderRef(savedOrder)
		suite.Require().True(ok)
		suite.Require().Equal(savedOrder.Id, ordRef.Id)
		suite.Require().Equal(savedOrder.MarketId, ordRef.MarketId)
		suite.Require().Equal(savedOrder.OrderType, ordRef.OrderType)

		//check order reference is present in user index
		ordRef2, ok := suite.getUserOrderRef(savedOrder)
		suite.Require().True(ok)
		suite.Require().Equal(ordRef, ordRef2)
	}
}

func (suite *IntegrationTestSuite) TestSaveOrder() {
	order := getRandomOrder(int64(2))
	savedOrder := suite.k.NewOrder(suite.ctx, order)
	suite.Require().NotEmpty(savedOrder.Id)
	suite.Require().Equal(savedOrder.OrderType, order.OrderType)
	suite.Require().Equal(savedOrder.MarketId, order.MarketId)
	suite.Require().Equal(savedOrder.Amount, order.Amount)

	savedOrder.Amount = "1"
	saveResult := suite.k.SaveOrder(suite.ctx, savedOrder)
	suite.Require().Equal(savedOrder, saveResult)

	foundOrder, ok := suite.k.GetOrder(suite.ctx, order.MarketId, order.OrderType, savedOrder.Id)
	suite.Require().True(ok)
	suite.Require().Equal(foundOrder, saveResult)
}

func (suite *IntegrationTestSuite) TestGetOrderCoins() {
	price := "0.91"
	minAmount := keeper.CalculateMinAmount(price)
	buyCoins, err := suite.k.GetOrderCoins(types.OrderTypeBuy, price, minAmount, &market)
	suite.Require().Nil(err)
	suite.Require().Equal(buyCoins.Amount.Int64(), int64(1820)) //result of amount * price
	suite.Require().Equal(buyCoins.Denom, market.Quote)

	sellCoins, err := suite.k.GetOrderCoins(types.OrderTypeSell, price, minAmount, &market)
	suite.Require().Nil(err)
	suite.Require().Equal(sellCoins.Amount, minAmount)
	suite.Require().Equal(sellCoins.Denom, market.Base)
}

func (suite *IntegrationTestSuite) TestGetOrderCoins_Error() {
	_, err := suite.k.GetOrderCoins("NOT_A_TYPE", "100", sdk.NewInt(2), &market)
	suite.Require().NotNil(err)

	_, err = suite.k.GetOrderCoins(types.OrderTypeBuy, "0.", sdk.NewInt(1), &market)
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestGetAllOrders() {
	initial := suite.k.GetAllOrder(suite.ctx)
	addedOrders := 7
	randCol := getRandomOrderCollection(addedOrders)
	for _, ord := range randCol {
		suite.k.NewOrder(suite.ctx, ord)
	}

	res := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(res), len(initial)+addedOrders)
}

func (suite *IntegrationTestSuite) TestRemoveOrder() {
	order := getRandomOrder(int64(2))
	savedOrder := suite.k.NewOrder(suite.ctx, order)
	suite.Require().NotEmpty(savedOrder.Id)
	suite.k.RemoveOrder(suite.ctx, savedOrder)

	_, ok := suite.k.GetOrder(suite.ctx, order.MarketId, order.OrderType, savedOrder.Id)
	suite.Require().False(ok)

	_, ok = suite.getUserOrderRef(savedOrder)
	suite.Require().False(ok)

	_, ok = suite.getPriceOrderRef(savedOrder)
	suite.Require().False(ok)
}

func (suite *IntegrationTestSuite) TestGetAllPriceOrder() {
	initial := suite.k.GetAllOrder(suite.ctx)
	addedOrders := 7
	randCol := getRandomOrderCollection(addedOrders)
	for _, ord := range randCol {
		suite.k.NewOrder(suite.ctx, ord)
	}

	res := suite.k.GetAllPriceOrder(suite.ctx)
	suite.Require().Equal(len(res), len(initial)+addedOrders)
}

func (suite *IntegrationTestSuite) TestAggregatedOrder() {
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
