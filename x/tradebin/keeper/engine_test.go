package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// newEngine builds the order book engine on the suite keeper and the suite bank mock with the default
// max fills per order
func (suite *IntegrationTestSuite) newEngine() *keeper.ProcessingEngine {
	return suite.newEngineWithMaxFills(v2types.DefaultMaxFillsPerOrder)
}

// newEngineWithMaxFills builds the order book engine on the suite keeper and the suite bank mock
func (suite *IntegrationTestSuite) newEngineWithMaxFills(maxFills uint64) *keeper.ProcessingEngine {
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, maxFills, suite.k.Logger())
	suite.Require().NoError(err)

	return engine
}

// limitOrder builds an order the way the message servers hand it to the engine (no id yet)
func limitOrder(orderType, amount, price string, owner sdk.AccAddress) types.Order {
	return types.Order{
		MarketId:  getMarketId(),
		OrderType: orderType,
		Amount:    amount,
		Price:     price,
		Owner:     owner.String(),
	}
}

// placeOrders executes every order with PlaceOrder on the suite context, in the given order
func (suite *IntegrationTestSuite) placeOrders(engine *keeper.ProcessingEngine, orders ...types.Order) {
	for _, o := range orders {
		suite.Require().NoError(engine.PlaceOrder(suite.ctx, o))
	}
}

func (suite *IntegrationTestSuite) checkAggregatedOrder(marketId, orderType, price string, expectedAmount string) {
	agg, ok := suite.k.GetAggregatedOrder(suite.ctx, marketId, orderType, price)
	suite.Require().True(ok)
	suite.Require().Equal(agg.Amount, expectedAmount)
}

func (suite *IntegrationTestSuite) TestEngine_AddMakerOrder() {
	engine := suite.newEngine()

	addr1 := sdk.AccAddress("addr1_______________")

	mBuyAmt, err := keeper.CalculateMinAmount("100")
	suite.Require().NoError(err)
	mBuy := limitOrder(types.OrderTypeBuy, mBuyAmt.String(), "100", addr1)
	mSellAmt, err := keeper.CalculateMinAmount("10")
	suite.Require().NoError(err)
	mSell := limitOrder(types.OrderTypeSell, mSellAmt.String(), "10", addr1)

	suite.placeOrders(engine, mBuy, mSell)

	//every placed order got an id from the order counter
	suite.Require().Equal(uint64(2), suite.k.GetOrderCounter(suite.ctx))

	//check orders are correctly aggregated
	aggBuy, ok := suite.k.GetAggregatedOrder(suite.ctx, mBuy.MarketId, mBuy.OrderType, mBuy.Price)
	suite.Require().True(ok)
	suite.Require().Equal(aggBuy.MarketId, mBuy.MarketId)
	suite.Require().Equal(aggBuy.OrderType, mBuy.OrderType)
	suite.Require().Equal(aggBuy.Price, mBuy.Price)
	suite.Require().Equal(mBuy.Amount, aggBuy.Amount)

	aggSell, ok := suite.k.GetAggregatedOrder(suite.ctx, mSell.MarketId, mSell.OrderType, mSell.Price)
	suite.Require().True(ok)
	suite.Require().Equal(aggSell.MarketId, mSell.MarketId)
	suite.Require().Equal(aggSell.OrderType, mSell.OrderType)
	suite.Require().Equal(aggSell.Price, mSell.Price)
	suite.Require().Equal(mSell.Amount, aggSell.Amount)

	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(allOrders)
	suite.Require().Len(allOrders, 2)

	var buyOrder types.Order
	var sellOrder types.Order
	for _, ord := range allOrders {
		if ord.OrderType == types.OrderTypeBuy {
			buyOrder = ord
		} else {
			sellOrder = ord
		}
	}
	suite.Require().Equal(mBuy.Amount, buyOrder.Amount)
	suite.Require().Equal(mSell.Amount, sellOrder.Amount)

	//check orders exist
	priceBuy, ok := suite.getPriceOrderRef(buyOrder)
	suite.Require().True(ok)
	priceSell, ok := suite.getPriceOrderRef(sellOrder)
	suite.Require().True(ok)

	_, ok = suite.k.GetOrder(suite.ctx, priceBuy.MarketId, priceBuy.OrderType, priceBuy.Id)
	suite.Require().True(ok)

	_, ok = suite.k.GetOrder(suite.ctx, priceSell.MarketId, priceSell.OrderType, priceSell.Id)
	suite.Require().True(ok)
}

