package keeper_test

import (
	"context"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// Amounts must be canonical base-10 integers: the staking reward message servers refuse any other form and
// move no coins.

const amountFormatDenom = "ubze"

// amountFormatCanonicalErr is the text of the refusal of an amount that is not a canonical base-10 integer.
const amountFormatCanonicalErr = "must be a canonical base-10 integer"

// setZeroCreationFees makes reward creation free so no fee path is exercised.
func (suite *IntegrationTestSuite) setZeroCreationFees() {
	params := types.DefaultParams()
	params.CreateStakingRewardFee = sdk.NewCoin(amountFormatDenom, math.ZeroInt())
	params.CreateTradingRewardFee = sdk.NewCoin(amountFormatDenom, math.ZeroInt())
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
}

// expectSends records every coin amount sent from an account into the module.
func (suite *IntegrationTestSuite) expectSends(sent *math.Int) {
	suite.bank.EXPECT().
		SpendableCoins(gomock.Any(), gomock.Any()).
		Return(sdk.NewCoins(sdk.NewCoin(amountFormatDenom, math.NewInt(1_000_000_000)))).
		AnyTimes()
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), types.ModuleName, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ sdk.AccAddress, _ string, amt sdk.Coins) error {
			*sent = sent.Add(amt.AmountOf(amountFormatDenom))
			return nil
		}).
		AnyTimes()
}

func amountFormatReward(id, prizeAmount string) types.StakingReward {
	return types.StakingReward{
		RewardId:         id,
		PrizeAmount:      prizeAmount,
		PrizeDenom:       amountFormatDenom,
		StakingDenom:     amountFormatDenom,
		Duration:         5,
		StakedAmount:     "1",
		DistributedStake: "0",
	}
}

func (suite *IntegrationTestSuite) TestStakingRewardAmountFormat_CreateNonCanonicalAmount() {
	suite.setZeroCreationFees()
	creator := sdk.AccAddress("creator")
	sent := math.ZeroInt()
	suite.bank.EXPECT().HasSupply(gomock.Any(), amountFormatDenom).Return(true).AnyTimes()
	suite.expectSends(&sent)

	counter := suite.k.GetStakingRewardsCounter(suite.ctx)
	_, err := suite.msgServer.CreateStakingReward(suite.ctx, &types.MsgCreateStakingReward{
		Creator:      creator.String(),
		PrizeAmount:  "0100",
		PrizeDenom:   amountFormatDenom,
		StakingDenom: amountFormatDenom,
		Duration:     "5",
		MinStake:     "0",
		Lock:         "0",
	})
	suite.Require().ErrorIs(err, types.ErrInvalidAmount, "amounts must be canonical base-10 integers")
	suite.Require().True(sent.IsZero(), "no coins may move when the request is refused")
	suite.Require().Equal(counter, suite.k.GetStakingRewardsCounter(suite.ctx), "no reward may be created")
}

func (suite *IntegrationTestSuite) TestStakingRewardAmountFormat_DistributeNonCanonicalAmount() {
	creator := sdk.AccAddress("creator")
	sent := math.ZeroInt()
	suite.expectSends(&sent)
	suite.k.SetStakingReward(suite.ctx, amountFormatReward("format-distribute", "1000"))

	_, err := suite.msgServer.DistributeStakingRewards(suite.ctx, &types.MsgDistributeStakingRewards{
		Creator:  creator.String(),
		RewardId: "format-distribute",
		Amount:   "0100",
	})
	suite.Require().ErrorIs(err, types.ErrInvalidAmount, "amounts must be canonical base-10 integers")
	suite.Require().True(sent.IsZero(), "no coins may move when the request is refused")

	after, found := suite.k.GetStakingReward(suite.ctx, "format-distribute")
	suite.Require().True(found)
	suite.Require().Equal("0", after.DistributedStake, "the reward must be left as it was")
}

