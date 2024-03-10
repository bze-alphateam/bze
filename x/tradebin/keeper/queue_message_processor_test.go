package keeper_test

import (
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func newStakeCoin(amt int64) sdk.Coin {
	return sdk.NewInt64Coin(denomStake, amt)
}

func newBzeCoin(amt int64) sdk.Coin {
	return sdk.NewInt64Coin(denomBze, amt)
}

func (suite *IntegrationTestSuite) TestQueueMessageProcessor_AddMakerOrder() {
	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper)
	suite.Require().Nil(err)

	addr1 := sdk.AccAddress("addr1_______________")

	mBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      100,
		Price:       "100",
		OrderType:   types.OrderTypeBuy,
		Owner:       addr1.String(),
	}
	mSell := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      1000,
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
	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper)
	suite.Require().Nil(err)

	//add some coins to module so it has what to send back on order cancel
	balances := sdk.NewCoins(newStakeCoin(10000), newBzeCoin(50000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	//create an user account
	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	//initial balances need to be 0
	initialUserBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().True(initialUserBalances.IsZero())

	//create two random orders
	mBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      100,
		Price:       "100",
		OrderType:   types.OrderTypeBuy,
		Owner:       addr1.String(),
	}
	buyCoins, err := suite.k.GetOrderCoins(mBuy.OrderType, mBuy.Price, mBuy.Amount, &market)
	suite.Require().Nil(err)
	mSell := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      1000,
		Price:       "10",
		OrderType:   types.OrderTypeSell,
		Owner:       addr1.String(),
	}
	sellCoins, err := suite.k.GetOrderCoins(mSell.OrderType, mSell.Price, mSell.Amount, &market)
	suite.Require().Nil(err)

	//set messages in queue
	suite.k.SetQueueMessage(suite.ctx, mBuy)
	suite.k.SetQueueMessage(suite.ctx, mSell)

	//call engine
	engine.ProcessQueueMessages(suite.ctx)

	//check orders were created
	allUserOrders, err := suite.k.UserMarketOrders(sdk.WrapSDKContext(suite.ctx), &types.QueryUserMarketOrdersRequest{
		Address:    addr1.String(),
		Market:     getMarketId(),
		Pagination: nil,
	})
	suite.Require().NotNil(allUserOrders)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(allUserOrders.List)

	//create cancel messages and store found orders to check storage later with their details
	var orders []types.OrderReference
	for _, or := range allUserOrders.List {
		qm := types.QueueMessage{
			MarketId:    or.MarketId,
			MessageType: types.OrderTypeCancel,
			OrderId:     or.Id,
			OrderType:   or.OrderType,
			Owner:       addr1.String(),
		}
		orders = append(orders, or)
		suite.k.SetQueueMessage(suite.ctx, qm)
	}
	//process cancel messages
	engine.ProcessQueueMessages(suite.ctx)

	//check user orders were canceled
	allUserOrders, err = suite.k.UserMarketOrders(sdk.WrapSDKContext(suite.ctx), &types.QueryUserMarketOrdersRequest{
		Address:    addr1.String(),
		Market:     getMarketId(),
		Pagination: nil,
	})
	suite.Require().NotNil(allUserOrders)
	//list now should be empty
	suite.Require().Empty(allUserOrders.List)

	//check order coins have been added to the owner account
	newBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().False(newBalances.IsZero())
	suite.Require().Equal(newBalances.AmountOf(buyCoins.Denom), buyCoins.Amount)
	suite.Require().Equal(newBalances.AmountOf(sellCoins.Denom), sellCoins.Amount)

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

