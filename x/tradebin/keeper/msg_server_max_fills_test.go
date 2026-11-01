package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"go.uber.org/mock/gomock"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// allowBank lets every bank call of the order book message servers succeed
func (suite *IntegrationTestSuite) allowBank() {
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
}

func (suite *IntegrationTestSuite) setMaxFillsPerOrder(maxFills uint64) {
	params := suite.k.GetParams(suite.ctx)
	params.MaxFillsPerOrder = maxFills
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
}

// MsgCreateOrder examines at most the maxFillsPerOrder param of resting orders (filled or skipped).
func (suite *IntegrationTestSuite) TestMsgCreateOrder_MaxFillsPerOrderParam() {
	suite.k.SetMarket(suite.ctx, market)
	suite.setMaxFillsPerOrder(2)
	makers := suite.restSells(4, "10", "1")
	taker := sdk.AccAddress("addr2_______________")
	suite.allowBank()

	suite.resetEvents()
	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Creator:   taker.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    "100",
		Price:     "1",
	})
	suite.Require().NoError(err)

	events := suite.tradebinTypedEvents()
	suite.requireFilledInOrder(events, makers[:2], "10")
	refunds := refundedEvents(events)
	suite.Require().Len(refunds, 1)
	suite.Require().Equal(types.RefundReasonMaxFillsReached, refunds[0].Reason)
	suite.Require().Equal("80", refunds[0].Amount)
	suite.requireResting(makers[2])
	suite.requireResting(makers[3])
}

// Each MsgFillOrders entry is its own order with its own cap.
func (suite *IntegrationTestSuite) TestMsgFillOrders_MaxFillsPerOrderPerEntry() {
	suite.k.SetMarket(suite.ctx, market)
	suite.setMaxFillsPerOrder(2)
	makers := suite.restSells(5, "10", "1")
	taker := sdk.AccAddress("addr2_______________")
	suite.allowBank()

	suite.resetEvents()
	_, err := suite.msgServer.FillOrders(suite.ctx, &types.MsgFillOrders{
		Creator:   taker.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{Amount: "30", Price: "1"},
			{Amount: "30", Price: "1"},
		},
	})
	suite.Require().NoError(err)

	events := suite.tradebinTypedEvents()
	suite.requireFilledInOrder(events, makers[:4], "10")
	refunds := refundedEvents(events)
	suite.Require().Len(refunds, 2)
	for _, r := range refunds {
		suite.Require().Equal(types.RefundReasonMaxFillsReached, r.Reason)
		suite.Require().Equal("10", r.Amount)
	}
	suite.Require().NotEqual(refunds[0].OrderId, refunds[1].OrderId)
	suite.requireResting(makers[4])
}
