package keeper_test

func (suite *IntegrationTestSuite) TestCounter() {
	key := []byte("test")
	initial := suite.k.GetCounter(suite.ctx, key)
	suite.Require().Equal(initial, uint64(0))

	suite.k.SetCounter(suite.ctx, key, 2)

	counter := suite.k.GetCounter(suite.ctx, key)
	suite.Require().Equal(counter, uint64(2))
}

func (suite *IntegrationTestSuite) TestStakingRewardsCounter() {
	initial := suite.k.GetStakingRewardsCounter(suite.ctx)
	suite.Require().Equal(initial, uint64(0))

	suite.k.SetStakingRewardsCounter(suite.ctx, uint64(2323))

	counter := suite.k.GetStakingRewardsCounter(suite.ctx)
	suite.Require().Equal(counter, uint64(2323))
}

func (suite *IntegrationTestSuite) TestTradingRewardsCounter() {
	initial := suite.k.GetTradingRewardsCounter(suite.ctx)
	suite.Require().Equal(initial, uint64(0))

	suite.k.SetTradingRewardsCounter(suite.ctx, uint64(2323))

	counter := suite.k.GetTradingRewardsCounter(suite.ctx)
	suite.Require().Equal(counter, uint64(2323))
}
