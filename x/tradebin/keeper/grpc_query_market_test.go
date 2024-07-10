package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestMarketAll_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.MarketAll(goCtx, nil)
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestMarketAll_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	fakeDenom1 := types.Market{
		Base:    "fake1",
		Quote:   denomBze,
		Creator: "fake_addr",
	}
	fakeDenom2 := types.Market{
		Base:    denomBze,
		Quote:   "fake2",
		Creator: "fake_addr",
	}
	suite.k.SetMarket(suite.ctx, market)
	suite.k.SetMarket(suite.ctx, fakeDenom1)
	suite.k.SetMarket(suite.ctx, fakeDenom2)

	res, err := suite.k.MarketAll(goCtx, &types.QueryAllMarketRequest{})
	suite.Require().Nil(err)
	suite.Require().NotEmpty(res.Market)
	suite.Require().Equal(len(res.Market), 3)
}

func (suite *IntegrationTestSuite) TestMarket_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.Market(goCtx, nil)
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestMarket_InvalidArguments() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.Market(goCtx, &types.QueryGetMarketRequest{Base: denomStake, Quote: ""})
	suite.Require().NotNil(err)

	_, err = suite.k.Market(goCtx, &types.QueryGetMarketRequest{Base: "", Quote: ""})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestMarket_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	fakeDenom1 := types.Market{
		Base:    "fake1",
		Quote:   denomBze,
		Creator: "fake_addr",
	}
	suite.k.SetMarket(suite.ctx, market)
	suite.k.SetMarket(suite.ctx, fakeDenom1)

	res, err := suite.k.Market(goCtx, &types.QueryGetMarketRequest{Base: denomStake, Quote: denomBze})
	suite.Require().Nil(err)
	suite.Require().Equal(res.Market, market)

	res, err = suite.k.Market(goCtx, &types.QueryGetMarketRequest{Base: "fake1", Quote: denomBze})
	suite.Require().Nil(err)
	suite.Require().Equal(res.Market, fakeDenom1)
}