func (suite *IntegrationTestSuite) TestEngine_CancelOrder() {
	//create test market
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()

	//create an user account
	addr1 := sdk.AccAddress("addr1_______________")

	//create two random orders
	mBuy := limitOrder(types.OrderTypeBuy, "100", "0.182", addr1)
	mSell := limitOrder(types.OrderTypeSell, "1000", "1.35", addr1)

	totalBuyAmount := math.ZeroInt()
	totalSellAmount := math.ZeroInt()
	//place the orders
	for i := 0; i < 3; i++ {
		suite.placeOrders(engine, mBuy)
		buyAmtInt, _ := math.NewIntFromString(mBuy.Amount)
		totalBuyAmount = totalBuyAmount.Add(buyAmtInt)

		suite.placeOrders(engine, mSell)
		sellAmtInt, _ := math.NewIntFromString(mSell.Amount)
		totalSellAmount = totalSellAmount.Add(sellAmtInt)
	}

	//check orders were created
	allUserOrders, err := suite.k.UserMarketOrders(suite.ctx, &types.QueryUserMarketOrdersRequest{
		Address:    addr1.String(),
		Market:     getMarketId(),
		Pagination: nil,
	})
	suite.Require().NotNil(allUserOrders)
	suite.Require().Nil(err)
	suite.Require().Len(allUserOrders.List, 6)

	//check aggregate total amount is equal tot total amounts of the orders we placed
	suite.checkAggregatedOrder(mBuy.MarketId, mBuy.OrderType, mBuy.Price, totalBuyAmount.String())
	suite.checkAggregatedOrder(mSell.MarketId, mSell.OrderType, mSell.Price, totalSellAmount.String())

	//cancel every order and check the refund happens in the same call
	var orders []types.OrderReference
	cancelCount := 0
	for _, or := range allUserOrders.List {
		toCancelOrder, ok := suite.k.GetOrder(suite.ctx, or.MarketId, or.OrderType, or.Id)
		suite.Require().True(ok)
		canceledAmount, ok := math.NewIntFromString(toCancelOrder.Amount)
		suite.Require().True(ok)
		canceledCoins, _, err := suite.k.GetOrderSdkCoin(toCancelOrder.OrderType, toCancelOrder.Price, canceledAmount, &market)
		suite.Require().Nil(err)
		orders = append(orders, or)

		aggBefore, ok := suite.k.GetAggregatedOrder(suite.ctx, toCancelOrder.MarketId, toCancelOrder.OrderType, toCancelOrder.Price)
		suite.Require().True(ok)
		aggBeforeInt, ok := math.NewIntFromString(aggBefore.Amount)
		suite.Require().True(ok)

		suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(canceledCoins)).Return(nil).Times(1)
		suite.Require().NoError(engine.CancelOrder(suite.ctx, toCancelOrder))
		cancelCount++

		_, ok = suite.k.GetOrder(suite.ctx, or.MarketId, or.OrderType, or.Id)
		suite.Require().False(ok)
		aggAfter, ok := suite.k.GetAggregatedOrder(suite.ctx, toCancelOrder.MarketId, toCancelOrder.OrderType, toCancelOrder.Price)
		if aggBeforeInt.Equal(canceledAmount) {
			suite.Require().False(ok)
		} else {
			suite.Require().True(ok)
			suite.Require().Equal(aggBeforeInt.Sub(canceledAmount).String(), aggAfter.Amount)
		}

		//check user order were canceled
		checkUserOrders, err := suite.k.UserMarketOrders(suite.ctx, &types.QueryUserMarketOrdersRequest{
			Address:    addr1.String(),
			Market:     getMarketId(),
			Pagination: nil,
		})
		suite.Require().Nil(err)
		suite.Require().NotNil(checkUserOrders)
		//list should now have fewer orders
		suite.Require().Equal(len(checkUserOrders.List), len(allUserOrders.List)-cancelCount)
	}

	//check aggregated orders were removed with the orders
	_, ok := suite.k.GetAggregatedOrder(suite.ctx, mBuy.MarketId, mBuy.OrderType, mBuy.Price)
	suite.Require().False(ok)

	_, ok = suite.k.GetAggregatedOrder(suite.ctx, mSell.MarketId, mSell.OrderType, mSell.Price)
	suite.Require().False(ok)

	//check orders no longer exist
	for _, o := range orders {
		_, ok = suite.k.GetOrder(suite.ctx, o.MarketId, o.OrderType, o.Id)
		suite.Require().False(ok)
	}
}

