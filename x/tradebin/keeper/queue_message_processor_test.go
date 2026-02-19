package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestQueueMessageProcessor_AddMakerOrder() {
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().Nil(err)

	addr1 := sdk.AccAddress("addr1_______________")

	mBuyAmt := keeper.CalculateMinAmount("100")
	mBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      mBuyAmt.String(),
		Price:       "100",
		OrderType:   types.OrderTypeBuy,
		Owner:       addr1.String(),
	}
	mSellAmt := keeper.CalculateMinAmount("10")
	mSell := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      mSellAmt.String(),
		Price:       "10",
		OrderType:   types.OrderTypeSell,
		Owner:       addr1.String(),
	}

	suite.k.SetQueueMessage(suite.ctx, mBuy)
	suite.k.SetQueueMessage(suite.ctx, mSell)

	//check message counter was incremented
	mCnt := suite.k.GetQueueMessageCounter(suite.ctx)
	suite.Require().Equal(mCnt, uint64(2))

	//call engine
	engine.ProcessQueueMessages(suite.ctx)

	//check the results
	//check orders are correctly aggregated
	aggBuy, ok := suite.k.GetAggregatedOrder(suite.ctx, mBuy.MarketId, mBuy.OrderType, mBuy.Price)
	suite.Require().True(ok)
	suite.Require().Equal(aggBuy.MarketId, mBuy.MarketId)
	suite.Require().Equal(aggBuy.OrderType, mBuy.OrderType)
	suite.Require().Equal(aggBuy.Price, mBuy.Price)

	aggSell, ok := suite.k.GetAggregatedOrder(suite.ctx, mSell.MarketId, mSell.OrderType, mSell.Price)
	suite.Require().True(ok)
	suite.Require().Equal(aggSell.MarketId, mSell.MarketId)
	suite.Require().Equal(aggSell.OrderType, mSell.OrderType)
	suite.Require().Equal(aggSell.Price, mSell.Price)

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

func (suite *IntegrationTestSuite) TestQueueMessageProcessor_CancelOrder() {
	//create test market
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().Nil(err)

	//create an user account
	addr1 := sdk.AccAddress("addr1_______________")

	//create two random orders
	mBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      "100",
		Price:       "0.182",
		OrderType:   types.OrderTypeBuy,
		Owner:       addr1.String(),
	}

	mSell := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      "1000",
		Price:       "1.35",
		OrderType:   types.OrderTypeSell,
		Owner:       addr1.String(),
	}

	totalBuyAmount := math.ZeroInt()
	totalSellAmount := math.ZeroInt()
	totalBuyCoins := sdk.NewCoin(market.Quote, math.ZeroInt())
	totalSellCoins := sdk.NewCoin(market.Base, math.ZeroInt())
	//set messages in queue
	for i := 0; i < 3; i++ {
		suite.k.SetQueueMessage(suite.ctx, mBuy)
		buyAmtInt, _ := math.NewIntFromString(mBuy.Amount)
		totalBuyAmount = totalBuyAmount.Add(buyAmtInt)
		buyCoins, _, err := suite.k.GetOrderSdkCoin(mBuy.OrderType, mBuy.Price, buyAmtInt, &market)
		suite.Require().Nil(err)
		totalBuyCoins = totalBuyCoins.Add(buyCoins)

		suite.k.SetQueueMessage(suite.ctx, mSell)
		sellAmtInt, _ := math.NewIntFromString(mSell.Amount)
		totalSellAmount = totalSellAmount.Add(sellAmtInt)
		sellCoins, _, err := suite.k.GetOrderSdkCoin(mSell.OrderType, mSell.Price, sellAmtInt, &market)
		suite.Require().Nil(err)
		totalSellCoins = totalSellCoins.Add(sellCoins)
	}

	//call engine
	engine.ProcessQueueMessages(suite.ctx)

	//check orders were created
	allUserOrders, err := suite.k.UserMarketOrders(suite.ctx, &types.QueryUserMarketOrdersRequest{
		Address:    addr1.String(),
		Market:     getMarketId(),
		Pagination: nil,
	})
	suite.Require().NotNil(allUserOrders)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(allUserOrders.List)

	//store aggregates so we can check the amounts
	aggOrderBuy, ok := suite.k.GetAggregatedOrder(suite.ctx, mBuy.MarketId, mBuy.OrderType, mBuy.Price)
	suite.Require().True(ok)
	//check aggregate total amount is equal tot total amounts of the orders we placed
	suite.Require().EqualValues(aggOrderBuy.Amount, totalBuyAmount.String())
	aggOrderSell, ok := suite.k.GetAggregatedOrder(suite.ctx, mSell.MarketId, mSell.OrderType, mSell.Price)
	suite.Require().True(ok)
	//check aggregate total amount is equal tot total amounts of the orders we placed
	suite.Require().EqualValues(aggOrderSell.Amount, totalSellAmount.String())

	//create cancel messages and store found orders to check storage later with their details
	var orders []types.OrderReference
	cancelCount := 0
	for _, or := range allUserOrders.List {
		toCancelOrder, ok := suite.k.GetOrder(suite.ctx, or.MarketId, or.OrderType, or.Id)
		suite.Require().True(ok)
		canceledAmount, ok := math.NewIntFromString(toCancelOrder.Amount)
		suite.Require().True(ok)
		canceledCoins, _, err := suite.k.GetOrderSdkCoin(toCancelOrder.OrderType, toCancelOrder.Price, canceledAmount, &market)
		suite.Require().Nil(err)
		qm := types.QueueMessage{
			MarketId:    or.MarketId,
			MessageType: types.MessageTypeCancel,
			OrderId:     or.Id,
			OrderType:   or.OrderType,
			Owner:       addr1.String(),
		}
		orders = append(orders, or)
		suite.k.SetQueueMessage(suite.ctx, qm)
		suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(canceledCoins))
		//process cancel message
		engine.ProcessQueueMessages(suite.ctx)
		cancelCount++
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
	_, ok = suite.k.GetAggregatedOrder(suite.ctx, mBuy.MarketId, mBuy.OrderType, mBuy.Price)
	suite.Require().False(ok)

	_, ok = suite.k.GetAggregatedOrder(suite.ctx, mSell.MarketId, mSell.OrderType, mSell.Price)
	suite.Require().False(ok)

	//check orders no longer exist
	for _, o := range orders {
		_, ok = suite.k.GetOrder(suite.ctx, o.MarketId, o.OrderType, o.Id)
		suite.Require().False(ok)
	}
}

