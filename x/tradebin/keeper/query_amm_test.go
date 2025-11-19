package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestQueryAmm_LiquidityPool() {
	liquidityPool := types.LiquidityPool{
		Id:      "atom/usdc",
		Base:    "atom",
		Quote:   "usdc",
		LpDenom: "lp/atom/usdc",
		Creator: "bze1creator",
		Fee:     math.LegacyMustNewDecFromStr("0.003"), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyMustNewDecFromStr("0.5"),
			Burner:    math.LegacyMustNewDecFromStr("0.3"),
			Providers: math.LegacyMustNewDecFromStr("0.2"),
		},
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Stable:       false,
	}

	suite.k.SetLiquidityPool(suite.ctx, liquidityPool)

	req := &types.QueryLiquidityPoolRequest{
		PoolId: "atom/usdc",
	}

	response, err := suite.k.LiquidityPool(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().NotNil(response.Pool)
	suite.Require().Equal(liquidityPool.Id, response.Pool.Id)
	suite.Require().Equal(liquidityPool.Base, response.Pool.Base)
	suite.Require().Equal(liquidityPool.Quote, response.Pool.Quote)
	suite.Require().Equal(liquidityPool.LpDenom, response.Pool.LpDenom)
	suite.Require().Equal(liquidityPool.Creator, response.Pool.Creator)
	suite.Require().Equal(liquidityPool.Fee, response.Pool.Fee)
	suite.Require().Equal(liquidityPool.ReserveBase, response.Pool.ReserveBase)
	suite.Require().Equal(liquidityPool.ReserveQuote, response.Pool.ReserveQuote)
	suite.Require().Equal(liquidityPool.Stable, response.Pool.Stable)
}

