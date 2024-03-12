package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestCancelOrder_MarketNotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CancelOrder(goCtx, &types.MsgCancelOrder{
		Creator:   "me",
		MarketId:  "me/me",
		OrderId:   "",
		OrderType: "",
	})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCancelOrder_OrderNotFound() {
	suite.k.SetMarket(suite.ctx, market)
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CancelOrder(goCtx, &types.MsgCancelOrder{
		Creator:   "me",
		MarketId:  getMarketId(),
		OrderId:   "123",
		OrderType: "dsa",
	})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCancelOrder_Unauthorized() {
	suite.k.SetMarket(suite.ctx, market)
	order := suite.k.NewOrder(suite.ctx, types.Order{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    "102",
		Price:     "1",
		Owner:     "me",
	})

	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CancelOrder(goCtx, &types.MsgCancelOrder{
		Creator:   "not_me",
		MarketId:  getMarketId(),
		OrderId:   order.Id,
		OrderType: types.OrderTypeBuy,
	})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCancelOrder_CancelBuy_Success() {
	suite.k.SetMarket(suite.ctx, market)
	order := suite.k.NewOrder(suite.ctx, types.Order{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    "102000",
		Price:     "1",
		Owner:     "me",
	})

	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CancelOrder(goCtx, &types.MsgCancelOrder{
		Creator:   "me",
		MarketId:  getMarketId(),
		OrderId:   order.Id,
		OrderType: types.OrderTypeBuy,
	})
	suite.Require().Nil(err)

	qms := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qms, 1)

	suite.Require().Equal(qms[0].MarketId, order.MarketId)
	suite.Require().Equal(qms[0].MessageType, types.OrderTypeCancel)
	suite.Require().Equal(qms[0].OrderId, order.Id)
	suite.Require().Equal(qms[0].OrderType, order.OrderType)
	suite.Require().Equal(qms[0].Owner, order.Owner)
}

func (suite *IntegrationTestSuite) TestCancelOrder_CancelSell_Success() {
	suite.k.SetMarket(suite.ctx, market)
	order := suite.k.NewOrder(suite.ctx, types.Order{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Amount:    "10200021",
		Price:     "1000",
		Owner:     "me",
	})

	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CancelOrder(goCtx, &types.MsgCancelOrder{
		Creator:   "me",
		MarketId:  getMarketId(),
		OrderId:   order.Id,
		OrderType: types.OrderTypeSell,
	})
	suite.Require().Nil(err)

	qms := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qms, 1)

	suite.Require().Equal(qms[0].MarketId, order.MarketId)
	suite.Require().Equal(qms[0].MessageType, types.OrderTypeCancel)
	suite.Require().Equal(qms[0].OrderId, order.Id)
	suite.Require().Equal(qms[0].OrderType, order.OrderType)
	suite.Require().Equal(qms[0].Owner, order.Owner)
}