func (suite *IntegrationTestSuite) TestQueueMessageProcessor_OrderMatching() {
	//create test market
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().Nil(err)

	//create accounts
	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")

	//sellPrice := int64(10)
	sellPriceStr := "1"
	sellAmt := keeper.CalculateMinAmount(sellPriceStr).MulRaw(5)
	//buyPrice := int64(20)
	buyPriceStr := "2"
	buyAmt := keeper.CalculateMinAmount(buyPriceStr).MulRaw(5)
	orderCounter := int64(10)
	for i := int64(0); i < orderCounter; i++ {
		qmSell := types.QueueMessage{
			MarketId:    getMarketId(),
			MessageType: types.OrderTypeSell,
			Amount:      sellAmt.String(),
			Price:       sellPriceStr,
			OrderType:   types.OrderTypeSell,
			Owner:       makerAddr.String(),
		}
		suite.k.SetQueueMessage(suite.ctx, qmSell)
		qmBuy := types.QueueMessage{
			MarketId:    getMarketId(),
			MessageType: types.OrderTypeBuy,
			Amount:      buyAmt.String(),
			Price:       buyPriceStr,
			OrderType:   types.OrderTypeBuy,
			Owner:       makerAddr.String(),
		}
		suite.k.SetQueueMessage(suite.ctx, qmBuy)
	}

	//process initial messages
	engine.ProcessQueueMessages(suite.ctx)
	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(allOrders)
	suite.Require().Equal(len(allOrders), int(orderCounter)*2) //all orders should be there
	//check aggregated orders
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).String())

	//1. fill 50% of an order -> check its amount is updated -> check the maker gets his coins ->
	//check module balances updated -> check the taker balances are updated -> check aggregated is updated
	qmAmountInt := sellAmt.QuoRaw(2)
	qmBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      qmAmountInt.String(),
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}
	makerCoins, _, err := suite.k.GetOrderSdkCoin(qmBuy.OrderType, qmBuy.Price, qmAmountInt, &market)
	suite.Require().Nil(err)
	takerCoins, _, err := suite.k.GetOrderSdkCoin(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, qmAmountInt, &market)
	suite.Require().Nil(err)

	tradedStakeCoins := takerCoins

	suite.k.SetQueueMessage(suite.ctx, qmBuy)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(takerCoins))
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, sdk.NewCoins(makerCoins))
	engine.ProcessQueueMessages(suite.ctx)

	//check all orders are still there since none of them have been filled
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2))

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).Sub(takerCoins.Amount).String())

	//2. fill 25% of the order -> check all above again
	qmAmountInt = sellAmt.QuoRaw(4)
	qmBuy = types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      qmAmountInt.String(),
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}
	makerCoins, _, err = suite.k.GetOrderSdkCoin(qmBuy.OrderType, qmBuy.Price, qmAmountInt, &market)
	suite.Require().Nil(err)
	takerCoins, _, err = suite.k.GetOrderSdkCoin(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, qmAmountInt, &market)
	suite.Require().Nil(err)

	suite.k.SetQueueMessage(suite.ctx, qmBuy)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(takerCoins))
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, sdk.NewCoins(makerCoins))
	engine.ProcessQueueMessages(suite.ctx)

	//check all orders are still there since none of them were filled
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2))

	tradedStakeCoins = tradedStakeCoins.Add(takerCoins)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).Sub(tradedStakeCoins.Amount).String())

	//3. fill 200% of orders (2 * order amount) -> check all of the above again
	qmAmountInt = sellAmt.MulRaw(2)
	qmBuy = types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      qmAmountInt.String(),
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}
	makerCoins, _, err = suite.k.GetOrderSdkCoin(qmBuy.OrderType, qmBuy.Price, sellAmt, &market)
	suite.Require().Nil(err)
	takerCoins, _, err = suite.k.GetOrderSdkCoin(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, qmAmountInt, &market)
	suite.Require().Nil(err)

	suite.k.SetQueueMessage(suite.ctx, qmBuy)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, gomock.Any()).Times(3)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, gomock.Any()).Times(3)
	engine.ProcessQueueMessages(suite.ctx)

	//check the correct amount of suborders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2)-2)

	tradedStakeCoins = tradedStakeCoins.Add(takerCoins)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).Sub(tradedStakeCoins.Amount).String())

	//4. fill the rest + some amount to also create an order
	qmAmountInt = sellAmt.MulRaw(8)
	qmBuy = types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      qmAmountInt.String(),
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}
	makerCoins, _, err = suite.k.GetOrderSdkCoin(qmBuy.OrderType, qmBuy.Price, sellAmt, &market)
	suite.Require().Nil(err)
	takerCoins, _, err = suite.k.GetOrderSdkCoin(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, sellAmt, &market)
	suite.Require().Nil(err)

	suite.k.SetQueueMessage(suite.ctx, qmBuy)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, gomock.Any()).Times(8)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, gomock.Any()).Times(8)
	engine.ProcessQueueMessages(suite.ctx)

	//check the correct amount of suborders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2)-10+1)
	//only sell orders should exist now
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

	newOrderMakerCoins, _, err := suite.k.GetOrderSdkCoin(types.TheOtherOrderType(smallOrders[0].OrderType), smallOrders[0].Price, smallOrdAmt, &market)
	newOrderTakerCoins, _, err := suite.k.GetOrderSdkCoin(smallOrders[0].OrderType, smallOrders[0].Price, smallOrdAmt, &market)
	suite.Require().Nil(err)
	suite.Require().Nil(err)
	tradedStakeCoins = tradedStakeCoins.Add(takerCoins).Sub(newOrderMakerCoins)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, sellPriceStr, newOrderTakerCoins.Amount.String())
	//sell order should not exist anymore
	_, ok := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, sellPriceStr)
	suite.Require().False(ok)

	//5. fill all remaining orders
	qmAmountInt = buyAmt.MulRaw(12)
	qmSell := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      qmAmountInt.String(),
		Price:       buyPriceStr,
		OrderType:   types.OrderTypeSell,
		Owner:       takerAddr.String(),
	}

	makerCoins, _, err = suite.k.GetOrderSdkCoin(qmSell.OrderType, qmSell.Price, buyAmt, &market)
	suite.Require().Nil(err)
	takerCoins, _, err = suite.k.GetOrderSdkCoin(types.TheOtherOrderType(qmSell.OrderType), qmSell.Price, buyAmt, &market)
	suite.Require().Nil(err)

	suite.k.SetQueueMessage(suite.ctx, qmSell)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, sdk.NewCoins(takerCoins)).Times(10)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, sdk.NewCoins(makerCoins)).Times(10)
	engine.ProcessQueueMessages(suite.ctx)

	//check the correct amount of orders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), 2)
}

