package keeper_test

import (
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cometbft/cometbft/libs/rand"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestMsgCancelOrder_MarketNotFound() {
	_, err := suite.msgServer.CancelOrder(suite.ctx, &types.MsgCancelOrder{
		Creator:   sdk.AccAddress("addr1_______________").String(),
		MarketId:  "me/me",
		OrderId:   "",
		OrderType: "",
	})
	suite.Require().ErrorIs(err, types.ErrMarketNotFound)
}

func (suite *IntegrationTestSuite) TestMsgCancelOrder_OrderNotFound() {
	suite.k.SetMarket(suite.ctx, market)

	_, err := suite.msgServer.CancelOrder(suite.ctx, &types.MsgCancelOrder{
		Creator:   sdk.AccAddress("addr1_______________").String(),
		MarketId:  getMarketId(),
		OrderId:   "123",
		OrderType: "dsa",
	})
	suite.Require().ErrorIs(err, types.ErrOrderNotFound)
}

func (suite *IntegrationTestSuite) TestMsgCancelOrder_Unauthorized() {
	suite.k.SetMarket(suite.ctx, market)
	owner := sdk.AccAddress("addr1_______________")
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeBuy, "102", "1", owner))
	order, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, orderId(0))
	suite.Require().True(found)

	_, err := suite.msgServer.CancelOrder(suite.ctx, &types.MsgCancelOrder{
		Creator:   sdk.AccAddress("addr2_______________").String(),
		MarketId:  getMarketId(),
		OrderId:   order.Id,
		OrderType: types.OrderTypeBuy,
	})
	suite.Require().ErrorIs(err, types.ErrUnauthorizedOrder)

	// the order still rests untouched
	stored, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, order.Id)
	suite.Require().True(found)
	suite.Require().Equal(order, stored)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, "1", "102")
}

// cancelAndCheckRefund cancels a resting order through the message server and checks that the refund,
// the order removal and the aggregate removal all happen in that call
func (suite *IntegrationTestSuite) cancelAndCheckRefund(orderType, amount, price string, refund sdk.Coin) {
	suite.k.SetMarket(suite.ctx, market)
	owner := sdk.AccAddress("addr1_______________")
	suite.placeOrders(suite.newEngine(), limitOrder(orderType, amount, price, owner))
	order, found := suite.k.GetOrder(suite.ctx, getMarketId(), orderType, orderId(0))
	suite.Require().True(found)
	suite.checkAggregatedOrder(getMarketId(), orderType, price, amount)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(refund)).Return(nil).Times(1)
	_, err := suite.msgServer.CancelOrder(suite.ctx, &types.MsgCancelOrder{
		Creator:   owner.String(),
		MarketId:  getMarketId(),
		OrderId:   order.Id,
		OrderType: orderType,
	})
	suite.Require().NoError(err)

	_, found = suite.k.GetOrder(suite.ctx, getMarketId(), orderType, order.Id)
	suite.Require().False(found)
	_, found = suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), orderType, price)
	suite.Require().False(found)
	_, found = suite.getUserOrderRef(order)
	suite.Require().False(found)
	_, found = suite.getPriceOrderRef(order)
	suite.Require().False(found)
	// a cancel does not take an order id
	suite.Require().Equal(uint64(1), suite.k.GetOrderCounter(suite.ctx))
}

func (suite *IntegrationTestSuite) TestMsgCancelOrder_CancelBuy_RefundsInTx() {
	suite.cancelAndCheckRefund(types.OrderTypeBuy, "102000", "1", sdk.NewInt64Coin(market.Quote, 102000))
}

func (suite *IntegrationTestSuite) TestMsgCancelOrder_CancelSell_RefundsInTx() {
	suite.cancelAndCheckRefund(types.OrderTypeSell, "10200021", "1000", sdk.NewInt64Coin(market.Base, 10200021))
}

// TestMsgCancelOrder_SecondCancelOrderNotFound: the first cancel refunds and removes the order in the
// transaction, so a second cancel of the same order finds nothing and refunds nothing.
func (suite *IntegrationTestSuite) TestMsgCancelOrder_SecondCancelOrderNotFound() {
	suite.k.SetMarket(suite.ctx, market)
	addr1 := sdk.AccAddress("addr1_______________")
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeBuy, "102000", "1", addr1))
	order, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, orderId(0))
	suite.Require().True(found)

	refunds := 0
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 102000))).
		DoAndReturn(func(_ sdk.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
			refunds++
			return nil
		}).Times(1)

	cancelMsg := &types.MsgCancelOrder{
		Creator:   addr1.String(),
		MarketId:  getMarketId(),
		OrderId:   order.Id,
		OrderType: types.OrderTypeBuy,
	}
	_, err := suite.msgServer.CancelOrder(suite.ctx, cancelMsg)
	suite.Require().NoError(err)
	suite.Require().Equal(1, refunds, "the first cancel refunds in its own call")
	_, found = suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, order.Id)
	suite.Require().False(found)

	_, err = suite.msgServer.CancelOrder(suite.ctx, cancelMsg)
	suite.Require().ErrorIs(err, types.ErrOrderNotFound)
	suite.Require().Equal(1, refunds, "the second cancel refunds nothing")
}