func (suite *IntegrationTestSuite) TestEngine_OrderMatching() {
	//create test market
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()

	//create accounts
	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")

	sellPriceStr := "1"
	sellAmt, err := keeper.CalculateMinAmount(sellPriceStr)
	suite.Require().NoError(err)
	sellAmt = sellAmt.MulRaw(5)
	buyPriceStr := "2"
	buyAmt, err := keeper.CalculateMinAmount(buyPriceStr)
	suite.Require().NoError(err)
	buyAmt = buyAmt.MulRaw(5)
	orderCounter := int64(10)
	for i := int64(0); i < orderCounter; i++ {
		suite.placeOrders(engine,
			limitOrder(types.OrderTypeSell, sellAmt.String(), sellPriceStr, makerAddr),
			limitOrder(types.OrderTypeBuy, buyAmt.String(), buyPriceStr, makerAddr),
		)
	}

	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(allOrders)
	suite.Require().Equal(len(allOrders), int(orderCounter)*2) //all orders should be there
	//check aggregated orders
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).String())

	//1. fill 50% of an order -> check its amount is updated -> check the maker gets his coins ->
	//check module balances updated -> check the taker balances are updated -> check aggregated is updated
	qmAmountInt := sellAmt.QuoRaw(2)
	qmBuy := limitOrder(types.OrderTypeBuy, qmAmountInt.String(), sellPriceStr, takerAddr)
	makerCoins, _, err := suite.k.GetOrderSdkCoin(qmBuy.OrderType, qmBuy.Price, qmAmountInt, &market)
	suite.Require().Nil(err)
	takerCoins, _, err := suite.k.GetOrderSdkCoin(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, qmAmountInt, &market)
	suite.Require().Nil(err)

	tradedStakeCoins := takerCoins

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(takerCoins)).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, sdk.NewCoins(makerCoins)).Return(nil).Times(1)
	suite.placeOrders(engine, qmBuy)

	//check all orders are still there since none of them have been filled
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2))

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).Sub(takerCoins.Amount).String())

	//2. fill 25% of the order -> check all above again
	qmAmountInt = sellAmt.QuoRaw(4)
	qmBuy = limitOrder(types.OrderTypeBuy, qmAmountInt.String(), sellPriceStr, takerAddr)
	makerCoins, _, err = suite.k.GetOrderSdkCoin(qmBuy.OrderType, qmBuy.Price, qmAmountInt, &market)
	suite.Require().Nil(err)
	takerCoins, _, err = suite.k.GetOrderSdkCoin(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, qmAmountInt, &market)
	suite.Require().Nil(err)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(takerCoins)).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, sdk.NewCoins(makerCoins)).Return(nil).Times(1)
	suite.placeOrders(engine, qmBuy)

	//check all orders are still there since none of them were filled
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2))

	tradedStakeCoins = tradedStakeCoins.Add(takerCoins)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).Sub(tradedStakeCoins.Amount).String())

	//3. fill 200% of orders (2 * order amount) -> check all of the above again
	qmAmountInt = sellAmt.MulRaw(2)
	qmBuy = limitOrder(types.OrderTypeBuy, qmAmountInt.String(), sellPriceStr, takerAddr)
	takerCoins, _, err = suite.k.GetOrderSdkCoin(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, qmAmountInt, &market)
	suite.Require().Nil(err)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, gomock.Any()).Return(nil).Times(3)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, gomock.Any()).Return(nil).Times(3)
	suite.placeOrders(engine, qmBuy)

	//check the correct amount of suborders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2)-2)

	tradedStakeCoins = tradedStakeCoins.Add(takerCoins)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).Sub(tradedStakeCoins.Amount).String())

	//4. fill the rest + some amount to also create an order
	qmAmountInt = sellAmt.MulRaw(8)
	qmBuy = limitOrder(types.OrderTypeBuy, qmAmountInt.String(), sellPriceStr, takerAddr)
	takerCoins, _, err = suite.k.GetOrderSdkCoin(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, sellAmt, &market)
	suite.Require().Nil(err)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, gomock.Any()).Return(nil).Times(8)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, gomock.Any()).Return(nil).Times(8)
	suite.placeOrders(engine, qmBuy)

	//check the correct amount of suborders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2)-10+1)
	//only buy orders should exist now
	//they should have either sellAmt amount or one should have 75%
	var smallOrders []types.Order
	for _, ord := range allOrders {
		suite.Require().Equal(ord.OrderType, types.OrderTypeBuy)
		ordAmtInt, _ := math.NewIntFromString(ord.Amount)
		if ordAmtInt.LT(sellAmt) {
			smallOrders = append(smallOrders, ord)
		}
	}

	//check the smaller order is there, and it has the right values
	suite.Require().Equal(len(smallOrders), 1)
	suite.Require().Equal(smallOrders[0].Amount, sellAmt.MulRaw(3).QuoRaw(4).String())
	suite.Require().Equal(smallOrders[0].Price, sellPriceStr)

	smallOrdAmt, _ := math.NewIntFromString(smallOrders[0].Amount)

	newOrderTakerCoins, _, err := suite.k.GetOrderSdkCoin(smallOrders[0].OrderType, smallOrders[0].Price, smallOrdAmt, &market)
	suite.Require().Nil(err)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, sellPriceStr, newOrderTakerCoins.Amount.String())
	//sell order should not exist anymore
	_, ok := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, sellPriceStr)
	suite.Require().False(ok)

	//5. fill all remaining orders
	qmAmountInt = buyAmt.MulRaw(12)
	qmSell := limitOrder(types.OrderTypeSell, qmAmountInt.String(), buyPriceStr, takerAddr)

	makerCoins, _, err = suite.k.GetOrderSdkCoin(qmSell.OrderType, qmSell.Price, buyAmt, &market)
	suite.Require().Nil(err)
	takerCoins, _, err = suite.k.GetOrderSdkCoin(types.TheOtherOrderType(qmSell.OrderType), qmSell.Price, buyAmt, &market)
	suite.Require().Nil(err)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(takerCoins)).Return(nil).Times(10)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, sdk.NewCoins(makerCoins)).Return(nil).Times(10)
	suite.placeOrders(engine, qmSell)

	//check the correct amount of orders removed: the small buy at 1 and the sell remainder at 2 rest
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), 2)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, buyPriceStr, buyAmt.MulRaw(2).String())
	_, ok = suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, buyPriceStr)
	suite.Require().False(ok)
}

