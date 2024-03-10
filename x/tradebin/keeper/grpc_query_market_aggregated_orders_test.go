package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestMarketAggregatedOrders_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.MarketAggregatedOrders(goCtx, nil)
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestMarketAggregatedOrders_InvalidMarket() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.MarketAggregatedOrders(goCtx, &types.QueryMarketAggregatedOrdersRequest{Market: ""})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestMarketAggregatedOrders_InvalidOrderType() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	suite.k.SetMarket(suite.ctx, market)

	_, err := suite.k.MarketAggregatedOrders(goCtx, &types.QueryMarketAggregatedOrdersRequest{Market: getMarketId(), OrderType: "yeahsureinvalidordertype"})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestMarketAggregatedOrders_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	suite.k.SetMarket(suite.ctx, market)
	aggBuy := types.AggregatedOrder{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    100,
		Price:     "2",
	}

	aggSell := types.AggregatedOrder{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Amount:    int64(9223372036854775806),
		Price:     "232183213232131221313",
	}

	suite.k.SetAggregatedOrder(suite.ctx, aggBuy)
	suite.k.SetAggregatedOrder(suite.ctx, aggSell)

	res, err := suite.k.MarketAggregatedOrders(goCtx, &types.QueryMarketAggregatedOrdersRequest{Market: getMarketId(), OrderType: types.OrderTypeBuy})
	suite.Require().Nil(err)
	suite.Require().Len(res.List, 1)
	suite.Require().Equal(res.List[0], aggBuy)

	res, err = suite.k.MarketAggregatedOrders(goCtx, &types.QueryMarketAggregatedOrdersRequest{Market: getMarketId(), OrderType: types.OrderTypeSell})
	suite.Require().Nil(err)
	suite.Require().Len(res.List, 1)
	suite.Require().Equal(res.List[0], aggSell)
}