func (suite *IntegrationTestSuite) TestCreateOrder_BuyAtMatchingSellPrice_OnlySell_Succeeds() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr2_______________")
	// Single sell order at price 2
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeSell, "10000", "2", maker))

	addr1 := sdk.AccAddress("addr1_______________")
	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)
	paidCoins := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(2_000_000)))

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, paidCoins).Return(nil).Times(1)
	// the resting sell is filled in the same call
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 20000))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 10000))).Return(nil).Times(1)

	// Buy at price 2 matches the only sell - should succeed
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "2",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().Nil(err)

	// the remainder rests as a buy with the taker's id
	_, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, "2")
	suite.Require().False(found)
	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().Equal(orderId(1), all[0].Id)
	suite.Require().Equal(types.OrderTypeBuy, all[0].OrderType)
	suite.Require().Equal("990000", all[0].Amount)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, "2", "990000")
}

func (suite *IntegrationTestSuite) TestCreateOrder_BuyAtHigherSellPrice_BetterSellExists_Fails() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr2_______________")
	// Two sell orders: one at price 3 (better) and one at price 5
	suite.placeOrders(suite.newEngine(),
		limitOrder(types.OrderTypeSell, "10000", "3", maker),
		limitOrder(types.OrderTypeSell, "10000", "5", maker),
	)

	addr1 := sdk.AccAddress("addr1_______________")

	// Buy at price 5 should fail because a better sell at price 3 exists
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "5",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().ErrorIs(err, types.ErrInvalidOrderPrice)

	// nothing executed: the book is untouched and no id was taken
	suite.Require().Len(suite.k.GetAllOrder(suite.ctx), 2)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, "3", "10000")
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, "5", "10000")
	suite.Require().Equal(uint64(2), suite.k.GetOrderCounter(suite.ctx))
}

func (suite *IntegrationTestSuite) TestCreateOrder_SellAtMatchingBuyPrice_OnlyBuy_Succeeds() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr2_______________")
	// Single buy order at price 1
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeBuy, "10000", "1", maker))

	addr1 := sdk.AccAddress("addr1_______________")
	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)
	paidCoins := sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(1000000)))

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, paidCoins).Return(nil).Times(1)
	// the resting buy is filled in the same call
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 10000))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 10000))).Return(nil).Times(1)

	// Sell at price 1 matches the only buy - should succeed
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "1",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().Nil(err)

	_, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, "1")
	suite.Require().False(found)
	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().Equal(orderId(1), all[0].Id)
	suite.Require().Equal(types.OrderTypeSell, all[0].OrderType)
	suite.Require().Equal("990000", all[0].Amount)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, "1", "990000")
}

func (suite *IntegrationTestSuite) TestCreateOrder_SellAtLowerBuyPrice_BetterBuyExists_Fails() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr2_______________")
	// Two buy orders: one at price 1 and one at price 5 (better for seller)
	suite.placeOrders(suite.newEngine(),
		limitOrder(types.OrderTypeBuy, "10000", "1", maker),
		limitOrder(types.OrderTypeBuy, "10000", "5", maker),
	)

	addr1 := sdk.AccAddress("addr1_______________")

	// Sell at price 1 should fail because a better buy at price 5 exists
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "1",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().ErrorIs(err, types.ErrInvalidOrderPrice)

	// nothing executed: the book is untouched and no id was taken
	suite.Require().Len(suite.k.GetAllOrder(suite.ctx), 2)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, "1", "10000")
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, "5", "10000")
	suite.Require().Equal(uint64(2), suite.k.GetOrderCounter(suite.ctx))
}

