package keeper_test

import (
	"github.com/bze-alphateam/bze/testutil/sample"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) seedDenomReward(stakingDenom, stakedAmount string) types.DenomReward {
	dr := types.DenomReward{
		StakingDenom: stakingDenom,
		Lock:         7,
		MinStake:     0,
		StakedAmount: stakedAmount,
	}
	suite.k.SetDenomReward(suite.ctx, dr)

	return dr
}

func (suite *IntegrationTestSuite) TestQueryDenomReward_InvalidRequest() {
	_, err := suite.k.DenomReward(suite.ctx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryDenomReward_NotFound() {
	_, err := suite.k.DenomReward(suite.ctx, &types.QueryDenomRewardRequest{Denom: "udenom1"})
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestQueryDenomReward_Success() {
	dr := suite.seedDenomReward("udenom1", "1000")
	suite.seedDenomReward("udenom2", "500")

	resp, err := suite.k.DenomReward(suite.ctx, &types.QueryDenomRewardRequest{Denom: "udenom1"})
	suite.Require().NoError(err)
	suite.Require().Equal(dr, resp.DenomReward)
}

func (suite *IntegrationTestSuite) TestQueryDenomRewardAll_Pagination() {
	dr1 := suite.seedDenomReward("udenom1", "1000")
	dr2 := suite.seedDenomReward("udenom2", "500")
	dr3 := suite.seedDenomReward("udenom3", "0")

	resp, err := suite.k.DenomRewardAll(suite.ctx, &types.QueryDenomRewardAllRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal([]types.DenomReward{dr1, dr2, dr3}, resp.List)

	// first page of 2
	resp, err = suite.k.DenomRewardAll(suite.ctx, &types.QueryDenomRewardAllRequest{
		Pagination: &query.PageRequest{Limit: 2},
	})
	suite.Require().NoError(err)
	suite.Require().Equal([]types.DenomReward{dr1, dr2}, resp.List)
	suite.Require().NotNil(resp.Pagination.NextKey)

	// second page via next key
	resp, err = suite.k.DenomRewardAll(suite.ctx, &types.QueryDenomRewardAllRequest{
		Pagination: &query.PageRequest{Key: resp.Pagination.NextKey},
	})
	suite.Require().NoError(err)
	suite.Require().Equal([]types.DenomReward{dr3}, resp.List)
}

func (suite *IntegrationTestSuite) TestQueryDenomRewardPrizes() {
	suite.seedDenomReward("udenom1", "1000")
	suite.seedDenomReward("udenom2", "500")

	prizeA := types.DenomRewardPrize{StakingDenom: "udenom1", PrizeDenom: "uprizea", DistributedStake: "1.5", LastDistributionEpoch: 10}
	prizeB := types.DenomRewardPrize{StakingDenom: "udenom1", PrizeDenom: "uprizeb", DistributedStake: "0.25", LastDistributionEpoch: 11}
	other := types.DenomRewardPrize{StakingDenom: "udenom2", PrizeDenom: "uprizec", DistributedStake: "3", LastDistributionEpoch: 12}
	suite.k.SetDenomRewardPrize(suite.ctx, prizeA)
	suite.k.SetDenomRewardPrize(suite.ctx, prizeB)
	suite.k.SetDenomRewardPrize(suite.ctx, other)

	resp, err := suite.k.DenomRewardPrizes(suite.ctx, &types.QueryDenomRewardPrizesRequest{Denom: "udenom1"})
	suite.Require().NoError(err)
	suite.Require().Equal([]types.DenomRewardPrize{prizeA, prizeB}, resp.List)

	// a denom without prizes returns an empty list, not an error
	resp, err = suite.k.DenomRewardPrizes(suite.ctx, &types.QueryDenomRewardPrizesRequest{Denom: "udenom3"})
	suite.Require().NoError(err)
	suite.Require().Empty(resp.List)
}

func (suite *IntegrationTestSuite) TestQueryDenomRewardSchedules_Pagination() {
	suite.seedDenomReward("udenom1", "1000")
	suite.seedDenomReward("udenom2", "500")

	sched1 := types.DenomRewardSchedule{ScheduleId: "000000000001", StakingDenom: "udenom1", PrizeDenom: "uprizea", DailyAmount: "100", Duration: 30, Payouts: 10}
	sched2 := types.DenomRewardSchedule{ScheduleId: "000000000002", StakingDenom: "udenom1", PrizeDenom: "uprizeb", DailyAmount: "50", Duration: 5}
	other := types.DenomRewardSchedule{ScheduleId: "000000000003", StakingDenom: "udenom2", PrizeDenom: "uprizec", DailyAmount: "10", Duration: 2}
	suite.k.SetDenomRewardSchedule(suite.ctx, sched1)
	suite.k.SetDenomRewardSchedule(suite.ctx, sched2)
	suite.k.SetDenomRewardSchedule(suite.ctx, other)

	resp, err := suite.k.DenomRewardSchedules(suite.ctx, &types.QueryDenomRewardSchedulesRequest{Denom: "udenom1"})
	suite.Require().NoError(err)
	suite.Require().Equal([]types.DenomRewardSchedule{sched1, sched2}, resp.List)

	// first page of 1
	resp, err = suite.k.DenomRewardSchedules(suite.ctx, &types.QueryDenomRewardSchedulesRequest{
		Denom:      "udenom1",
		Pagination: &query.PageRequest{Limit: 1},
	})
	suite.Require().NoError(err)
	suite.Require().Equal([]types.DenomRewardSchedule{sched1}, resp.List)
	suite.Require().NotNil(resp.Pagination.NextKey)

	// second page via next key
	resp, err = suite.k.DenomRewardSchedules(suite.ctx, &types.QueryDenomRewardSchedulesRequest{
		Denom:      "udenom1",
		Pagination: &query.PageRequest{Key: resp.Pagination.NextKey},
	})
	suite.Require().NoError(err)
	suite.Require().Equal([]types.DenomRewardSchedule{sched2}, resp.List)
}

func (suite *IntegrationTestSuite) TestQueryDenomRewardParticipant_NotFound() {
	suite.seedDenomReward("udenom1", "1000")

	_, err := suite.k.DenomRewardParticipant(suite.ctx, &types.QueryDenomRewardParticipantRequest{
		Address: sample.AccAddress(),
		Denom:   "udenom1",
	})
	suite.Require().Error(err)
}

// TestQueryDenomRewardParticipant_PendingMatchesClaim seeds a multi-prize position with a
// stamped index, a lazy-zero (missing) index and a dust-only accumulator, and checks that the
// query's pending equals exactly what the settle engine then pays — and that computing it
// wrote nothing to the store.
func (suite *IntegrationTestSuite) TestQueryDenomRewardParticipant_PendingMatchesClaim() {
	addr := sample.AccAddress()
	dr := suite.seedDenomReward("udenom1", "1000")

	// prize A: S = 1.5, index stamped at 0.5 → pending = 400 × 1.0 = 400
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "udenom1", PrizeDenom: "uprizea", DistributedStake: "1.5"})
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{
		Address: addr, StakingDenom: "udenom1", PrizeDenom: "uprizea", Index: "0.5",
	})
	// prize B: S = 0.25, no index (lazy zero) → pending = 400 × 0.25 = 100
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "udenom1", PrizeDenom: "uprizeb", DistributedStake: "0.25"})
	// prize C: S = 0.001, no index → 400 × 0.001 = 0.4 → dust, excluded
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "udenom1", PrizeDenom: "uprizec", DistributedStake: "0.001"})

	participant := types.DenomRewardParticipant{Address: addr, StakingDenom: "udenom1", Amount: "400"}
	suite.k.SetDenomRewardParticipant(suite.ctx, participant)

	// snapshot the full module state before the query
	before := rewards.ExportGenesis(suite.ctx, *suite.k)

	resp, err := suite.k.DenomRewardParticipant(suite.ctx, &types.QueryDenomRewardParticipantRequest{
		Address: addr,
		Denom:   "udenom1",
	})
	suite.Require().NoError(err)
	suite.Require().Equal(participant, resp.Participant)

	expected := sdk.NewCoins(sdk.NewInt64Coin("uprizea", 400), sdk.NewInt64Coin("uprizeb", 100))
	suite.Require().Equal(expected, sdk.NewCoins(resp.Pending...))

	// the query performed zero writes: exported state is unchanged
	after := rewards.ExportGenesis(suite.ctx, *suite.k)
	suite.Require().Equal(before, after)

	// a real claim pays exactly what the query reported
	acc, err := sdk.AccAddressFromBech32(addr)
	suite.Require().NoError(err)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("uprizea", 400))).
		Return(nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("uprizeb", 100))).
		Return(nil).Times(1)

	paid, err := suite.k.SettleDenomParticipant(suite.ctx, dr, participant)
	suite.Require().NoError(err)
	suite.Require().Equal(expected, paid)
}