func (suite *IntegrationTestSuite) checkAggregatedOrder(marketId, orderType, price string, expectedAmount string) {
	agg, ok := suite.k.GetAggregatedOrder(suite.ctx, marketId, orderType, price)
	suite.Require().True(ok)
	suite.Require().Equal(agg.Amount, expectedAmount)
}

func (suite *IntegrationTestSuite) TestQueueMessageProcessor_OrderMatching_WithDust() {
	//create test market
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().Nil(err)

	//create accounts
	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")

	//sellPrice := int64(10)
	sellPriceStr := "0.7612"
	sellAmt := keeper.CalculateMinAmount(sellPriceStr).MulRaw(5)
	//buyPrice := int64(20)
	buyPriceStr := "0.950012"
	buyAmt := keeper.CalculateMinAmount(buyPriceStr).MulRaw(6)
	orderCounter := int64(10)
	for i := int64(0); i < orderCounter; i++ {
		qmSell := types.QueueMessage{
			MarketId:    getMarketId(),
			MessageType: types.OrderTypeSell,
			Amount:      sellAmt.String(),
			Price:       sellPriceStr,
			OrderType:   types.OrderTypeSell,
			Owner:       makerAddr.String(),
		}
		suite.k.SetQueueMessage(suite.ctx, qmSell)
		qmBuy := types.QueueMessage{
			MarketId:    getMarketId(),
			MessageType: types.OrderTypeBuy,
			Amount:      buyAmt.String(),
			Price:       buyPriceStr,
			OrderType:   types.OrderTypeBuy,
			Owner:       makerAddr.String(),
		}
		suite.k.SetQueueMessage(suite.ctx, qmBuy)
	}

	//process initial messages
	engine.ProcessQueueMessages(suite.ctx)
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
	qmBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      qmAmountInt.String(),
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}

	suite.k.SetQueueMessage(suite.ctx, qmBuy)
	engine.ProcessQueueMessages(suite.ctx)

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
	qmBuy = types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      qmAmountInt.String(),
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}

	suite.k.SetQueueMessage(suite.ctx, qmBuy)
	engine.ProcessQueueMessages(suite.ctx)

	//check all orders are still there since none of them were filled
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2))

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).Sub(math.NewInt(15)).String())

	//3. fill 200% of orders (2 * order amount) -> check all of the above again
	qmAmountInt = sellAmt.MulRaw(2)
	qmBuy = types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      qmAmountInt.String(),
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}

	suite.k.SetQueueMessage(suite.ctx, qmBuy)
	engine.ProcessQueueMessages(suite.ctx)

	//check the correct amount of suborders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2)-2)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt.MulRaw(orderCounter).String())
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt.MulRaw(orderCounter).Sub(math.NewInt(55)).String())

	//4. fill the rest + some amount to also create an order
	qmAmountInt = sellAmt.MulRaw(8)
	qmBuy = types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      qmAmountInt.String(),
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}

	suite.k.SetQueueMessage(suite.ctx, qmBuy)
	engine.ProcessQueueMessages(suite.ctx)

	//check the correct amount of suborders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2)-10+1)
	//only sell orders should exist now
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
	qmSell := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      qmAmountInt.String(),
		Price:       buyPriceStr,
		OrderType:   types.OrderTypeSell,
		Owner:       takerAddr.String(),
	}

	suite.k.SetQueueMessage(suite.ctx, qmSell)
	engine.ProcessQueueMessages(suite.ctx)

	//check the correct amount of orders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), 2)
}