func (suite *IntegrationTestSuite) Msg_TestCreateMarket_InvalidDenom() {

	//same denom for both
	_, err := suite.msgServer.CreateMarket(suite.ctx, &types.MsgCreateMarket{
		Creator: "me",
		Base:    denomBze,
		Quote:   denomBze,
	})
	suite.Require().NotNil(err)

	suite.bankMock.EXPECT().HasSupply(gomock.Any(), gomock.AnyOf(denomStake, denomBze)).Return(false).Times(1)
	//denom has no supply
	_, err = suite.msgServer.CreateMarket(suite.ctx, &types.MsgCreateMarket{
		Creator: "me",
		Base:    denomStake,
		Quote:   denomBze,
	})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) Msg_TestCreateMarket_MarketAlreadyExist() {
	suite.k.SetMarket(suite.ctx, market)

	_, err := suite.msgServer.CreateMarket(suite.ctx, &types.MsgCreateMarket{
		Creator: "me",
		Base:    denomStake,
		Quote:   denomBze,
	})
	suite.Require().NotNil(err)

	_, err = suite.msgServer.CreateMarket(suite.ctx, &types.MsgCreateMarket{
		Creator: "me",
		Base:    denomBze,
		Quote:   denomStake,
	})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) Msg_TestCreateMarket_NotEnoughCoinsForFee() {

	addr1 := sdk.AccAddress("addr1_______________")
	marketFee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(25000000000)))

	suite.bankMock.EXPECT().HasSupply(gomock.Any(), denomStake).Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), denomBze).Return(true).Times(1)

	//expect fee capture to module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, marketFee).
		Times(1).
		Return(nil)

	//expect market fee to be sent to fee collector
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), marketFee).
		Times(1).
		Return(fmt.Errorf("not enough balance"))

	_, err := suite.msgServer.CreateMarket(suite.ctx, &types.MsgCreateMarket{
		Creator: addr1.String(),
		Base:    denomStake,
		Quote:   denomBze,
	})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) Msg_TestCreateMarket_Success() {

	addr1 := sdk.AccAddress("addr1_______________")
	marketFee := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(25000000000)))

	suite.bankMock.EXPECT().HasSupply(gomock.Any(), denomStake).Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), denomBze).Return(true).Times(1)

	//expect fee capture to module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, marketFee).
		Times(1).
		Return(nil)

	//expect market fee to be sent to fee collector
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), marketFee).
		Times(1).
		Return(nil)

	newMarket := types.Market{
		Creator: addr1.String(),
		Base:    denomStake,
		Quote:   denomBze,
	}
	_, err := suite.msgServer.CreateMarket(suite.ctx, &types.MsgCreateMarket{
		Creator: addr1.String(),
		Base:    denomStake,
		Quote:   denomBze,
	})

	suite.Require().Nil(err)
	storageMarket, ok := suite.k.GetMarket(suite.ctx, newMarket.Base, newMarket.Quote)
	suite.Require().True(ok)
	suite.Require().Equal(newMarket, storageMarket)
}

func (suite *IntegrationTestSuite) Msg_TestCreateOrder_InvalidAmount() {

	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Amount: "hdsihdshdshids",
		Price:  "1",
	})

	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "amount could not be converted to Int")
}

func (suite *IntegrationTestSuite) Msg_TestCreateOrder_AmountTooLow() {

	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Amount: "1",
		Price:  "1",
	})

	suite.Require().NotNil(err)
	//amount should be bigger than
	suite.Require().Contains(err.Error(), "amount should be bigger than")
}

func (suite *IntegrationTestSuite) Msg_TestCreateOrder_MarketNotFound() {

	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Amount:   "1000000",
		Price:    "1",
		MarketId: "notamarket/notatall",
	})

	suite.Require().NotNil(err)
	//market id
	suite.Require().Contains(err.Error(), "market id")
}

func (suite *IntegrationTestSuite) Msg_TestCreateOrder_InvalidOrderType() {
	suite.k.SetMarket(suite.ctx, market)

	addr1 := sdk.AccAddress("addr1_______________")
	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "1",
		MarketId:  getMarketId(),
		OrderType: "notatype",
		Creator:   addr1.String(),
	})

	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "order type")
}

func (suite *IntegrationTestSuite) Msg_TestCreateOrder_InvalidCreator() {
	suite.k.SetMarket(suite.ctx, market)

	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "1",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   "notanaddress",
	})

	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "bech32")
}

func (suite *IntegrationTestSuite) TestMsgCreateOrder_MarketMaker_Buy_Success_ZeroDust() {
	suite.k.SetMarket(suite.ctx, market)

	addr1 := sdk.AccAddress("addr1_______________")
	//fee should be captured
	params := suite.k.GetParams(suite.ctx)
	makerFee := sdk.NewCoins(params.MarketMakerFee)
	paidCoins := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(2_000_000)))

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, paidCoins).Return(nil).Times(1)

	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "2",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().Nil(err)

	suite.requireRestingOrder(orderId(0), orderMsg)

	_, ok := suite.k.GetUserDust(suite.ctx, addr1.String(), market.Quote)
	suite.Require().False(ok)

	_, ok = suite.k.GetUserDust(suite.ctx, addr1.String(), market.Base)
	suite.Require().False(ok)
}

// requireRestingOrder checks the message rests on the book, whole, under the given id
func (suite *IntegrationTestSuite) requireRestingOrder(id string, orderMsg types.MsgCreateOrder) {
	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().Equal(id, all[0].Id)
	suite.Require().Equal(orderMsg.MarketId, all[0].MarketId)
	suite.Require().Equal(orderMsg.Amount, all[0].Amount)
	suite.Require().Equal(orderMsg.OrderType, all[0].OrderType)
	suite.Require().Equal(orderMsg.Price, all[0].Price)
	suite.Require().Equal(orderMsg.Creator, all[0].Owner)
	suite.checkAggregatedOrder(orderMsg.MarketId, orderMsg.OrderType, orderMsg.Price, orderMsg.Amount)
}

