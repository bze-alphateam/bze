package keeper_test

func (suite *IntegrationTestSuite) TestCreatePoolId() {
	base := "abc"
	quote := "xyz"
	newBase, newQuote, poolId := suite.k.CreatePoolId(base, quote)
	suite.Require().Equal(newBase, base)
	suite.Require().Equal(newQuote, quote)
	suite.Require().Contains(poolId, quote)
	suite.Require().Contains(poolId, base)

	newBase, newQuote, poolId = suite.k.CreatePoolId(quote, base)
	suite.Require().Equal(newBase, base)
	suite.Require().Equal(newQuote, quote)
	suite.Require().Contains(poolId, quote)
	suite.Require().Contains(poolId, base)
}