func (suite *IntegrationTestSuite) TestQueryDenomRewardParticipations() {
	addr := sample.AccAddress()
	otherAddr := sample.AccAddress()

	suite.seedDenomReward("udenom1", "1000")
	suite.seedDenomReward("udenom2", "500")

	p1 := types.DenomRewardParticipant{Address: addr, StakingDenom: "udenom1", Amount: "400"}
	p2 := types.DenomRewardParticipant{Address: addr, StakingDenom: "udenom2", Amount: "100"}
	other := types.DenomRewardParticipant{Address: otherAddr, StakingDenom: "udenom2", Amount: "77"}
	suite.k.SetDenomRewardParticipant(suite.ctx, p1)
	suite.k.SetDenomRewardParticipant(suite.ctx, p2)
	suite.k.SetDenomRewardParticipant(suite.ctx, other)

	// exactly the marker-indexed set of the address, nothing from other users
	resp, err := suite.k.DenomRewardParticipations(suite.ctx, &types.QueryDenomRewardParticipationsRequest{Address: addr})
	suite.Require().NoError(err)
	suite.Require().Equal([]types.DenomRewardParticipant{p1, p2}, resp.List)

	// first page of 1
	resp, err = suite.k.DenomRewardParticipations(suite.ctx, &types.QueryDenomRewardParticipationsRequest{
		Address:    addr,
		Pagination: &query.PageRequest{Limit: 1},
	})
	suite.Require().NoError(err)
	suite.Require().Equal([]types.DenomRewardParticipant{p1}, resp.List)
	suite.Require().NotNil(resp.Pagination.NextKey)

	// second page via next key
	resp, err = suite.k.DenomRewardParticipations(suite.ctx, &types.QueryDenomRewardParticipationsRequest{
		Address:    addr,
		Pagination: &query.PageRequest{Key: resp.Pagination.NextKey},
	})
	suite.Require().NoError(err)
	suite.Require().Equal([]types.DenomRewardParticipant{p2}, resp.List)
}
