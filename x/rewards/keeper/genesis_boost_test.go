package keeper_test

import (
	"cosmossdk.io/math"
	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// TestBoostGenesis_RoundTripSettleParity exports a state with an active and a
// finished boost, imports it into a fresh chain, and asserts the same claim
// settles identically on both — the imported chain owes exactly what the
// exported one did.
func (suite *IntegrationTestSuite) TestBoostGenesis_RoundTripSettleParity() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "100")
	active := suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", false)
	finished := suite.setBoostWithAccumulator("000000000001", "000000000002", "ubst2", "1.5", true)
	entry := types.BoostParticipant{
		Address: creator.String(), RewardId: "000000000001", BoostId: "000000000002", JoinedAt: "0.5",
	}
	suite.k.SetBoostParticipant(suite.ctx, entry)
	suite.k.SetBoostsCounter(suite.ctx, 7)

	exported := rewards.ExportGenesis(suite.ctx, *suite.k)
	suite.Require().NoError(exported.Validate())
	suite.Require().ElementsMatch([]types.Boost{active, finished}, exported.BoostList)
	suite.Require().Equal([]types.BoostParticipant{entry}, exported.BoostParticipantList)
	suite.Require().Equal(uint64(7), exported.BoostCounter)

	// fresh chain importing the export
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()
	bank2 := testutil.NewMockBankKeeper(ctrl)
	k2, ctx2 := keepertest.RewardsKeeper(
		suite.T(), bank2, testutil.NewMockEpochKeeper(ctrl), testutil.NewMockTradingKeeper(ctrl), testutil.NewMockAccountKeeper(ctrl),
	)
	rewards.InitGenesis(ctx2, k2, *exported)

	suite.Require().ElementsMatch(exported.BoostList, k2.GetAllBoosts(ctx2))
	suite.Require().Equal(exported.BoostParticipantList, k2.GetAllBoostParticipant(ctx2))
	suite.Require().Equal(uint64(7), k2.GetBoostsCounter(ctx2))

	// same claim on both chains: active boost has no entry (S0 = 0) and pays
	// 100 x 2 = 200 uboost; finished boost settles from its stored entry and
	// pays 100 x (1.5 - 0.5) = 100 ubst2; the base reward owes nothing.
	activePayout := sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(200)))
	finishedPayout := sdk.NewCoins(sdk.NewCoin("ubst2", math.NewInt(100)))
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, activePayout).Return(nil).Times(1)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, finishedPayout).Return(nil).Times(1)
	bank2.EXPECT().SendCoinsFromModuleToAccount(ctx2, types.ModuleName, creator, activePayout).Return(nil).Times(1)
	bank2.EXPECT().SendCoinsFromModuleToAccount(ctx2, types.ModuleName, creator, finishedPayout).Return(nil).Times(1)

	msg := &types.MsgClaimStakingRewards{Creator: creator.String(), RewardId: "000000000001"}
	response1, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().NoError(err)

	msgServer2 := keeper.NewMsgServerImpl(k2)
	response2, err := msgServer2.ClaimStakingRewards(ctx2, msg)
	suite.Require().NoError(err)
	suite.Require().Equal(response1, response2)

	// both chains stamped the same baselines
	for _, boostId := range []string{"000000000001", "000000000002"} {
		entry1, found := suite.k.GetBoostParticipant(suite.ctx, creator.String(), "000000000001", boostId)
		suite.Require().True(found)
		entry2, found := k2.GetBoostParticipant(ctx2, creator.String(), "000000000001", boostId)
		suite.Require().True(found)
		suite.Require().Equal(entry1, entry2)
	}
}

// TestBoostGenesis_CounterNeverRestarts: the imported counter keeps issuing
// fresh ids — a restarted counter would reuse ids and corrupt the
// absent-entry rule (retired exploit A1).
func (suite *IntegrationTestSuite) TestBoostGenesis_CounterNeverRestarts() {
	suite.k.SetBoostsCounter(suite.ctx, 7)

	exported := rewards.ExportGenesis(suite.ctx, *suite.k)

	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()
	k2, ctx2 := keepertest.RewardsKeeper(
		suite.T(), testutil.NewMockBankKeeper(ctrl), testutil.NewMockEpochKeeper(ctrl), testutil.NewMockTradingKeeper(ctrl), testutil.NewMockAccountKeeper(ctrl),
	)
	rewards.InitGenesis(ctx2, k2, *exported)

	suite.Require().Equal("000000000008", k2.ReserveBoostId(ctx2))
}
