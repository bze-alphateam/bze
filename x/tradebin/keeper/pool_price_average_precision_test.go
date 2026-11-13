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

// Known precision limit of the price average, pinned as it is: the fold keeps 18 decimals, so for a price in the
// 1e-16 range the gap to the price is a handful of 1e-18 units. Each block keeps d = 0.5^(1/20) ~ 0.966 of the
// gap and rounds it back to a whole unit; once the gap is 14 units, 14 x 0.966 = 13.52 rounds to 14 again and the
// average stops moving. With the pool saved once per block at price 1e-16 for 200 blocks (10 half-lives, where
// exact arithmetic would leave ~0.1 unit of gap), the average stays at 1.14e-16, 14% above the price. It does not
// converge, and the relative gap is larger than the default fee and module drift limits (10%), so such a pool
// counts as drifted for as long as it keeps being saved every block. A save after more than 20 half-lives without
// one resets the average to the price.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_TinyPriceFoldStallsAboveThePrice() {
	suite.Require().Equal(uint64(20), suite.k.GetParams(suite.ctx).PoolPriceHalfLifeBlocks)
	base := math.NewIntWithDecimal(1, 16)
	price := ppaDec("0.0000000000000001")

	// block 10 seeds the average at 2e-16; block 11 moves the pool to 1e-16
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolWith(base, math.NewInt(2)))
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolWith(base, math.NewInt(1)))
	suite.Require().Equal(ppaDec("0.0000000000000002"), suite.ppaRecord(suite.ppaAt(11)).AvgPrice)

	averages := suite.ppaSaveEveryBlock(func() (math.Int, math.Int) { return base, math.NewInt(1) }, 12, 211)

	// the first folds move by ~3% of the gap, as expected
	suite.Require().Equal(ppaDec("0.000000000000000197"), averages[12])
	suite.Require().Equal(ppaDec("0.000000000000000194"), averages[13])
	// the average never goes past the price and never moves away from it
	for h := int64(13); h <= 211; h++ {
		suite.Require().True(averages[h].LTE(averages[h-1]), "height %d", h)
		suite.Require().True(averages[h].GT(price), "height %d", h)
	}
	// it stalls 14 units above the price well before the 200 blocks are over
	stalled := ppaDec("0.000000000000000114")
	for h := int64(111); h <= 211; h++ {
		suite.Require().Equal(stalled, averages[h], "height %d", h)
	}

	// the read-only view agrees, and the relative gap is over the default drift limits
	avg, found := suite.k.GetPoolPriceAverage(suite.ppaAt(212), ppaPool)
	suite.Require().True(found)
	suite.Require().Equal(stalled, avg)
	gap := avg.Sub(price).Quo(avg)
	params := suite.k.GetParams(suite.ctx)
	suite.Require().True(gap.GT(params.FeeMaxPriceDrift), "gap %s", gap)
	suite.Require().True(gap.GT(params.ModuleMaxPriceDrift), "gap %s", gap)
}

// The same limit from below: an average seeded at 1e-16 that follows a pool moved to 2e-16 stalls 14 units under it.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_TinyPriceFoldStallsBelowThePrice() {
	base := math.NewIntWithDecimal(1, 16)

	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolWith(base, math.NewInt(1)))
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolWith(base, math.NewInt(2)))
	averages := suite.ppaSaveEveryBlock(func() (math.Int, math.Int) { return base, math.NewInt(2) }, 12, 211)

	suite.Require().Equal(ppaDec("0.000000000000000186"), averages[211])
	suite.Require().Equal(averages[150], averages[211])
}