func (suite *IntegrationTestSuite) TestEngine_OrderMatching_WithDust() {
	//create test market
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()

	//create accounts
	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")

	sellPriceStr := "0.7612"
	sellAmt, err := keeper.CalculateMinAmount(sellPriceStr)
	suite.Require().NoError(err)
	sellAmt = sellAmt.MulRaw(5)
	buyPriceStr := "0.950012"
	buyAmt, err := keeper.CalculateMinAmount(buyPriceStr)
	suite.Require().NoError(err)
	buyAmt = buyAmt.MulRaw(6)
	orderCounter := int64(10)
	for i := int64(0); i < orderCounter; i++ {
		suite.placeOrders(engine,
			limitOrder(types.OrderTypeSell, sellAmt.String(), sellPriceStr, makerAddr),
			limitOrder(types.OrderTypeBuy, buyAmt.String(), buyPriceStr, makerAddr),
		)
	}

	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(allOrders)
	suite.Require().Equal(len(allOrders), int(orderCounter)*2) //all orders should be there
	//check aggregated orders
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).String())

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.AnyOf(makerAddr, takerAddr), gomock.Any()).AnyTimes()
	//1. fill 50% of an order -> check its amount is updated -> check the maker gets his coins ->
	//check module balances updated -> check the taker balances are updated -> check aggregated is updated
	qmAmountInt := sellAmt.QuoRaw(2)
	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, qmAmountInt.String(), sellPriceStr, takerAddr))

	//check all orders are still there since none of them have been filled
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2))

	//check maker dust
	makerDust, ok := suite.k.GetUserDust(suite.ctx, makerAddr.String(), market.Quote)
	suite.Require().True(ok)
	suite.Require().Equal(makerDust.Denom, market.Quote)
	suite.Require().Equal(makerDust.Owner, makerAddr.String())
	suite.Require().Equal(makerDust.Amount, "0.612000000000000000")

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).SubRaw(10).String())

	//2. fill 25% of the order -> check all above again
	qmAmountInt = sellAmt.QuoRaw(4)
	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, qmAmountInt.String(), sellPriceStr, takerAddr))

	//check all orders are still there since none of them were filled
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2))

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).Sub(math.NewInt(15)).String())

	//3. fill 200% of orders (2 * order amount) -> check all of the above again
	qmAmountInt = sellAmt.MulRaw(2)
	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, qmAmountInt.String(), sellPriceStr, takerAddr))

	//check the correct amount of suborders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2)-2)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).Sub(math.NewInt(55)).String())

	//4. fill the rest + some amount to also create an order
	qmAmountInt = sellAmt.MulRaw(8)
	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, qmAmountInt.String(), sellPriceStr, takerAddr))

	//check the correct amount of suborders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2)-10+1)
	//only buy orders should exist now
	//they should have either sellAmt amount or one should have 75%
	var smallOrders []types.Order
	for _, ord := range allOrders {
		suite.Require().Equal(ord.OrderType, types.OrderTypeBuy)
		ordAmtInt, _ := math.NewIntFromString(ord.Amount)
		if ordAmtInt.LT(sellAmt) {
			smallOrders = append(smallOrders, ord)
		}
	}

	//check the smaller order is there, and it has the right values
	suite.Require().Equal(len(smallOrders), 1)
	suite.Require().Equal(smallOrders[0].Amount, sellAmt.MulRaw(3).QuoRaw(4).String())
	suite.Require().Equal(smallOrders[0].Price, sellPriceStr)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	//sell order should not exist anymore
	_, ok = suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, sellPriceStr)
	suite.Require().False(ok)

	//5. fill all remaining orders
	qmAmountInt = buyAmt.MulRaw(12)
	suite.placeOrders(engine, limitOrder(types.OrderTypeSell, qmAmountInt.String(), buyPriceStr, takerAddr))

	//check the correct amount of orders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), 2)
}