func (suite *IntegrationTestSuite) TestQueueMessageProcessor_OrderBookPerBlockMessagesLimit() {
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().Nil(err)

	addr1 := sdk.AccAddress("addr1_______________")

	// Create more messages than the limit allows
	messageCount := 10
	limit := uint64(5)

	// Update params to set a lower limit for testing
	params := suite.k.GetParams(suite.ctx)
	params.OrderBookPerBlockMessages = limit
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	// Add messages to queue
	for i := 0; i < messageCount; i++ {
		mBuy := types.QueueMessage{
			MarketId:    getMarketId(),
			MessageType: types.OrderTypeBuy,
			Amount:      keeper.CalculateMinAmount("100").String(),
			Price:       "100",
			OrderType:   types.OrderTypeBuy,
			Owner:       addr1.String(),
		}
		suite.k.SetQueueMessage(suite.ctx, mBuy)
	}

	// Check message counter was incremented
	mCnt := suite.k.GetQueueMessageCounter(suite.ctx)
	suite.Require().Equal(uint64(messageCount), mCnt)

	// Process queue messages
	engine.ProcessQueueMessages(suite.ctx)

	// Check that only 'limit' orders were created (not all messages were processed)
	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(int(limit), len(allOrders))

	// Check that HasQueueMessages returns true (remaining messages in queue)
	hasMessages := suite.k.HasQueueMessages(suite.ctx)
	suite.Require().True(hasMessages)

	// Check that counter was NOT reset (because queue still has messages)
	// Counter should still be at the original value since we didn't reset
	mCnt = suite.k.GetQueueMessageCounter(suite.ctx)
	suite.Require().Equal(uint64(messageCount), mCnt)

	// Process remaining messages
	engine.ProcessQueueMessages(suite.ctx)

	// Now all orders should be created
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(messageCount, len(allOrders))

	// Queue should be empty now
	hasMessages = suite.k.HasQueueMessages(suite.ctx)
	suite.Require().False(hasMessages)

	// Counter should be reset now
	mCnt = suite.k.GetQueueMessageCounter(suite.ctx)
	suite.Require().Equal(uint64(0), mCnt)
}