func (suite *IntegrationTestSuite) TestMsgCreateOrder_MarketTaker_Buy_Success_ZeroDust() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr2_______________")
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeSell, "10000", "2", maker))

	addr1 := sdk.AccAddress("addr1_______________")
	//fee should be captured
	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)
	paidCoins := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(2_000_000)))

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, paidCoins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 20000))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 10000))).Return(nil).Times(1)

	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "2",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().Nil(err)

	// the remainder rests under the taker's id
	remainderMsg := orderMsg
	remainderMsg.Amount = "990000"
	suite.requireRestingOrder(orderId(1), remainderMsg)

	_, ok := suite.k.GetUserDust(suite.ctx, addr1.String(), market.Quote)
	suite.Require().False(ok)

	_, ok = suite.k.GetUserDust(suite.ctx, addr1.String(), market.Base)
	suite.Require().False(ok)
}

func (suite *IntegrationTestSuite) TestMsgCreateOrder_MarketTaker_Buy_Success_WithDust() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr2_______________")
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeSell, "100000", "0.02331", maker))

	addr1 := sdk.AccAddress("addr1_______________")
	//fee should be captured
	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)
	paidCoins := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(3)))

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, paidCoins).Return(nil).Times(1)
	// 87 * 0.02331 = 2.02797: the maker gets 2, the taker gets the 87 base it bought
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 2))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 87))).Return(nil).Times(1)
	orderMsg := types.MsgCreateOrder{
		Amount:    "87",
		Price:     "0.02331",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().Nil(err)

	// fully filled: nothing of the taker rests, the maker keeps the rest of its order
	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().Equal(orderId(0), all[0].Id)
	suite.Require().Equal("99913", all[0].Amount)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, "0.02331", "99913")
	_, ok := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, "0.02331")
	suite.Require().False(ok)

	udQuote, ok := suite.k.GetUserDust(suite.ctx, addr1.String(), market.Quote)
	suite.Require().True(ok)
	suite.Require().Equal(udQuote.Denom, market.Quote)
	suite.Require().Equal(udQuote.Owner, addr1.String())
	suite.Require().EqualValues(udQuote.Amount, "0.972030000000000000")

	_, ok = suite.k.GetUserDust(suite.ctx, addr1.String(), market.Base)
	suite.Require().False(ok)

	makerDust, ok := suite.k.GetUserDust(suite.ctx, maker.String(), market.Quote)
	suite.Require().True(ok)
	suite.Require().EqualValues("0.027970000000000000", makerDust.Amount)
}

func (suite *IntegrationTestSuite) TestMsgCreateOrder_MarketMaker_Sell_Success() {
	suite.k.SetMarket(suite.ctx, market)

	addr1 := sdk.AccAddress("addr1_______________")

	//fee should be captured
	params := suite.k.GetParams(suite.ctx)
	makerFee := sdk.NewCoins(params.MarketMakerFee)
	paidCoins := sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(1000000)))

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, paidCoins).Return(nil).Times(1)
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "1",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().Nil(err)

	suite.requireRestingOrder(orderId(0), orderMsg)

	//should never have dust on sell orders
	_, ok := suite.k.GetUserDust(suite.ctx, addr1.String(), market.Quote)
	suite.Require().False(ok)

	_, ok = suite.k.GetUserDust(suite.ctx, addr1.String(), market.Base)
	suite.Require().False(ok)
}

func (suite *IntegrationTestSuite) TestMsgCreateOrder_MarketTaker_Sell_Success() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr2_______________")
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeBuy, "10000", "1", maker))

	addr1 := sdk.AccAddress("addr1_______________")

	//fee should be captured
	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)
	paidCoins := sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(1000000)))

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, paidCoins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 10000))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 10000))).Return(nil).Times(1)
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "1",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().Nil(err)

	remainderMsg := orderMsg
	remainderMsg.Amount = "990000"
	suite.requireRestingOrder(orderId(1), remainderMsg)

	//should never have dust on sell orders
	_, ok := suite.k.GetUserDust(suite.ctx, addr1.String(), market.Quote)
	suite.Require().False(ok)

	_, ok = suite.k.GetUserDust(suite.ctx, addr1.String(), market.Base)
	suite.Require().False(ok)
}

