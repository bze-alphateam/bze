package keeper_test

import (
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (suite *IntegrationTestSuite) TestMsgJoinStaking_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	_, err := suite.msgServer.JoinStaking(goCtx, nil)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (suite *IntegrationTestSuite) TestMsgJoinStaking_InvalidCreator() {
	//dependencies
	goCtx := sdk.WrapSDKContext(suite.ctx)
	msg := types.MsgJoinStaking{
		Creator:  "",
		RewardId: "",
		Amount:   "",
	}

	_, err := suite.msgServer.JoinStaking(goCtx, &msg)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestMsgJoinStaking_MissingStakingReward() {
	//dependencies
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	msg := types.MsgJoinStaking{
		Creator:  addr1.String(),
		RewardId: "0001",
		Amount:   "",
	}

	_, err := suite.msgServer.JoinStaking(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "reward with provided id not found")
}

func (suite *IntegrationTestSuite) TestMsgJoinStaking_AmountLowerThanMinStake() {
	//dependencies
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("ubze", 10000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
	sr := types.StakingReward{
		RewardId:         "01",
		PrizeDenom:       denomBze,
		StakedAmount:     "50",
		DistributedStake: "5",
		Lock:             100,
		StakingDenom:     denomBze,
		Duration:         100,
		Payouts:          5,
		MinStake:         1000,
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	msg := types.MsgJoinStaking{
		Creator:  addr1.String(),
		RewardId: sr.RewardId,
		Amount:   "1",
	}

	_, err := suite.msgServer.JoinStaking(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "amount is smaller than staking reward min stake")
}

func (suite *IntegrationTestSuite) TestMsgJoinStaking_AllowedAmountLowerThanMinStake() {
	//dependencies
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("ubze", 10000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
	sr := types.StakingReward{
		RewardId:         "01",
		PrizeDenom:       denomBze,
		StakedAmount:     "50",
		DistributedStake: "5",
		Lock:             100,
		StakingDenom:     denomBze,
		Duration:         100,
		Payouts:          5,
		MinStake:         1000,
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	msg := types.MsgJoinStaking{
		Creator:  addr1.String(),
		RewardId: sr.RewardId,
		Amount:   "1000",
	}
	//first stake the min amount allowed
	_, err := suite.msgServer.JoinStaking(goCtx, &msg)
	suite.Require().NoError(err)

	//try to stake an amount lower than min stake
	//it should be allowed since we already have a stake greater than/equal to min stake
	msg.Amount = "50"
	_, err = suite.msgServer.JoinStaking(goCtx, &msg)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestMsgJoinStaking_NotEnoughBalance() {
	//dependencies
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	sr := types.StakingReward{
		RewardId:         "01",
		PrizeDenom:       denomBze,
		StakedAmount:     "50",
		DistributedStake: "5",
		Lock:             100,
		StakingDenom:     denomBze,
		Duration:         100,
		Payouts:          5,
		MinStake:         1,
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	//test and assertions
	msg := types.MsgJoinStaking{
		Creator:  addr1.String(),
		RewardId: sr.RewardId,
		Amount:   "10",
	}

	_, err := suite.msgServer.JoinStaking(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "user balance is too low")
}

func (suite *IntegrationTestSuite) TestMsgJoinStaking_Success_NewParticipant() {
	//dependencies
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")

	balances := sdk.NewCoins(sdk.NewInt64Coin("ubze", 10000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))

	sr := types.StakingReward{
		RewardId:         "01",
		PrizeDenom:       denomBze,
		StakedAmount:     "0",
		DistributedStake: "0",
		Lock:             100,
		StakingDenom:     denomBze,
		Duration:         100,
		Payouts:          5,
		MinStake:         1,
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	//test and assertions
	msg := types.MsgJoinStaking{
		Creator:  addr1.String(),
		RewardId: sr.RewardId,
		Amount:   "10",
	}

	_, err := suite.msgServer.JoinStaking(goCtx, &msg)
	suite.Require().NoError(err)

	part, f := suite.k.GetStakingRewardParticipant(suite.ctx, msg.Creator, sr.RewardId)
	suite.Require().True(f)
	suite.Require().EqualValues(part.JoinedAt, sr.DistributedStake)
	suite.Require().EqualValues(part.Address, msg.Creator)
	suite.Require().EqualValues(part.Amount, msg.Amount)
	suite.Require().EqualValues(part.RewardId, msg.RewardId)

	storageSr, f := suite.k.GetStakingReward(suite.ctx, sr.RewardId)
	suite.Require().True(f)
	suite.Require().EqualValues(storageSr.StakedAmount, "10")

	creatorBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	//check the user retrieves the unclaimed rewards first
	suite.Require().EqualValues(creatorBalance.AmountOf(denomBze).String(), "9990")

	//check balances were subtracted from module
	moduleAddr := suite.app.AccountKeeper.GetModuleAddress(types.ModuleName)
	newBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, moduleAddr)
	suite.Require().EqualValues(newBalances.AmountOf(denomBze).String(), "10")
}

func (suite *IntegrationTestSuite) TestMsgJoinStaking_Success_ExistingParticipant() {
	//dependencies
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")

	balances := sdk.NewCoins(sdk.NewInt64Coin("ubze", 10000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))

	sr := types.StakingReward{
		RewardId:         "01",
		PrizeDenom:       denomBze,
		StakedAmount:     "50",
		DistributedStake: "0",
		Lock:             100,
		StakingDenom:     denomBze,
		Duration:         100,
		Payouts:          0,
		MinStake:         10,
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	srp := types.StakingRewardParticipant{
		Address:  addr1.String(),
		RewardId: sr.RewardId,
		Amount:   "50",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, srp)

	//test and assertions
	msg := types.MsgJoinStaking{
		Creator:  addr1.String(),
		RewardId: sr.RewardId,
		Amount:   "10",
	}

	_, err := suite.msgServer.JoinStaking(goCtx, &msg)
	suite.Require().NoError(err)

	part, f := suite.k.GetStakingRewardParticipant(suite.ctx, msg.Creator, sr.RewardId)
	suite.Require().True(f)
	suite.Require().EqualValues(part.JoinedAt, sr.DistributedStake)
	suite.Require().EqualValues(part.Address, msg.Creator)
	suite.Require().EqualValues(part.Amount, "60")
	suite.Require().EqualValues(part.RewardId, msg.RewardId)

	storageSr, f := suite.k.GetStakingReward(suite.ctx, sr.RewardId)
	suite.Require().True(f)
	suite.Require().EqualValues(storageSr.StakedAmount, "60")

	creatorBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	//check the user retrieves the unclaimed rewards first
	suite.Require().EqualValues(creatorBalance.AmountOf(denomBze).String(), "9990")

	//check balances were subtracted from module
	moduleAddr := suite.app.AccountKeeper.GetModuleAddress(types.ModuleName)
	newBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, moduleAddr)
	suite.Require().EqualValues(newBalances.AmountOf(denomBze).String(), "10")
}
