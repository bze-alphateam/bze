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

	minAmount = keeper.CalculateMinAmount("2.5532")
	suite.Require().Equal(minAmount.Int64(), int64(2))

	minAmount = keeper.CalculateMinAmount("1.28382713")
	suite.Require().Equal(minAmount.Int64(), int64(2))
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

type GetOrderCoinsWithDustTestCase struct {
	Price     string
	BuyDust   float64
	BuyAmount string
}

func (suite *IntegrationTestSuite) TestGetOrderCoinsWithDust_UserDoesNotReceive() {
	tc := []GetOrderCoinsWithDustTestCase{
		{
			Price:     "0.91",
			BuyDust:   0.36,
			BuyAmount: "4",
		},
		{
			Price:     "0.00004",
			BuyDust:   0,
			BuyAmount: "2",
		},
		{
			Price:     "0.06617",
			BuyDust:   0.88256,
			BuyAmount: "3",
		},
		{
			Price:     "0.13891",
			BuyDust:   0.77744,
			BuyAmount: "3",
		},
		{
			Price:     "2.118322",
			BuyDust:   0.763356,
			BuyAmount: "5",
		},
	}

	for _, tc := range tc {
		minAmount := keeper.CalculateMinAmount(tc.Price)
		coinReq := types.OrderCoinsArguments{
			OrderType:    types.OrderTypeBuy,
			OrderPrice:   tc.Price,
			OrderAmount:  minAmount,
			Market:       &market,
			UserAddress:  "addr1",
			UserReceives: false,
		}
		buyCoins, err := suite.k.GetOrderCoinsWithDust(suite.ctx, coinReq)
		suite.Require().Nil(err)
		suite.Require().Equal(buyCoins.Coin.Amount.String(), tc.BuyAmount) //result of amount * price
		suite.Require().Equal(buyCoins.Coin.Denom, market.Quote)
		suite.Require().EqualValues(buyCoins.Dust.MustFloat64(), tc.BuyDust)
		if tc.BuyDust > 0 {
			dustFromStorage, err := sdk.NewDecFromStr(buyCoins.UserDust.Amount)
			suite.Require().Nil(err)
			suite.Require().EqualValues(buyCoins.Dust, dustFromStorage)
			suite.Require().EqualValues(buyCoins.UserDust.Owner, "addr1")
			suite.Require().EqualValues(buyCoins.UserDust.Denom, market.Quote)
		}

		coinReq = types.OrderCoinsArguments{
			OrderType:    types.OrderTypeSell,
			OrderPrice:   tc.Price,
			OrderAmount:  minAmount,
			Market:       &market,
			UserAddress:  "addr2",
			UserReceives: false,
		}

		buyCoins, err = suite.k.GetOrderCoinsWithDust(suite.ctx, coinReq)
		suite.Require().Nil(err)
		suite.Require().Equal(buyCoins.Coin.Amount, minAmount)
		suite.Require().Equal(buyCoins.Coin.Denom, market.Base)
		suite.Require().EqualValues(buyCoins.Dust.MustFloat64(), 0)
		suite.Require().Nil(err)
		suite.Require().Nil(buyCoins.UserDust)
		suite.Require().EqualValues(buyCoins.Dust, sdk.ZeroDec())
	}
}

func (suite *IntegrationTestSuite) TestGetOrderCoinsWithDust_UserReceive() {
	tc := []GetOrderCoinsWithDustTestCase{
		{
			Price:     "0.91",
			BuyDust:   0.64,
			BuyAmount: "3",
		},
		{
			Price:     "0.00004",
			BuyDust:   0,
			BuyAmount: "2",
		},
		{
			Price:     "0.06617",
			BuyDust:   0.11744,
			BuyAmount: "2",
		},
		{
			Price:     "0.13891",
			BuyDust:   0.22256,
			BuyAmount: "2",
		},
		{
			Price:     "2.118322",
			BuyDust:   0.236644,
			BuyAmount: "4",
		},
	}

	for _, tc := range tc {
		minAmount := keeper.CalculateMinAmount(tc.Price)
		coinReq := types.OrderCoinsArguments{
			OrderType:    types.OrderTypeBuy,
			OrderPrice:   tc.Price,
			OrderAmount:  minAmount,
			Market:       &market,
			UserAddress:  "addr1",
			UserReceives: true,
		}
		buyCoins, err := suite.k.GetOrderCoinsWithDust(suite.ctx, coinReq)
		suite.Require().Nil(err)
		suite.Require().Equal(buyCoins.Coin.Amount.String(), tc.BuyAmount)
		suite.Require().Equal(buyCoins.Coin.Denom, market.Quote)
		suite.Require().EqualValues(buyCoins.Dust.MustFloat64(), tc.BuyDust)
		if tc.BuyDust > 0 {
			dustFromStorage, err := sdk.NewDecFromStr(buyCoins.UserDust.Amount)
			suite.Require().Nil(err)
			suite.Require().EqualValues(buyCoins.Dust, dustFromStorage)
			suite.Require().EqualValues(buyCoins.UserDust.Owner, "addr1")
			suite.Require().EqualValues(buyCoins.UserDust.Denom, market.Quote)
		}

		coinReq = types.OrderCoinsArguments{
			OrderType:    types.OrderTypeSell,
			OrderPrice:   tc.Price,
			OrderAmount:  minAmount,
			Market:       &market,
			UserAddress:  "addr2",
			UserReceives: false,
		}

		buyCoins, err = suite.k.GetOrderCoinsWithDust(suite.ctx, coinReq)
		suite.Require().Nil(err)
		suite.Require().Equal(buyCoins.Coin.Amount.String(), minAmount.String())
		suite.Require().Equal(buyCoins.Coin.Denom, market.Base)
		suite.Require().EqualValues(buyCoins.Dust.MustFloat64(), 0.0)
		suite.Require().Nil(err)
		suite.Require().Nil(buyCoins.UserDust)
		suite.Require().EqualValues(buyCoins.Dust, sdk.ZeroDec())
	}
}
