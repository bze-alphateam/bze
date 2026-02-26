package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ============ CalculateMinAmount Tests ============

func (suite *IntegrationTestSuite) TestCalculateMinAmount_ValidPrice() {
	result, err := keeper.CalculateMinAmount("1")
	suite.Require().NoError(err)
	// 1/1 = 1, ceil = 1, * 2 = 2
	suite.Require().Equal(math.NewInt(2), result)
}

func (suite *IntegrationTestSuite) TestCalculateMinAmount_HighPrice() {
	result, err := keeper.CalculateMinAmount("100")
	suite.Require().NoError(err)
	// 1/100 = 0.01, ceil = 1, * 2 = 2
	suite.Require().Equal(math.NewInt(2), result)
}

func (suite *IntegrationTestSuite) TestCalculateMinAmount_LowPrice() {
	result, err := keeper.CalculateMinAmount("0.001")
	suite.Require().NoError(err)
	// 1/0.001 = 1000, ceil = 1000, * 2 = 2000
	suite.Require().Equal(math.NewInt(2000), result)
}

func (suite *IntegrationTestSuite) TestCalculateMinAmount_FractionalPrice() {
	result, err := keeper.CalculateMinAmount("0.5")
	suite.Require().NoError(err)
	// 1/0.5 = 2, ceil = 2, * 2 = 4
	suite.Require().Equal(math.NewInt(4), result)
}

func (suite *IntegrationTestSuite) TestCalculateMinAmount_NonRoundResult() {
	result, err := keeper.CalculateMinAmount("3")
	suite.Require().NoError(err)
	// 1/3 = 0.333..., ceil = 1, * 2 = 2
	suite.Require().Equal(math.NewInt(2), result)
}

func (suite *IntegrationTestSuite) TestCalculateMinAmount_ZeroPrice() {
	_, err := keeper.CalculateMinAmount("0")
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "price cannot be zero")
}

func (suite *IntegrationTestSuite) TestCalculateMinAmount_InvalidPrice() {
	_, err := keeper.CalculateMinAmount("invalid")
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "error converting price to Dec")
}

func (suite *IntegrationTestSuite) TestCalculateMinAmount_EmptyPrice() {
	_, err := keeper.CalculateMinAmount("")
	suite.Require().Error(err)
}

// ============ CalculateMinAmountFromDecPrice Tests ============

func (suite *IntegrationTestSuite) TestCalculateMinAmountFromDecPrice_ZeroPrice() {
	_, err := keeper.CalculateMinAmountFromDecPrice(math.LegacyZeroDec())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "price cannot be zero")
}

func (suite *IntegrationTestSuite) TestCalculateMinAmountFromDecPrice_PriceOne() {
	result, err := keeper.CalculateMinAmountFromDecPrice(math.LegacyOneDec())
	suite.Require().NoError(err)
	// 1/1 = 1, ceil = 1, * 2 = 2
	suite.Require().Equal(math.NewInt(2), result)
}

func (suite *IntegrationTestSuite) TestCalculateMinAmountFromDecPrice_SmallPrice() {
	price := math.LegacyMustNewDecFromStr("0.01")
	result, err := keeper.CalculateMinAmountFromDecPrice(price)
	suite.Require().NoError(err)
	// 1/0.01 = 100, ceil = 100, * 2 = 200
	suite.Require().Equal(math.NewInt(200), result)
}

func (suite *IntegrationTestSuite) TestCalculateMinAmountFromDecPrice_LargePrice() {
	price := math.LegacyMustNewDecFromStr("1000")
	result, err := keeper.CalculateMinAmountFromDecPrice(price)
	suite.Require().NoError(err)
	// 1/1000 = 0.001, ceil = 1, * 2 = 2
	suite.Require().Equal(math.NewInt(2), result)
}

func (suite *IntegrationTestSuite) TestCalculateMinAmountFromDecPrice_FractionalPrice() {
	price := math.LegacyMustNewDecFromStr("0.3")
	result, err := keeper.CalculateMinAmountFromDecPrice(price)
	suite.Require().NoError(err)
	// 1/0.3 = 3.333..., ceil = 4, * 2 = 8
	suite.Require().Equal(math.NewInt(8), result)
}

// ============ GetOrderSdkCoin Tests ============

func (suite *IntegrationTestSuite) TestGetOrderSdkCoin_BuyOrder() {
	orderAmount := math.NewInt(100)
	coin, dust, err := suite.k.GetOrderSdkCoin(types.OrderTypeBuy, "2", orderAmount, &market)
	suite.Require().NoError(err)
	// Buy order: coin denom = market.Quote, amount = orderAmount * price = 100 * 2 = 200
	suite.Require().Equal(market.Quote, coin.Denom)
	suite.Require().Equal(math.NewInt(200), coin.Amount)
	suite.Require().True(dust.IsZero())
}

