package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestStoreDenomReward_SetGetHasRemove() {
	dr := types.DenomReward{
		StakingDenom: "ubze",
		Lock:         7,
		MinStake:     0,
		StakedAmount: "1000",
	}

	// not present initially
	suite.Require().False(suite.k.HasDenomReward(suite.ctx, "ubze"))
	_, found := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().False(found)

	// set + get
	suite.k.SetDenomReward(suite.ctx, dr)
	suite.Require().True(suite.k.HasDenomReward(suite.ctx, "ubze"))

	got, found := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().True(found)
	suite.Require().Equal(dr.StakingDenom, got.StakingDenom)
	suite.Require().Equal(dr.Lock, got.Lock)
	suite.Require().Equal(dr.MinStake, got.MinStake)
	suite.Require().Equal(dr.StakedAmount, got.StakedAmount)

	// remove
	suite.k.RemoveDenomReward(suite.ctx, "ubze")
	suite.Require().False(suite.k.HasDenomReward(suite.ctx, "ubze"))
	_, found = suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreDenomReward_GetAllAndIterate() {
	denoms := []string{"aaa", "bbb", "ccc"}
	for _, d := range denoms {
		suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: d, StakedAmount: "0"})
	}

	all := suite.k.GetAllDenomReward(suite.ctx)
	suite.Require().Len(all, 3)

	// iteration is deterministic (lexicographic by staking denom)
	var seen []string
	suite.k.IterateAllDenomRewards(suite.ctx, func(_ sdk.Context, dr types.DenomReward) bool {
		seen = append(seen, dr.StakingDenom)
		return false
	})
	suite.Require().Equal([]string{"aaa", "bbb", "ccc"}, seen)

	// early stop
	var count int
	suite.k.IterateAllDenomRewards(suite.ctx, func(_ sdk.Context, _ types.DenomReward) bool {
		count++
		return true
	})
	suite.Require().Equal(1, count)
}
