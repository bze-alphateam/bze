package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// seedOrphanedStamps writes n boost participant stamps referencing boosts
// that do not exist — the state a dormant participant accumulates as cleanup
// cycles stamp them and then delete the boost records. Ids start above the
// live boosts' so the fixture mirrors reality (orphans come from earlier,
// already-recycled slots, but any id works — only the record's absence
// matters).
func (suite *IntegrationTestSuite) seedOrphanedStamps(address, rewardId string, n int) {
	for i := 0; i < n; i++ {
		suite.k.SetBoostParticipant(suite.ctx, types.BoostParticipant{
			Address: address, RewardId: rewardId, BoostId: fmt.Sprintf("%012d", 100+i), JoinedAt: "1",
		})
	}
}

// TestOrphanPurge_BoundedAndConverges: a settle scans at most the bound,
// reaps only orphans, leaves live stamps intact, and repeated settles
// converge to zero orphans.
func (suite *IntegrationTestSuite) TestOrphanPurge_BoundedAndConverges() {
	creator := sdk.AccAddress("creator")
	sr, participant := suite.seedRewardAndParticipant("000000000001", "100")

	//live boost stamped at its accumulator: zero pending, must survive purges
	live := suite.setBoostWithAccumulator(sr.RewardId, "000000000001", "uboost", "1", false)
	suite.k.SetBoostParticipant(suite.ctx, types.BoostParticipant{
		Address: participant.Address, RewardId: sr.RewardId, BoostId: live.Id, JoinedAt: "1",
	})

	//1.5x the scan bound worth of orphans
	orphans := keeper.MaxOrphanedBoostParticipantScan + keeper.MaxOrphanedBoostParticipantScan/2
	suite.seedOrphanedStamps(participant.Address, sr.RewardId, orphans)
	suite.Require().Len(suite.k.GetAllBoostParticipant(suite.ctx), orphans+1)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(2)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(50)))).
		Return(nil).
		Times(2)

	//first settle (top-up join) scans the bound: the live stamp plus
	//(bound - 1) orphans — exactly (bound - 1) entries reaped
	_, err := suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{
		Creator: participant.Address, RewardId: sr.RewardId, Amount: "50",
	})
	suite.Require().NoError(err)
	remaining := orphans + 1 - (keeper.MaxOrphanedBoostParticipantScan - 1)
	suite.Require().Len(suite.k.GetAllBoostParticipant(suite.ctx), remaining)

	//second settle reaps the rest; only the live stamp survives
	_, err = suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{
		Creator: participant.Address, RewardId: sr.RewardId, Amount: "50",
	})
	suite.Require().NoError(err)
	suite.Require().Len(suite.k.GetAllBoostParticipant(suite.ctx), 1)
	_, found := suite.k.GetBoostParticipant(suite.ctx, participant.Address, sr.RewardId, live.Id)
	suite.Require().True(found)
}

// TestExitStaking_HeavyOrphanHistoryBounded: an exit under a heavy orphan
// backlog does bounded work — live stamps go via exact-key deletes, one
// bounded purge runs, and the leftover orphans stay behind as inert state
// that genesis export already filters out.
func (suite *IntegrationTestSuite) TestExitStaking_HeavyOrphanHistoryBounded() {
	creator := sdk.AccAddress("creator")
	sr, participant := suite.seedRewardAndParticipant("000000000001", "500")
	suite.k.SetStakingRewardParticipantIndexEntry(suite.ctx, sr.RewardId, participant.Address)

	//live boost stamped at its accumulator: nothing pending at exit
	live := suite.setBoostWithAccumulator(sr.RewardId, "000000000001", "uboost", "2", false)
	suite.k.SetBoostParticipant(suite.ctx, types.BoostParticipant{
		Address: participant.Address, RewardId: sr.RewardId, BoostId: live.Id, JoinedAt: "2",
	})

	orphans := keeper.MaxOrphanedBoostParticipantScan + keeper.MaxOrphanedBoostParticipantScan/2
	suite.seedOrphanedStamps(participant.Address, sr.RewardId, orphans)

	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "hour").Return(int64(100), nil).AnyTimes()
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	_, err := suite.msgServer.ExitStaking(suite.ctx, &types.MsgExitStaking{Creator: creator.String(), RewardId: sr.RewardId})
	suite.Require().NoError(err)

	//the live stamp went via the exact-key delete
	_, found := suite.k.GetBoostParticipant(suite.ctx, participant.Address, sr.RewardId, live.Id)
	suite.Require().False(found)
	//one bounded purge ran: (bound - 1) orphans reaped, the rest is residue
	residue := orphans - (keeper.MaxOrphanedBoostParticipantScan - 1)
	suite.Require().Len(suite.k.GetAllBoostParticipant(suite.ctx), residue)
	//the residue is invisible to genesis export
	suite.Require().Empty(suite.k.GetAllLiveBoostParticipant(suite.ctx))
	//and the exit itself completed fully
	_, found = suite.k.GetStakingRewardParticipant(suite.ctx, participant.Address, sr.RewardId)
	suite.Require().False(found)
	suite.Require().False(suite.k.HasStakingRewardParticipantIndexEntry(suite.ctx, sr.RewardId, participant.Address))
}
