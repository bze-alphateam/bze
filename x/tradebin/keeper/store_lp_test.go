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
