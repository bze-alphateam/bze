package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

func (suite *IntegrationTestSuite) TestSnapshotModifiedLiquidityPools_SinglePool() {
	// Create and save a pool (this adds it to modification queue)
	lp := types.LiquidityPool{
		Id:           "pool1",
		Base:         "tokenA",
		Quote:        "tokenB",
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp)

	// Verify pool is in modification queue
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 1)

	// Take snapshot of modified pools
	suite.k.SnapshotModifiedLiquidityPools(suite.ctx)

	// Verify snapshot was created
	snapshot, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "pool1")
	suite.Require().True(ok)
	suite.Require().Equal(lp.String(), snapshot.String())

	// Verify modification queue was cleared
	modifiedIds = suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 0)
}

func (suite *IntegrationTestSuite) TestSnapshotModifiedLiquidityPools_MultiplePools() {
	// Create and save multiple pools
	lp1 := types.LiquidityPool{
		Id:           "pool1",
		Base:         "tokenA",
		Quote:        "tokenB",
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp1)

	lp2 := types.LiquidityPool{
		Id:           "pool2",
		Base:         "tokenC",
		Quote:        "tokenD",
		Fee:          math.LegacyNewDecWithPrec(5, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(3000),
		ReserveQuote: math.NewInt(4000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp2)

	lp3 := types.LiquidityPool{
		Id:           "pool3",
		Base:         "tokenE",
		Quote:        "tokenF",
		Fee:          math.LegacyNewDecWithPrec(1, 2),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(5000),
		ReserveQuote: math.NewInt(6000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp3)

	// Verify all pools are in modification queue
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 3)

	// Take snapshot of modified pools
	suite.k.SnapshotModifiedLiquidityPools(suite.ctx)

	// Verify all snapshots were created
	snapshot1, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "pool1")
	suite.Require().True(ok)
	suite.Require().Equal(lp1.String(), snapshot1.String())

	snapshot2, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "pool2")
	suite.Require().True(ok)
	suite.Require().Equal(lp2.String(), snapshot2.String())

	snapshot3, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "pool3")
	suite.Require().True(ok)
	suite.Require().Equal(lp3.String(), snapshot3.String())

	// Verify modification queue was cleared
	modifiedIds = suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 0)

	// Verify we have exactly 3 snapshots
	allSnapshots := suite.k.GetAllLiquidityPoolSnapshots(suite.ctx)
	suite.Require().Len(allSnapshots, 3)
}

func (suite *IntegrationTestSuite) TestSnapshotModifiedLiquidityPools_UpdatedPool() {
	// Create and save a pool
	lp := types.LiquidityPool{
		Id:           "pool_update",
		Base:         "tokenA",
		Quote:        "tokenB",
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp)

	// Take first snapshot
	suite.k.SnapshotModifiedLiquidityPools(suite.ctx)

	// Verify first snapshot
	snapshot1, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "pool_update")
	suite.Require().True(ok)
	suite.Require().Equal(math.NewInt(1000), snapshot1.ReserveBase)
	suite.Require().Equal(math.NewInt(2000), snapshot1.ReserveQuote)

	// Update the pool
	lp.ReserveBase = math.NewInt(5000)
	lp.ReserveQuote = math.NewInt(10000)
	suite.k.SetLiquidityPool(suite.ctx, lp)

	// Take second snapshot
	suite.k.SnapshotModifiedLiquidityPools(suite.ctx)

	// Verify snapshot was updated
	snapshot2, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "pool_update")
	suite.Require().True(ok)
	suite.Require().Equal(math.NewInt(5000), snapshot2.ReserveBase)
	suite.Require().Equal(math.NewInt(10000), snapshot2.ReserveQuote)
}

func (suite *IntegrationTestSuite) TestSnapshotModifiedLiquidityPools_EmptyQueue() {
	// Take snapshot when no pools are modified
	suite.k.SnapshotModifiedLiquidityPools(suite.ctx)

	// Verify no snapshots were created
	allSnapshots := suite.k.GetAllLiquidityPoolSnapshots(suite.ctx)
	suite.Require().Len(allSnapshots, 0)

	// Verify modification queue is still empty
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 0)
}

