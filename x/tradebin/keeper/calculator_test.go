package keeper_test

import "github.com/bze-alphateam/bze/x/tradebin/keeper"

func (suite *IntegrationTestSuite) TestCalculateMinAmount() {
	minAmount := keeper.CalculateMinAmount("0.9")
	suite.Require().Equal(minAmount.Int64(), int64(4))

	minAmount = keeper.CalculateMinAmount("0.09")
	suite.Require().Equal(minAmount.Int64(), int64(24))

	minAmount = keeper.CalculateMinAmount("0.009")
	suite.Require().Equal(minAmount.Int64(), int64(224))

	minAmount = keeper.CalculateMinAmount("0.0009")
	suite.Require().Equal(minAmount.Int64(), int64(2224))

	minAmount = keeper.CalculateMinAmount("0.0143331")
	suite.Require().Equal(minAmount.Int64(), int64(140))

	minAmount = keeper.CalculateMinAmount("0.09000123")
	suite.Require().Equal(minAmount.Int64(), int64(24))

	minAmount = keeper.CalculateMinAmount("0.0497999")
	suite.Require().Equal(minAmount.Int64(), int64(42))
}
