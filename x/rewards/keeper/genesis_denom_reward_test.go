package keeper_test

import (
	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// TestGenesisDenomRewardsRoundTrip exports a mid-lifecycle Denom Rewards state (active
// schedules, multiple prize denoms, participants with and without indexes, a pending
// distribution queue), imports it into a fresh keeper and asserts the imported chain is
// indistinguishable from the control: same re-export, rebuilt drp/a/ markers, and identical
// settle results for every participant.
func (suite *IntegrationTestSuite) TestGenesisDenomRewardsRoundTrip() {
	addr1 := sample.AccAddress()
	addr2 := sample.AccAddress()

	// control keeper (suite.k): build mid-lifecycle state through the regular setters
	suite.seedDenomReward("udenom1", "1000")
	suite.seedDenomReward("udenom2", "0")

	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "udenom1", PrizeDenom: "uprizea", DistributedStake: "1.5", LastDistributionEpoch: 10})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "udenom1", PrizeDenom: "uprizeb", DistributedStake: "0.25", LastDistributionEpoch: 11})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "udenom2", PrizeDenom: "uprizec", DistributedStake: "0", LastDistributionEpoch: 0})

	// addr1 has an index on uprizea only (uprizeb reads lazy-zero); addr2 has both stamped
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: addr1, StakingDenom: "udenom1", Amount: "400"})
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{Address: addr1, StakingDenom: "udenom1", PrizeDenom: "uprizea", Index: "0.5"})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: addr2, StakingDenom: "udenom1", Amount: "600"})
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{Address: addr2, StakingDenom: "udenom1", PrizeDenom: "uprizea", Index: "1.5"})
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{Address: addr2, StakingDenom: "udenom1", PrizeDenom: "uprizeb", Index: "0"})

	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{ScheduleId: "000000000001", StakingDenom: "udenom1", PrizeDenom: "uprizea", DailyAmount: "100", Duration: 30, Payouts: 10})
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{ScheduleId: "000000000002", StakingDenom: "udenom2", PrizeDenom: "uprizec", DailyAmount: "50", Duration: 5, Payouts: 0})
	suite.k.SetDenomRewardScheduleCounter(suite.ctx, 2)

	// a distribution caught mid-drain: pending with a cursor pointing at schedule 1
	suite.k.SetDenomRewardsDistributionQueue(suite.ctx, types.DenomRewardsDistributionQueue{
		Pending: true,
		Cursor:  string(types.DenomRewardScheduleKey("udenom1", "000000000001")),
	})

	exported := rewards.ExportGenesis(suite.ctx, *suite.k)
	suite.Require().NoError(exported.Validate())

	// import into a fresh chain
	mockCtrl := gomock.NewController(suite.T())
	freshBank := testutil.NewMockBankKeeper(mockCtrl)
	freshK, freshCtx := keepertest.RewardsKeeper(suite.T(), freshBank, testutil.NewMockEpochKeeper(mockCtrl), testutil.NewMockTradingKeeper(mockCtrl), testutil.NewMockAccountKeeper(mockCtrl))
	rewards.InitGenesis(freshCtx, freshK, *exported)

	// the imported chain re-exports to the identical genesis
	suite.Require().Equal(exported, rewards.ExportGenesis(freshCtx, freshK))

	// the drp/a/ markers (derivable state, not exported) were rebuilt on import
	suite.Require().True(freshK.HasDenomRewardParticipantMarker(freshCtx, addr1, "udenom1"))
	suite.Require().True(freshK.HasDenomRewardParticipantMarker(freshCtx, addr2, "udenom1"))
	suite.Require().False(freshK.HasDenomRewardParticipantMarker(freshCtx, addr1, "udenom2"))

	// settle every participant on both chains and require identical payouts:
	// addr1 → uprizea 400×(1.5−0.5)=400, uprizeb 400×0.25=100 (lazy zero)
	// addr2 → uprizea nothing (index at S), uprizeb 600×0.25=150
	for _, tc := range []struct {
		address string
		paid    sdk.Coins
	}{
		{addr1, sdk.NewCoins(sdk.NewInt64Coin("uprizea", 400), sdk.NewInt64Coin("uprizeb", 100))},
		{addr2, sdk.NewCoins(sdk.NewInt64Coin("uprizeb", 150))},
	} {
		acc, err := sdk.AccAddressFromBech32(tc.address)
		suite.Require().NoError(err)

		for _, k := range []struct {
			keeper keeper.Keeper
			ctx    sdk.Context
			bank   *testutil.MockBankKeeper
		}{
			{*suite.k, suite.ctx, suite.bank},
			{freshK, freshCtx, freshBank},
		} {
			dr, found := k.keeper.GetDenomReward(k.ctx, "udenom1")
			suite.Require().True(found)
			participant, found := k.keeper.GetDenomRewardParticipant(k.ctx, "udenom1", tc.address)
			suite.Require().True(found)

			for _, coin := range tc.paid {
				k.bank.EXPECT().
					SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, acc, sdk.NewCoins(coin)).
					Return(nil).Times(1)
			}

			paid, err := k.keeper.SettleDenomParticipant(k.ctx, dr, participant)
			suite.Require().NoError(err)
			suite.Require().Equal(tc.paid, paid)
		}
	}
}