func (suite *IntegrationTestSuite) TestStakingRewardAmountFormat_DistributeNonCanonicalForms() {
	creator := sdk.AccAddress("creator")
	suite.k.SetStakingReward(suite.ctx, amountFormatReward("format-distribute-forms", "1000"))
	sent := math.ZeroInt()
	suite.expectSends(&sent)

	for _, amount := range []string{"0x10", "0o10", "0b10", "1_0", "+10"} {
		suite.Run(amount, func() {
			sent = math.ZeroInt()
			_, err := suite.msgServer.DistributeStakingRewards(suite.ctx, &types.MsgDistributeStakingRewards{
				Creator:  creator.String(),
				RewardId: "format-distribute-forms",
				Amount:   amount,
			})
			suite.Require().ErrorIs(err, types.ErrInvalidAmount, "amount %q must be refused", amount)
			suite.Require().True(sent.IsZero(), "amount %q: no coins may move", amount)

			after, found := suite.k.GetStakingReward(suite.ctx, "format-distribute-forms")
			suite.Require().True(found)
			suite.Require().Equal("0", after.DistributedStake, "amount %q: the reward must be left as it was", amount)
		})
	}
}

func (suite *IntegrationTestSuite) TestStakingRewardAmountFormat_UpdateStoredNonCanonicalAmount() {
	creator := sdk.AccAddress("creator")
	sent := math.ZeroInt()
	suite.expectSends(&sent)
	suite.k.SetStakingReward(suite.ctx, amountFormatReward("format-update", "0100"))

	_, err := suite.msgServer.UpdateStakingReward(suite.ctx, &types.MsgUpdateStakingReward{
		Creator:  creator.String(),
		RewardId: "format-update",
		Duration: "3",
	})
	suite.Require().ErrorContains(err, amountFormatCanonicalErr, "amounts must be canonical base-10 integers")
	suite.Require().True(sent.IsZero(), "no coins may move when the request is refused")

	after, found := suite.k.GetStakingReward(suite.ctx, "format-update")
	suite.Require().True(found)
	suite.Require().Equal(amountFormatReward("format-update", "0100"), after, "the reward must be left as it was")
}

func (suite *IntegrationTestSuite) TestStakingRewardAmountFormat_JoinNonCanonicalAmount() {
	joiner := sdk.AccAddress("joiner")
	sent := math.ZeroInt()
	suite.expectSends(&sent)
	suite.k.SetStakingReward(suite.ctx, amountFormatReward("format-join", "1000"))

	_, err := suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{
		Creator:  joiner.String(),
		RewardId: "format-join",
		Amount:   "0100",
	})
	suite.Require().ErrorContains(err, amountFormatCanonicalErr, "amounts must be canonical base-10 integers")
	suite.Require().True(sent.IsZero(), "no coins may move when the request is refused")

	_, found := suite.k.GetStakingRewardParticipant(suite.ctx, joiner.String(), "format-join")
	suite.Require().False(found, "no participant may be written")

	after, found := suite.k.GetStakingReward(suite.ctx, "format-join")
	suite.Require().True(found)
	suite.Require().Equal(amountFormatReward("format-join", "1000"), after, "the reward must be left as it was")
}

// The daily distribution credits only a stored prize amount written in canonical base-10 form: a
// reward holding any other form is skipped for the day while its canonical neighbours are paid.
func (suite *IntegrationTestSuite) TestStakingRewardAmountFormat_DailyDistributionStoredNonCanonicalAmount() {
	staker := sdk.AccAddress("staker")
	stored := amountFormatReward("format-daily-a", "0100")
	stored.StakedAmount = "1000"
	suite.k.SetStakingReward(suite.ctx, stored)
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address:  staker.String(),
		RewardId: stored.RewardId,
		Amount:   "1000",
		JoinedAt: "0",
	})

	canonical := amountFormatReward("format-daily-b", "100")
	canonical.StakedAmount = "1000"
	suite.k.SetStakingReward(suite.ctx, canonical)

	suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
	suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)

	after, found := suite.k.GetStakingReward(suite.ctx, stored.RewardId)
	suite.Require().True(found)
	suite.Require().Equal("0", after.DistributedStake, "nothing may be credited from a non-canonical stored amount")
	suite.Require().Equal(uint32(0), after.Payouts, "the payout must not be counted")

	paid, found := suite.k.GetStakingReward(suite.ctx, canonical.RewardId)
	suite.Require().True(found)
	suite.Require().Equal(uint32(1), paid.Payouts, "a canonical reward in the same batch is paid")
	suite.Require().Equal(math.LegacyMustNewDecFromStr("0.1").String(), paid.DistributedStake)

	_, found = suite.k.GetStakingRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found, "the distribution pass must complete")
}