func (suite *IntegrationTestSuite) TestQueueMessageProcessor_CounterResetOnlyWhenEmpty() {
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().Nil(err)

	addr1 := sdk.AccAddress("addr1_______________")

	// Add one message
	mBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      keeper.CalculateMinAmount("100").String(),
		Price:       "100",
		OrderType:   types.OrderTypeBuy,
		Owner:       addr1.String(),
	}
	suite.k.SetQueueMessage(suite.ctx, mBuy)

	// Check counter is 1
	mCnt := suite.k.GetQueueMessageCounter(suite.ctx)
	suite.Require().Equal(uint64(1), mCnt)

	// Process the message
	engine.ProcessQueueMessages(suite.ctx)

	// Queue should be empty
	hasMessages := suite.k.HasQueueMessages(suite.ctx)
	suite.Require().False(hasMessages)

	// Counter should be reset to 0
	mCnt = suite.k.GetQueueMessageCounter(suite.ctx)
	suite.Require().Equal(uint64(0), mCnt)

	// Add another message to verify counter starts from 0
	suite.k.SetQueueMessage(suite.ctx, mBuy)
	mCnt = suite.k.GetQueueMessageCounter(suite.ctx)
	suite.Require().Equal(uint64(1), mCnt)
}

func (suite *IntegrationTestSuite) TestHasQueueMessages() {
	// Initially empty queue
	hasMessages := suite.k.HasQueueMessages(suite.ctx)
	suite.Require().False(hasMessages)

	addr1 := sdk.AccAddress("addr1_______________")

	// Add a message
	mBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      keeper.CalculateMinAmount("100").String(),
		Price:       "100",
		OrderType:   types.OrderTypeBuy,
		Owner:       addr1.String(),
	}
	suite.k.SetQueueMessage(suite.ctx, mBuy)

	// Should have messages now
	hasMessages = suite.k.HasQueueMessages(suite.ctx)
	suite.Require().True(hasMessages)

	// Get all messages to retrieve the actual messageId
	allMessages := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(allMessages, 1)

	// Remove the message using the retrieved messageId
	suite.k.RemoveQueueMessage(suite.ctx, allMessages[0].MarketId, allMessages[0].MessageId)

	// Should be empty again
	hasMessages = suite.k.HasQueueMessages(suite.ctx)
	suite.Require().False(hasMessages)
}

