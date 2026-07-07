package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
)

func (suite *IntegrationTestSuite) TestIsLucky_ZeroChances_NeverLucky() {
	raffle := &types.Raffle{
		Chances: 0,
	}

	// With 0 chances, should never be lucky
	result := suite.k.IsLucky(suite.ctx, raffle, "anyaddress")
	suite.Require().False(result)
}

func (suite *IntegrationTestSuite) TestIsLucky_MaxChances_AlwaysLucky() {
	raffle := &types.Raffle{
		Chances: 1_000_000, // max range = 1,000,000, so any number < 1,000,000 is lucky
	}

	// With max chances, should always be lucky
	result := suite.k.IsLucky(suite.ctx, raffle, "anyaddress")
	suite.Require().True(result)
}

func (suite *IntegrationTestSuite) TestIsLucky_Deterministic() {
	raffle := &types.Raffle{
		Chances: 500_000,
	}

	result1 := suite.k.IsLucky(suite.ctx, raffle, "addr1")
	result2 := suite.k.IsLucky(suite.ctx, raffle, "addr1")
	suite.Require().Equal(result1, result2, "same context and address should produce same result")
}

func (suite *IntegrationTestSuite) TestIsLucky_DifferentAddresses() {
	raffle := &types.Raffle{
		Chances: 500_000,
	}

	// Different addresses should potentially produce different results
	results := make(map[bool]int)
	for i := 0; i < 20; i++ {
		addr := "testaddr" + string(rune('a'+i))
		result := suite.k.IsLucky(suite.ctx, raffle, addr)
		results[result]++
	}

	// With 50% chance and 20 addresses, we should see both true and false
	suite.Require().Greater(len(results), 1, "different addresses should produce varied results with 50%% chance")
}
