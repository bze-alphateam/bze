package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestQueryStakingReward_StakingReward() {
	stakingReward := types.StakingReward{
		RewardId:         "query-test-reward",
		PrizeAmount:      math.NewInt(1000),
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         30,
		Payouts:          5,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     math.NewInt(5000),
		DistributedStake: math.LegacyMustNewDecFromStr("500"),
	}

	suite.k.SetStakingReward(suite.ctx, stakingReward)

	req := &types.QueryGetStakingRewardRequest{
		RewardId: "query-test-reward",
	}

	response, err := suite.k.StakingReward(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal(stakingReward.RewardId, response.StakingReward.RewardId)
	suite.Require().Equal(stakingReward.PrizeAmount, response.StakingReward.PrizeAmount)
	suite.Require().Equal(stakingReward.PrizeDenom, response.StakingReward.PrizeDenom)
	suite.Require().Equal(stakingReward.StakingDenom, response.StakingReward.StakingDenom)
	suite.Require().Equal(stakingReward.Duration, response.StakingReward.Duration)
	suite.Require().Equal(stakingReward.Payouts, response.StakingReward.Payouts)
	suite.Require().Equal(stakingReward.MinStake, response.StakingReward.MinStake)
	suite.Require().Equal(stakingReward.Lock, response.StakingReward.Lock)
	suite.Require().Equal(stakingReward.StakedAmount, response.StakingReward.StakedAmount)
	suite.Require().Equal(stakingReward.DistributedStake, response.StakingReward.DistributedStake)
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_StakingRewardNilRequest() {
	response, err := suite.k.StakingReward(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_StakingRewardNotFound() {
	req := &types.QueryGetStakingRewardRequest{
		RewardId: "non-existent-reward",
	}

	response, err := suite.k.StakingReward(suite.ctx, req)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.NotFound, status.Code(err))
	suite.Require().Contains(err.Error(), "not found")
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_AllStakingRewards() {
	stakingRewards := []types.StakingReward{
		{
			RewardId:         "all-query-reward-1",
			PrizeAmount:      math.NewInt(1000),
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         30,
			Payouts:          5,
			MinStake:         100,
			Lock:             7,
			StakedAmount:     math.NewInt(5000),
			DistributedStake: math.LegacyMustNewDecFromStr("500"),
		},
		{
			RewardId:         "all-query-reward-2",
			PrizeAmount:      math.NewInt(2000),
			PrizeDenom:       "utoken",
			StakingDenom:     "ustake",
			Duration:         60,
			Payouts:          10,
			MinStake:         200,
			Lock:             14,
			StakedAmount:     math.NewInt(10000),
			DistributedStake: math.LegacyMustNewDecFromStr("1000"),
		},
		{
			RewardId:         "all-query-reward-3",
			PrizeAmount:      math.NewInt(1500),
			PrizeDenom:       "ucoin",
			StakingDenom:     "ucoin",
			Duration:         45,
			Payouts:          8,
			MinStake:         150,
			Lock:             10,
			StakedAmount:     math.NewInt(7500),
			DistributedStake: math.LegacyMustNewDecFromStr("750"),
		},
	}

	for _, reward := range stakingRewards {
		suite.k.SetStakingReward(suite.ctx, reward)
	}

	req := &types.QueryAllStakingRewardsRequest{
		Pagination: nil,
	}

	response, err := suite.k.AllStakingRewards(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 3)

	// Verify all rewards are present
	rewardIds := make(map[string]bool)
	for _, reward := range response.List {
		rewardIds[reward.RewardId] = true
	}

	suite.Require().True(rewardIds["all-query-reward-1"])
	suite.Require().True(rewardIds["all-query-reward-2"])
	suite.Require().True(rewardIds["all-query-reward-3"])
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_AllStakingRewardsNilRequest() {
	response, err := suite.k.AllStakingRewards(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_AllStakingRewardsEmpty() {
	req := &types.QueryAllStakingRewardsRequest{
		Pagination: nil,
	}

	response, err := suite.k.AllStakingRewards(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Empty(response.List)
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_AllStakingRewardsPagination() {
	stakingRewards := []types.StakingReward{
		{
			RewardId:         "page-reward-1",
			PrizeAmount:      math.NewInt(1000),
			PrizeDenom:       "ubze",
			StakingDenom:     "ubze",
			Duration:         30,
			Payouts:          5,
			MinStake:         100,
			Lock:             7,
			StakedAmount:     math.NewInt(5000),
			DistributedStake: math.LegacyMustNewDecFromStr("500"),
		},
		{
			RewardId:         "page-reward-2",
			PrizeAmount:      math.NewInt(2000),
			PrizeDenom:       "utoken",
			StakingDenom:     "ustake",
			Duration:         60,
			Payouts:          10,
			MinStake:         200,
			Lock:             14,
			StakedAmount:     math.NewInt(10000),
			DistributedStake: math.LegacyMustNewDecFromStr("1000"),
		},
	}

	for _, reward := range stakingRewards {
		suite.k.SetStakingReward(suite.ctx, reward)
	}

	req := &types.QueryAllStakingRewardsRequest{
		Pagination: &query.PageRequest{
			Limit: 1,
		},
	}

	response, err := suite.k.AllStakingRewards(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 1)
	suite.Require().NotNil(response.Pagination)
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_StakingRewardParticipant() {
	participants := []types.StakingRewardParticipant{
		{
			Address:  "bze1user1",
			RewardId: "participant-reward-1",
			Amount:   "1000",
			JoinedAt: "500",
		},
		{
			Address:  "bze1user1",
			RewardId: "participant-reward-2",
			Amount:   "2000",
			JoinedAt: "1000",
		},
		{
			Address:  "bze1user2",
			RewardId: "participant-reward-1",
			Amount:   "1500",
			JoinedAt: "750",
		},
	}

	for _, participant := range participants {
		suite.k.SetStakingRewardParticipant(suite.ctx, participant)
	}

	req := &types.QueryStakingRewardParticipantRequest{
		Address:    "bze1user1",
		Pagination: nil,
	}

	response, err := suite.k.StakingRewardParticipant(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 2)

	// Verify only user1's participations are returned
	for _, participant := range response.List {
		suite.Require().Equal("bze1user1", participant.Address)
	}
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_StakingRewardParticipantNilRequest() {
	response, err := suite.k.StakingRewardParticipant(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_StakingRewardParticipantEmpty() {
	req := &types.QueryStakingRewardParticipantRequest{
		Address:    "bze1nonexistent",
		Pagination: nil,
	}

	response, err := suite.k.StakingRewardParticipant(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Empty(response.List)
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_AllStakingRewardParticipants() {
	participants := []types.StakingRewardParticipant{
		{
			Address:  "bze1user1",
			RewardId: "all-participant-reward-1",
			Amount:   "1000",
			JoinedAt: "500",
		},
		{
			Address:  "bze1user2",
			RewardId: "all-participant-reward-2",
			Amount:   "2000",
			JoinedAt: "1000",
		},
		{
			Address:  "bze1user3",
			RewardId: "all-participant-reward-3",
			Amount:   "1500",
			JoinedAt: "750",
		},
	}

	for _, participant := range participants {
		suite.k.SetStakingRewardParticipant(suite.ctx, participant)
	}

	req := &types.QueryAllStakingRewardParticipantsRequest{
		Pagination: nil,
	}

	response, err := suite.k.AllStakingRewardParticipants(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 3)

	// Verify all participants are present
	participantKeys := make(map[string]bool)
	for _, participant := range response.List {
		key := participant.Address + "-" + participant.RewardId
		participantKeys[key] = true
	}

	suite.Require().True(participantKeys["bze1user1-all-participant-reward-1"])
	suite.Require().True(participantKeys["bze1user2-all-participant-reward-2"])
	suite.Require().True(participantKeys["bze1user3-all-participant-reward-3"])
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_AllStakingRewardParticipantsNilRequest() {
	response, err := suite.k.AllStakingRewardParticipants(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_AllStakingRewardParticipantsEmpty() {
	req := &types.QueryAllStakingRewardParticipantsRequest{
		Pagination: nil,
	}

	response, err := suite.k.AllStakingRewardParticipants(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Empty(response.List)
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_AllPendingUnlockParticipants() {
	epochNumber := int64(100)
	addr1 := sdk.AccAddress("addr1")
	addr2 := sdk.AccAddress("addr2")

	participants := []types.PendingUnlockParticipant{
		{
			Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr1.String())),
			Address: addr1.String(),
			Amount:  "1000",
			Denom:   "ubze",
		},
		{
			Index:   fmt.Sprintf("%d/%s", epochNumber+1, fmt.Sprintf("%s/%s", "reward_2", addr2.String())),
			Address: addr2.String(),
			Amount:  "2000",
			Denom:   "utoken",
		},
	}

	for _, participant := range participants {
		suite.k.SetPendingUnlockParticipant(suite.ctx, participant)
	}

	req := &types.QueryAllPendingUnlockParticipantsRequest{
		Pagination: nil,
	}

	response, err := suite.k.AllPendingUnlockParticipants(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 2)

	// Verify all participants are present
	addresses := make(map[string]bool)
	for _, participant := range response.List {
		addresses[participant.Address] = true
	}

	suite.Require().True(addresses[addr1.String()])
	suite.Require().True(addresses[addr2.String()])
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_AllPendingUnlockParticipantsNilRequest() {
	response, err := suite.k.AllPendingUnlockParticipants(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_AllPendingUnlockParticipantsEmpty() {
	req := &types.QueryAllPendingUnlockParticipantsRequest{
		Pagination: nil,
	}

	response, err := suite.k.AllPendingUnlockParticipants(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Empty(response.List)
}

func (suite *IntegrationTestSuite) TestQueryStakingReward_AllPendingUnlockParticipantsPagination() {
	epochNumber := int64(100)
	addr1 := sdk.AccAddress("addr1")
	addr2 := sdk.AccAddress("addr2")

	participants := []types.PendingUnlockParticipant{
		{
			Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr1.String())),
			Address: addr1.String(),
			Amount:  "1000",
			Denom:   "ubze",
		},
		{
			Index:   fmt.Sprintf("%d/%s", epochNumber+1, fmt.Sprintf("%s/%s", "reward_2", addr2.String())),
			Address: addr2.String(),
			Amount:  "2000",
			Denom:   "utoken",
		},
	}

	for _, participant := range participants {
		suite.k.SetPendingUnlockParticipant(suite.ctx, participant)
	}

	req := &types.QueryAllPendingUnlockParticipantsRequest{
		Pagination: &query.PageRequest{
			Limit: 1,
		},
	}

	response, err := suite.k.AllPendingUnlockParticipants(suite.ctx, req)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.List, 1)
	suite.Require().NotNil(response.Pagination)
}
