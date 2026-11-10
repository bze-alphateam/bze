package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// These tests pin how an order stored before canonical amount validation is read today, for the
// refund, the fills, the aggregate and a merge. The state is built through the engine, the same way
// the message servers stored it.

const (
	storedAmountNonCanonical = "0100"
	storedAmountReadAs       = int64(64)
	storedAmountPrice        = "1"
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

// restStoredNonCanonicalOrder rests an order with the amount "0100" on the book and returns it.
func (suite *IntegrationTestSuite) restStoredNonCanonicalOrder(orderType string, owner sdk.AccAddress) types.Order {
	suite.k.SetMarket(suite.ctx, market)
	suite.placeOrders(suite.newEngine(), limitOrder(orderType, storedAmountNonCanonical, storedAmountPrice, owner))
	order, found := suite.k.GetOrder(suite.ctx, getMarketId(), orderType, orderId(0))
	suite.Require().True(found)
	suite.Require().Equal(storedAmountNonCanonical, order.Amount, "the order keeps the amount as stored")
	suite.checkAggregatedOrder(getMarketId(), orderType, storedAmountPrice, math.NewInt(storedAmountReadAs).String())

	return order
}

func (suite *IntegrationTestSuite) cancelStoredNonCanonicalOrder(orderType, refundDenom string) {
	owner := sdk.AccAddress("addr1_______________")
	order := suite.restStoredNonCanonicalOrder(orderType, owner)
	_, toAccount := suite.recordBankTransfers()

	_, err := suite.msgServer.CancelOrder(suite.ctx, &types.MsgCancelOrder{
		Creator:   owner.String(),
		MarketId:  getMarketId(),
		OrderId:   order.Id,
		OrderType: orderType,
	})
	suite.Require().NoError(err)

	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(refundDenom, storedAmountReadAs)), transfersTo(*toAccount, owner),
		"stored amount is read as stored before canonical validation")
	_, found := suite.k.GetOrder(suite.ctx, getMarketId(), orderType, order.Id)
	suite.Require().False(found)
	_, found = suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), orderType, storedAmountPrice)
	suite.Require().False(found, "the aggregate is emptied by the stored amount")
}

func (suite *IntegrationTestSuite) TestCancelOrder_StoredAmountNonCanonical_Sell() {
	suite.cancelStoredNonCanonicalOrder(types.OrderTypeSell, market.Base)
}

func (suite *IntegrationTestSuite) TestCancelOrder_StoredAmountNonCanonical_Buy() {
	suite.cancelStoredNonCanonicalOrder(types.OrderTypeBuy, market.Quote)
}

func (suite *IntegrationTestSuite) fillStoredNonCanonicalOrder(fillAmount int64) (maker, taker sdk.AccAddress, toModule, toAccount []bankTransfer) {
	maker = sdk.AccAddress("addr1_______________")
	taker = sdk.AccAddress("addr2_______________")
	suite.restStoredNonCanonicalOrder(types.OrderTypeSell, maker)
	in, out := suite.recordBankTransfers()

	_, err := suite.msgServer.FillOrders(suite.ctx, &types.MsgFillOrders{
		Creator:   taker.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders:    []*types.FillOrderItem{{Amount: math.NewInt(fillAmount).String(), Price: storedAmountPrice}},
	})
	suite.Require().NoError(err)

	return maker, taker, *in, *out
}

func (suite *IntegrationTestSuite) TestFillOrders_StoredAmountNonCanonical_FullFill() {
	maker, taker, toModule, toAccount := suite.fillStoredNonCanonicalOrder(storedAmountReadAs)

	takerFee := suite.k.GetParams(suite.ctx).MarketTakerFee
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Quote, storedAmountReadAs)).Add(takerFee), transfersTo(toModule, taker),
		"the taker escrows the fill and the fee")
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Base, storedAmountReadAs)), transfersTo(toAccount, taker),
		"stored amount is read as stored before canonical validation")
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Quote, storedAmountReadAs)), transfersTo(toAccount, maker),
		"stored amount is read as stored before canonical validation")

	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx), "the stored order is fully filled")
	_, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, storedAmountPrice)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestFillOrders_StoredAmountNonCanonical_PartialFill() {
	maker, taker, _, toAccount := suite.fillStoredNonCanonicalOrder(32)

	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Base, 32)), transfersTo(toAccount, taker))
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 32)), transfersTo(toAccount, maker))

	remaining, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, orderId(0))
	suite.Require().True(found)
	suite.Require().Equal("32", remaining.Amount, "stored amount is read as stored before canonical validation")
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, storedAmountPrice, "32")
}

func (suite *IntegrationTestSuite) TestCreateOrder_MatchesStoredAmountNonCanonical() {
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	suite.restStoredNonCanonicalOrder(types.OrderTypeSell, maker)
	_, toAccount := suite.recordBankTransfers()

	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Creator:   taker.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    math.NewInt(storedAmountReadAs).String(),
		Price:     storedAmountPrice,
	})
	suite.Require().NoError(err)

	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Base, storedAmountReadAs)), transfersTo(*toAccount, taker),
		"stored amount is read as stored before canonical validation")
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(market.Quote, storedAmountReadAs)), transfersTo(*toAccount, maker),
		"stored amount is read as stored before canonical validation")
	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx), "the stored order is fully filled")
}

func (suite *IntegrationTestSuite) TestPlaceOrder_MergesIntoStoredAmountNonCanonical() {
	owner := sdk.AccAddress("addr1_______________")
	suite.restStoredNonCanonicalOrder(types.OrderTypeSell, owner)

	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeSell, "10", storedAmountPrice, owner))

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().Equal("74", all[0].Amount, "stored amount is read as stored before canonical validation")
	suite.checkAggregatedOrder(getMarketId(), types.OrderTypeSell, storedAmountPrice, "74")
}
