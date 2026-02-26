package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"time"
)

func newHistoryOrder(marketId, orderType, maker, taker string, amount math.Int, price math.LegacyDec, executedAt int64) types.HistoryOrder {
	return types.HistoryOrder{
		MarketId:   marketId,
		OrderType:  orderType,
		Amount:     amount,
		Price:      price,
		ExecutedAt: executedAt,
		Maker:      maker,
		Taker:      taker,
	}
}

func (suite *IntegrationTestSuite) TestStore_SetAndGetAllHistoryOrder() {
	// Verify no history orders exist initially
	initial := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Empty(initial)

	// Create and store a single history order
	ho := newHistoryOrder(
		getMarketId(),
		types.OrderTypeBuy,
		"maker1",
		"taker1",
		math.NewInt(100),
		math.LegacyMustNewDecFromStr("50"),
		time.Now().Unix(),
	)
	suite.k.SetHistoryOrder(suite.ctx, ho, "idx1")

	// Verify it can be retrieved
	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().Equal(ho.MarketId, all[0].MarketId)
	suite.Require().Equal(ho.OrderType, all[0].OrderType)
	suite.Require().True(ho.Amount.Equal(all[0].Amount))
	suite.Require().True(ho.Price.Equal(all[0].Price))
	suite.Require().Equal(ho.ExecutedAt, all[0].ExecutedAt)
	suite.Require().Equal(ho.Maker, all[0].Maker)
	suite.Require().Equal(ho.Taker, all[0].Taker)
}

func (suite *IntegrationTestSuite) TestStore_SetHistoryOrder_MultipleOrders() {
	marketId := getMarketId()
	now := time.Now().Unix()

	orders := []struct {
		order types.HistoryOrder
		index string
	}{
		{
			order: newHistoryOrder(marketId, types.OrderTypeBuy, "maker1", "taker1", math.NewInt(100), math.LegacyMustNewDecFromStr("10"), now),
			index: "idx1",
		},
		{
			order: newHistoryOrder(marketId, types.OrderTypeSell, "maker2", "taker2", math.NewInt(200), math.LegacyMustNewDecFromStr("20"), now+1),
			index: "idx2",
		},
		{
			order: newHistoryOrder(marketId, types.OrderTypeBuy, "maker3", "taker3", math.NewInt(300), math.LegacyMustNewDecFromStr("30"), now+2),
			index: "idx3",
		},
	}

	for _, o := range orders {
		suite.k.SetHistoryOrder(suite.ctx, o.order, o.index)
	}

	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, 3)

	// Verify all amounts are present (order may vary due to key sorting)
	amounts := make(map[string]bool)
	for _, ho := range all {
		amounts[ho.Amount.String()] = true
	}
	suite.Require().True(amounts["100"])
	suite.Require().True(amounts["200"])
	suite.Require().True(amounts["300"])
}

func (suite *IntegrationTestSuite) TestStore_SetHistoryOrder_OverwriteSameIndex() {
	marketId := getMarketId()
	now := time.Now().Unix()
	index := "idx_overwrite"

	// Store first order
	ho1 := newHistoryOrder(marketId, types.OrderTypeBuy, "maker1", "taker1", math.NewInt(100), math.LegacyMustNewDecFromStr("10"), now)
	suite.k.SetHistoryOrder(suite.ctx, ho1, index)

	// Overwrite with same market and timestamp but different data using same index
	ho2 := newHistoryOrder(marketId, types.OrderTypeSell, "maker2", "taker2", math.NewInt(999), math.LegacyMustNewDecFromStr("55"), now)
	suite.k.SetHistoryOrder(suite.ctx, ho2, index)

	// Should still have exactly 1 order (overwritten)
	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().True(all[0].Amount.Equal(math.NewInt(999)))
	suite.Require().True(all[0].Price.Equal(math.LegacyMustNewDecFromStr("55")))
	suite.Require().Equal("maker2", all[0].Maker)
	suite.Require().Equal("taker2", all[0].Taker)
	suite.Require().Equal(types.OrderTypeSell, all[0].OrderType)
}

