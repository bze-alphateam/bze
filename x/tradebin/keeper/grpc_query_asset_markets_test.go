package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestAssetMarkets_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.AssetMarkets(goCtx, nil)
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestAssetMarkets_InvalidRequest_InvalidAsset() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.AssetMarkets(goCtx, &types.QueryAssetMarketsRequest{Asset: ""})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestAssetMarkets_OneMarketAsBaseDenom_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	suite.k.SetMarket(suite.ctx, market)

	res, err := suite.k.AssetMarkets(goCtx, &types.QueryAssetMarketsRequest{Asset: denomStake})
	suite.Require().Nil(err)

	suite.Require().Empty(res.Quote)
	suite.Require().NotEmpty(res.Base)
	suite.Require().Equal(len(res.Base), 1)
	suite.Require().Equal(res.Base[0], market)
}

func (suite *IntegrationTestSuite) TestAssetMarkets_OneMarketAsQuoteDenom_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	suite.k.SetMarket(suite.ctx, market)

	res, err := suite.k.AssetMarkets(goCtx, &types.QueryAssetMarketsRequest{Asset: denomBze})
	suite.Require().Nil(err)

	suite.Require().Empty(res.Base)
	suite.Require().NotEmpty(res.Quote)
	suite.Require().Equal(len(res.Quote), 1)
	suite.Require().Equal(res.Quote[0], market)
}

func (suite *IntegrationTestSuite) TestAssetMarkets_MoreMarket_Success() {
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

	res, err := suite.k.AssetMarkets(goCtx, &types.QueryAssetMarketsRequest{Asset: denomBze})
	suite.Require().Nil(err)

	suite.Require().Equal(len(res.Base), 1)
	suite.Require().Equal(res.Base[0], fakeDenom2)

	suite.Require().Equal(len(res.Quote), 2)
	suite.Require().Equal(res.Quote[0], fakeDenom1)
	suite.Require().Equal(res.Quote[1], market)
}
