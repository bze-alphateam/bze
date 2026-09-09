package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// --- JoinStaking additional tests ---

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_JoinStakingNilRequest() {
	response, err := suite.msgServer.JoinStaking(suite.ctx, nil)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(sdkerrors.ErrInvalidRequest, err)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_JoinStakingInvalidCreator() {
	msg := &types.MsgJoinStaking{
		Creator:  "invalid-address",
		RewardId: "some-reward",
		Amount:   "500",
	}

	response, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_JoinStakingRewardNotFound() {
	creator := sdk.AccAddress("creator")

	msg := &types.MsgJoinStaking{
		Creator:  creator.String(),
		RewardId: "non-existent-reward",
		Amount:   "500",
	}

	response, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "reward with provided id not found")
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_JoinStakingInsufficientBalance() {
	creator := sdk.AccAddress("creator")

	stakingReward := types.StakingReward{
		RewardId:         "join-insufficient-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          0,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "0",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(10)), // Not enough
		)).
		Times(1)

	msg := &types.MsgJoinStaking{
		Creator:  creator.String(),
		RewardId: "join-insufficient-reward",
		Amount:   "500",
	}

	response, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "user balance is too low")
}

// --- ExitStaking additional tests ---

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_ExitStakingNilRequest() {
	response, err := suite.msgServer.ExitStaking(suite.ctx, nil)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(sdkerrors.ErrInvalidRequest, err)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_ExitStakingRewardNotFound() {
	creator := sdk.AccAddress("creator")

	msg := &types.MsgExitStaking{
		Creator:  creator.String(),
		RewardId: "non-existent-reward",
	}

	response, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "reward with provided id not found")
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_ExitStakingNotAParticipant() {
	creator := sdk.AccAddress("creator")

	stakingReward := types.StakingReward{
		RewardId:         "exit-notparticipant-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "500",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	msg := &types.MsgExitStaking{
		Creator:  creator.String(),
		RewardId: "exit-notparticipant-reward",
	}

	response, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "you are not a participant in this staking reward")
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_ExitStakingFinalizesReward() {
	creator := sdk.AccAddress("creator")

	// Staking reward where all payouts are done and this is the last participant
	stakingReward := types.StakingReward{
		RewardId:         "exit-finalize-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          5, // All payouts done
		MinStake:         100,
		Lock:             0,
		StakedAmount:     "500",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	participant := types.StakingRewardParticipant{
		Address:  creator.String(),
		RewardId: "exit-finalize-reward",
		Amount:   "500",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			creator,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500))),
		).
		Return(nil).
		Times(1)

	msg := &types.MsgExitStaking{
		Creator:  creator.String(),
		RewardId: "exit-finalize-reward",
	}

	response, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify the staking reward was removed since all payouts done and no stakers left
	_, found := suite.k.GetStakingReward(suite.ctx, "exit-finalize-reward")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_ExitStakingWithPendingRewards() {
	creator := sdk.AccAddress("creator")

	// Staking reward with distributed stake so participant has pending rewards
	stakingReward := types.StakingReward{
		RewardId:         "exit-pending-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             0,
		StakedAmount:     "1000",
		DistributedStake: "2.0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	participant := types.StakingRewardParticipant{
		Address:  creator.String(),
		RewardId: "exit-pending-reward",
		Amount:   "500",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	// Mock claim pending rewards: 500 * (2.0 - 0) = 1000
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			creator,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000))),
		).
		Return(nil).
		Times(1)

	// Mock unlock staked amount
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			creator,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500))),
		).
		Return(nil).
		Times(1)

	msg := &types.MsgExitStaking{
		Creator:  creator.String(),
		RewardId: "exit-pending-reward",
	}

	response, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify participant was removed
	_, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "exit-pending-reward")
	suite.Require().False(found)

	// Verify staked amount was updated
	updatedReward, found := suite.k.GetStakingReward(suite.ctx, "exit-pending-reward")
	suite.Require().True(found)
	suite.Require().Equal("500", updatedReward.StakedAmount)
}

// --- ClaimStakingRewards additional tests ---

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_ClaimStakingRewardsNilRequest() {
	response, err := suite.msgServer.ClaimStakingRewards(suite.ctx, nil)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(sdkerrors.ErrInvalidRequest, err)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_ClaimStakingRewardsRewardNotFound() {
	creator := sdk.AccAddress("creator")

	msg := &types.MsgClaimStakingRewards{
		Creator:  creator.String(),
		RewardId: "non-existent-reward",
	}

	response, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "reward with provided id not found")
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_ClaimStakingRewardsNotAParticipant() {
	creator := sdk.AccAddress("creator")

	stakingReward := types.StakingReward{
		RewardId:         "claim-notparticipant-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "500",
		DistributedStake: "1.0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	msg := &types.MsgClaimStakingRewards{
		Creator:  creator.String(),
		RewardId: "claim-notparticipant-reward",
	}

	response, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "you are not a participant in this staking reward")
}