// TestFillAggregatedOrder_ContinuePastUnfillableOrder verifies that when getExecutedAmount returns zero for an order
// (because the order amount is between minAmount and 2*minAmount, making it impossible to partially fill without
// leaving a remainder below minAmount), the iteration continues to the next order instead of breaking.
// This ensures that all compatible orders at the same price level are considered for filling.
func (suite *IntegrationTestSuite) TestFillAggregatedOrder_ContinuePastUnfillableOrder() {
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().Nil(err)

	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")

	price := "1"
	minAmount := keeper.CalculateMinAmount(price) // ceil(1/1)*2 = 2

	// Create a small sell order with amount between minAmount and 2*minAmount.
	// For msgAmount=minAmount=2 and orderAmount=3:
	//   getExecutedAmount(2, 3, 2): orderRemaining=1 < minAmount, executedAmount=1 < minAmount → returns 0
	smallSellAmount := minAmount.AddRaw(1) // 3
	qmSmallSell := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      smallSellAmount.String(),
		Price:       price,
		OrderType:   types.OrderTypeSell,
		Owner:       makerAddr.String(),
	}
	suite.k.SetQueueMessage(suite.ctx, qmSmallSell)

	// Create a large sell order that can be partially filled.
	// For msgAmount=2 and orderAmount=100: orderRemaining=98 >= minAmount → returns 2
	largeSellAmount := minAmount.MulRaw(50) // 100
	qmLargeSell := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      largeSellAmount.String(),
		Price:       price,
		OrderType:   types.OrderTypeSell,
		Owner:       makerAddr.String(),
	}
	suite.k.SetQueueMessage(suite.ctx, qmLargeSell)

	// Process sell orders to place them in the order book
	engine.ProcessQueueMessages(suite.ctx)

	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(allOrders, 2)

	totalSellAmount := smallSellAmount.Add(largeSellAmount) // 103
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, price, totalSellAmount.String())

	// Submit a buy message with amount = minAmount.
	// The small sell order cannot be filled (getExecutedAmount returns 0) but with 'continue',
	// the large sell order should still be reached and partially filled.
	buyAmount := minAmount // 2
	qmBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      buyAmount.String(),
		Price:       price,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}
	suite.k.SetQueueMessage(suite.ctx, qmBuy)

	// Expect exactly 1 fill: maker gets quote coins, taker gets base coins
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, gomock.Any()).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, gomock.Any()).Times(1)

	engine.ProcessQueueMessages(suite.ctx)

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

// TestFillAggregatedOrder_EarlyExitMsgBelowMinAmount verifies that when the remaining message amount drops below
// minAmount after a partial fill, the iteration breaks early. The remaining amount is refunded to the message owner.
func (suite *IntegrationTestSuite) TestFillAggregatedOrder_EarlyExitMsgBelowMinAmount() {
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().Nil(err)

	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")

	price := "1"
	minAmount := keeper.CalculateMinAmount(price) // 2

	// Create two sell orders at the same price, each with amount = minAmount (fully fillable)
	sellAmount := minAmount // 2
	for i := 0; i < 2; i++ {
		qmSell := types.QueueMessage{
			MarketId:    getMarketId(),
			MessageType: types.OrderTypeSell,
			Amount:      sellAmount.String(),
			Price:       price,
			OrderType:   types.OrderTypeSell,
			Owner:       makerAddr.String(),
		}
		suite.k.SetQueueMessage(suite.ctx, qmSell)
	}

	engine.ProcessQueueMessages(suite.ctx)

	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(allOrders, 2)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, price, sellAmount.MulRaw(2).String()) // 4

	// Submit a buy message with amount = minAmount + 1 = 3.
	// First sell order (amount=2): getExecutedAmount(3, 2, 2) → msgAmount >= orderAmount → returns 2 (fully fills)
	//   msgAmountInt = 3 - 2 = 1
	// Second iteration: msgAmountInt(1) < minAmount(2) → break (new early exit)
	// Second sell order is never reached.
	// Remaining 1 is refunded to the buyer.
	buyAmount := minAmount.AddRaw(1) // 3
	qmBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      buyAmount.String(),
		Price:       price,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}
	suite.k.SetQueueMessage(suite.ctx, qmBuy)

	// Expect 1 fill (2 bank calls) + 1 refund (1 bank call to taker)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, gomock.Any()).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, gomock.Any()).Times(2)

	engine.ProcessQueueMessages(suite.ctx)

	// First sell order should be removed (fully filled), second should remain untouched
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(allOrders, 1)
	suite.Require().Equal(sellAmount.String(), allOrders[0].Amount)

	// Aggregated sell should be just the remaining second order: 2
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, price, sellAmount.String())
}