func (suite *IntegrationTestSuite) TestGetOrderSdkCoin_BuyOrderWithDust() {
	orderAmount := math.NewInt(100)
	// 100 * 0.333 = 33.3 → truncated to 33, dust = 0.3
	coin, dust, err := suite.k.GetOrderSdkCoin(types.OrderTypeBuy, "0.333", orderAmount, &market)
	suite.Require().NoError(err)
	suite.Require().Equal(market.Quote, coin.Denom)
	suite.Require().Equal(math.NewInt(33), coin.Amount)
	suite.Require().False(dust.IsZero())
	// dust = 33.3 - 33 = 0.3
	expectedDust := math.LegacyMustNewDecFromStr("0.3")
	suite.Require().Equal(expectedDust, dust)
}

func (suite *IntegrationTestSuite) TestGetOrderSdkCoin_BuyOrderNoDust() {
	orderAmount := math.NewInt(100)
	// 100 * 0.5 = 50 → no dust
	coin, dust, err := suite.k.GetOrderSdkCoin(types.OrderTypeBuy, "0.5", orderAmount, &market)
	suite.Require().NoError(err)
	suite.Require().Equal(market.Quote, coin.Denom)
	suite.Require().Equal(math.NewInt(50), coin.Amount)
	suite.Require().True(dust.IsZero())
}

func (suite *IntegrationTestSuite) TestGetOrderSdkCoin_SellOrder() {
	orderAmount := math.NewInt(500)
	coin, dust, err := suite.k.GetOrderSdkCoin(types.OrderTypeSell, "2", orderAmount, &market)
	suite.Require().NoError(err)
	// Sell order: coin denom = market.Base, amount = orderAmount
	suite.Require().Equal(market.Base, coin.Denom)
	suite.Require().Equal(math.NewInt(500), coin.Amount)
	suite.Require().True(dust.IsZero())
}

func (suite *IntegrationTestSuite) TestGetOrderSdkCoin_SellOrderPriceIgnored() {
	orderAmount := math.NewInt(300)
	// Price is irrelevant for sell orders
	coin, dust, err := suite.k.GetOrderSdkCoin(types.OrderTypeSell, "999", orderAmount, &market)
	suite.Require().NoError(err)
	suite.Require().Equal(market.Base, coin.Denom)
	suite.Require().Equal(math.NewInt(300), coin.Amount)
	suite.Require().True(dust.IsZero())
}

func (suite *IntegrationTestSuite) TestGetOrderSdkCoin_InvalidOrderType() {
	orderAmount := math.NewInt(100)
	_, _, err := suite.k.GetOrderSdkCoin("invalid", "1", orderAmount, &market)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInvalidOrderType)
}

func (suite *IntegrationTestSuite) TestGetOrderSdkCoin_InvalidPrice() {
	orderAmount := math.NewInt(100)
	_, _, err := suite.k.GetOrderSdkCoin(types.OrderTypeBuy, "notanumber", orderAmount, &market)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInvalidOrderPrice)
}

func (suite *IntegrationTestSuite) TestGetOrderSdkCoin_EmptyPrice() {
	orderAmount := math.NewInt(100)
	_, _, err := suite.k.GetOrderSdkCoin(types.OrderTypeBuy, "", orderAmount, &market)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInvalidOrderPrice)
}

func (suite *IntegrationTestSuite) TestGetOrderSdkCoin_BuyOrderLargeAmount() {
	orderAmount := math.NewInt(1_000_000)
	coin, dust, err := suite.k.GetOrderSdkCoin(types.OrderTypeBuy, "1.5", orderAmount, &market)
	suite.Require().NoError(err)
	suite.Require().Equal(market.Quote, coin.Denom)
	// 1_000_000 * 1.5 = 1_500_000
	suite.Require().Equal(math.NewInt(1_500_000), coin.Amount)
	suite.Require().True(dust.IsZero())
}

// ============ GetOrderCoinsWithDust Tests ============

func (suite *IntegrationTestSuite) TestGetOrderCoinsWithDust_BuyOrder() {
	args := types.OrderCoinsArguments{
		OrderType:    types.OrderTypeBuy,
		OrderPrice:   "2",
		OrderAmount:  math.NewInt(100),
		Market:       &market,
		UserAddress:  "bze1testaddr",
		UserReceives: false,
	}

	result, err := suite.k.GetOrderCoinsWithDust(suite.ctx, args)
	suite.Require().NoError(err)
	suite.Require().Equal(market.Quote, result.Coin.Denom)
	suite.Require().Equal(math.NewInt(200), result.Coin.Amount)
	suite.Require().True(result.Dust.IsZero())
	suite.Require().Nil(result.UserDust)
}