func (suite *IntegrationTestSuite) TestSnapshotModifiedLiquidityPools_MissingPool() {
	// Manually add a pool ID to modification queue without creating the pool
	// This simulates the edge case mentioned in the error handling
	suite.k.AddLpToModificationQueue(suite.ctx, "non_existent_pool")

	// Create and save a real pool
	lp := types.LiquidityPool{
		Id:           "real_pool",
		Base:         "tokenA",
		Quote:        "tokenB",
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp)

	// Verify both IDs are in modification queue
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 2)

	// Take snapshot (should skip the non-existent pool and log error)
	suite.k.SnapshotModifiedLiquidityPools(suite.ctx)

	// Verify only the real pool snapshot was created
	_, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "non_existent_pool")
	suite.Require().False(ok)

	snapshot, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "real_pool")
	suite.Require().True(ok)
	suite.Require().Equal(lp.String(), snapshot.String())

	// Verify modification queue was cleared (even though one pool was missing)
	modifiedIds = suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 0)

	// Verify we have exactly 1 snapshot
	allSnapshots := suite.k.GetAllLiquidityPoolSnapshots(suite.ctx)
	suite.Require().Len(allSnapshots, 1)
}

func (suite *IntegrationTestSuite) TestSnapshotModifiedLiquidityPools_MultipleSnapshots() {
	// First batch: Create and save pools
	lp1 := types.LiquidityPool{
		Id:           "pool1",
		Base:         "tokenA",
		Quote:        "tokenB",
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp1)

	lp2 := types.LiquidityPool{
		Id:           "pool2",
		Base:         "tokenC",
		Quote:        "tokenD",
		Fee:          math.LegacyNewDecWithPrec(5, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(3000),
		ReserveQuote: math.NewInt(4000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp2)

	// Take first snapshot
	suite.k.SnapshotModifiedLiquidityPools(suite.ctx)

	// Verify first snapshot
	allSnapshots := suite.k.GetAllLiquidityPoolSnapshots(suite.ctx)
	suite.Require().Len(allSnapshots, 2)

	// Second batch: Create new pools
	lp3 := types.LiquidityPool{
		Id:           "pool3",
		Base:         "tokenE",
		Quote:        "tokenF",
		Fee:          math.LegacyNewDecWithPrec(1, 2),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(5000),
		ReserveQuote: math.NewInt(6000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp3)

	// Take second snapshot
	suite.k.SnapshotModifiedLiquidityPools(suite.ctx)

	// Verify all snapshots exist (old + new)
	allSnapshots = suite.k.GetAllLiquidityPoolSnapshots(suite.ctx)
	suite.Require().Len(allSnapshots, 3)

	// Verify modification queue is empty after second snapshot
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 0)
}

func (suite *IntegrationTestSuite) TestSnapshotModifiedLiquidityPools_PartialUpdate() {
	// Create three pools
	lp1 := types.LiquidityPool{
		Id:           "pool1",
		Base:         "tokenA",
		Quote:        "tokenB",
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp1)

	lp2 := types.LiquidityPool{
		Id:           "pool2",
		Base:         "tokenC",
		Quote:        "tokenD",
		Fee:          math.LegacyNewDecWithPrec(5, 3),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(3000),
		ReserveQuote: math.NewInt(4000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp2)

	lp3 := types.LiquidityPool{
		Id:           "pool3",
		Base:         "tokenE",
		Quote:        "tokenF",
		Fee:          math.LegacyNewDecWithPrec(1, 2),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(5000),
		ReserveQuote: math.NewInt(6000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp3)

	// Take snapshot
	suite.k.SnapshotModifiedLiquidityPools(suite.ctx)

	// Update only pool2
	lp2.ReserveBase = math.NewInt(9000)
	suite.k.SetLiquidityPool(suite.ctx, lp2)

	// Verify only pool2 is in modification queue
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 1)
	suite.Require().Contains(modifiedIds, "pool2")

	// Take another snapshot
	suite.k.SnapshotModifiedLiquidityPools(suite.ctx)

	// Verify pool2 snapshot was updated
	snapshot2, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "pool2")
	suite.Require().True(ok)
	suite.Require().Equal(math.NewInt(9000), snapshot2.ReserveBase)

	// Verify pool1 and pool3 snapshots are unchanged
	snapshot1, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "pool1")
	suite.Require().True(ok)
	suite.Require().Equal(math.NewInt(1000), snapshot1.ReserveBase)

	snapshot3, ok := suite.k.GetLiquidityPoolSnapshot(suite.ctx, "pool3")
	suite.Require().True(ok)
	suite.Require().Equal(math.NewInt(5000), snapshot3.ReserveBase)
}
