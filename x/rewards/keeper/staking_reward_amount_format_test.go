package keeper_test

import (
	"context"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// Amounts must be canonical base-10 integers: whatever is taken from the payer for a staking reward
// has to match what is later credited to the reward's stakers.

const amountFormatDenom = "ubze"

// setZeroCreationFees makes reward creation free so no fee path is exercised.
func (suite *IntegrationTestSuite) setZeroCreationFees() {
	params := types.DefaultParams()
	params.CreateStakingRewardFee = sdk.NewCoin(amountFormatDenom, math.ZeroInt())
	params.CreateTradingRewardFee = sdk.NewCoin(amountFormatDenom, math.ZeroInt())
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
}

// expectCapture records every coin amount moved from an account into the module.
func (suite *IntegrationTestSuite) expectCapture(captured *math.Int) {
	suite.bank.EXPECT().
		SpendableCoins(gomock.Any(), gomock.Any()).
		Return(sdk.NewCoins(sdk.NewCoin(amountFormatDenom, math.NewInt(1_000_000_000)))).
		AnyTimes()
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), types.ModuleName, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ sdk.AccAddress, _ string, amt sdk.Coins) error {
			*captured = captured.Add(amt.AmountOf(amountFormatDenom))
			return nil
		}).
		AnyTimes()
}

// creditedPerPayout runs one daily distribution for the given reward (with a total stake of 1, so the
// accumulator delta equals the amount credited) and returns the credited amount.
func (suite *IntegrationTestSuite) creditedPerPayout(rewardId string) math.LegacyDec {
	sr, found := suite.k.GetStakingReward(suite.ctx, rewardId)
	suite.Require().True(found)
	sr.StakedAmount = "1"
	sr.DistributedStake = "0"
	sr.Payouts = 0
	suite.k.SetStakingReward(suite.ctx, sr)

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	after, found := suite.k.GetStakingReward(suite.ctx, rewardId)
	suite.Require().True(found)
	credited, err := math.LegacyNewDecFromStr(after.DistributedStake)
	suite.Require().NoError(err)

	return credited
}

func (suite *IntegrationTestSuite) TestStakingRewardAmountFormat_CreateLeadingZeroPrize() {
	suite.setZeroCreationFees()
	creator := sdk.AccAddress("creator")
	captured := math.ZeroInt()
	suite.bank.EXPECT().HasSupply(gomock.Any(), amountFormatDenom).Return(true).AnyTimes()
	suite.expectCapture(&captured)

	const duration = 5
	res, err := suite.msgServer.CreateStakingReward(suite.ctx, &types.MsgCreateStakingReward{
		Creator:      creator.String(),
		PrizeAmount:  "0100",
		PrizeDenom:   amountFormatDenom,
		StakingDenom: amountFormatDenom,
		Duration:     "5",
		MinStake:     "0",
		Lock:         "0",
	})
	if err != nil {
		suite.Require().ErrorIs(err, types.ErrInvalidAmount)
		suite.Require().True(captured.IsZero(), "nothing may be captured when the request is refused")
		return
	}

	credited := suite.creditedPerPayout(res.RewardId)
	suite.Require().Equal(
		math.LegacyNewDecFromInt(captured).String(),
		credited.MulInt64(duration).String(),
		"captured from creator (left) must equal credited per payout x duration (right)",
	)
}

func (suite *IntegrationTestSuite) TestStakingRewardAmountFormat_DistributeLeadingZeroAmount() {
	creator := sdk.AccAddress("creator")
	captured := math.ZeroInt()
	suite.expectCapture(&captured)

	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "format-distribute",
		PrizeAmount:      "1000",
		PrizeDenom:       amountFormatDenom,
		StakingDenom:     amountFormatDenom,
		Duration:         5,
		StakedAmount:     "1",
		DistributedStake: "0",
	})

	_, err := suite.msgServer.DistributeStakingRewards(suite.ctx, &types.MsgDistributeStakingRewards{
		Creator:  creator.String(),
		RewardId: "format-distribute",
		Amount:   "0100",
	})
	if err != nil {
		suite.Require().ErrorIs(err, types.ErrInvalidAmount)
		suite.Require().True(captured.IsZero(), "nothing may be captured when the request is refused")
		return
	}

	after, found := suite.k.GetStakingReward(suite.ctx, "format-distribute")
	suite.Require().True(found)
	credited, err := math.LegacyNewDecFromStr(after.DistributedStake)
	suite.Require().NoError(err)
	suite.Require().Equal(
		math.LegacyNewDecFromInt(captured).String(),
		credited.String(),
		"captured from distributor (left) must equal credited to stakers (right)",
	)
}

func (suite *IntegrationTestSuite) TestStakingRewardAmountFormat_DistributeRejectsNonDecimal() {
	creator := sdk.AccAddress("creator")
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "format-distribute-nondecimal",
		PrizeAmount:      "1000",
		PrizeDenom:       amountFormatDenom,
		StakingDenom:     amountFormatDenom,
		Duration:         5,
		StakedAmount:     "1",
		DistributedStake: "0",
	})

	captured := math.ZeroInt()
	suite.expectCapture(&captured)

	for _, amount := range []string{"0x10", "0o10", "0b10", "1_0", "+10"} {
		suite.Run(amount, func() {
			captured = math.ZeroInt()
			_, err := suite.msgServer.DistributeStakingRewards(suite.ctx, &types.MsgDistributeStakingRewards{
				Creator:  creator.String(),
				RewardId: "format-distribute-nondecimal",
				Amount:   amount,
			})
			suite.Require().Error(err, "amount %q must be refused", amount)
			suite.Require().True(captured.IsZero(), "amount %q: nothing may be captured", amount)
		})
	}
}

func (suite *IntegrationTestSuite) TestStakingRewardAmountFormat_UpdateStoredLeadingZeroPrize() {
	creator := sdk.AccAddress("creator")
	captured := math.ZeroInt()
	suite.expectCapture(&captured)

	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "format-update",
		PrizeAmount:      "0100",
		PrizeDenom:       amountFormatDenom,
		StakingDenom:     amountFormatDenom,
		Duration:         5,
		StakedAmount:     "1",
		DistributedStake: "0",
	})

	const extraDays = 3
	_, err := suite.msgServer.UpdateStakingReward(suite.ctx, &types.MsgUpdateStakingReward{
		Creator:  creator.String(),
		RewardId: "format-update",
		Duration: "3",
	})
	if err != nil {
		suite.Require().True(captured.IsZero(), "nothing may be captured when the request is refused")
		return
	}

	credited := suite.creditedPerPayout("format-update")
	suite.Require().Equal(
		math.LegacyNewDecFromInt(captured).String(),
		credited.MulInt64(extraDays).String(),
		"captured for the extension (left) must equal credited per payout x extra days (right)",
	)
}