func (suite *IntegrationTestSuite) TestQueryAmm_LiquidityPoolNilRequest() {
	response, err := suite.k.LiquidityPool(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryAmm_LiquidityPoolNotFound() {
	req := &types.QueryLiquidityPoolRequest{
		PoolId: "non-existent-pool",
	}

	response, err := suite.k.LiquidityPool(suite.ctx, req)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.NotFound, status.Code(err))
	suite.Require().Contains(err.Error(), "not found")
}

func (suite *IntegrationTestSuite) TestQueryAmm_LiquidityPoolStable() {
	liquidityPool := types.LiquidityPool{
		Id:      "usdt/usdc",
		Base:    "usdt",
		Quote:   "usdc",
		LpDenom: "lp/usdt/usdc",
		Creator: "bze1creator",
		Fee:     math.LegacyMustNewDecFromStr("0.001"), // 0.1%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyMustNewDecFromStr("0.6"),
			Burner:    math.LegacyMustNewDecFromStr("0.4"),
			Providers: math.LegacyMustNewDecFromStr("0.0"),
		},
		ReserveBase:  math.NewInt(5000000),
		ReserveQuote: math.NewInt(4950000),
		Stable:       true, // Stable pool
	}

	suite.k.SetLiquidityPool(suite.ctx, liquidityPool)

	req := &types.QueryLiquidityPoolRequest{
		PoolId: "usdt/usdc",
	}

	response, err := suite.k.LiquidityPool(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().NotNil(response.Pool)
	suite.Require().Equal(liquidityPool.Id, response.Pool.Id)
	suite.Require().True(response.Pool.Stable)
	suite.Require().Equal(liquidityPool.FeeDest.Treasury, response.Pool.FeeDest.Treasury)
	suite.Require().Equal(liquidityPool.FeeDest.Burner, response.Pool.FeeDest.Burner)
	suite.Require().Equal(liquidityPool.FeeDest.Providers, response.Pool.FeeDest.Providers)
}

func (suite *IntegrationTestSuite) TestQueryAmm_AllLiquidityPools() {
	liquidityPools := []types.LiquidityPool{
		{
			Id:      "atom/usdc",
			Base:    "atom",
			Quote:   "usdc",
			LpDenom: "lp/atom/usdc",
			Creator: "bze1creator1",
			Fee:     math.LegacyMustNewDecFromStr("0.003"),
			FeeDest: &types.FeeDestination{
				Treasury:  math.LegacyMustNewDecFromStr("0.5"),
				Burner:    math.LegacyMustNewDecFromStr("0.5"),
				Providers: math.LegacyMustNewDecFromStr("0.0"),
			},
			ReserveBase:  math.NewInt(1000000),
			ReserveQuote: math.NewInt(2000000),
			Stable:       false,
		},
		{
			Id:      "eth/usdc",
			Base:    "eth",
			Quote:   "usdc",
			LpDenom: "lp/eth/usdc",
			Creator: "bze1creator2",
			Fee:     math.LegacyMustNewDecFromStr("0.005"),
			FeeDest: &types.FeeDestination{
				Treasury:  math.LegacyMustNewDecFromStr("0.3"),
				Burner:    math.LegacyMustNewDecFromStr("0.3"),
				Providers: math.LegacyMustNewDecFromStr("0.4"),
			},
			ReserveBase:  math.NewInt(3000000),
			ReserveQuote: math.NewInt(4000000),
			Stable:       false,
		},
		{
			Id:      "usdt/usdc",
			Base:    "usdt",
			Quote:   "usdc",
			LpDenom: "lp/usdt/usdc",
			Creator: "bze1creator3",
			Fee:     math.LegacyMustNewDecFromStr("0.001"),
			FeeDest: &types.FeeDestination{
				Treasury:  math.LegacyMustNewDecFromStr("1.0"),
				Burner:    math.LegacyMustNewDecFromStr("0.0"),
				Providers: math.LegacyMustNewDecFromStr("0.0"),
			},
			ReserveBase:  math.NewInt(5000000),
			ReserveQuote: math.NewInt(6000000),
			Stable:       true,
		},
	}

	for _, pool := range liquidityPools {
		suite.k.SetLiquidityPool(suite.ctx, pool)
	}

	req := &types.QueryAllLiquidityPoolsRequest{
		Pagination: nil,
	}

	response, err := suite.k.AllLiquidityPools(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 3)

	// Verify all pools are present
	poolIds := make(map[string]bool)
	for _, pool := range response.List {
		poolIds[pool.Id] = true
	}

	suite.Require().True(poolIds["atom/usdc"])
	suite.Require().True(poolIds["eth/usdc"])
	suite.Require().True(poolIds["usdt/usdc"])
}

func (suite *IntegrationTestSuite) TestQueryAmm_AllLiquidityPoolsNilRequest() {
	response, err := suite.k.AllLiquidityPools(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryAmm_AllLiquidityPoolsEmpty() {
	req := &types.QueryAllLiquidityPoolsRequest{
		Pagination: nil,
	}

	response, err := suite.k.AllLiquidityPools(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Empty(response.List)
}

func (suite *IntegrationTestSuite) TestQueryAmm_AllLiquidityPoolsPagination() {
	liquidityPools := []types.LiquidityPool{
		{
			Id:      "atom/usdc",
			Base:    "atom",
			Quote:   "usdc",
			LpDenom: "lp/atom/usdc",
			Creator: "bze1creator1",
			Fee:     math.LegacyMustNewDecFromStr("0.003"),
			FeeDest: &types.FeeDestination{
				Treasury:  math.LegacyMustNewDecFromStr("0.5"),
				Burner:    math.LegacyMustNewDecFromStr("0.5"),
				Providers: math.LegacyMustNewDecFromStr("0.0"),
			},
			ReserveBase:  math.NewInt(1000000),
			ReserveQuote: math.NewInt(2000000),
			Stable:       false,
		},
		{
			Id:      "eth/usdc",
			Base:    "eth",
			Quote:   "usdc",
			LpDenom: "lp/eth/usdc",
			Creator: "bze1creator2",
			Fee:     math.LegacyMustNewDecFromStr("0.005"),
			FeeDest: &types.FeeDestination{
				Treasury:  math.LegacyMustNewDecFromStr("0.3"),
				Burner:    math.LegacyMustNewDecFromStr("0.3"),
				Providers: math.LegacyMustNewDecFromStr("0.4"),
			},
			ReserveBase:  math.NewInt(3000000),
			ReserveQuote: math.NewInt(4000000),
			Stable:       false,
		},
	}

	for _, pool := range liquidityPools {
		suite.k.SetLiquidityPool(suite.ctx, pool)
	}

	req := &types.QueryAllLiquidityPoolsRequest{
		Pagination: &query.PageRequest{
			Limit: 1,
		},
	}

	response, err := suite.k.AllLiquidityPools(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 1)
	suite.Require().NotNil(response.Pagination)
}

func (suite *IntegrationTestSuite) TestQueryAmm_AllLiquidityPoolsVariousFees() {
	liquidityPools := []types.LiquidityPool{
		{
			Id:      "btc/usdc",
			Base:    "btc",
			Quote:   "usdc",
			LpDenom: "lp/btc/usdc",
			Creator: "bze1creator1",
			Fee:     math.LegacyMustNewDecFromStr("0.01"), // 1%
			FeeDest: &types.FeeDestination{
				Treasury:  math.LegacyMustNewDecFromStr("0.2"),
				Burner:    math.LegacyMustNewDecFromStr("0.3"),
				Providers: math.LegacyMustNewDecFromStr("0.5"),
			},
			ReserveBase:  math.NewInt(1000000),
			ReserveQuote: math.NewInt(2000000),
			Stable:       false,
		},
		{
			Id:      "dai/usdc",
			Base:    "dai",
			Quote:   "usdc",
			LpDenom: "lp/dai/usdc",
			Creator: "bze1creator2",
			Fee:     math.LegacyMustNewDecFromStr("0.0001"), // 0.01%
			FeeDest: &types.FeeDestination{
				Treasury:  math.LegacyMustNewDecFromStr("0.0"),
				Burner:    math.LegacyMustNewDecFromStr("0.0"),
				Providers: math.LegacyMustNewDecFromStr("1.0"),
			},
			ReserveBase:  math.NewInt(3000000),
			ReserveQuote: math.NewInt(4000000),
			Stable:       true,
		},
	}

	for _, pool := range liquidityPools {
		suite.k.SetLiquidityPool(suite.ctx, pool)
	}

	req := &types.QueryAllLiquidityPoolsRequest{
		Pagination: nil,
	}

	response, err := suite.k.AllLiquidityPools(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 2)

	// Verify fee structures are included in response
	for _, pool := range response.List {
		if pool.Id == "btc/usdc" {
			suite.Require().Equal(math.LegacyMustNewDecFromStr("0.01"), pool.Fee)
			suite.Require().Equal(math.LegacyMustNewDecFromStr("0.2"), pool.FeeDest.Treasury)
			suite.Require().Equal(math.LegacyMustNewDecFromStr("0.3"), pool.FeeDest.Burner)
			suite.Require().Equal(math.LegacyMustNewDecFromStr("0.5"), pool.FeeDest.Providers)
		} else if pool.Id == "dai/usdc" {
			suite.Require().Equal(math.LegacyMustNewDecFromStr("0.0001"), pool.Fee)
			suite.Require().Equal(math.LegacyMustNewDecFromStr("1.0"), pool.FeeDest.Providers)
			suite.Require().True(pool.Stable)
		}
	}
}

func (suite *IntegrationTestSuite) TestQueryAmm_LiquidityPoolZeroReserves() {
	liquidityPool := types.LiquidityPool{
		Id:      "empty/pool",
		Base:    "empty",
		Quote:   "pool",
		LpDenom: "lp/empty/pool",
		Creator: "bze1creator",
		Fee:     math.LegacyMustNewDecFromStr("0.003"),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyMustNewDecFromStr("1.0"),
			Burner:    math.LegacyMustNewDecFromStr("0.0"),
			Providers: math.LegacyMustNewDecFromStr("0.0"),
		},
		ReserveBase:  math.ZeroInt(),
		ReserveQuote: math.ZeroInt(),
		Stable:       false,
	}

	suite.k.SetLiquidityPool(suite.ctx, liquidityPool)

	req := &types.QueryLiquidityPoolRequest{
		PoolId: "empty/pool",
	}

	response, err := suite.k.LiquidityPool(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().NotNil(response.Pool)
	suite.Require().Equal(math.ZeroInt(), response.Pool.ReserveBase)
	suite.Require().Equal(math.ZeroInt(), response.Pool.ReserveQuote)
}