func (suite *IntegrationTestSuite) Msg_TestCreateOrder_MarketTaker_StressBalance() {
	suite.k.SetMarket(suite.ctx, market)

	//create initial random markets
	testDenom1 := "test1"
	testDenom2 := "test2"
	market1 := types.Market{
		Base:    testDenom1,
		Quote:   testDenom2,
		Creator: "addr1",
	}
	market2 := types.Market{
		Base:    denomStake,
		Quote:   testDenom2,
		Creator: "addr1",
	}
	market3 := types.Market{
		Base:    denomBze,
		Quote:   testDenom1,
		Creator: "addr1",
	}

	market4 := types.Market{
		Base:    denomBze,
		Quote:   testDenom2,
		Creator: "addr1",
	}
	marketsMap := make(map[string]types.Market)
	marketsMap[types.CreateMarketId(market.Base, market.Quote)] = market
	marketsMap[types.CreateMarketId(market1.Base, market1.Quote)] = market1
	marketsMap[types.CreateMarketId(market2.Base, market2.Quote)] = market2
	marketsMap[types.CreateMarketId(market3.Base, market3.Quote)] = market3
	marketsMap[types.CreateMarketId(market4.Base, market4.Quote)] = market4
	suite.k.SetMarket(suite.ctx, market1)
	suite.k.SetMarket(suite.ctx, market2)
	suite.k.SetMarket(suite.ctx, market3)
	suite.k.SetMarket(suite.ctx, market4)

	var creators []string
	for i := 0; i < 10; i++ {
		addr1 := sdk.AccAddress(fmt.Sprintf("addr%d_______________", i))
		creators = append(creators, addr1.String())
	}

	allPaid := sdk.NewCoins()
	// every order executes in its own CreateOrder call: settlements and refunds are paid out of what was escrowed
	paidOut := sdk.NewCoins()
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx sdk.Context, moduleName string, recipient sdk.AccAddress, coins sdk.Coins) error {
			paidOut = paidOut.Add(coins...)
			return nil
		}).AnyTimes()
	for i := 0; i < 5; i++ {
		_, paid := suite.randomOrderCreateMessages(suite.randomNumber(1000), creators, market)
		allPaid = allPaid.Add(paid...)
		_, paid = suite.randomOrderCreateMessages(suite.randomNumber(1000), creators, market1)
		allPaid = allPaid.Add(paid...)
		_, paid = suite.randomOrderCreateMessages(suite.randomNumber(1000), creators, market2)
		allPaid = allPaid.Add(paid...)
		_, paid = suite.randomOrderCreateMessages(suite.randomNumber(1000), creators, market3)
		allPaid = allPaid.Add(paid...)
		_, paid = suite.randomOrderCreateMessages(suite.randomNumber(1000), creators, market4)
		allPaid = allPaid.Add(paid...)
	}

	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(allOrders)
	amounts := sdk.NewCoins()
	//check module balance is equal to the coins found
	foundOrderTypesByPrice := make(map[string]string)
	aggregatedOrders := make(map[string]types.AggregatedOrder)
	for _, or := range allOrders {
		//search to see if an opposite order with the same price already exists
		//if so it means the engine failed to match them when they were processed
		oppositeKey := fmt.Sprintf("%s_%s_%s", or.MarketId, or.Price, types.TheOtherOrderType(or.OrderType))
		id, ok := foundOrderTypesByPrice[oppositeKey]
		suite.Require().False(ok, fmt.Sprintf("order [%s] has the same price [%s] as [%s]", id, or.Price, or.Id))

		//save locally to check later
		key := fmt.Sprintf("%s_%s_%s", or.MarketId, or.Price, or.OrderType)
		foundOrderTypesByPrice[key] = or.Id
		amtInt, ok := math.NewIntFromString(or.Amount)
		suite.Require().True(ok)
		pickMarket, ok := marketsMap[or.MarketId]
		suite.Require().True(ok)
		coins, _, err := suite.k.GetOrderSdkCoin(or.OrderType, or.Price, amtInt, &pickMarket)
		suite.Require().NoError(err)

		amounts = amounts.Add(coins)
		//save in aggregated orders map so we can check it later
		agg, found := aggregatedOrders[key]
		if !found {
			agg = types.AggregatedOrder{
				MarketId:  or.MarketId,
				OrderType: or.OrderType,
				Amount:    "0",
				Price:     or.Price,
			}
		}
		aggAmt, ok := math.NewIntFromString(agg.Amount)
		suite.Require().True(ok)
		aggAmt = aggAmt.Add(amtInt)
		agg.Amount = aggAmt.String()
		aggregatedOrders[key] = agg
	}

	for _, agg := range aggregatedOrders {
		foundAgg, ok := suite.k.GetAggregatedOrder(suite.ctx, agg.MarketId, agg.OrderType, agg.Price)
		suite.Require().True(ok)
		suite.Require().Equal(foundAgg.Amount, agg.Amount)
	}

	// what was escrowed is either paid out or still rests on the book
	suite.Require().Equal(allPaid, amounts.Add(paidOut...))
}

func (suite *IntegrationTestSuite) TestMsgCreateOrder_CheckPrice_Fail() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr2_______________")
	suite.placeOrders(suite.newEngine(),
		limitOrder(types.OrderTypeBuy, "10000", "1", maker),
		limitOrder(types.OrderTypeSell, "10000", "5", maker),
	)

	addr1 := sdk.AccAddress("addr1_______________")

	//check price error on sell order
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "0.5",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().ErrorIs(err, types.ErrInvalidOrderPrice)
	suite.Require().Len(suite.k.GetAllOrder(suite.ctx), 2)

	//check price error on buy
	orderMsg = types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "5.1",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err = suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().ErrorIs(err, types.ErrInvalidOrderPrice)
	suite.Require().Len(suite.k.GetAllOrder(suite.ctx), 2)

	//fee should be captured
	params := suite.k.GetParams(suite.ctx)
	makerFee := sdk.NewCoins(params.MarketMakerFee)

	paidCoins := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(4000000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, paidCoins).Return(nil).Times(1)
	//add 2 new resting orders that narrow the spread, then check the price validator against them
	orderMsg = types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "4",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err = suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().NoError(err)

	paidCoins = sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(1000000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, paidCoins).Return(nil).Times(1)
	orderMsg = types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "4.5",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err = suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().NoError(err)
	suite.Require().Len(suite.k.GetAllOrder(suite.ctx), 4)
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeBuy, "4", "1000000")
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, "4.5", "1000000")

	orderMsg = types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "3.5",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err = suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().ErrorIs(err, types.ErrInvalidOrderPrice)

	orderMsg = types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "4.55",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err = suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
	suite.Require().ErrorIs(err, types.ErrInvalidOrderPrice)
	suite.Require().Len(suite.k.GetAllOrder(suite.ctx), 4)
}

