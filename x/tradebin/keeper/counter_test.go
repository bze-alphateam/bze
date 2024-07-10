package keeper_test

func (suite *IntegrationTestSuite) TestQueueMessageCounter() {
	initial := suite.k.GetQueueMessageCounter(suite.ctx)
	suite.Require().Equal(initial, uint64(0))

	nextValue := uint64(100)
	suite.k.SetQueueMessageCounter(suite.ctx, nextValue)

	end := suite.k.GetQueueMessageCounter(suite.ctx)
	suite.Require().Equal(end, nextValue)
}

func (suite *IntegrationTestSuite) TestOrderCounter() {
	initial := suite.k.GetOrderCounter(suite.ctx)
	suite.Require().Equal(initial, uint64(0))

	nextValue := uint64(100)
	suite.k.SetOrderCounter(suite.ctx, nextValue)

	end := suite.k.GetOrderCounter(suite.ctx)
	suite.Require().Equal(end, nextValue)
}