func (suite *IntegrationTestSuite) TestQueueMessageProcessor_OrderMatching() {
	//create test market
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper)
	suite.Require().Nil(err)

	//add some coins to module, so it has what to send back on order cancel
	initialModuleBalances := sdk.NewCoins(newStakeCoin(10000000), newBzeCoin(50000000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, initialModuleBalances))
	moduleAddr := suite.app.AccountKeeper.GetModuleAddress(types.ModuleName)

	//create accounts
	makerAddr := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, makerAddr)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	takerAddr := sdk.AccAddress("addr2_______________")
	takerAccount := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, takerAddr)
	suite.app.AccountKeeper.SetAccount(suite.ctx, takerAccount)
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, takerAddr, initialModuleBalances))

	//initial initialModuleBalances need to be 0
	makerBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, makerAddr)
	suite.Require().True(makerBalance.IsZero())
	//initial initialModuleBalances the same as previously added
	takerBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, takerAddr)
	suite.Require().True(takerBalance.IsEqual(initialModuleBalances))

	//sellPrice := int64(10)
	sellPriceStr := "1"
	sellAmt := keeper.CalculateMinAmount(sellPriceStr) * 5
	//buyPrice := int64(20)
	buyPriceStr := "2"
	buyAmt := keeper.CalculateMinAmount(buyPriceStr) * 5
	orderCounter := int64(10)
	for i := int64(0); i < orderCounter; i++ {
		qmSell := types.QueueMessage{
			MarketId:    getMarketId(),
			MessageType: types.OrderTypeSell,
			Amount:      sellAmt,
			Price:       sellPriceStr,
			OrderType:   types.OrderTypeSell,
			Owner:       makerAddr.String(),
		}
		suite.k.SetQueueMessage(suite.ctx, qmSell)
		qmBuy := types.QueueMessage{
			MarketId:    getMarketId(),
			MessageType: types.OrderTypeBuy,
			Amount:      buyAmt,
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
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt*orderCounter)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, sellAmt*orderCounter)

	//1. fill 50% of an order -> check its amount is updated -> check the maker gets his coins ->
	//check module balances updated -> check the taker balances are updated -> check aggregated is updated
	qmBuy := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      sellAmt / 2,
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}
	makerCoins, err := suite.k.GetOrderCoins(qmBuy.OrderType, qmBuy.Price, qmBuy.Amount, &market)
	suite.Require().Nil(err)
	takerCoins, err := suite.k.GetOrderCoins(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, qmBuy.Amount, &market)
	suite.Require().Nil(err)

	tradedUbzeCoins := makerCoins
	tradedStakeCoins := takerCoins

	suite.k.SetQueueMessage(suite.ctx, qmBuy)
	engine.ProcessQueueMessages(suite.ctx)

	//check all orders are still there since none of them have been filled
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2))

	//check maker and taker new balances after the trade was filled
	makerBalance = suite.app.BankKeeper.GetAllBalances(suite.ctx, makerAddr)
	suite.Require().Equal(makerBalance.AmountOf(tradedUbzeCoins.Denom), tradedUbzeCoins.Amount)
	takerNewBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, takerAddr)
	suite.Require().Equal(takerNewBalance.AmountOf(tradedStakeCoins.Denom), tradedStakeCoins.Amount.Add(takerBalance.AmountOf(tradedStakeCoins.Denom)))

	//check module amounts were subtracted
	suite.checkModuleBalances(moduleAddr, tradedUbzeCoins, tradedStakeCoins, initialModuleBalances)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt*orderCounter)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, (sellAmt*orderCounter)-takerCoins.Amount.Int64())

	//2. fill 25% of the order -> check all above again
	qmBuy = types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      sellAmt / 4,
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}
	makerCoins, err = suite.k.GetOrderCoins(qmBuy.OrderType, qmBuy.Price, qmBuy.Amount, &market)
	suite.Require().Nil(err)
	takerCoins, err = suite.k.GetOrderCoins(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, qmBuy.Amount, &market)
	suite.Require().Nil(err)

	suite.k.SetQueueMessage(suite.ctx, qmBuy)
	engine.ProcessQueueMessages(suite.ctx)

	//check all orders are still there since none of them were filled
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2))

	tradedUbzeCoins = tradedUbzeCoins.Add(makerCoins)
	tradedStakeCoins = tradedStakeCoins.Add(takerCoins)

	makerBalance = suite.app.BankKeeper.GetAllBalances(suite.ctx, makerAddr)
	suite.Require().Equal(makerBalance.AmountOf(tradedUbzeCoins.Denom), tradedUbzeCoins.Amount)
	takerNewBalance = suite.app.BankKeeper.GetAllBalances(suite.ctx, takerAddr)
	suite.Require().Equal(takerNewBalance.AmountOf(tradedStakeCoins.Denom), tradedStakeCoins.Amount.Add(takerBalance.AmountOf(tradedStakeCoins.Denom)))

	suite.checkModuleBalances(moduleAddr, tradedUbzeCoins, tradedStakeCoins, initialModuleBalances)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt*orderCounter)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, (sellAmt*orderCounter)-tradedStakeCoins.Amount.Int64())

	//3. fill 200% of orders (2 * order amount) -> check all of the above again
	qmBuy = types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      sellAmt * 2,
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}
	makerCoins, err = suite.k.GetOrderCoins(qmBuy.OrderType, qmBuy.Price, qmBuy.Amount, &market)
	suite.Require().Nil(err)
	takerCoins, err = suite.k.GetOrderCoins(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, qmBuy.Amount, &market)
	suite.Require().Nil(err)

	suite.k.SetQueueMessage(suite.ctx, qmBuy)
	engine.ProcessQueueMessages(suite.ctx)

	//check the correct amount of suborders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), int(orderCounter*2)-2)

	tradedUbzeCoins = tradedUbzeCoins.Add(makerCoins)
	tradedStakeCoins = tradedStakeCoins.Add(takerCoins)

	makerBalance = suite.app.BankKeeper.GetAllBalances(suite.ctx, makerAddr)
	suite.Require().Equal(makerBalance.AmountOf(tradedUbzeCoins.Denom), tradedUbzeCoins.Amount)
	takerNewBalance = suite.app.BankKeeper.GetAllBalances(suite.ctx, takerAddr)
	suite.Require().Equal(takerNewBalance.AmountOf(tradedStakeCoins.Denom), tradedStakeCoins.Amount.Add(takerBalance.AmountOf(tradedStakeCoins.Denom)))

	suite.checkModuleBalances(moduleAddr, tradedUbzeCoins, tradedStakeCoins, initialModuleBalances)

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt*orderCounter)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, sellPriceStr, (sellAmt*orderCounter)-tradedStakeCoins.Amount.Int64())

	//4. fill the rest + some amount to also create an order
	qmBuy = types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeBuy,
		Amount:      sellAmt * 8,
		Price:       sellPriceStr,
		OrderType:   types.OrderTypeBuy,
		Owner:       takerAddr.String(),
	}
	makerCoins, err = suite.k.GetOrderCoins(qmBuy.OrderType, qmBuy.Price, qmBuy.Amount, &market)
	suite.Require().Nil(err)
	takerCoins, err = suite.k.GetOrderCoins(types.TheOtherOrderType(qmBuy.OrderType), qmBuy.Price, qmBuy.Amount, &market)
	suite.Require().Nil(err)

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
		if ord.Amount < sellAmt {
			smallOrders = append(smallOrders, ord)
		}
	}

	//check the smaller order is there, and it has the right values
	suite.Require().Equal(len(smallOrders), 1)
	suite.Require().Equal(smallOrders[0].Amount, sellAmt*3/4)
	suite.Require().Equal(smallOrders[0].Price, sellPriceStr)

	newOrderMakerCoins, err := suite.k.GetOrderCoins(types.TheOtherOrderType(smallOrders[0].OrderType), smallOrders[0].Price, smallOrders[0].Amount, &market)
	newOrderTakerCoins, err := suite.k.GetOrderCoins(smallOrders[0].OrderType, smallOrders[0].Price, smallOrders[0].Amount, &market)
	suite.Require().Nil(err)
	tradedUbzeCoins = tradedUbzeCoins.Add(makerCoins).Sub(newOrderTakerCoins)
	suite.Require().Nil(err)
	tradedStakeCoins = tradedStakeCoins.Add(takerCoins).Sub(newOrderMakerCoins)

	makerBalance = suite.app.BankKeeper.GetAllBalances(suite.ctx, makerAddr)
	suite.Require().Equal(makerBalance.AmountOf(tradedUbzeCoins.Denom), tradedUbzeCoins.Amount)
	takerNewBalance = suite.app.BankKeeper.GetAllBalances(suite.ctx, takerAddr)
	suite.Require().Equal(takerNewBalance.AmountOf(tradedStakeCoins.Denom), tradedStakeCoins.Amount.Add(takerBalance.AmountOf(tradedStakeCoins.Denom)))

	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, buyPriceStr, buyAmt*orderCounter)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, sellPriceStr, newOrderTakerCoins.Amount.Int64())
	//sell order should not exist anymore
	_, ok := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, sellPriceStr)
	suite.Require().False(ok)

	//5. fill all remaining orders
	qmSell := types.QueueMessage{
		MarketId:    getMarketId(),
		MessageType: types.OrderTypeSell,
		Amount:      buyAmt * 12,
		Price:       buyPriceStr,
		OrderType:   types.OrderTypeSell,
		Owner:       takerAddr.String(),
	}

	makerCoins, err = suite.k.GetOrderCoins(qmSell.OrderType, qmSell.Price, qmSell.Amount, &market)
	suite.Require().Nil(err)
	takerCoins, err = suite.k.GetOrderCoins(types.TheOtherOrderType(qmSell.OrderType), qmSell.Price, qmSell.Amount, &market)
	suite.Require().Nil(err)

	suite.k.SetQueueMessage(suite.ctx, qmSell)
	engine.ProcessQueueMessages(suite.ctx)

	//check the correct amount of orders removed
	allOrders = suite.k.GetAllOrder(suite.ctx)
	suite.Require().Equal(len(allOrders), 2)
}

func (suite *IntegrationTestSuite) checkModuleBalances(moduleAddr sdk.AccAddress, tradedUbzeCoins, tradedStakeCoins sdk.Coin, initialBalances sdk.Coins) {
	moduleBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, moduleAddr)
	suite.Require().Equal(moduleBalance.AmountOf(tradedUbzeCoins.Denom), initialBalances.AmountOf(tradedUbzeCoins.Denom).Sub(tradedUbzeCoins.Amount))
	suite.Require().Equal(moduleBalance.AmountOf(tradedStakeCoins.Denom), initialBalances.AmountOf(tradedStakeCoins.Denom).Sub(tradedStakeCoins.Amount))
}

func (suite *IntegrationTestSuite) checkAggregatedOrder(marketId, orderType, price string, expectedAmount int64) {
	agg, ok := suite.k.GetAggregatedOrder(suite.ctx, marketId, orderType, price)
	suite.Require().True(ok)
	suite.Require().Equal(agg.Amount, expectedAmount)
}
