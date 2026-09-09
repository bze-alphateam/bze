package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- smallZeroFillId tests ---

func (suite *IntegrationTestSuite) TestKeeper_SmallZeroFillId_Zero() {
	// Access via creating a staking reward and checking the ID format
	counter := suite.k.GetStakingRewardsCounter(suite.ctx)
	suite.Require().Equal(uint64(0), counter)

	// Set up a staking reward to observe the generated ID
	creator := sdk.AccAddress("creator")

	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(2)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100000)))).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000)))).
		Return(nil).
		Times(1)

	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(0)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(0)),
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	msg := &types.MsgCreateStakingReward{
		Creator:      creator.String(),
		PrizeAmount:  "1000",
		PrizeDenom:   "ubze",
		StakingDenom: "ubze",
		Duration:     "1",
		MinStake:     "100",
		Lock:         "0",
	}

	response, err := suite.msgServer.CreateStakingReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	// ID should be zero-padded to 12 digits
	suite.Require().Equal("000000000000", response.RewardId)
	suite.Require().Len(response.RewardId, 12)
}

// --- getAmountToCapture tests ---
// Tested indirectly through DistributeStakingRewards

func (suite *IntegrationTestSuite) TestKeeper_GetAmountToCapture_InvalidStakedAmount() {
	creator := sdk.AccAddress("creator")

	// distributeStakingRewards parses StakedAmount - make it invalid
	stakingReward := types.StakingReward{
		RewardId:         "capture-invalid-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "not-a-number", // Invalid staked amount
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100000)))).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	msg := &types.MsgDistributeStakingRewards{
		Creator:  creator.String(),
		RewardId: "capture-invalid-reward",
		Amount:   "500",
	}

	response, err := suite.msgServer.DistributeStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "could not transform staked amount")
}

// --- distributeStakingRewards (math function) tests ---
// Tested through ProcessStakingRewardsDistributionQueue with different staked amounts

func (suite *IntegrationTestSuite) TestDistributeStakingRewards_MathPrecision() {
	// Set up a staking reward with a large staked amount to test precision
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "precision-reward",
		PrizeAmount:      "1",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          0,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "3", // 1/3 = 0.333...
		DistributedStake: "0",
	})

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	reward, found := suite.k.GetStakingReward(suite.ctx, "precision-reward")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), reward.Payouts)

	// DistributedStake should be 1/3 = 0.333...
	ds, err := math.LegacyNewDecFromStr(reward.DistributedStake)
	suite.Require().NoError(err)
	suite.Require().True(ds.IsPositive())

	expected := math.LegacyNewDec(1).Quo(math.LegacyNewDec(3))
	suite.Require().Equal(expected.String(), ds.String())
}

func (suite *IntegrationTestSuite) TestDistributeStakingRewards_LargeValues() {
	// Test with large staked amounts to ensure no overflow
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "large-reward",
		PrizeAmount:      "1000000000000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          0,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "999999999999",
		DistributedStake: "0",
	})

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	reward, found := suite.k.GetStakingReward(suite.ctx, "large-reward")
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), reward.Payouts)

	ds, err := math.LegacyNewDecFromStr(reward.DistributedStake)
	suite.Require().NoError(err)
	suite.Require().True(ds.IsPositive())
}

func (suite *IntegrationTestSuite) TestDistributeStakingRewards_AccumulatesOverMultiplePayouts() {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "accumulate-reward",
		PrizeAmount:      "100",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         3,
		Payouts:          0,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "1000",
		DistributedStake: "0",
	})

	// Process 3 distributions
	for i := 0; i < 3; i++ {
		suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
		suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)
	}

	reward, found := suite.k.GetStakingReward(suite.ctx, "accumulate-reward")
	suite.Require().True(found)
	suite.Require().Equal(uint32(3), reward.Payouts)

	// DistributedStake = 3 * (100/1000) = 0.3
	ds, err := math.LegacyNewDecFromStr(reward.DistributedStake)
	suite.Require().NoError(err)
	expected := math.LegacyNewDecWithPrec(3, 1) // 0.3
	suite.Require().Equal(expected.String(), ds.String())
}