// TestEngine_FillAggregatedOrder_ContinuePastUnfillableOrder verifies that when getExecutedAmount returns zero for an order
// (because the order amount is between minAmount and 2*minAmount, making it impossible to partially fill without
// leaving a remainder below minAmount), the iteration continues to the next order instead of breaking.
// This ensures that all compatible orders at the same price level are considered for filling.
func (suite *IntegrationTestSuite) TestEngine_FillAggregatedOrder_ContinuePastUnfillableOrder() {
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()

	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")

	price := "1"
	minAmount, err := keeper.CalculateMinAmount(price) // ceil(1/1)*2 = 2
	suite.Require().NoError(err)

	// Create a small sell order with amount between minAmount and 2*minAmount.
	// For msgAmount=minAmount=2 and orderAmount=3:
	//   getExecutedAmount(2, 3, 2): orderRemaining=1 < minAmount, executedAmount=1 < minAmount → returns 0
	smallSellAmount := minAmount.AddRaw(1) // 3
	// Create a large sell order that can be partially filled.
	// For msgAmount=2 and orderAmount=100: orderRemaining=98 >= minAmount → returns 2
	largeSellAmount := minAmount.MulRaw(50) // 100

	// place the sell orders in the order book
	suite.placeOrders(engine,
		limitOrder(types.OrderTypeSell, smallSellAmount.String(), price, makerAddr),
		limitOrder(types.OrderTypeSell, largeSellAmount.String(), price, makerAddr),
	)

	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(allOrders, 2)

	totalSellAmount := smallSellAmount.Add(largeSellAmount) // 103
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, price, totalSellAmount.String())

	// Submit a buy order with amount = minAmount.
	// The small sell order cannot be filled (getExecutedAmount returns 0) but with 'continue',
	// the large sell order should still be reached and partially filled.
	buyAmount := minAmount // 2

	// Expect exactly 1 fill: maker gets quote coins, taker gets base coins
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, sdk.NewCoins(sdk.NewCoin(market.Quote, buyAmount))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(sdk.NewCoin(market.Base, buyAmount))).Return(nil).Times(1)

	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, buyAmount.String(), price, takerAddr))

	// Both orders should still exist: small untouched, large partially filled
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(allOrders, 2)

	for _, ord := range allOrders {
		ordAmt, ok := math.NewIntFromString(ord.Amount)
		suite.Require().True(ok)
		if ordAmt.Equal(smallSellAmount) {
			// Small order should be untouched
			suite.Require().Equal(smallSellAmount.String(), ord.Amount)
		} else {
			// Large order should be partially filled: 100 - 2 = 98
			suite.Require().Equal(largeSellAmount.Sub(buyAmount).String(), ord.Amount)
		}
	}

	// Aggregated sell should reflect the fill: 103 - 2 = 101
	expectedAggAmount := totalSellAmount.Sub(buyAmount)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, price, expectedAggAmount.String())
}

