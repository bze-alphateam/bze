package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (suite *IntegrationTestSuite) TestMsgExitStaking_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	_, err := suite.msgServer.ExitStaking(goCtx, nil)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (suite *IntegrationTestSuite) TestMsgExitStaking_MissingStakingReward() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	msg := types.MsgExitStaking{
		Creator:  "",
		RewardId: "a",
	}

	_, err := suite.msgServer.ExitStaking(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "reward with provided id not found")
}

func (suite *IntegrationTestSuite) TestMsgExitStaking_MissingStakingRewardParticipant() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	sr := types.StakingReward{
		RewardId:         "01",
		PrizeDenom:       denomBze,
		StakedAmount:     "50",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	msg := types.MsgExitStaking{
		Creator:  "aaa",
		RewardId: sr.RewardId,
	}
	_, err := suite.msgServer.ExitStaking(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "you are not a participant in this staking reward")
}

func (suite *IntegrationTestSuite) TestMsgExitStaking_Success_OngoingStakingReward() {
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
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	untouchedSrp := types.StakingRewardParticipant{
		Address:  "asadsadasda",
		RewardId: sr.RewardId,
		Amount:   "50",
		JoinedAt: "0",
	}
	srp := types.StakingRewardParticipant{
		Address:  addr1.String(),
		RewardId: sr.RewardId,
		Amount:   "22",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, srp)
	suite.k.SetStakingRewardParticipant(suite.ctx, untouchedSrp)

	balances := sdk.NewCoins(newStakeCoin(10000), newBzeCoin(50000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	//tests and asserts
	msg := types.MsgExitStaking{
		Creator:  addr1.String(),
		RewardId: sr.RewardId,
	}
	_, err := suite.msgServer.ExitStaking(goCtx, &msg)
	suite.Require().NoError(err)

	creatorBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	//check the user retrieves the unclaimed rewards first
	suite.Require().EqualValues(creatorBalance.AmountOf(denomBze).String(), "110")

	//check the unlock was created
	lockKey := types.CreatePendingUnlockParticipantKey(int64(sr.Lock*24), fmt.Sprintf("%s/%s", sr.RewardId, srp.Address))
	unlockList := suite.k.GetAllPendingUnlockParticipant(suite.ctx)
	suite.Require().NotEmpty(unlockList)
	suite.Require().EqualValues(unlockList[0].Index, lockKey)
	suite.Require().EqualValues(unlockList[0].Address, srp.Address)
	suite.Require().EqualValues(unlockList[0].Amount, "22")
	suite.Require().EqualValues(unlockList[0].Denom, sr.StakingDenom)

	//check the staking reward participant was deleted
	_, f := suite.k.GetStakingRewardParticipant(suite.ctx, srp.Address, sr.RewardId)
	suite.Require().False(f)

	//check the staking reward was updated
	newSr, f := suite.k.GetStakingReward(suite.ctx, sr.RewardId)
	suite.Require().True(f)
	suite.Require().EqualValues(newSr.StakedAmount, "28")

	//check that the dummy srp is not touched since it wasn't belonging to the message creator
	untouchedSrpStorage, f := suite.k.GetStakingRewardParticipant(suite.ctx, untouchedSrp.Address, untouchedSrp.RewardId)
	suite.Require().True(f)
	suite.Require().EqualValues(untouchedSrpStorage, untouchedSrp)
}

func (suite *IntegrationTestSuite) TestMsgExitStaking_Success_EmptyingStakingReward() {
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
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	untouchedSrp := types.StakingRewardParticipant{
		Address:  "asadsadasda",
		RewardId: sr.RewardId,
		Amount:   "50",
		JoinedAt: "0",
	}

	srp := types.StakingRewardParticipant{
		Address:  addr1.String(),
		RewardId: sr.RewardId,
		Amount:   "50",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, srp)
	suite.k.SetStakingRewardParticipant(suite.ctx, untouchedSrp)

	balances := sdk.NewCoins(newStakeCoin(10000), newBzeCoin(50000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	//tests and asserts
	msg := types.MsgExitStaking{
		Creator:  addr1.String(),
		RewardId: sr.RewardId,
	}
	_, err := suite.msgServer.ExitStaking(goCtx, &msg)
	suite.Require().NoError(err)

	creatorBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	//check the user retrieves the unclaimed rewards first
	suite.Require().EqualValues(creatorBalance.AmountOf(denomBze).String(), "250")

	//check the unlock was created
	lockKey := types.CreatePendingUnlockParticipantKey(int64(sr.Lock*24), fmt.Sprintf("%s/%s", sr.RewardId, srp.Address))
	unlockList := suite.k.GetAllPendingUnlockParticipant(suite.ctx)
	suite.Require().NotEmpty(unlockList)
	suite.Require().EqualValues(unlockList[0].Index, lockKey)
	suite.Require().EqualValues(unlockList[0].Address, srp.Address)
	suite.Require().EqualValues(unlockList[0].Amount, "50")
	suite.Require().EqualValues(unlockList[0].Denom, sr.StakingDenom)

	//check the staking reward participant was deleted
	_, f := suite.k.GetStakingRewardParticipant(suite.ctx, srp.Address, sr.RewardId)
	suite.Require().False(f)

	//check the staking reward was updated
	newSr, f := suite.k.GetStakingReward(suite.ctx, sr.RewardId)
	suite.Require().True(f)
	suite.Require().EqualValues(newSr.StakedAmount, "0")

	//check that the dummy srp is not touched since it wasn't belonging to the message creator
	untouchedSrpStorage, f := suite.k.GetStakingRewardParticipant(suite.ctx, untouchedSrp.Address, untouchedSrp.RewardId)
	suite.Require().True(f)
	suite.Require().EqualValues(untouchedSrpStorage, untouchedSrp)
}

func (suite *IntegrationTestSuite) TestMsgExitStaking_Success_RemovingStakingReward() {
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
		Payouts:          100,
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	srp := types.StakingRewardParticipant{
		Address:  addr1.String(),
		RewardId: sr.RewardId,
		Amount:   "50",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, srp)

	balances := sdk.NewCoins(newStakeCoin(10000), newBzeCoin(50000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	//tests and asserts
	msg := types.MsgExitStaking{
		Creator:  addr1.String(),
		RewardId: sr.RewardId,
	}
	_, err := suite.msgServer.ExitStaking(goCtx, &msg)
	suite.Require().NoError(err)

	creatorBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	//check the user retrieves the unclaimed rewards first
	suite.Require().EqualValues(creatorBalance.AmountOf(denomBze).String(), "250")

	//check the unlock was created
	lockKey := types.CreatePendingUnlockParticipantKey(int64(sr.Lock*24), fmt.Sprintf("%s/%s", sr.RewardId, srp.Address))
	unlockList := suite.k.GetAllPendingUnlockParticipant(suite.ctx)
	suite.Require().NotEmpty(unlockList)
	suite.Require().EqualValues(unlockList[0].Index, lockKey)
	suite.Require().EqualValues(unlockList[0].Address, srp.Address)
	suite.Require().EqualValues(unlockList[0].Amount, "50")
	suite.Require().EqualValues(unlockList[0].Denom, sr.StakingDenom)

	//check the staking reward participant was deleted
	_, f := suite.k.GetStakingRewardParticipant(suite.ctx, srp.Address, sr.RewardId)
	suite.Require().False(f)

	//check the staking reward was deleted
	_, f = suite.k.GetStakingReward(suite.ctx, sr.RewardId)
	suite.Require().False(f)
}

func (suite *IntegrationTestSuite) TestMsgExitStaking_Success_RemovingStakingReward_WithoutLock() {
	//dependencies
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	sr := types.StakingReward{
		RewardId:         "01",
		PrizeDenom:       denomBze,
		StakedAmount:     "50",
		DistributedStake: "5",
		Lock:             0,
		StakingDenom:     denomBze,
		Duration:         100,
		Payouts:          100,
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	srp := types.StakingRewardParticipant{
		Address:  addr1.String(),
		RewardId: sr.RewardId,
		Amount:   "50",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, srp)

	balances := sdk.NewCoins(newStakeCoin(10000), newBzeCoin(50000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	//tests and asserts
	msg := types.MsgExitStaking{
		Creator:  addr1.String(),
		RewardId: sr.RewardId,
	}
	_, err := suite.msgServer.ExitStaking(goCtx, &msg)
	suite.Require().NoError(err)

	creatorBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	//check the user retrieves the unclaimed rewards + the staked balance
	suite.Require().EqualValues(creatorBalance.AmountOf(denomBze).String(), "300")

	//check the unlock was NOT created since the funds should be released immediately
	unlockList := suite.k.GetAllPendingUnlockParticipant(suite.ctx)
	suite.Require().Empty(unlockList)

	//check the staking reward participant was deleted
	_, f := suite.k.GetStakingRewardParticipant(suite.ctx, srp.Address, sr.RewardId)
	suite.Require().False(f)

	//check the staking reward was deleted
	_, f = suite.k.GetStakingReward(suite.ctx, sr.RewardId)
	suite.Require().False(f)
}
