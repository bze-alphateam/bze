package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestStoreLp() {
	_, ok := suite.k.GetLiquidityPool(suite.ctx, "not_a_pool_id")
	suite.Require().False(ok)

	lp := types.LiquidityPool{
		Id:      "test",
		Fee:     sdk.ZeroDec(),
		FeeDest: &types.FeeDestination{},
	}

	//save LP
	suite.k.SetLiquidityPool(suite.ctx, lp)
	stored, ok := suite.k.GetLiquidityPool(suite.ctx, lp.GetId())
	//check it was saved
	suite.Require().True(ok)
	suite.Require().Equal(lp, stored)

	//check for a random ID, shouldn't exist
	_, ok = suite.k.GetLiquidityPool(suite.ctx, "not_a_pool_id")
	suite.Require().False(ok)

	//let's get the list
	all := suite.k.GetAllLiquidityPool(suite.ctx)
	//check the list contains only the saved LP
	suite.Require().Len(all, 1)
	suite.Require().Equal(all[0], lp)

	lp2 := types.LiquidityPool{
		Id:           "test2",
		Base:         "abc",
		Quote:        "xyz",
		LpDenom:      "",
		Creator:      "address",
		Fee:          sdk.ZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  0,
		ReserveQuote: 0,
		Stable:       false,
	}

	//save a second LP
	suite.k.SetLiquidityPool(suite.ctx, lp2)
	//make sure second LP was saved
	stored2, ok := suite.k.GetLiquidityPool(suite.ctx, lp2.GetId())
	suite.Require().True(ok)
	suite.Require().Equal(lp2, stored2)
	//make sure second LP is not the same as initial LP
	suite.Require().NotEqual(stored, stored2)

	//let's get the list
	all = suite.k.GetAllLiquidityPool(suite.ctx)
	//check the list contains only the saved LP
	suite.Require().Len(all, 2)
	suite.Require().NotEqual(all[0], all[1])
}