// TestEngine_FillAggregatedOrder_EarlyExitMsgBelowMinAmount verifies that when the remaining taker amount drops below
// minAmount after a partial fill, the iteration breaks early. The remaining amount is refunded to the taker.
func (suite *IntegrationTestSuite) TestEngine_FillAggregatedOrder_EarlyExitMsgBelowMinAmount() {
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()

	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")

	price := "1"
	minAmount, err := keeper.CalculateMinAmount(price) // 2
	suite.Require().NoError(err)

	// Create two sell orders at the same price, each with amount = minAmount (fully fillable)
	sellAmount := minAmount // 2
	for i := 0; i < 2; i++ {
		suite.placeOrders(engine, limitOrder(types.OrderTypeSell, sellAmount.String(), price, makerAddr))
	}

	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(allOrders, 2)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, price, sellAmount.MulRaw(2).String()) // 4

	// Submit a buy order with amount = minAmount + 1 = 3.
	// First sell order (amount=2): getExecutedAmount(3, 2, 2) → msgAmount >= orderAmount → returns 2 (fully fills)
	//   msgAmountInt = 3 - 2 = 1
	// Second iteration: msgAmountInt(1) < minAmount(2) → break (early exit)
	// Second sell order is never reached.
	// Remaining 1 is refunded to the buyer.
	buyAmount := minAmount.AddRaw(1) // 3

	// Expect 1 fill (2 bank calls) + 1 refund (1 bank call to taker)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, sdk.NewCoins(sdk.NewCoin(market.Quote, sellAmount))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(sdk.NewCoin(market.Base, sellAmount))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 1))).Return(nil).Times(1)

	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, buyAmount.String(), price, takerAddr))

	// First sell order should be removed (fully filled), second should remain untouched
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(allOrders, 1)
	suite.Require().Equal(sellAmount.String(), allOrders[0].Amount)
	suite.Require().Equal(types.OrderTypeSell, allOrders[0].OrderType)

	// Aggregated sell should be just the remaining second order: 2
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, price, sellAmount.String())
	// the refunded remainder does not rest on the book
	_, ok := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, price)
	suite.Require().False(ok)
}

