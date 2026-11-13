package keeper_test

import (
	"cosmossdk.io/math"
)

// ppaSaveEveryBlock saves pool once per block from height from to height to, both included, and returns the
// stored average after each save, indexed by height.
func (suite *IntegrationTestSuite) ppaSaveEveryBlock(pool func() (math.Int, math.Int), from, to int64) map[int64]math.LegacyDec {
	averages := make(map[int64]math.LegacyDec, to-from+1)
	for h := from; h <= to; h++ {
		base, quote := pool()
		suite.k.SetLiquidityPool(suite.ppaAt(h), ppaPoolWith(base, quote))
		averages[h] = suite.ppaRecord(suite.ppaAt(h)).AvgPrice
	}

	return averages
}

// The fold keeps 18 decimals; at very small prices it stops short of the price. This pins only the direction:
// with the pool saved once per block at a price in the 1e-16 range, the average moves toward the price, never
// moves away from it and never crosses it.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_TinyPriceMovesDownTowardThePrice() {
	base := math.NewIntWithDecimal(1, 16)
	price := ppaDec("0.0000000000000001")

	// block 10 seeds the average at 2e-16; block 11 moves the pool to 1e-16
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolWith(base, math.NewInt(2)))
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolWith(base, math.NewInt(1)))
	seed := suite.ppaRecord(suite.ppaAt(11)).AvgPrice
	suite.Require().Equal(ppaDec("0.0000000000000002"), seed)

	averages := suite.ppaSaveEveryBlock(func() (math.Int, math.Int) { return base, math.NewInt(1) }, 12, 211)
	averages[11] = seed

	suite.Require().True(averages[211].LT(seed), "the average moves toward the price")
	for h := int64(12); h <= 211; h++ {
		suite.Require().True(averages[h].LTE(averages[h-1]), "height %d", h)
		suite.Require().True(averages[h].GTE(price), "height %d", h)
	}
}

// The same from below: an average seeded at 1e-16 that follows a pool moved to 2e-16 moves up toward it.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_TinyPriceMovesUpTowardThePrice() {
	base := math.NewIntWithDecimal(1, 16)
	price := ppaDec("0.0000000000000002")

	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolWith(base, math.NewInt(1)))
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolWith(base, math.NewInt(2)))
	seed := suite.ppaRecord(suite.ppaAt(11)).AvgPrice
	suite.Require().Equal(ppaDec("0.0000000000000001"), seed)

	averages := suite.ppaSaveEveryBlock(func() (math.Int, math.Int) { return base, math.NewInt(2) }, 12, 211)
	averages[11] = seed

	suite.Require().True(averages[211].GT(seed), "the average moves toward the price")
	for h := int64(12); h <= 211; h++ {
		suite.Require().True(averages[h].GTE(averages[h-1]), "height %d", h)
		suite.Require().True(averages[h].LTE(price), "height %d", h)
	}
}
