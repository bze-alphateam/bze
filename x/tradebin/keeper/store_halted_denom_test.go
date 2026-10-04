package keeper_test

const (
	haltedIbcDenom     = "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"
	haltedFactoryDenom = "factory/bze1m33n82r5x3eyjmjtwjkl82zzdlrnv8pevd8u9r/spam"
)

func (suite *IntegrationTestSuite) TestHaltedDenomStore_SetHasRemove() {
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomHalted), "nothing is halted by default")
	suite.Require().Empty(suite.k.GetAllHaltedDenoms(suite.ctx))

	suite.k.SetHaltedDenom(suite.ctx, denomHalted)
	suite.Require().True(suite.k.IsDenomHalted(suite.ctx, denomHalted))
	suite.Require().Equal([]string{denomHalted}, suite.k.GetAllHaltedDenoms(suite.ctx))

	// idempotent
	suite.k.SetHaltedDenom(suite.ctx, denomHalted)
	suite.Require().Equal([]string{denomHalted}, suite.k.GetAllHaltedDenoms(suite.ctx))

	suite.k.RemoveHaltedDenom(suite.ctx, denomHalted)
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomHalted))
	suite.Require().Empty(suite.k.GetAllHaltedDenoms(suite.ctx))

	// removing a denom that is not halted is a no-op
	suite.k.RemoveHaltedDenom(suite.ctx, denomHalted)
	suite.Require().Empty(suite.k.GetAllHaltedDenoms(suite.ctx))
}

// The key is the denom itself: a prefix, an extension, a market id or a pool id containing the denom
// never matches.
func (suite *IntegrationTestSuite) TestHaltedDenomStore_ExactMatchOnly() {
	suite.k.SetHaltedDenom(suite.ctx, denomHalted)

	suite.Require().True(suite.k.IsDenomHalted(suite.ctx, denomHalted))
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomHalted[:len(denomHalted)-1]))
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomHalted+"x"))
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomStake+"/"+denomHalted))
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomStake+"_"+denomHalted))
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, ""))
}

// GetAll walks the prefix store, so denoms come back complete and in byte order whatever the insertion
// order; ibc and factory denoms (with "/" in them) are ordinary keys.
func (suite *IntegrationTestSuite) TestHaltedDenomStore_GetAllIsSortedAndComplete() {
	for _, d := range []string{denomHalted, haltedIbcDenom, haltedFactoryDenom} {
		suite.k.SetHaltedDenom(suite.ctx, d)
	}

	suite.Require().Equal([]string{haltedFactoryDenom, haltedIbcDenom, denomHalted}, suite.k.GetAllHaltedDenoms(suite.ctx))
	for _, d := range []string{denomHalted, haltedIbcDenom, haltedFactoryDenom} {
		suite.Require().True(suite.k.IsDenomHalted(suite.ctx, d), d)
	}
}