func (suite *IntegrationTestSuite) randomOrderCreateMessages(count int, creators []string, market types.Market) ([]types.MsgCreateOrder, sdk.Coins) {
	var msgs []types.MsgCreateOrder
	orderTypes := []string{types.OrderTypeBuy, types.OrderTypeSell}

	allPaid := sdk.NewCoins()
	const ExecPrice = 15
	for i := 0; i < count; i++ {
		randomOrderType := orderTypes[i%2]
		randomPrice := suite.randomNumber(24) + 1 //make sure it's always higher than 0
		if randomOrderType == types.OrderTypeBuy && randomPrice > ExecPrice {
			randomPrice = ExecPrice
		} else if randomOrderType == types.OrderTypeSell && randomPrice < ExecPrice {
			randomPrice = ExecPrice
		}
		randomPriceStr := strconv.Itoa(randomPrice)
		minAmount, err := keeper.CalculateMinAmount(randomPriceStr)
		suite.Require().NoError(err)
		orderAmount := minAmount.AddRaw(int64(suite.randomNumber(1000)))
		orderMsg := types.MsgCreateOrder{
			Amount:    orderAmount.String(),
			Price:     randomPriceStr,
			MarketId:  types.CreateMarketId(market.Base, market.Quote),
			OrderType: randomOrderType,
			Creator:   creators[suite.randomNumber(len(creators))],
		}

		//fee should be captured
		params := suite.k.GetParams(suite.ctx)
		takerFee := sdk.NewCoins(params.MarketTakerFee)
		makerFee := sdk.NewCoins(params.MarketMakerFee)
		var paidCoins sdk.Coins
		if orderMsg.OrderType == types.OrderTypeBuy {
			paidCoins = sdk.NewCoins(sdk.NewCoin(market.Quote, orderAmount.MulRaw(int64(randomPrice))))
		} else {
			paidCoins = sdk.NewCoins(sdk.NewCoin(market.Base, orderAmount))
		}
		allPaid = allPaid.Add(paidCoins...)

		creatorAcc, err := sdk.AccAddressFromBech32(orderMsg.Creator)
		suite.Require().NoError(err)
		suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), creatorAcc, types.ModuleName, gomock.AnyOf(takerFee, makerFee)).Return(nil).Times(1)
		suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.AnyOf(takerFee, makerFee)).Return(nil).Times(1)
		suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), creatorAcc, types.ModuleName, paidCoins).Return(nil).Times(1)

		_, err = suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
		suite.Require().NoError(err)
	}

	return msgs, allPaid
}

func (suite *IntegrationTestSuite) randomNumber(to int) int {
	// Seed the random number generator to get different results each run
	// Uses the current time as the seed
	rand.Seed(time.Now().UnixNano())

	// Generate a random number between 0 and to
	num := rand.Intn(to)

	return num
}