// TestFillAggregatedOrder_ContinueThenEarlyExit exercises both code changes together:
// 1. An unfillable order is skipped (continue instead of break)
// 2. A subsequent order is partially filled
// 3. The remaining message amount drops below minAmount, causing an early exit
// 4. Further orders are not reached, and the remainder is refunded
func (suite *IntegrationTestSuite) TestFillAggregatedOrder_ContinueThenEarlyExit() {
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.k, suite.bankMock, suite.k.Logger())
	suite.Require().Nil(err)

	makerAddr := sdk.AccAddress("addr1_______________")
	takerAddr := sdk.AccAddress("addr2_______________")

	// Use price "0.1" for a larger minAmount, giving more room for the test scenario
	price := "0.1"
	minAmount := keeper.CalculateMinAmount(price) // ceil(1/0.1)*2 = 20

	// Order 1: unfillable (amount between minAmount and 2*minAmount)
	// For msgAmount=25 and orderAmount=30:
	//   getExecutedAmount(25, 30, 20): orderRemaining=5 < 20, executedAmount=10 < 20 → returns 0
	unfillableAmount := minAmount.AddRaw(10) // 30
	qmSell1 := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      unfillableAmount.String(),
		Price:       price,
		OrderType:   types.OrderTypeSell,
		Owner:       makerAddr.String(),
	}
	suite.k.SetQueueMessage(suite.ctx, qmSell1)

	// Order 2: partially fillable, but the partial fill leaves msgAmount < minAmount
	// For msgAmount=25 and orderAmount=35:
	//   getExecutedAmount(25, 35, 20): orderRemaining=10 < 20, executedAmount=15 >= 20? No, 15 < 20 → returns 0
	// Hmm, let me recalculate. executedAmount = 35 - 20 = 15. 15 >= 20? No → returns 0.
	// That's also unfillable! I need a different amount.
	//
	// For orderAmount=45 and msgAmount=25:
	//   getExecutedAmount(25, 45, 20): orderRemaining=20 >= 20 → returns 25
	// That fills the full msgAmount. I need partial fill.
	//
	// For orderAmount=40 and msgAmount=25:
	//   getExecutedAmount(25, 40, 20): orderRemaining=15 < 20, executedAmount=20 >= 20 → returns 20
	//   msgAmountInt = 25-20 = 5 < minAmount(20) → early exit!
	partialFillAmount := minAmount.MulRaw(2) // 40
	qmSell2 := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      partialFillAmount.String(),
		Price:       price,
		OrderType:   types.OrderTypeSell,
		Owner:       makerAddr.String(),
	}
	suite.k.SetQueueMessage(suite.ctx, qmSell2)

	// Order 3: large order that should NOT be reached due to early exit
	largeAmount := minAmount.MulRaw(10) // 200
	qmSell3 := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      largeAmount.String(),
		Price:       price,
		OrderType:   types.OrderTypeSell,
		Owner:       makerAddr.String(),
	}
	suite.k.SetQueueMessage(suite.ctx, qmSell3)

	engine.ProcessQueueMessages(suite.ctx)

	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(allOrders, 3)
	totalSellAmount := unfillableAmount.Add(partialFillAmount).Add(largeAmount) // 30+40+200=270
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, price, totalSellAmount.String())

	// Submit buy message with amount=25 (> minAmount=20)
	// Flow:
	// 1. Order 1 (30): getExecutedAmount(25,30,20) → orderRemaining=5<20, executedAmount=10<20 → 0 → continue
	// 2. Order 2 (40): getExecutedAmount(25,40,20) → orderRemaining=15<20, executedAmount=20>=20 → 20
	//    msgAmountInt = 25-20 = 5, orderAmount = 40-20 = 20
	// 3. Next iteration: msgAmountInt(5) < minAmount(20) → break
	// 4. Order 3 (200): not reached
	// Remaining 5 is refunded to buyer
	buyAmount := minAmount.AddRaw(5) // 25
	qmBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      buyAmount.String(),
		Price:       price,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}
	suite.k.SetQueueMessage(suite.ctx, qmBuy)

	// 1 fill (maker + taker) + 1 refund (taker)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, makerAddr, gomock.Any()).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, takerAddr, gomock.Any()).Times(2)

	engine.ProcessQueueMessages(suite.ctx)

	// All 3 orders should still exist
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(allOrders, 3)

	// Verify each order's amount
	executedAmount := minAmount // 20 (what was filled from order 2)
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
