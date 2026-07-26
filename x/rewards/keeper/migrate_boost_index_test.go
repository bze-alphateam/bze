package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestMigrate4to5_A5_PreUpgradeParticipantsSweptAfterBackfill is the A5
// regression: participants that joined before the boosts upgrade have a
// StakingRewardParticipant record but NO reverse-index entry, so the
// finalization sweep cannot see them and they would silently lose their boost
// payout. After the v4->v5 migration backfills the index they must be reachable
// by the sweep and actually get paid.
func (suite *IntegrationTestSuite) TestMigrate4to5_A5_PreUpgradeParticipantsSweptAfterBackfill() {
	rewardId := "000000000001"
	suite.seedCleanupReward(rewardId)

	// finalizing boost with s_final = 2
	suite.k.SetBoost(suite.ctx, boostRecord(rewardId, "uvdl", 7, "2", 0))

	// pre-upgrade participants: record set WITHOUT the reverse index entry.
	addr1 := sdk.AccAddress("a5-pre-addr-1")
	addr2 := sdk.AccAddress("a5-pre-addr-2")
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: addr1.String(), RewardId: rewardId, Amount: "100", JoinedAt: "0",
	})
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: addr2.String(), RewardId: rewardId, Amount: "50", JoinedAt: "0",
	})

	// before the migration they are invisible to the sweep (the A5 bug)
	suite.Require().Empty(suite.k.GetBoostParticipantsFromCursor(suite.ctx, rewardId, "", 0))

	// run the v4->v5 migration through the keeper wiring
	suite.Require().NoError(keeper.NewMigrator(*suite.k, nil).Migrate4to5(suite.ctx))

	// now both are reachable, exactly once each
	addresses := suite.k.GetBoostParticipantsFromCursor(suite.ctx, rewardId, "", 0)
	suite.Require().ElementsMatch([]string{addr1.String(), addr2.String()}, addresses)

	// and the finalization sweep actually pays them: 100×2=200, 50×2=100
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin("uvdl", math.NewInt(200))),
	).Return(nil).Times(1)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin("uvdl", math.NewInt(100))),
	).Return(nil).Times(1)

	resp, err := suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: addr1.String(), RewardId: rewardId, Limit: 200,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(2), resp.Processed)
	suite.Require().True(resp.Completed)

	// migration also set the boost params to their defaults
	params := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(types.DefaultCreateBoostFee, params.CreateBoostFee)
	suite.Require().Equal(types.DefaultMaxBoostsPerReward, params.MaxBoostsPerReward)
}

// TestMigrate4to5_Idempotent verifies re-running the migration is a no-op: the
// reverse index still holds exactly one entry per participant.
func (suite *IntegrationTestSuite) TestMigrate4to5_Idempotent() {
	rewardId := "000000000007"
	addr := sdk.AccAddress("a5-idem-addr-1")
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: addr.String(), RewardId: rewardId, Amount: "10", JoinedAt: "0",
	})

	migrator := keeper.NewMigrator(*suite.k, nil)
	suite.Require().NoError(migrator.Migrate4to5(suite.ctx))
	suite.Require().NoError(migrator.Migrate4to5(suite.ctx))

	addresses := suite.k.GetBoostParticipantsFromCursor(suite.ctx, rewardId, "", 0)
	suite.Require().Equal([]string{addr.String()}, addresses)
}
