package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestDistributeAllStakingRewards() {
	initial := types.StakingReward{
		RewardId:         "01",
		PrizeAmount:      "100",
		PrizeDenom:       denomBze,
		StakingDenom:     denomStake,
		Duration:         10,
		Payouts:          0,
		MinStake:         0,
		Lock:             10,
		StakedAmount:     "10",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, initial)

	rewardAmt, ok := sdk.NewIntFromString(initial.PrizeAmount)
	suite.Require().True(ok)

	for i := uint32(0); i < initial.Duration; i++ {
		suite.k.DistributeAllStakingRewards(suite.ctx)
		storage, f := suite.k.GetStakingReward(suite.ctx, initial.RewardId)
		suite.Require().True(f)
		suite.Require().EqualValues(storage.Payouts, i+1)

		staked, ok := sdk.NewIntFromString(storage.StakedAmount)
		suite.Require().True(ok)

		newDistribution := rewardAmt.ToDec().Quo(staked.ToDec())
		distributed, err := sdk.NewDecFromStr(initial.DistributedStake)
		suite.Require().NoError(err)

		distributed = distributed.Add(newDistribution)
		suite.Require().EqualValuesf(distributed.String(), storage.DistributedStake, fmt.Sprintf("values not  equal on %d iteration", i))

		initial.DistributedStake = distributed.String()
		staked = staked.AddRaw(int64(i))
		initial.StakedAmount = staked.String()
		storage.StakedAmount = initial.StakedAmount

		suite.k.SetStakingReward(suite.ctx, storage)
	}

	suite.k.DistributeAllStakingRewards(suite.ctx)
	suite.k.DistributeAllStakingRewards(suite.ctx)
	suite.k.DistributeAllStakingRewards(suite.ctx)
	storage, f := suite.k.GetStakingReward(suite.ctx, initial.RewardId)
	suite.Require().True(f)
	suite.Require().EqualValues(storage.Payouts, initial.Duration)
	suite.Require().EqualValues(storage.DistributedStake, initial.DistributedStake)
}
