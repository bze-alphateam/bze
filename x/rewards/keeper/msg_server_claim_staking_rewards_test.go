package keeper_test

import (
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	denomBze   = "ubze"
	denomStake = "stake"
)

func newStakeCoin(amt int64) sdk.Coin {
	return sdk.NewInt64Coin(denomStake, amt)
}

func newBzeCoin(amt int64) sdk.Coin {
	return sdk.NewInt64Coin(denomBze, amt)
}

func (suite *IntegrationTestSuite) TestMsgClaimStakingRewards_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	_, err := suite.msgServer.ClaimStakingRewards(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestMsgClaimStakingRewards_StakingRewardNotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	msg := types.MsgClaimStakingRewards{
		Creator:  "",
		RewardId: "asd",
	}

	_, err := suite.msgServer.ClaimStakingRewards(goCtx, &msg)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestMsgClaimStakingRewards_ParticipantNotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	sr := types.StakingReward{RewardId: "a"}
	suite.k.SetStakingReward(suite.ctx, sr)

	msg := types.MsgClaimStakingRewards{
		Creator:  "not_a_participant",
		RewardId: "asd",
	}

	_, err := suite.msgServer.ClaimStakingRewards(goCtx, &msg)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestMsgClaimStakingRewards_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")

	balances := sdk.NewCoins(newStakeCoin(10000), newBzeCoin(50000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	sr := types.StakingReward{
		RewardId:         "0001",
		DistributedStake: "100",
		PrizeDenom:       denomBze,
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	srp := types.StakingRewardParticipant{
		Address:  addr1.String(),
		RewardId: sr.RewardId,
		Amount:   "12",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, srp)

	msg := types.MsgClaimStakingRewards{
		Creator:  addr1.String(),
		RewardId: sr.RewardId,
	}

	resp, err := suite.msgServer.ClaimStakingRewards(goCtx, &msg)
	suite.Require().NoError(err)
	suite.Require().EqualValues(resp.Amount, "1200")

	resultedParticipant, f := suite.k.GetStakingRewardParticipant(suite.ctx, addr1.String(), sr.RewardId)
	suite.Require().True(f)
	//check if new participant is different than the old one (after claim some fields are updated)
	suite.Require().NotEqualValues(resultedParticipant, srp)

	//check joined at is equal to sr.DistributedStake
	suite.Require().EqualValues(resultedParticipant.JoinedAt, sr.DistributedStake)

	//check balances were subtracted from module
	moduleAddr := suite.app.AccountKeeper.GetModuleAddress(types.ModuleName)
	newBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, moduleAddr)
	suite.Require().EqualValues(newBalances.AmountOf(denomBze).String(), "48800")

	//check user was awarded with the amount
	takerBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().EqualValues(takerBalance.AmountOf(denomBze).String(), "1200")
}
