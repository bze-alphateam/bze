package keeper_test

func (suite *IntegrationTestSuite) TestMarket() {
	//create market
	suite.k.SetMarket(suite.ctx, market)

	//check it exists in all indexes
	found, ok := suite.k.GetMarket(suite.ctx, market.Base, market.Quote)
	suite.Require().True(ok)
	suite.Equal(found.Base, market.Base)
	suite.Equal(found.Quote, market.Quote)

	found, ok = suite.k.GetMarketAlias(suite.ctx, market.Quote, market.Base)
	suite.Require().True(ok)
	suite.Equal(found.Base, market.Base)
	suite.Equal(found.Quote, market.Quote)

	found, ok = suite.k.GetMarketById(suite.ctx, getMarketId())
	suite.Require().True(ok)
	suite.Equal(found.Base, market.Base)
	suite.Equal(found.Quote, market.Quote)

	list := suite.k.GetAllAssetMarkets(suite.ctx, market.Base)
	suite.Require().NotEmpty(list)

	list = suite.k.GetAllAssetMarketAliases(suite.ctx, market.Quote)
	suite.Require().NotEmpty(list)
}
