package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

func (suite *IntegrationTestSuite) TestStoreLpSnapshot() {
	// Try to get a snapshot that doesn't exist
	_, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "not_a_pool_id")
	suite.Require().False(ok)

	lp := types.LiquidityPool{
		Id:           "test_snapshot",
		Base:         "tokenA",
		Quote:        "tokenB",
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}

	// Save snapshot
	suite.k.SetLiquidityPoolSnapshot(suite.ctx, lp)

	// Retrieve snapshot
	stored, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, lp.GetId())
	suite.Require().True(ok)
	suite.Require().Equal(lp.String(), stored.String())

	// Check for a random ID, shouldn't exist
	_, ok = suite.k.GetLiquidityPoolSnapshot(suite.ctx, "not_a_pool_id")
	suite.Require().False(ok)

	// Get all snapshots
	all := suite.k.GetAllLiquidityPoolSnapshots(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().Equal(all[0].String(), lp.String())
}

func (suite *IntegrationTestSuite) TestStoreLpSnapshot_MultipleSnapshots() {
	lp1 := types.LiquidityPool{
		Id:           "snapshot1",
		Base:         "tokenA",
		Quote:        "tokenB",
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPoolSnapshot(suite.ctx, lp1)

	lp2 := types.LiquidityPool{
		Id:           "snapshot2",
		Base:         "tokenC",
		Quote:        "tokenD",
		Fee:          math.LegacyNewDecWithPrec(5, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(3000),
		ReserveQuote: math.NewInt(4000),
	}
	suite.k.SetLiquidityPoolSnapshot(suite.ctx, lp2)

	lp3 := types.LiquidityPool{
		Id:           "snapshot3",
		Base:         "tokenE",
		Quote:        "tokenF",
		Fee:          math.LegacyNewDecWithPrec(1, 2),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(5000),
		ReserveQuote: math.NewInt(6000),
	}
	suite.k.SetLiquidityPoolSnapshot(suite.ctx, lp3)

	// Verify all snapshots are stored
	all := suite.k.GetAllLiquidityPoolSnapshots(suite.ctx)
	suite.Require().Len(all, 3)

	// Verify each snapshot can be retrieved
	stored1, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "snapshot1")
	suite.Require().True(ok)
	suite.Require().Equal(lp1.String(), stored1.String())

	stored2, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "snapshot2")
	suite.Require().True(ok)
	suite.Require().Equal(lp2.String(), stored2.String())

	stored3, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "snapshot3")
	suite.Require().True(ok)
	suite.Require().Equal(lp3.String(), stored3.String())
}

func (suite *IntegrationTestSuite) TestStoreLpSnapshot_OverwriteExisting() {
	lp := types.LiquidityPool{
		Id:           "snapshot_overwrite",
		Base:         "tokenA",
		Quote:        "tokenB",
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPoolSnapshot(suite.ctx, lp)

	// Retrieve initial snapshot
	stored, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, lp.GetId())
	suite.Require().True(ok)
	suite.Require().Equal(math.NewInt(1000), stored.ReserveBase)
	suite.Require().Equal(math.NewInt(2000), stored.ReserveQuote)

	// Update the pool and save snapshot again
	lp.ReserveBase = math.NewInt(5000)
	lp.ReserveQuote = math.NewInt(10000)
	suite.k.SetLiquidityPoolSnapshot(suite.ctx, lp)

	// Verify the snapshot was overwritten
	updated, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, lp.GetId())
	suite.Require().True(ok)
	suite.Require().Equal(math.NewInt(5000), updated.ReserveBase)
	suite.Require().Equal(math.NewInt(10000), updated.ReserveQuote)

	// Verify only one snapshot exists (not duplicated)
	all := suite.k.GetAllLiquidityPoolSnapshots(suite.ctx)
	suite.Require().Len(all, 1)
}

func (suite *IntegrationTestSuite) TestStoreLpSnapshot_IndependentFromActualPool() {
	// Create and save an actual pool
	lp := types.LiquidityPool{
		Id:           "independent_pool",
		Base:         "tokenA",
		Quote:        "tokenB",
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp)

	// Save a snapshot
	suite.k.SetLiquidityPoolSnapshot(suite.ctx, lp)

	// Modify the actual pool
	lp.ReserveBase = math.NewInt(5000)
	lp.ReserveQuote = math.NewInt(10000)
	suite.k.SetLiquidityPool(suite.ctx, lp)

	// Verify the snapshot is unchanged
	snapshot, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "independent_pool")
	suite.Require().True(ok)
	suite.Require().Equal(math.NewInt(1000), snapshot.ReserveBase)
	suite.Require().Equal(math.NewInt(2000), snapshot.ReserveQuote)

	// Verify the actual pool was updated
	actualPool, ok := suite.k.GetLiquidityPool(suite.ctx, "independent_pool")
	suite.Require().True(ok)
	suite.Require().Equal(math.NewInt(5000), actualPool.ReserveBase)
	suite.Require().Equal(math.NewInt(10000), actualPool.ReserveQuote)
}

func (suite *IntegrationTestSuite) TestStoreLpSnapshot_EmptySnapshots() {
	// Get all snapshots when none exist
	all := suite.k.GetAllLiquidityPoolSnapshots(suite.ctx)
	suite.Require().Len(all, 0)
}