func (suite *IntegrationTestSuite) TestGetOrderCoinsWithDust_SellOrder() {
	args := types.OrderCoinsArguments{
		OrderType:    types.OrderTypeSell,
		OrderPrice:   "2",
		OrderAmount:  math.NewInt(100),
		Market:       &market,
		UserAddress:  "bze1testaddr",
		UserReceives: false,
	}

	result, err := suite.k.GetOrderCoinsWithDust(suite.ctx, args)
	suite.Require().NoError(err)
	suite.Require().Equal(market.Base, result.Coin.Denom)
	suite.Require().Equal(math.NewInt(100), result.Coin.Amount)
	suite.Require().True(result.Dust.IsZero())
	suite.Require().Nil(result.UserDust)
}

func (suite *IntegrationTestSuite) TestGetOrderCoinsWithDust_InvalidOrderType() {
	args := types.OrderCoinsArguments{
		OrderType:    "invalid",
		OrderPrice:   "1",
		OrderAmount:  math.NewInt(100),
		Market:       &market,
		UserAddress:  "bze1testaddr",
		UserReceives: false,
	}

	_, err := suite.k.GetOrderCoinsWithDust(suite.ctx, args)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInvalidOrderType)
}

func (suite *IntegrationTestSuite) TestGetOrderCoinsWithDust_WithExistingDust() {
	addr := "bze1testaddr"
	// Store some existing dust for the user
	ud := types.UserDust{
		Owner:  addr,
		Amount: "0.5",
		Denom:  market.Quote,
	}
	suite.k.SetUserDust(suite.ctx, ud)

	args := types.OrderCoinsArguments{
		OrderType:    types.OrderTypeBuy,
		OrderPrice:   "0.333",
		OrderAmount:  math.NewInt(100),
		Market:       &market,
		UserAddress:  addr,
		UserReceives: false,
	}

	result, err := suite.k.GetOrderCoinsWithDust(suite.ctx, args)
	suite.Require().NoError(err)
	suite.Require().Equal(market.Quote, result.Coin.Denom)
	suite.Require().NotNil(result.UserDust)
}

func (suite *IntegrationTestSuite) TestGetOrderCoinsWithDust_BuyOrderWithDust() {
	args := types.OrderCoinsArguments{
		OrderType:    types.OrderTypeBuy,
		OrderPrice:   "0.333",
		OrderAmount:  math.NewInt(100),
		Market:       &market,
		UserAddress:  "bze1testaddr",
		UserReceives: false,
	}

	result, err := suite.k.GetOrderCoinsWithDust(suite.ctx, args)
	suite.Require().NoError(err)
	suite.Require().Equal(market.Quote, result.Coin.Denom)
	// 100 * 0.333 = 33.3, truncate to 33, dust = 0.3
	// Payer dust: coin += 1 (becomes 34), userDust stores 1 - 0.3 = 0.7
	suite.Require().Equal(math.NewInt(34), result.Coin.Amount)
	suite.Require().NotNil(result.UserDust)
	expectedStoredDust := math.LegacyOneDec().Sub(math.LegacyMustNewDecFromStr("0.3"))
	suite.Require().Equal(expectedStoredDust.String(), result.UserDust.Amount)
}

func (suite *IntegrationTestSuite) TestGetOrderCoinsWithDust_ReceiverWithDust() {
	args := types.OrderCoinsArguments{
		OrderType:    types.OrderTypeBuy,
		OrderPrice:   "0.333",
		OrderAmount:  math.NewInt(100),
		Market:       &market,
		UserAddress:  "bze1testaddr",
		UserReceives: true,
	}

	result, err := suite.k.GetOrderCoinsWithDust(suite.ctx, args)
	suite.Require().NoError(err)
	suite.Require().Equal(market.Quote, result.Coin.Denom)
	// 100 * 0.333 = 33.3, truncate to 33, dust = 0.3
	// Receiver: coin stays 33, dust stored = 0.3
	suite.Require().Equal(math.NewInt(33), result.Coin.Amount)
	suite.Require().NotNil(result.UserDust)
	expectedStoredDust := math.LegacyMustNewDecFromStr("0.3")
	suite.Require().Equal(expectedStoredDust.String(), result.UserDust.Amount)
}

func (suite *IntegrationTestSuite) TestGetOrderCoinsWithDust_SellOrderNoDust() {
	args := types.OrderCoinsArguments{
		OrderType:    types.OrderTypeSell,
		OrderPrice:   "0.333",
		OrderAmount:  math.NewInt(100),
		Market:       &market,
		UserAddress:  "bze1testaddr",
		UserReceives: false,
	}

	result, err := suite.k.GetOrderCoinsWithDust(suite.ctx, args)
	suite.Require().NoError(err)
	// Sell order never produces dust
	suite.Require().Equal(market.Base, result.Coin.Denom)
	suite.Require().Equal(sdk.NewCoin(market.Base, math.NewInt(100)), result.Coin)
	suite.Require().True(result.Dust.IsZero())
	suite.Require().Nil(result.UserDust)
}