// TestEngine_FillAggregatedOrder_ContinueThenEarlyExit exercises both behaviours together:
// 1. An unfillable order is skipped (continue instead of break)
// 2. A subsequent order is partially filled
// 3. The remaining taker amount drops below minAmount, causing an early exit
// 4. Further orders are not reached, and the remainder is refunded
func (suite *IntegrationTestSuite) TestEngine_FillAggregatedOrder_ContinueThenEarlyExit() {
	suite.k.SetMarket(suite.ctx, market)
	engine := suite.newEngine()

	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")

	// Use price "0.1" for a larger minAmount, giving more room for the test scenario
	price := "0.1"
	minAmount, err := keeper.CalculateMinAmount(price) // ceil(1/0.1)*2 = 20
	suite.Require().NoError(err)

	// Order 1: unfillable (amount between minAmount and 2*minAmount)
	// For msgAmount=25 and orderAmount=30:
	//   getExecutedAmount(25, 30, 20): orderRemaining=5 < 20, executedAmount=10 < 20 → returns 0
	unfillableAmount := minAmount.AddRaw(10) // 30

	// Order 2: partially fillable, the partial fill leaves the taker amount < minAmount
	// For orderAmount=40 and msgAmount=25:
	//   getExecutedAmount(25, 40, 20): orderRemaining=15 < 20, executedAmount=20 >= 20 → returns 20
	//   msgAmountInt = 25-20 = 5 < minAmount(20) → early exit!
	partialFillAmount := minAmount.MulRaw(2) // 40

	// Order 3: large order that should NOT be reached due to early exit
	largeAmount := minAmount.MulRaw(10) // 200

	suite.placeOrders(engine,
		limitOrder(types.OrderTypeSell, unfillableAmount.String(), price, makerAddr),
		limitOrder(types.OrderTypeSell, partialFillAmount.String(), price, makerAddr),
		limitOrder(types.OrderTypeSell, largeAmount.String(), price, makerAddr),
	)

	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(allOrders, 3)
	totalSellAmount := unfillableAmount.Add(partialFillAmount).Add(largeAmount) // 30+40+200=270
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, price, totalSellAmount.String())

	// Submit buy order with amount=25 (> minAmount=20)
	// Flow:
	// 1. Order 1 (30): getExecutedAmount(25,30,20) → orderRemaining=5<20, executedAmount=10<20 → 0 → continue
	// 2. Order 2 (40): getExecutedAmount(25,40,20) → orderRemaining=15<20, executedAmount=20>=20 → 20
	//    msgAmountInt = 25-20 = 5, orderAmount = 40-20 = 20
	// 3. Next iteration: msgAmountInt(5) < minAmount(20) → break
	// 4. Order 3 (200): not reached
	// Remaining 5 is refunded to buyer: 5 * 0.1 = 0.5 quote, which rounds to zero and is kept as dust
	buyAmount := minAmount.AddRaw(5) // 25
	executedAmount := minAmount      // 20 (what is filled from order 2)

	// 1 fill (maker + taker), the refund of 0.5 quote is below one unit
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 2))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(sdk.NewCoin(market.Base, executedAmount))).Return(nil).Times(1)

	suite.placeOrders(engine, limitOrder(types.OrderTypeBuy, buyAmount.String(), price, takerAddr))

	// All 3 orders should still exist
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(allOrders, 3)

	// Verify each order's amount
	for _, ord := range allOrders {
		ordAmt, ok := math.NewIntFromString(ord.Amount)
		suite.Require().True(ok)
		if ordAmt.Equal(unfillableAmount) {
			// Order 1: untouched at 30
			suite.Require().Equal(unfillableAmount.String(), ord.Amount)
		} else if ordAmt.Equal(partialFillAmount.Sub(executedAmount)) {
			// Order 2: partially filled, 40 - 20 = 20
			suite.Require().Equal(partialFillAmount.Sub(executedAmount).String(), ord.Amount)
		} else {
			// Order 3: untouched at 200
			suite.Require().Equal(largeAmount.String(), ord.Amount)
		}
	}

	// Aggregated sell: 270 - 20 = 250
	expectedAggAmount := totalSellAmount.Sub(executedAmount)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, price, expectedAggAmount.String())
}
