package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// Covers orders stored before input validation.

const (
	storedAmount         = "0100"
	expectedStoredAmount = int64(64)
	storedOrderPrice     = "1"
)

type bankTransfer struct {
	addr  sdk.AccAddress
	coins sdk.Coins
}

// recordBankTransfers answers every bank call the order book makes and records the transfers between the
// module and the accounts.
func (suite *IntegrationTestSuite) recordBankTransfers() (toModule, toAccount *[]bankTransfer) {
	in := []bankTransfer{}
	out := []bankTransfer{}
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), types.ModuleName, gomock.Any()).
		DoAndReturn(func(_ sdk.Context, addr sdk.AccAddress, _ string, coins sdk.Coins) error {
			in = append(in, bankTransfer{addr: addr, coins: coins})
			return nil
		}).AnyTimes()
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ sdk.Context, _ string, addr sdk.AccAddress, coins sdk.Coins) error {
			out = append(out, bankTransfer{addr: addr, coins: coins})
			return nil
		}).AnyTimes()
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	return &in, &out
}

func transfersTo(transfers []bankTransfer, addr sdk.AccAddress) sdk.Coins {
	total := sdk.NewCoins()
	for _, t := range transfers {
		if t.addr.Equals(addr) {
			total = total.Add(t.coins...)
		}
	}

	return total
}

// restStoredOrder rests an order with storedAmount on the book and returns it.
func (suite *IntegrationTestSuite) restStoredOrder(orderType string, owner sdk.AccAddress) types.Order {
	suite.k.SetMarket(suite.ctx, market)
	suite.placeOrders(suite.newEngine(), limitOrder(orderType, storedAmount, storedOrderPrice, owner))
	order, found := suite.k.GetOrder(suite.ctx, getMarketId(), orderType, orderId(0))
	suite.Require().True(found)
	suite.Require().Equal(storedAmount, order.Amount, "stored order amount")
	suite.checkAggregatedOrder(getMarketId(), orderType, storedOrderPrice, math.NewInt(expectedStoredAmount).String())

	return order
}

func (suite *IntegrationTestSuite) cancelStoredOrder(orderType, refundDenom string) {
	owner := sdk.AccAddress("addr1_______________")
	order := suite.restStoredOrder(orderType, owner)
	_, toAccount := suite.recordBankTransfers()

	_, err := suite.msgServer.CancelOrder(suite.ctx, &types.MsgCancelOrder{
		Creator:   owner.String(),
		MarketId:  getMarketId(),
		OrderId:   order.Id,
		OrderType: orderType,
	})
	suite.Require().NoError(err)

	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(refundDenom, expectedStoredAmount)), transfersTo(*toAccount, owner),
		"stored order amount")
	_, found := suite.k.GetOrder(suite.ctx, getMarketId(), orderType, order.Id)
	suite.Require().False(found)
	_, found = suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), orderType, storedOrderPrice)
	suite.Require().False(found, "stored order aggregate")
}

func (suite *IntegrationTestSuite) TestCancelOrder_StoredAmountNonCanonical_Sell() {
	suite.cancelStoredOrder(types.OrderTypeSell, market.Base)
}

func (suite *IntegrationTestSuite) TestCancelOrder_StoredAmountNonCanonical_Buy() {
	suite.cancelStoredOrder(types.OrderTypeBuy, market.Quote)
}

func (suite *IntegrationTestSuite) fillStoredOrder(fillAmount int64) (maker, taker sdk.AccAddress, toModule, toAccount []bankTransfer) {
	maker = sdk.AccAddress("addr1_______________")
	taker = sdk.AccAddress("addr2_______________")
	suite.restStoredOrder(types.OrderTypeSell, maker)
	in, out := suite.recordBankTransfers()

	_, err := suite.msgServer.FillOrders(suite.ctx, &types.MsgFillOrders{
		Creator:   taker.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders:    []*types.FillOrderItem{{Amount: math.NewInt(fillAmount).String(), Price: storedOrderPrice}},
	})
	suite.Require().NoError(err)

	return maker, taker, *in, *out
}