func (suite *IntegrationTestSuite) msgOrderFillSetup(orderType string) (allPrices []string, addr1, addr2 sdk.AccAddress) {
	suite.k.SetMarket(suite.ctx, market)

	addr1 = sdk.AccAddress("addr1_______________")
	addr2 = sdk.AccAddress("addr2_______________")
	params := suite.k.GetParams(suite.ctx)
	makerFee := sdk.NewCoins(params.MarketMakerFee)

	initialPrice := math.ZeroInt()
	staticAmount := "1000000"
	for i := 1; i <= 10; i++ {
		initialPrice = initialPrice.AddRaw(int64(i))
		orderMsg := types.MsgCreateOrder{
			Amount:    staticAmount,
			Price:     initialPrice.String(),
			MarketId:  getMarketId(),
			OrderType: orderType,
			Creator:   addr1.String(),
		}

		var paidCoins sdk.Coins
		if orderType == types.OrderTypeBuy {
			paidCoins = sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000).Mul(initialPrice)))
		} else {
			paidCoins = sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))
		}

		// Maker fee is captured to tradebin module first, then sent to fee collector
		suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, makerFee).Return(nil).Times(1)
		suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), makerFee).Return(nil).Times(1)
		suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, paidCoins).Return(nil).Times(1)
		_, err := suite.msgServer.CreateOrder(suite.ctx, &orderMsg)
		suite.Require().NoError(err)

		allPrices = append(allPrices, initialPrice.String())
	}

	return
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_OneOrderPartialFill_Sell() {
	_, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "500000",
				Price:  "1",
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(500000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)

	receivedCoins := sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(500000)))
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, receivedCoins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, paidCoins).Return(nil).Times(1)
	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 10)
	remainingOrderChecked := false
	for _, order := range all {
		//let's find our order and check if it contains the correct amount
		if order.Price != "1" {
			continue
		}

		suite.Require().EqualValues(order.Amount, fmt.Sprintf("%d", 1000000-500000))
		remainingOrderChecked = true
		break
	}

	suite.Require().True(remainingOrderChecked)
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_OneOrderPartialFill_Buy() {
	_, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)
	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "500000",
				Price:  "1",
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(500000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	receivedCoins := sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(500000)))
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, receivedCoins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, paidCoins).Return(nil).Times(1)

	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 10)
	remainingOrderChecked := false
	for _, order := range all {
		//let's find our order and check if it contains the correct amount
		if order.Price != "1" {
			continue
		}

		suite.Require().EqualValues(order.Amount, fmt.Sprintf("%d", 1000000-500000))
		remainingOrderChecked = true
		break
	}

	suite.Require().True(remainingOrderChecked)
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_OneOrderFullFill_Sell() {
	_, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  "1",
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	receivedCoins := sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, receivedCoins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, paidCoins).Return(nil).Times(1)
	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	for _, order := range all {
		//let's find our order and check if it contains the correct amount
		suite.Require().NotEqual(order.Price, "1")
	}
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_OneOrderFullFill_Buy() {
	_, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  "1",
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	receivedCoins := sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000)))
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, receivedCoins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, paidCoins).Return(nil).Times(1)

	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	for _, order := range all {
		//let's find our order and check if it contains the correct amount
		suite.Require().NotEqual(order.Price, "1000000")
	}
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_TwoOrdersOnePartialFill_Sell() {
	allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  allPrices[0],
			},
			{
				Amount: "500000",
				Price:  allPrices[1],
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(2500000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	//they receive in multiple bank calls: one for each message
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1500000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(500000)))).Return(nil).Times(1)
	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	remainingOrderChecked := false
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		//let's find our order and check if it contains the correct amount
		if order.Price != allPrices[1] {
			continue
		}

		suite.Require().EqualValues(order.Amount, fmt.Sprintf("%d", 1000000-500000))
		remainingOrderChecked = true
		break
	}

	suite.Require().True(remainingOrderChecked)
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_TwoOrdersOnePartialFill_Buy() {
	allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  allPrices[0],
			},
			{
				Amount: "500000",
				Price:  allPrices[1],
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1500000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	//they receive in multiple bank calls: one for each message
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(500000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1500000)))).Return(nil).Times(1)
	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	remainingOrderChecked := false
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		//let's find our order and check if it contains the correct amount
		if order.Price != allPrices[1] {
			continue
		}

		suite.Require().EqualValues(order.Amount, fmt.Sprintf("%d", 1000000-500000))
		remainingOrderChecked = true
		break
	}

	suite.Require().True(remainingOrderChecked)
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_TwoFullyFilledOrders_Buy() {
	allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  allPrices[0],
			},
			{
				Amount: "1000000",
				Price:  allPrices[1],
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(2000000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	//they receive in multiple bank calls: one for each message
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(3000000)))).Return(nil).Times(1)
	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 8)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		suite.Require().NotEqual(order.Price, allPrices[1])
	}
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_TwoFullyFilledOrders_Sell() {
	allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  allPrices[0],
			},
			{
				Amount: "1000000",
				Price:  allPrices[1],
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(4000000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	//they receive in multiple bank calls: one for each message
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(3000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)
	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 8)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		suite.Require().NotEqual(order.Price, allPrices[1])
	}
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_OneFullyFilledOrderWithExtraAmount_Buy() {
	allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1500000",
				Price:  allPrices[0],
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1500000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	receivedCoins := sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000)))
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, receivedCoins).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(500000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)

	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
	}
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_OneFullyFilledOrderWithExtraAmount_Sell() {
	allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1500000",
				Price:  allPrices[0],
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1500000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(500000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000)))).Return(nil).Times(1)

	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
	}
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_TwoFullyFilledOrdersWithExtraAmounts_Buy() {
	allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "2000000",
				Price:  allPrices[0],
			},
			{
				Amount: "1500000",
				Price:  allPrices[1],
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(3500000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(3000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(500000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)

	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 8)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		suite.Require().NotEqual(order.Price, allPrices[1])
		suite.Require().Equal(order.Amount, "1000000")
	}
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_TwoFullyFilledOrdersWithExtraAmounts_Sell() {
	allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "2000000",
				Price:  allPrices[0],
			},
			{
				Amount: "1500000",
				Price:  allPrices[1],
			},
		},
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(6500000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	//they receive in multiple bank calls: one for each message
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Base, math.NewInt(1000000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(1500000)))).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin(market.Quote, math.NewInt(3000000)))).Return(nil).Times(1)
	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 8)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		suite.Require().NotEqual(order.Price, allPrices[1])
		suite.Require().Equal(order.Amount, "1000000")
	}
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_FillAllWithExtraAmounts_Sell() {
	allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)
	suite.Require().NotEmpty(allPrices)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders:    []*types.FillOrderItem{},
	}

	quoteAmount := math.ZeroInt()
	baseAmount := math.ZeroInt()
	expectedQuoteAmount := math.ZeroInt()
	expectedBaseAmount := math.ZeroInt()
	for _, pr := range allPrices {
		randAmount := 1000000 + suite.randomNumber(999999)
		fillOrder.Orders = append(fillOrder.Orders, &types.FillOrderItem{
			Price:  pr,
			Amount: fmt.Sprintf("%d", randAmount),
		})

		price, ok := math.NewIntFromString(pr)
		suite.Require().True(ok)
		baseAmount = baseAmount.AddRaw(int64(randAmount))
		quoteAmount = quoteAmount.Add(price.MulRaw(int64(randAmount)))

		expectedQuoteAmount = expectedQuoteAmount.Add(price.MulRaw(1000000))
		expectedBaseAmount = expectedBaseAmount.AddRaw(1000000)
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Quote, quoteAmount))
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	//they receive in multiple bank calls: one for each FillOrderItem
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, gomock.Any()).Return(nil).Times(len(allPrices))
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, gomock.Any()).
		DoAndReturn(func(ctx sdk.Context, moduleName string, recipient sdk.AccAddress, coins sdk.Coins) error {
			for _, c := range coins {
				if c.Denom == market.Base {
					expectedBaseAmount = expectedBaseAmount.Sub(c.Amount)
				} else if c.Denom == market.Quote {
					quoteAmount = quoteAmount.Sub(c.Amount)
				}
			}

			return nil
		}).Times(len(allPrices) * 2)

	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	suite.Require().True(expectedBaseAmount.IsZero())
	suite.Require().True(quoteAmount.Equal(expectedQuoteAmount))

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Empty(all)
}