func (suite *IntegrationTestSuite) TestStore_SetHistoryOrder_DifferentIndexSameTimestamp() {
	marketId := getMarketId()
	now := time.Now().Unix()

	// Two orders at the same timestamp but different indices
	ho1 := newHistoryOrder(marketId, types.OrderTypeBuy, "maker1", "taker1", math.NewInt(100), math.LegacyMustNewDecFromStr("10"), now)
	ho2 := newHistoryOrder(marketId, types.OrderTypeSell, "maker2", "taker2", math.NewInt(200), math.LegacyMustNewDecFromStr("20"), now)

	suite.k.SetHistoryOrder(suite.ctx, ho1, "idxA")
	suite.k.SetHistoryOrder(suite.ctx, ho2, "idxB")

	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, 2)
}

func (suite *IntegrationTestSuite) TestStore_SetHistoryOrder_MultipleMarkets() {
	market1 := "base1/quote1"
	market2 := "base2/quote2"
	now := time.Now().Unix()

	ho1 := newHistoryOrder(market1, types.OrderTypeBuy, "maker1", "taker1", math.NewInt(100), math.LegacyMustNewDecFromStr("10"), now)
	ho2 := newHistoryOrder(market2, types.OrderTypeSell, "maker2", "taker2", math.NewInt(200), math.LegacyMustNewDecFromStr("20"), now)

	suite.k.SetHistoryOrder(suite.ctx, ho1, "idx1")
	suite.k.SetHistoryOrder(suite.ctx, ho2, "idx2")

	// GetAllHistoryOrder should return orders from all markets
	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, 2)

	marketIds := make(map[string]bool)
	for _, ho := range all {
		marketIds[ho.MarketId] = true
	}
	suite.Require().True(marketIds[market1])
	suite.Require().True(marketIds[market2])
}

func (suite *IntegrationTestSuite) TestStore_GetAllHistoryOrder_Empty() {
	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Empty(all)
	suite.Require().Nil(all)
}

func (suite *IntegrationTestSuite) TestStore_SetHistoryOrder_PreservesAllFields() {
	ho := types.HistoryOrder{
		MarketId:   "custom/market",
		OrderType:  types.OrderTypeSell,
		Amount:     math.NewInt(123456789),
		Price:      math.LegacyMustNewDecFromStr("999.123456789"),
		ExecutedAt: 1700000000,
		Maker:      "bze1maker_address_here",
		Taker:      "bze1taker_address_here",
	}

	suite.k.SetHistoryOrder(suite.ctx, ho, "preserve_test")

	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, 1)

	retrieved := all[0]
	suite.Require().Equal(ho.MarketId, retrieved.MarketId)
	suite.Require().Equal(ho.OrderType, retrieved.OrderType)
	suite.Require().True(ho.Amount.Equal(retrieved.Amount))
	suite.Require().True(ho.Price.Equal(retrieved.Price))
	suite.Require().Equal(ho.ExecutedAt, retrieved.ExecutedAt)
	suite.Require().Equal(ho.Maker, retrieved.Maker)
	suite.Require().Equal(ho.Taker, retrieved.Taker)
}

func (suite *IntegrationTestSuite) TestStore_SetHistoryOrder_LargeAmountAndPrice() {
	// Test with very large values to ensure no precision loss
	largeAmount := math.NewIntFromBigInt(math.NewInt(1).BigInt().Mul(
		math.NewInt(1).BigInt(),
		math.NewInt(1_000_000_000_000_000).BigInt(),
	))
	largePrice := math.LegacyMustNewDecFromStr("99999999999999.999999999999999999")

	ho := newHistoryOrder(
		getMarketId(),
		types.OrderTypeBuy,
		"maker",
		"taker",
		largeAmount,
		largePrice,
		time.Now().Unix(),
	)

	suite.k.SetHistoryOrder(suite.ctx, ho, "large_vals")

	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().True(all[0].Amount.Equal(largeAmount))
	suite.Require().True(all[0].Price.Equal(largePrice))
}