func (suite *IntegrationTestSuite) TestFillOrders_StoredAmountNonCanonical_FullFill() {
	maker, taker, toModule, toAccount := suite.fillStoredOrder(expectedStoredAmount)

	takerFee := suite.k.GetParams(suite.ctx).MarketTakerFee
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Quote, expectedStoredAmount)).Add(takerFee), transfersTo(toModule, taker),
		"the taker escrows the fill and the fee")
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Base, expectedStoredAmount)), transfersTo(toAccount, taker),
		"stored order amount")
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Quote, expectedStoredAmount)), transfersTo(toAccount, maker),
		"stored order amount")

	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx), "the stored order is fully filled")
	_, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, storedOrderPrice)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestFillOrders_StoredAmountNonCanonical_PartialFill() {
	maker, taker, _, toAccount := suite.fillStoredOrder(32)

	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Base, 32)), transfersTo(toAccount, taker))
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 32)), transfersTo(toAccount, maker))

	remaining, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, orderId(0))
	suite.Require().True(found)
	suite.Require().Equal("32", remaining.Amount, "stored order amount")
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, storedOrderPrice, "32")
}

func (suite *IntegrationTestSuite) TestCreateOrder_MatchesStoredAmountNonCanonical() {
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	suite.restStoredOrder(types.OrderTypeSell, maker)
	_, toAccount := suite.recordBankTransfers()

	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Creator:   taker.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    math.NewInt(expectedStoredAmount).String(),
		Price:     storedOrderPrice,
	})
	suite.Require().NoError(err)

	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Base, expectedStoredAmount)), transfersTo(*toAccount, taker),
		"stored order amount")
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Quote, expectedStoredAmount)), transfersTo(*toAccount, maker),
		"stored order amount")
	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx), "the stored order is fully filled")
}

func (suite *IntegrationTestSuite) TestPlaceOrder_MergesIntoStoredAmountNonCanonical() {
	owner := sdk.AccAddress("addr1_______________")
	suite.restStoredOrder(types.OrderTypeSell, owner)

	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeSell, "10", storedOrderPrice, owner))

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().Equal("74", all[0].Amount, "stored order amount")
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, storedOrderPrice, "74")
}

func (suite *IntegrationTestSuite) TestPlaceOrder_MergesIntoStoredAmountNonCanonical_Event() {
	owner := sdk.AccAddress("addr1_______________")
	stored := suite.restStoredOrder(types.OrderTypeSell, owner)

	suite.resetEvents()
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeSell, "10", storedOrderPrice, owner))

	suite.Require().Equal([]*types.OrderAmendedEvent{{
		Id:          orderId(1),
		MarketId:    getMarketId(),
		OrderType:   types.OrderTypeSell,
		Price:       storedOrderPrice,
		Owner:       owner.String(),
		AddedAmount: "10",
		Amount:      "74",
		PreviousId:  stored.Id,
	}}, amendedEvents(suite.tradebinTypedEvents()), "stored order amount")
}

func (suite *IntegrationTestSuite) TestCreateOrder_PartiallyFillsStoredAmountNonCanonical() {
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	suite.restStoredOrder(types.OrderTypeSell, maker)
	toModule, toAccount := suite.recordBankTransfers()

	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Creator:   taker.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    "40",
		Price:     storedOrderPrice,
	})
	suite.Require().NoError(err)

	takerFee := suite.k.GetParams(suite.ctx).MarketTakerFee
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 40)).Add(takerFee), transfersTo(*toModule, taker),
		"the taker escrows the order and the fee")
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Base, 40)), transfersTo(*toAccount, taker))
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 40)), transfersTo(*toAccount, maker))

	remaining, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, orderId(0))
	suite.Require().True(found)
	suite.Require().Equal("24", remaining.Amount, "stored order amount")
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, storedOrderPrice, "24")
	_, found = suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, storedOrderPrice)
	suite.Require().False(found, "the taker order is filled in full and does not rest")
}
