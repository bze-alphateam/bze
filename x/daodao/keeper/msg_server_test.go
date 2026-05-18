package keeper_test

// Smoke test: the keeper, msg server and context are wired correctly by
// SetupTest. Covered by the suite's own setup-validation path; this file
// is intentionally tiny — every real test lives elsewhere as an
// IntegrationTestSuite method.

func (suite *IntegrationTestSuite) TestMsgServer_Setup() {
	suite.Require().NotNil(suite.msgServer)
	suite.Require().NotNil(suite.k)
}
