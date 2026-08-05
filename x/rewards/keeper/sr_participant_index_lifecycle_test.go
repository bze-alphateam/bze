package keeper_test

import (
	"cosmossdk.io/math"
	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// TestSrParticipantIndex_JoinCreatesEntry covers C5: a fresh join writes the
// joiner's reverse-index entry.
func (suite *IntegrationTestSuite) TestSrParticipantIndex_JoinCreatesEntry() {
	joiner := sdk.AccAddress("joiner.")
	sr, _ := suite.seedRewardAndParticipant("000000000001", "100")

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, joiner).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, joiner, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	msg := &types.MsgJoinStaking{Creator: joiner.String(), RewardId: sr.RewardId, Amount: "500"}
	_, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().True(suite.k.HasStakingRewardParticipantIndexEntry(suite.ctx, sr.RewardId, joiner.String()))
}

// TestSrParticipantIndex_TopUpCreatesEntry covers C5: a top-up join by a
// pre-index participant (entry seeded directly, no index write) creates the
// entry too — the maintenance path is idempotent and self-healing.
func (suite *IntegrationTestSuite) TestSrParticipantIndex_TopUpCreatesEntry() {
	creator := sdk.AccAddress("creator")
	sr, _ := suite.seedRewardAndParticipant("000000000001", "100")
	suite.Require().False(suite.k.HasStakingRewardParticipantIndexEntry(suite.ctx, sr.RewardId, creator.String()))

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(50)))).
		Return(nil).
		Times(1)

	msg := &types.MsgJoinStaking{Creator: creator.String(), RewardId: sr.RewardId, Amount: "50"}
	_, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().Equal([]string{creator.String()}, suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, sr.RewardId, "", 100))
}

// TestSrParticipantIndex_ExitRemovesEntry covers C5: exit removes the index
// entry (no boosts present).
func (suite *IntegrationTestSuite) TestSrParticipantIndex_ExitRemovesEntry() {
	creator := sdk.AccAddress("creator")
	sr, participant := suite.seedRewardAndParticipant("000000000001", "500")
	suite.k.SetStakingRewardParticipantIndexEntry(suite.ctx, sr.RewardId, participant.Address)

	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "hour").Return(int64(100), nil).AnyTimes()
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	msg := &types.MsgExitStaking{Creator: creator.String(), RewardId: sr.RewardId}
	_, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().False(suite.k.HasStakingRewardParticipantIndexEntry(suite.ctx, sr.RewardId, creator.String()))
}

// TestSrParticipantIndex_ExitWithBoostsRemovesEntry covers C5: exit removes
// the index entry with a boost present (boost settle runs in the same tx).
func (suite *IntegrationTestSuite) TestSrParticipantIndex_ExitWithBoostsRemovesEntry() {
	creator := sdk.AccAddress("creator")
	sr, participant := suite.seedRewardAndParticipant("000000000001", "500")
	suite.setBoostWithAccumulator(sr.RewardId, "000000000001", "uboost", "2", false)
	suite.k.SetStakingRewardParticipantIndexEntry(suite.ctx, sr.RewardId, participant.Address)

	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "hour").Return(int64(100), nil).AnyTimes()
	//boost pending paid inside the exit tx: 500 x 2 = 1000
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(1000)))).
		Return(nil).
		Times(1)
	//the unlocked stake
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	msg := &types.MsgExitStaking{Creator: creator.String(), RewardId: sr.RewardId}
	_, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().False(suite.k.HasStakingRewardParticipantIndexEntry(suite.ctx, sr.RewardId, creator.String()))
	suite.Require().Empty(suite.k.GetAllBoostParticipant(suite.ctx))
}

// TestSrParticipantIndex_InitGenesisRebuilds covers C5: the index is not part
// of the genesis file; InitGenesis rebuilds it exactly from the imported
// participant list.
func (suite *IntegrationTestSuite) TestSrParticipantIndex_InitGenesisRebuilds() {
	sr, participant := suite.seedRewardAndParticipant("000000000001", "100")
	second := types.StakingRewardParticipant{
		Address:  sdk.AccAddress("second.").String(),
		RewardId: sr.RewardId,
		Amount:   "50",
		JoinedAt: sr.DistributedStake,
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, second)

	exported := rewards.ExportGenesis(suite.ctx, *suite.k)
	suite.Require().ElementsMatch(
		[]types.StakingRewardParticipant{participant, second},
		exported.StakingRewardParticipantList,
	)

	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()
	k2, ctx2 := keepertest.RewardsKeeper(
		suite.T(), testutil.NewMockBankKeeper(ctrl), testutil.NewMockEpochKeeper(ctrl), testutil.NewMockTradingKeeper(ctrl), testutil.NewMockAccountKeeper(ctrl),
	)
	rewards.InitGenesis(ctx2, k2, *exported)

	//exactly the imported participants, nothing else
	suite.Require().ElementsMatch(
		[]string{participant.Address, second.Address},
		k2.GetStakingRewardParticipantIndexAddresses(ctx2, sr.RewardId, "", 100),
	)
}
