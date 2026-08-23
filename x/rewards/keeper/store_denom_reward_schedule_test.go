package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestStoreDenomRewardSchedule_SetGetRemove() {
	s := types.DenomRewardSchedule{
		ScheduleId:   "000000000001",
		StakingDenom: "ubze",
		PrizeDenom:   "uprize",
		DailyAmount:  math.NewInt(100),
		Duration:     30,
		Payouts:      0,
	}

	_, found := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000001")
	suite.Require().False(found)

	suite.k.SetDenomRewardSchedule(suite.ctx, s)
	got, found := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(s.PrizeDenom, got.PrizeDenom)
	suite.Require().Equal(s.DailyAmount, got.DailyAmount)
	suite.Require().Equal(s.Duration, got.Duration)

	suite.k.RemoveDenomRewardSchedule(suite.ctx, "ubze", "000000000001")
	_, found = suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000001")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreDenomRewardSchedule_IterateByDenom_Isolation() {
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{ScheduleId: "000000000001", StakingDenom: "ubze", PrizeDenom: "p1"})
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{ScheduleId: "000000000002", StakingDenom: "ubze", PrizeDenom: "p2"})
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{ScheduleId: "000000000003", StakingDenom: "other", PrizeDenom: "p1"})

	var ubze []string
	suite.k.IterateDenomSchedules(suite.ctx, "ubze", func(_ sdk.Context, s types.DenomRewardSchedule) bool {
		suite.Require().Equal("ubze", s.StakingDenom)
		ubze = append(ubze, s.ScheduleId)
		return false
	})
	suite.Require().Equal([]string{"000000000001", "000000000002"}, ubze)

	suite.Require().Len(suite.k.GetAllDenomRewardSchedule(suite.ctx), 3)
}

func (suite *IntegrationTestSuite) TestStoreDenomRewardSchedule_GetBatch_CursorSemantics() {
	// global composite-key order is "{denom}/{id}/": other/...1, ubze/...1, ubze/...2
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{ScheduleId: "000000000001", StakingDenom: "ubze", PrizeDenom: "p1"})
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{ScheduleId: "000000000002", StakingDenom: "ubze", PrizeDenom: "p2"})
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{ScheduleId: "000000000001", StakingDenom: "other", PrizeDenom: "p1"})

	// empty cursor starts from the beginning, limit is respected
	first := suite.k.GetBatchDenomRewardSchedules(suite.ctx, "", 2)
	suite.Require().Len(first, 2)
	suite.Require().Equal("other", first[0].StakingDenom)
	suite.Require().Equal("ubze", first[1].StakingDenom)
	suite.Require().Equal("000000000001", first[1].ScheduleId)

	// resume strictly after the last returned key
	cursor := string(types.DenomRewardScheduleKey(first[1].StakingDenom, first[1].ScheduleId))
	second := suite.k.GetBatchDenomRewardSchedules(suite.ctx, cursor, 2)
	suite.Require().Len(second, 1)
	suite.Require().Equal("ubze", second[0].StakingDenom)
	suite.Require().Equal("000000000002", second[0].ScheduleId)

	// cursor at the very last key yields nothing
	lastCursor := string(types.DenomRewardScheduleKey("ubze", "000000000002"))
	suite.Require().Len(suite.k.GetBatchDenomRewardSchedules(suite.ctx, lastCursor, 2), 0)
}

func (suite *IntegrationTestSuite) TestStoreDenomRewardScheduleCounter_IndependentFromSRAndTrading() {
	// seed the SR and trading counters (they share the sr/c/ store, sub-keys 1 and 2)
	suite.k.SetStakingRewardsCounter(suite.ctx, 100)
	suite.k.SetTradingRewardsCounter(suite.ctx, 200)

	// the DR schedule counter lives in its own dr/c/ store and starts at 0
	suite.Require().Equal(uint64(0), suite.k.GetDenomRewardScheduleCounter(suite.ctx))

	// the counter only advances through CreateDenomRewardSchedule (each schedule consumes one id);
	// zero fees keep the flow free so no trade/fee-collector expectations are needed
	creator := sdk.AccAddress("drs-counter-01")
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{MaxPrizeDenomsPerDr: 50}))
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", StakedAmount: math.ZeroInt()})
	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(2)
	suite.richBalance(creator)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(10_000)))).
		Return(nil).Times(2)

	// advancing the DR counter does not touch the SR/trading counters
	first, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), "10"))
	suite.Require().NoError(err)
	suite.Require().Equal("000000000000", first.ScheduleId)
	suite.Require().Equal(uint64(1), suite.k.GetDenomRewardScheduleCounter(suite.ctx))
	second, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), "10"))
	suite.Require().NoError(err)
	suite.Require().Equal("000000000001", second.ScheduleId)
	suite.Require().Equal(uint64(2), suite.k.GetDenomRewardScheduleCounter(suite.ctx))

	suite.Require().Equal(uint64(100), suite.k.GetStakingRewardsCounter(suite.ctx))
	suite.Require().Equal(uint64(200), suite.k.GetTradingRewardsCounter(suite.ctx))

	// and advancing the SR counter does not touch the DR counter
	suite.k.SetStakingRewardsCounter(suite.ctx, 101)
	suite.Require().Equal(uint64(2), suite.k.GetDenomRewardScheduleCounter(suite.ctx))
}

func (suite *IntegrationTestSuite) TestStoreDenomRewardsDistributionQueue_SetGetRemove() {
	_, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)

	suite.k.SetDenomRewardsDistributionQueue(suite.ctx, types.DenomRewardsDistributionQueue{Pending: true, Cursor: "ubze/000000000001/"})
	q, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(q.Pending)
	suite.Require().Equal("ubze/000000000001/", q.Cursor)

	suite.k.RemoveDenomRewardsDistributionQueue(suite.ctx)
	_, found = suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
}
