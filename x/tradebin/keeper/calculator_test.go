package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestCalculateMinAmount() {
	minAmount := keeper.CalculateMinAmount("0.9")
	suite.Require().Equal(minAmount.Int64(), int64(4))

	minAmount = keeper.CalculateMinAmount("0.09")
	suite.Require().Equal(minAmount.Int64(), int64(24))

	minAmount = keeper.CalculateMinAmount("0.009")
	suite.Require().Equal(minAmount.Int64(), int64(224))

	minAmount = keeper.CalculateMinAmount("0.0009")
	suite.Require().Equal(minAmount.Int64(), int64(2224))

	minAmount = keeper.CalculateMinAmount("0.0143331")
	suite.Require().Equal(minAmount.Int64(), int64(140))

	minAmount = keeper.CalculateMinAmount("0.09000123")
	suite.Require().Equal(minAmount.Int64(), int64(24))

	minAmount = keeper.CalculateMinAmount("0.0497999")
	suite.Require().Equal(minAmount.Int64(), int64(42))
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

func (suite *IntegrationTestSuite) TestGetOrderSdkCoin() {
	price := "0.91"
	minAmount := keeper.CalculateMinAmount(price)
	buyCoins, _, err := suite.k.GetOrderSdkCoin(types.OrderTypeBuy, price, minAmount, &market)
	suite.Require().Nil(err)
	suite.Require().Equal(buyCoins.Amount.Int64(), int64(3)) //result of amount * price
	suite.Require().Equal(buyCoins.Denom, market.Quote)

	sellCoins, _, err := suite.k.GetOrderSdkCoin(types.OrderTypeSell, price, minAmount, &market)
	suite.Require().Nil(err)
	suite.Require().Equal(sellCoins.Amount, minAmount)
	suite.Require().Equal(sellCoins.Denom, market.Base)
}

func (suite *IntegrationTestSuite) TestGetOrderCoinsWithDust_UserDoesNotReceive() {
	price := "0.91"
	minAmount := keeper.CalculateMinAmount(price)
	coinReq := types.OrderCoinsArguments{
		OrderType:    types.OrderTypeBuy,
		OrderPrice:   price,
		OrderAmount:  minAmount,
		Market:       &market,
		UserAddress:  "addr1",
		UserReceives: false,
	}
	buyCoins, err := suite.k.GetOrderCoinsWithDust(suite.ctx, coinReq)
	suite.Require().Nil(err)
	suite.Require().Equal(buyCoins.Coin.Amount.Int64(), int64(4)) //result of amount * price
	suite.Require().Equal(buyCoins.Coin.Denom, market.Quote)
	suite.Require().EqualValues(buyCoins.Dust.MustFloat64(), 0.36)
	dustFromStorage, err := sdk.NewDecFromStr(buyCoins.UserDust.Amount)
	suite.Require().Nil(err)
	suite.Require().EqualValues(buyCoins.Dust, dustFromStorage)
	suite.Require().EqualValues(buyCoins.UserDust.Owner, "addr1")
	suite.Require().EqualValues(buyCoins.UserDust.Denom, market.Quote)

	coinReq = types.OrderCoinsArguments{
		OrderType:    types.OrderTypeSell,
		OrderPrice:   price,
		OrderAmount:  minAmount,
		Market:       &market,
		UserAddress:  "addr2",
		UserReceives: false,
	}

	buyCoins, err = suite.k.GetOrderCoinsWithDust(suite.ctx, coinReq)
	suite.Require().Nil(err)
	suite.Require().Equal(buyCoins.Coin.Amount.Int64(), int64(4)) //result of amount * price
	suite.Require().Equal(buyCoins.Coin.Denom, market.Base)
	suite.Require().EqualValues(buyCoins.Dust.MustFloat64(), 0)
	suite.Require().Nil(err)
	suite.Require().Nil(buyCoins.UserDust)
	suite.Require().EqualValues(buyCoins.Dust, sdk.ZeroDec())
}

func (suite *IntegrationTestSuite) TestGetOrderCoinsWithDust_UserReceive() {
	price := "0.91"
	minAmount := keeper.CalculateMinAmount(price)
	coinReq := types.OrderCoinsArguments{
		OrderType:    types.OrderTypeBuy,
		OrderPrice:   price,
		OrderAmount:  minAmount,
		Market:       &market,
		UserAddress:  "addr1",
		UserReceives: true,
	}
	buyCoins, err := suite.k.GetOrderCoinsWithDust(suite.ctx, coinReq)
	suite.Require().Nil(err)
	suite.Require().Equal(buyCoins.Coin.Amount.Int64(), int64(4)) //result of amount * price
	suite.Require().Equal(buyCoins.Coin.Denom, market.Quote)
	suite.Require().EqualValues(buyCoins.Dust.MustFloat64(), 0.36)
	dustFromStorage, err := sdk.NewDecFromStr(buyCoins.UserDust.Amount)
	suite.Require().Nil(err)
	suite.Require().EqualValues(buyCoins.Dust, dustFromStorage)
	suite.Require().EqualValues(buyCoins.UserDust.Owner, "addr1")
	suite.Require().EqualValues(buyCoins.UserDust.Denom, market.Quote)

	coinReq = types.OrderCoinsArguments{
		OrderType:    types.OrderTypeSell,
		OrderPrice:   price,
		OrderAmount:  minAmount,
		Market:       &market,
		UserAddress:  "addr2",
		UserReceives: false,
	}

	buyCoins, err = suite.k.GetOrderCoinsWithDust(suite.ctx, coinReq)
	suite.Require().Nil(err)
	suite.Require().Equal(buyCoins.Coin.Amount.Int64(), int64(4)) //result of amount * price
	suite.Require().Equal(buyCoins.Coin.Denom, market.Base)
	suite.Require().EqualValues(buyCoins.Dust.MustFloat64(), 0)
	suite.Require().Nil(err)
	suite.Require().Nil(buyCoins.UserDust)
	suite.Require().EqualValues(buyCoins.Dust, sdk.ZeroDec())
}
