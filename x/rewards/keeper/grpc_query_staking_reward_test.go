package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestQueryStakingRewardAll_Success_EmptyList() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	l, err := suite.k.StakingRewardAll(goCtx, &types.QueryAllStakingRewardRequest{})
	suite.Require().NoError(err)
	suite.Require().Empty(l.List)
}

func (suite *IntegrationTestSuite) TestQueryStakingRewardAll_Success_OneItem() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	suite.k.SetStakingReward(suite.ctx, types.StakingReward{RewardId: "123"})

	l, err := suite.k.StakingRewardAll(goCtx, &types.QueryAllStakingRewardRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(len(l.List), 1)
}

func (suite *IntegrationTestSuite) TestQueryStakingRewardAll_Success_TwoItems() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	suite.k.SetStakingReward(suite.ctx, types.StakingReward{RewardId: "123"})
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{RewardId: "456"})

	l, err := suite.k.StakingRewardAll(goCtx, &types.QueryAllStakingRewardRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(len(l.List), 2)
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.StakingReward(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_NotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	_, err := suite.k.StakingReward(goCtx, &types.QueryGetStakingRewardRequest{RewardId: "1"})
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	sr := types.StakingReward{RewardId: "1"}
	suite.k.SetStakingReward(suite.ctx, sr)

	l, err := suite.k.StakingReward(goCtx, &types.QueryGetStakingRewardRequest{RewardId: sr.RewardId})
	suite.Require().NoError(err)
	suite.Require().EqualValues(l.StakingReward, sr)
}

func (suite *IntegrationTestSuite) TestQueryStakingRewardParticipantAll_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.StakingRewardParticipantAll(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryStakingRewardParticipantAll_Success_EmptyList() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	list, err := suite.k.StakingRewardParticipantAll(goCtx, &types.QueryAllStakingRewardParticipantRequest{})
	suite.Require().NoError(err)
	suite.Require().Empty(list.List)
}

func (suite *IntegrationTestSuite) TestQueryStakingRewardParticipantAll_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	srp := types.StakingRewardParticipant{
		Address:  "abc",
		RewardId: "001",
		Amount:   "123",
		JoinedAt: "2002",
	}

	suite.k.SetStakingRewardParticipant(suite.ctx, srp)

	list, err := suite.k.StakingRewardParticipantAll(goCtx, &types.QueryAllStakingRewardParticipantRequest{})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(list.List)
	suite.Require().EqualValues(srp, list.List[0])
}

func (suite *IntegrationTestSuite) TestQueryStakingRewardParticipant_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.StakingRewardParticipant(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryStakingRewardParticipant_NotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	l, err := suite.k.StakingRewardParticipant(goCtx, &types.QueryGetStakingRewardParticipantRequest{Address: "1"})
	suite.Require().NoError(err)
	suite.Require().Empty(l.List)
}

func (suite *IntegrationTestSuite) TestQueryStakingRewardParticipant_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	sr := types.StakingRewardParticipant{RewardId: "1", Address: "abc123"}
	suite.k.SetStakingRewardParticipant(suite.ctx, sr)

	l, err := suite.k.StakingRewardParticipant(goCtx, &types.QueryGetStakingRewardParticipantRequest{Address: "abc123"})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(l.List)
	suite.Require().EqualValues(l.List[0], sr)

	//check a random address returns an empty list
	l, err = suite.k.StakingRewardParticipant(goCtx, &types.QueryGetStakingRewardParticipantRequest{Address: "abc1233332"})
	suite.Require().NoError(err)
	suite.Require().Empty(l.List)
}
