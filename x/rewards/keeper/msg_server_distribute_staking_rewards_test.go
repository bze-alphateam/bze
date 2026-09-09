package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_DistributeStakingRewardsNilRequest() {
	response, err := suite.msgServer.DistributeStakingRewards(suite.ctx, nil)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(sdkerrors.ErrInvalidRequest, err)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_DistributeStakingRewardsInvalidCreator() {
	msg := &types.MsgDistributeStakingRewards{
		Creator:  "invalid-address",
		RewardId: "some-reward",
		Amount:   "500",
	}

	response, err := suite.msgServer.DistributeStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_DistributeStakingRewardsInvalidAmount() {
	creator := sdk.AccAddress("creator")

	msg := &types.MsgDistributeStakingRewards{
		Creator:  creator.String(),
		RewardId: "some-reward",
		Amount:   "not-a-number",
	}

	response, err := suite.msgServer.DistributeStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "could not convert order amount")
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_DistributeStakingRewardsZeroAmount() {
	creator := sdk.AccAddress("creator")

	msg := &types.MsgDistributeStakingRewards{
		Creator:  creator.String(),
		RewardId: "some-reward",
		Amount:   "0",
	}

	response, err := suite.msgServer.DistributeStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "amount should be greater than 0")
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_DistributeStakingRewardsNegativeAmount() {
	creator := sdk.AccAddress("creator")

	msg := &types.MsgDistributeStakingRewards{
		Creator:  creator.String(),
		RewardId: "some-reward",
		Amount:   "-100",
	}

	response, err := suite.msgServer.DistributeStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "amount should be greater than 0")
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_DistributeStakingRewardsNotFound() {
	creator := sdk.AccAddress("creator")

	msg := &types.MsgDistributeStakingRewards{
		Creator:  creator.String(),
		RewardId: "non-existent-reward",
		Amount:   "500",
	}

	response, err := suite.msgServer.DistributeStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "staking reward not found")
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_DistributeStakingRewardsInsufficientFunds() {
	creator := sdk.AccAddress("creator")

	stakingReward := types.StakingReward{
		RewardId:         "dist-insufficient-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "1000",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(10)), // Not enough
		)).
		Times(1)

	msg := &types.MsgDistributeStakingRewards{
		Creator:  creator.String(),
		RewardId: "dist-insufficient-reward",
		Amount:   "500",
	}

	response, err := suite.msgServer.DistributeStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_DistributeStakingRewardsBankSendError() {
	creator := sdk.AccAddress("creator")

	stakingReward := types.StakingReward{
		RewardId:         "dist-bankerr-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "1000",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(10000)),
		)).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500))),
		).
		Return(sdkerrors.ErrInsufficientFunds).
		Times(1)

	msg := &types.MsgDistributeStakingRewards{
		Creator:  creator.String(),
		RewardId: "dist-bankerr-reward",
		Amount:   "500",
	}

	response, err := suite.msgServer.DistributeStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
}