func (suite *IntegrationTestSuite) TestMsg_OrderFill_FillAllWithExtraAmounts_Buy() {
	allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)
	suite.Require().NotEmpty(allPrices)

	params := suite.k.GetParams(suite.ctx)
	takerFee := sdk.NewCoins(params.MarketTakerFee)

	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders:    []*types.FillOrderItem{},
	}

	quoteAmount := math.ZeroInt()
	baseAmount := math.ZeroInt()
	expectedQuoteAmount := math.ZeroInt()
	expectedBaseAmount := math.ZeroInt()
	for _, pr := range allPrices {
		randAmount := 1000000 + suite.randomNumber(999999)
		fillOrder.Orders = append(fillOrder.Orders, &types.FillOrderItem{
			Price:  pr,
			Amount: fmt.Sprintf("%d", randAmount),
		})

		price, ok := math.NewIntFromString(pr)
		suite.Require().True(ok)
		baseAmount = baseAmount.AddRaw(int64(randAmount))
		quoteAmount = quoteAmount.Add(price.MulRaw(int64(randAmount)))

		expectedQuoteAmount = expectedQuoteAmount.Add(price.MulRaw(1000000))
		expectedBaseAmount = expectedBaseAmount.AddRaw(1000000)
	}

	paidCoins := sdk.NewCoins(sdk.NewCoin(market.Base, baseAmount))
	suite.expectFillEscrow(addr2, &fillOrder, paidCoins)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	//they receive in multiple bank calls: one for each FillOrderItem
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, gomock.Any()).Return(nil).Times(len(allPrices))
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, gomock.Any()).
		DoAndReturn(func(ctx sdk.Context, moduleName string, recipient sdk.AccAddress, coins sdk.Coins) error {
			for _, c := range coins {
				if c.Denom == market.Base {
					baseAmount = baseAmount.Sub(c.Amount)
				} else if c.Denom == market.Quote {
					expectedQuoteAmount = expectedQuoteAmount.Sub(c.Amount)
				}
			}

			return nil
		}).Times(len(allPrices) * 2)

	_, err := suite.msgServer.FillOrders(suite.ctx, &fillOrder)
	suite.Require().NoError(err)

	suite.Require().True(expectedQuoteAmount.IsZero())
	suite.Require().True(baseAmount.Equal(expectedBaseAmount))

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Empty(all)
}

// expectFillEscrow expects one escrow send per FillOrders entry, with that entry's coins, and checks
// that together they add up to total
func (suite *IntegrationTestSuite) expectFillEscrow(taker sdk.AccAddress, msg *types.MsgFillOrders, total sdk.Coins) {
	sum := sdk.NewCoins()
	for _, fo := range msg.Orders {
		amt, ok := math.NewIntFromString(fo.Amount)
		suite.Require().True(ok)
		coin, dust, err := suite.k.GetOrderSdkCoin(types.TheOtherOrderType(msg.OrderType), fo.Price, amt, &market)
		suite.Require().NoError(err)
		suite.Require().True(dust.IsZero(), "the fill scenarios use whole-unit escrows")
		suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), taker, types.ModuleName, sdk.NewCoins(coin)).Return(nil).Times(1)
		sum = sum.Add(coin)
	}
	suite.Require().Equal(total, sum)
}
