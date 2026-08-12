package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestStoreDenomRewardPrize_SetGetRemove() {
	prize := types.DenomRewardPrize{
		StakingDenom:          "ubze",
		PrizeDenom:            "uprize",
		DistributedStake:      "0",
		LastDistributionEpoch: 10,
	}

	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().False(found)

	suite.k.SetDenomRewardPrize(suite.ctx, prize)
	got, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().True(found)
	suite.Require().Equal(prize.DistributedStake, got.DistributedStake)
	suite.Require().Equal(prize.LastDistributionEpoch, got.LastDistributionEpoch)

	suite.k.RemoveDenomRewardPrize(suite.ctx, "ubze", "uprize")
	_, found = suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreDenomRewardPrize_IterateAndCount_PerDenomIsolation() {
	// ubze has prizes p1, p2, p3; other has a single prize p1 (same prize denom, different DR)
	for _, p := range []string{"p1", "p2", "p3"} {
		suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: p, DistributedStake: "0"})
	}
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "other", PrizeDenom: "p1", DistributedStake: "0"})

	// counts are isolated per staking denom
	suite.Require().Equal(uint32(3), suite.k.CountDenomRewardPrizes(suite.ctx, "ubze"))
	suite.Require().Equal(uint32(1), suite.k.CountDenomRewardPrizes(suite.ctx, "other"))
	suite.Require().Equal(uint32(0), suite.k.CountDenomRewardPrizes(suite.ctx, "missing"))

	// iteration only yields the requested denom's prizes, in lexicographic prize order
	var seen []string
	suite.k.IterateDenomRewardPrizes(suite.ctx, "ubze", func(_ sdk.Context, prize types.DenomRewardPrize) bool {
		suite.Require().Equal("ubze", prize.StakingDenom)
		seen = append(seen, prize.PrizeDenom)
		return false
	})
	suite.Require().Equal([]string{"p1", "p2", "p3"}, seen)

	// GetAll (across denoms) returns every prize
	suite.Require().Len(suite.k.GetAllDenomRewardPrize(suite.ctx), 4)

	// early stop
	var count int
	suite.k.IterateDenomRewardPrizes(suite.ctx, "ubze", func(_ sdk.Context, _ types.DenomRewardPrize) bool {
		count++
		return true
	})
	suite.Require().Equal(1, count)
}