func (suite *IntegrationTestSuite) TestStore_SetHistoryOrder_ZeroAmount() {
	ho := newHistoryOrder(
		getMarketId(),
		types.OrderTypeBuy,
		"maker",
		"taker",
		math.ZeroInt(),
		math.LegacyMustNewDecFromStr("10"),
		time.Now().Unix(),
	)

	suite.k.SetHistoryOrder(suite.ctx, ho, "zero_amt")

	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().True(all[0].Amount.IsZero())
}

func (suite *IntegrationTestSuite) TestStore_SetHistoryOrder_ManyOrders() {
	marketId := getMarketId()
	count := 50
	baseTime := time.Now().Unix()

	for i := 0; i < count; i++ {
		ho := newHistoryOrder(
			marketId,
			types.OrderTypeBuy,
			"maker",
			"taker",
			math.NewInt(int64(i+1)*10),
			math.LegacyMustNewDecFromStr("5"),
			baseTime+int64(i),
		)
		suite.k.SetHistoryOrder(suite.ctx, ho, fmt.Sprintf("idx_%d", i))
	}

	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, count)
}

func (suite *IntegrationTestSuite) TestStore_SetHistoryOrder_BuyAndSellTypes() {
	marketId := getMarketId()
	now := time.Now().Unix()

	buyOrder := newHistoryOrder(marketId, types.OrderTypeBuy, "maker", "taker", math.NewInt(100), math.LegacyMustNewDecFromStr("10"), now)
	sellOrder := newHistoryOrder(marketId, types.OrderTypeSell, "maker", "taker", math.NewInt(200), math.LegacyMustNewDecFromStr("20"), now)

	suite.k.SetHistoryOrder(suite.ctx, buyOrder, "buy_idx")
	suite.k.SetHistoryOrder(suite.ctx, sellOrder, "sell_idx")

	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, 2)

	orderTypes := make(map[string]bool)
	for _, ho := range all {
		orderTypes[ho.OrderType] = true
	}
	suite.Require().True(orderTypes[types.OrderTypeBuy])
	suite.Require().True(orderTypes[types.OrderTypeSell])
}

func (suite *IntegrationTestSuite) TestStore_SetHistoryOrder_TimestampZero() {
	ho := newHistoryOrder(
		getMarketId(),
		types.OrderTypeBuy,
		"maker",
		"taker",
		math.NewInt(100),
		math.LegacyMustNewDecFromStr("10"),
		0,
	)

	suite.k.SetHistoryOrder(suite.ctx, ho, "ts_zero")

	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().Equal(int64(0), all[0].ExecutedAt)
}

func (suite *IntegrationTestSuite) TestStore_SetHistoryOrder_DecimalPrice() {
	// Verify decimal precision is preserved through marshal/unmarshal
	prices := []string{
		"0.000000000000000001",
		"1.5",
		"100.123456789012345678",
		"0.1",
	}

	for i, priceStr := range prices {
		price := math.LegacyMustNewDecFromStr(priceStr)
		ho := newHistoryOrder(
			getMarketId(),
			types.OrderTypeBuy,
			"maker",
			"taker",
			math.NewInt(int64(i+1)),
			price,
			int64(i+1),
		)
		suite.k.SetHistoryOrder(suite.ctx, ho, fmt.Sprintf("dec_%d", i))
	}

	all := suite.k.GetAllHistoryOrder(suite.ctx)
	suite.Require().Len(all, len(prices))

	// Verify all prices are preserved exactly
	retrievedPrices := make(map[string]bool)
	for _, ho := range all {
		retrievedPrices[ho.Price.String()] = true
	}
	for _, priceStr := range prices {
		expected := math.LegacyMustNewDecFromStr(priceStr)
		suite.Require().True(retrievedPrices[expected.String()], "price %s should be preserved", priceStr)
	}
}
