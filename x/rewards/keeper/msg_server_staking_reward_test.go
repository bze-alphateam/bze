package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_CreateStakingRewardSuccess() {
	creator := sdk.AccAddress("creator")

	// Set up params with fee
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(100)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(200)),
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock bank keeper calls
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(2) // Called for both staking and prize denom

	// Mock spendable coins check
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(10000)),
		)).
		Times(1)

	// Mock sending coins to module
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(5000))), // 1000 * 5 duration
		).
		Return(nil).
		Times(1)

	// Mock fee capture and swap
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(
			suite.ctx,
			creator,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100))),
			types.ModuleName,
		).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100))), nil).
		Times(1)

	// Mock sending fee to fee collector
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(
			suite.ctx,
			types.ModuleName,
			gomock.Any(),
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100))),
		).
		Return(nil).
		Times(1)

	msg := &types.MsgCreateStakingReward{
		Creator:      creator.String(),
		PrizeAmount:  "1000",
		PrizeDenom:   "ubze",
		StakingDenom: "ubze",
		Duration:     "5",
		MinStake:     "100",
		Lock:         "7",
	}

	counter := suite.k.GetStakingRewardsCounter(suite.ctx)
	response, err := suite.msgServer.CreateStakingReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().NotEmpty(response.RewardId)
	newCounter := suite.k.GetStakingRewardsCounter(suite.ctx)
	suite.Require().Equal(counter+1, newCounter)

	// Verify staking reward was created
	stakingReward, found := suite.k.GetStakingReward(suite.ctx, response.RewardId)
	suite.Require().True(found)
	suite.Require().Equal("1000", stakingReward.PrizeAmount)
	suite.Require().Equal("ubze", stakingReward.PrizeDenom)
	suite.Require().Equal("ubze", stakingReward.StakingDenom)
	suite.Require().Equal(uint32(5), stakingReward.Duration)
	suite.Require().Equal(uint64(100), stakingReward.MinStake)
	suite.Require().Equal(uint32(7), stakingReward.Lock)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_CreateStakingRewardNilRequest() {
	response, err := suite.msgServer.CreateStakingReward(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(sdkerrors.ErrInvalidRequest, err)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_CreateStakingRewardInvalidCreator() {
	msg := &types.MsgCreateStakingReward{
		Creator:      "invalid-address",
		PrizeAmount:  "1000",
		PrizeDenom:   "ubze",
		StakingDenom: "ubze",
		Duration:     "5",
		MinStake:     "100",
		Lock:         "7",
	}

	response, err := suite.msgServer.CreateStakingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_CreateStakingRewardInvalidStakingDenom() {
	creator := sdk.AccAddress("creator")

	// Mock bank keeper to return false for staking denom
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "invalid-denom").
		Return(false).
		Times(1)

	msg := &types.MsgCreateStakingReward{
		Creator:      creator.String(),
		PrizeAmount:  "1000",
		PrizeDenom:   "ubze",
		StakingDenom: "invalid-denom",
		Duration:     "5",
		MinStake:     "100",
		Lock:         "7",
	}

	response, err := suite.msgServer.CreateStakingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(types.ErrInvalidStakingDenom, err)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_CreateStakingRewardInvalidPrizeDenom() {
	creator := sdk.AccAddress("creator")

	// Mock bank keeper calls
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(1) // Called for staking denom

	suite.bank.EXPECT().
		HasSupply(suite.ctx, "invalid-prize").
		Return(false).
		Times(1) // Called for prize denom

	msg := &types.MsgCreateStakingReward{
		Creator:      creator.String(),
		PrizeAmount:  "1000",
		PrizeDenom:   "invalid-prize",
		StakingDenom: "ubze",
		Duration:     "5",
		MinStake:     "100",
		Lock:         "7",
	}

	response, err := suite.msgServer.CreateStakingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(types.ErrInvalidPrizeDenom, err)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_CreateStakingRewardInsufficientFunds() {
	creator := sdk.AccAddress("creator")

	// Set up params with fee
	params := types.Params{
		CreateStakingRewardFee: sdk.NewCoin("ubze", math.NewInt(100)),
		CreateTradingRewardFee: sdk.NewCoin("ubze", math.NewInt(200)),
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Mock bank keeper calls
	suite.bank.EXPECT().
		HasSupply(suite.ctx, "ubze").
		Return(true).
		Times(2)

	// Mock insufficient spendable coins
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(100)), // Not enough for 5000 + 100 fee
		)).
		Times(1)

	msg := &types.MsgCreateStakingReward{
		Creator:      creator.String(),
		PrizeAmount:  "1000",
		PrizeDenom:   "ubze",
		StakingDenom: "ubze",
		Duration:     "5",
		MinStake:     "100",
		Lock:         "7",
	}

	response, err := suite.msgServer.CreateStakingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "user balance is too low")
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_UpdateStakingRewardSuccess() {
	creator := sdk.AccAddress("creator")

	// Set up existing staking reward
	existingReward := types.StakingReward{
		RewardId:         "update-test-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "0",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, existingReward)

	// Mock bank keeper calls
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
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(3000))), // 1000 * 3 additional duration
		).
		Return(nil).
		Times(1)

	msg := &types.MsgUpdateStakingReward{
		Creator:  creator.String(),
		RewardId: "update-test-reward",
		Duration: "3",
	}

	counter := suite.k.GetStakingRewardsCounter(suite.ctx)
	response, err := suite.msgServer.UpdateStakingReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	newCounter := suite.k.GetStakingRewardsCounter(suite.ctx)
	//check the counter was not incremented when the staking reward was updated
	suite.Require().Equal(counter, newCounter)

	// Verify duration was updated
	updatedReward, found := suite.k.GetStakingReward(suite.ctx, "update-test-reward")
	suite.Require().True(found)
	suite.Require().Equal(uint32(8), updatedReward.Duration) // 5 + 3
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_UpdateStakingRewardNotFound() {
	creator := sdk.AccAddress("creator")

	msg := &types.MsgUpdateStakingReward{
		Creator:  creator.String(),
		RewardId: "non-existent-reward",
		Duration: "3",
	}

	response, err := suite.msgServer.UpdateStakingReward(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "staking reward not found")
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_JoinStakingSuccess() {
	creator := sdk.AccAddress("creator")

	// Set up existing staking reward
	stakingReward := types.StakingReward{
		RewardId:         "join-test-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "0",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	// Mock bank keeper calls
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
		Return(nil).
		Times(1)

	msg := &types.MsgJoinStaking{
		Creator:  creator.String(),
		RewardId: "join-test-reward",
		Amount:   "500",
	}

	counter := suite.k.GetStakingRewardsCounter(suite.ctx)
	response, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	newCounter := suite.k.GetStakingRewardsCounter(suite.ctx)
	//check the counter was NOT incremented when the staking reward was joined
	suite.Require().Equal(counter, newCounter)

	// Verify participant was created
	participant, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "join-test-reward")
	suite.Require().True(found)
	suite.Require().Equal("500", participant.Amount)

	// Verify staked amount was updated
	updatedReward, found := suite.k.GetStakingReward(suite.ctx, "join-test-reward")
	suite.Require().True(found)
	suite.Require().Equal("500", updatedReward.StakedAmount)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_JoinStakingMinStakeNotMet() {
	creator := sdk.AccAddress("creator")

	// Set up existing staking reward with high min stake
	stakingReward := types.StakingReward{
		RewardId:         "join-minstake-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         1000, // High min stake
		Lock:             7,
		StakedAmount:     "0",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	// Mock bank keeper calls
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(10000)),
		)).
		Times(1)

	msg := &types.MsgJoinStaking{
		Creator:  creator.String(),
		RewardId: "join-minstake-reward",
		Amount:   "500", // Less than min stake
	}

	response, err := suite.msgServer.JoinStaking(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "amount is smaller than staking reward min stake")
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_ExitStakingSuccess() {
	creator := sdk.AccAddress("creator")

	// Set up existing staking reward and participant
	stakingReward := types.StakingReward{
		RewardId:         "exit-test-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             0, // No lock for immediate unlock
		StakedAmount:     "500",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	participant := types.StakingRewardParticipant{
		Address:  creator.String(),
		RewardId: "exit-test-reward",
		Amount:   "500",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	// Mock epoch keeper call (even for lock = 0, beginUnlock still calls it)
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(100)).
		Times(1)

	// Mock bank keeper call for unlock (immediate since lock = 0)
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
		RewardId: "exit-test-reward",
	}

	counter := suite.k.GetStakingRewardsCounter(suite.ctx)
	response, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	newCounter := suite.k.GetStakingRewardsCounter(suite.ctx)
	//check the counter was NOT incremented when the user exited staking
	suite.Require().Equal(counter, newCounter)

	// Verify participant was removed
	_, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "exit-test-reward")
	suite.Require().False(found)

	// Verify staked amount was updated
	updatedReward, found := suite.k.GetStakingReward(suite.ctx, "exit-test-reward")
	suite.Require().True(found)
	suite.Require().Equal("0", updatedReward.StakedAmount)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_ExitStakingWithLock() {
	creator := sdk.AccAddress("creator")

	// Set up existing staking reward and participant with lock
	stakingReward := types.StakingReward{
		RewardId:         "exit-lock-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7, // 7 day lock
		StakedAmount:     "500",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	participant := types.StakingRewardParticipant{
		Address:  creator.String(),
		RewardId: "exit-lock-reward",
		Amount:   "500",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	// Mock epoch keeper call for lock calculation
	suite.epoch.EXPECT().
		GetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(100)).
		Times(1)

	msg := &types.MsgExitStaking{
		Creator:  creator.String(),
		RewardId: "exit-lock-reward",
	}

	response, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify participant was removed
	_, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "exit-lock-reward")
	suite.Require().False(found)

	// Verify pending unlock participant was created
	expectedKey := fmt.Sprintf("%d/%s", 100+7*24, fmt.Sprintf("%s/%s", "exit-lock-reward", creator.String()))
	pendingParticipant, found := suite.k.GetPendingUnlockParticipant(suite.ctx, expectedKey)
	suite.Require().True(found)
	suite.Require().Equal(creator.String(), pendingParticipant.Address)
	suite.Require().Equal("500", pendingParticipant.Amount)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_ClaimStakingRewardsSuccess() {
	creator := sdk.AccAddress("creator")

	// Set up staking reward with distributed stake
	stakingReward := types.StakingReward{
		RewardId:         "claim-test-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "500",
		DistributedStake: "1.0", // Some rewards distributed
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	participant := types.StakingRewardParticipant{
		Address:  creator.String(),
		RewardId: "claim-test-reward",
		Amount:   "500",
		JoinedAt: "0", // Joined before any distribution
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	// Mock bank keeper call for reward claim
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			creator,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500))), // 500 * (1.0 - 0) = 500
		).
		Return(nil).
		Times(1)

	msg := &types.MsgClaimStakingRewards{
		Creator:  creator.String(),
		RewardId: "claim-test-reward",
	}

	response, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal("500", response.Amount)

	// Verify participant's JoinedAt was updated
	updatedParticipant, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "claim-test-reward")
	suite.Require().True(found)
	suite.Require().Equal("1.0", updatedParticipant.JoinedAt)
}

func (suite *IntegrationTestSuite) TestMsgServerStakingReward_DistributeStakingRewardsSuccess() {
	creator := sdk.AccAddress("creator")

	// Set up existing staking reward
	stakingReward := types.StakingReward{
		RewardId:         "distribute-test-reward",
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

	// Mock bank keeper calls
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
		Return(nil).
		Times(1)

	msg := &types.MsgDistributeStakingRewards{
		Creator:  creator.String(),
		RewardId: "distribute-test-reward",
		Amount:   "500",
	}

	counter := suite.k.GetStakingRewardsCounter(suite.ctx)
	response, err := suite.msgServer.DistributeStakingRewards(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	newCounter := suite.k.GetStakingRewardsCounter(suite.ctx)
	//check the counter was NOT incremented when distributing rewards
	suite.Require().Equal(counter, newCounter)

	// Verify distributed stake was updated
	updatedReward, found := suite.k.GetStakingReward(suite.ctx, "distribute-test-reward")
	suite.Require().True(found)
	suite.Require().Equal("0.500000000000000000", updatedReward.DistributedStake) // 500/1000 = 0.5
}
