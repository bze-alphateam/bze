package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

func (suite *IntegrationTestSuite) TestStoreLp() {
	_, ok := suite.k.GetLiquidityPool(suite.ctx, "not_a_pool_id")
	suite.Require().False(ok)

	lp := types.LiquidityPool{
		Id:           "test",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.ZeroInt(),
		ReserveQuote: math.ZeroInt(),
	}

	//save LP
	suite.k.SetLiquidityPool(suite.ctx, lp)
	stored, ok := suite.k.GetLiquidityPool(suite.ctx, lp.GetId())
	//check it was saved
	suite.Require().True(ok)
	suite.Require().Equal(lp.String(), stored.String())

	//check for a random ID, shouldn't exist
	_, ok = suite.k.GetLiquidityPool(suite.ctx, "not_a_pool_id")
	suite.Require().False(ok)

	//let's get the list
	all := suite.k.GetAllLiquidityPool(suite.ctx)
	//check the list contains only the saved LP
	suite.Require().Len(all, 1)
	suite.Require().Equal(all[0].String(), lp.String())

	lp2 := types.LiquidityPool{
		Id:           "test2",
		Base:         "abc",
		Quote:        "xyz",
		LpDenom:      "",
		Creator:      "address",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.ZeroInt(),
		ReserveQuote: math.ZeroInt(),
		Stable:       false,
	}

	//save a second LP
	suite.k.SetLiquidityPool(suite.ctx, lp2)
	//make sure second LP was saved
	stored2, ok := suite.k.GetLiquidityPool(suite.ctx, lp2.GetId())
	suite.Require().True(ok)
	suite.Require().Equal(lp2.String(), stored2.String())
	//make sure second LP is not the same as initial LP
	suite.Require().NotEqual(stored.String(), stored2.String())

	//let's get the list
	all = suite.k.GetAllLiquidityPool(suite.ctx)
	//check the list contains only the saved LP
	suite.Require().Len(all, 2)
	suite.Require().NotEqual(all[0].String(), all[1].String())
}

func (suite *IntegrationTestSuite) TestLpModificationQueue_SinglePool() {
	// Initially, modification queue should be empty
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 0)

	// Create and save a pool
	lp := types.LiquidityPool{
		Id:           "pool1",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp)

	// Check that pool ID was added to modification queue
	modifiedIds = suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 1)
	suite.Require().Equal("pool1", modifiedIds[0])
}

func (suite *IntegrationTestSuite) TestLpModificationQueue_MultiplePools() {
	// Create and save multiple pools
	lp1 := types.LiquidityPool{
		Id:           "pool1",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp1)

	lp2 := types.LiquidityPool{
		Id:           "pool2",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(3000),
		ReserveQuote: math.NewInt(4000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp2)

	lp3 := types.LiquidityPool{
		Id:           "pool3",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(5000),
		ReserveQuote: math.NewInt(6000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp3)

	// Check that all pool IDs were added to modification queue
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 3)
	suite.Require().Contains(modifiedIds, "pool1")
	suite.Require().Contains(modifiedIds, "pool2")
	suite.Require().Contains(modifiedIds, "pool3")
}

func (suite *IntegrationTestSuite) TestLpModificationQueue_DuplicateModifications() {
	// Create and save a pool
	lp := types.LiquidityPool{
		Id:           "pool1",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp)

	// Modify and save the same pool again
	lp.ReserveBase = math.NewInt(1500)
	suite.k.SetLiquidityPool(suite.ctx, lp)

	// Modify and save the same pool a third time
	lp.ReserveQuote = math.NewInt(2500)
	suite.k.SetLiquidityPool(suite.ctx, lp)

	// Check that pool ID appears only once in modification queue (no duplicates)
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 1)
	suite.Require().Equal("pool1", modifiedIds[0])
}

func (suite *IntegrationTestSuite) TestLpModificationQueue_ClearQueue() {
	// Create and save multiple pools
	lp1 := types.LiquidityPool{
		Id:           "pool1",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp1)

	lp2 := types.LiquidityPool{
		Id:           "pool2",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(3000),
		ReserveQuote: math.NewInt(4000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp2)

	// Verify pools were added to queue
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 2)

	// Clear the queue with specific pool IDs
	suite.k.ClearLpModificationQueue(suite.ctx, modifiedIds)

	// Verify queue is empty
	modifiedIds = suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 0)
}

func (suite *IntegrationTestSuite) TestLpModificationQueue_ClearAndReuse() {
	// Create and save a pool
	lp1 := types.LiquidityPool{
		Id:           "pool1",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp1)

	// Get the modified IDs and clear the queue
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.k.ClearLpModificationQueue(suite.ctx, modifiedIds)

	// Verify queue is empty
	modifiedIds = suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 0)

	// Create and save new pools
	lp2 := types.LiquidityPool{
		Id:           "pool2",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(3000),
		ReserveQuote: math.NewInt(4000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp2)

	lp3 := types.LiquidityPool{
		Id:           "pool3",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(5000),
		ReserveQuote: math.NewInt(6000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp3)

	// Verify only new pools are in the queue
	modifiedIds = suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 2)
	suite.Require().Contains(modifiedIds, "pool2")
	suite.Require().Contains(modifiedIds, "pool3")
	suite.Require().NotContains(modifiedIds, "pool1")
}

func (suite *IntegrationTestSuite) TestLpModificationQueue_PartialClear() {
	// Create and save multiple pools
	lp1 := types.LiquidityPool{
		Id:           "pool1",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp1)

	lp2 := types.LiquidityPool{
		Id:           "pool2",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(3000),
		ReserveQuote: math.NewInt(4000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp2)

	lp3 := types.LiquidityPool{
		Id:           "pool3",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  math.NewInt(5000),
		ReserveQuote: math.NewInt(6000),
	}
	suite.k.SetLiquidityPool(suite.ctx, lp3)

	// Verify all pools were added
	modifiedIds := suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 3)

	// Clear only pool1 and pool3 from the queue
	suite.k.ClearLpModificationQueue(suite.ctx, []string{"pool1", "pool3"})

	// Verify only pool2 remains in the queue
	modifiedIds = suite.k.GetModifiedLpIds(suite.ctx)
	suite.Require().Len(modifiedIds, 1)
	suite.Require().Contains(modifiedIds, "pool2")
	suite.Require().NotContains(modifiedIds, "pool1")
	suite.Require().NotContains(modifiedIds, "pool3")
}
