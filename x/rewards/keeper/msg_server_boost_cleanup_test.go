package keeper_test

import (
	"fmt"
	"sort"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/testutil/sample"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// seedIndexedParticipant stores a participant and its reverse-index entry —
// the state JoinStaking leaves behind. JoinedAt matches the seeded reward's
// accumulator so base pending stays zero and payouts are boost-driven only.
func (suite *IntegrationTestSuite) seedIndexedParticipant(rewardId, address, amount, joinedAt string) {
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: address, RewardId: rewardId, Amount: amount, JoinedAt: joinedAt,
	})
	suite.k.SetStakingRewardParticipantIndexEntry(suite.ctx, rewardId, address)
}

// sortedSampleAddresses returns n random bech32 addresses in index key order.
func sortedSampleAddresses(n int) []string {
	addrs := make([]string, n)
	for i := range addrs {
		addrs[i] = sample.AccAddress()
	}
	sort.Strings(addrs)

	return addrs
}

// TestCleanupBoost_MissingBoostErrors covers C4's missing-record half: a
// cleanup on a boost that does not exist errors cleanly.
func (suite *IntegrationTestSuite) TestCleanupBoost_MissingBoostErrors() {
	suite.seedRewardAndParticipant("000000000001", "100")

	msg := types.NewMsgCleanupBoost(sample.AccAddress(), "000000000001", "000000000001", 0)
	response, err := suite.msgServer.CleanupBoost(suite.ctx, msg)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrInvalidBoostId)
}

// TestCleanupBoost_ActiveBoostRejected covers C4: cleaning up an active boost
// would strand its future accrual and escrow — it must be rejected.
func (suite *IntegrationTestSuite) TestCleanupBoost_ActiveBoostRejected() {
	suite.seedRewardAndParticipant("000000000001", "100")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", false)

	msg := types.NewMsgCleanupBoost(sample.AccAddress(), "000000000001", "000000000001", 0)
	response, err := suite.msgServer.CleanupBoost(suite.ctx, msg)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrBoostNotFinished)

	_, found := suite.k.GetBoost(suite.ctx, "000000000001", "000000000001")
	suite.Require().True(found)
}

// TestCleanupBoost_SweepPaysDormantAndMidRunSettler: a dormant pre-boost
// staker (no entry, S0 = 0) is paid exactly amount x s_final; a participant
// with a stored baseline exactly the remainder. Both entries end stamped at
// s_final and no index entry is deleted (C1's stamp-never-delete rule).
func (suite *IntegrationTestSuite) TestCleanupBoost_SweepPaysDormantAndMidRunSettler() {
	addrs := sortedSampleAddresses(2)
	sr, _ := suite.seedRewardAndParticipant("000000000001", "100")
	suite.seedIndexedParticipant(sr.RewardId, addrs[0], "100", sr.DistributedStake)
	suite.seedIndexedParticipant(sr.RewardId, addrs[1], "200", sr.DistributedStake)
	boost := suite.setBoostWithAccumulator(sr.RewardId, "000000000001", "uboost", "2", true)
	//addrs[1] settled mid-run at accumulator 1.5
	suite.k.SetBoostParticipant(suite.ctx, types.BoostParticipant{
		Address: addrs[1], RewardId: sr.RewardId, BoostId: boost.Id, JoinedAt: "1.5",
	})

	//dormant: 100 x (2 - 0) = 200; mid-run settler: 200 x (2 - 1.5) = 100
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sdk.MustAccAddressFromBech32(addrs[0]), sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(200)))).
		Return(nil).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sdk.MustAccAddressFromBech32(addrs[1]), sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(100)))).
		Return(nil).
		Times(1)

	msg := types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 0)
	response, err := suite.msgServer.CleanupBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(2), response.Processed)
	suite.Require().True(response.Completed)

	//the record is gone, the entries are stamped, the index is untouched
	_, found := suite.k.GetBoost(suite.ctx, sr.RewardId, boost.Id)
	suite.Require().False(found)
	for _, addr := range addrs {
		entry, entryFound := suite.k.GetBoostParticipant(suite.ctx, addr, sr.RewardId, boost.Id)
		suite.Require().True(entryFound)
		suite.Require().Equal("2", entry.JoinedAt)
		suite.Require().True(suite.k.HasStakingRewardParticipantIndexEntry(suite.ctx, sr.RewardId, addr))
	}
}

// TestCleanupBoost_C1_SweptUserClaimPaysZero covers C1: a swept user claiming
// while the boost record still exists is paid nothing the second time — the
// sweep stamped (never deleted) their entry.
func (suite *IntegrationTestSuite) TestCleanupBoost_C1_SweptUserClaimPaysZero() {
	addrs := sortedSampleAddresses(2)
	sr, _ := suite.seedRewardAndParticipant("000000000001", "100")
	for _, addr := range addrs {
		suite.seedIndexedParticipant(sr.RewardId, addr, "100", sr.DistributedStake)
	}
	boost := suite.setBoostWithAccumulator(sr.RewardId, "000000000001", "uboost", "2", true)

	//the sweep's payout is the only transfer this test allows
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sdk.MustAccAddressFromBech32(addrs[0]), sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(200)))).
		Return(nil).
		Times(1)

	msg := types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 1)
	response, err := suite.msgServer.CleanupBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(1), response.Processed)
	suite.Require().False(response.Completed)
	_, found := suite.k.GetBoost(suite.ctx, sr.RewardId, boost.Id)
	suite.Require().True(found)

	//the swept user's claim finds nothing pending anywhere
	claim := &types.MsgClaimStakingRewards{Creator: addrs[0], RewardId: sr.RewardId}
	_, err = suite.msgServer.ClaimStakingRewards(suite.ctx, claim)
	suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim)
}

// TestCleanupBoost_C3_LimitOneConverges covers C3: sequential small-limit
// calls (same-block cranks) never double-pay, never skip, persist the cursor
// between calls and converge to completion.
func (suite *IntegrationTestSuite) TestCleanupBoost_C3_LimitOneConverges() {
	addrs := sortedSampleAddresses(3)
	sr, _ := suite.seedRewardAndParticipant("000000000001", "100")
	for _, addr := range addrs {
		suite.seedIndexedParticipant(sr.RewardId, addr, "100", sr.DistributedStake)
		//each address is paid exactly once across the whole run
		suite.bank.EXPECT().
			SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sdk.MustAccAddressFromBech32(addr), sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(200)))).
			Return(nil).
			Times(1)
	}
	boost := suite.setBoostWithAccumulator(sr.RewardId, "000000000001", "uboost", "2", true)

	for i := range addrs {
		response, err := suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 1))
		suite.Require().NoError(err)
		suite.Require().Equal(uint32(1), response.Processed)
		suite.Require().False(response.Completed)

		stored, found := suite.k.GetBoost(suite.ctx, sr.RewardId, boost.Id)
		suite.Require().True(found)
		suite.Require().Equal(addrs[i], stored.CleanupCursor)
	}

	//the exhausted iteration completes as a no-op and deletes the record
	response, err := suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 1))
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(0), response.Processed)
	suite.Require().True(response.Completed)
	_, found := suite.k.GetBoost(suite.ctx, sr.RewardId, boost.Id)
	suite.Require().False(found)
}

// TestCleanupBoost_C6_LimitClamp covers C6: limit 0 uses the param, a limit
// above the param is clamped to it, a limit within bounds is honored exactly.
func (suite *IntegrationTestSuite) TestCleanupBoost_C6_LimitClamp() {
	params := types.DefaultParams()
	params.CleanupBatchSize = 2
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	addrs := sortedSampleAddresses(5)
	sr, _ := suite.seedRewardAndParticipant("000000000001", "100")
	for _, addr := range addrs {
		suite.seedIndexedParticipant(sr.RewardId, addr, "100", sr.DistributedStake)
		suite.bank.EXPECT().
			SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sdk.MustAccAddressFromBech32(addr), sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(200)))).
			Return(nil).
			Times(1)
	}
	boost := suite.setBoostWithAccumulator(sr.RewardId, "000000000001", "uboost", "2", true)

	//limit 0 -> the param default (2)
	response, err := suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 0))
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(2), response.Processed)

	//limit above the param -> clamped to the param, never the caller's value
	response, err = suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 10))
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(2), response.Processed)

	//limit within bounds -> honored exactly
	response, err = suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 1))
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(1), response.Processed)
	suite.Require().False(response.Completed)

	response, err = suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 0))
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(0), response.Processed)
	suite.Require().True(response.Completed)
}

// TestCleanupBoost_C7_ExtensionResetsCursorAndPaysSecondSegment covers C7:
// sweep half, extend the boost (cursor must reset), run it to finished again,
// clean up fully — the first-half participant receives the second accrual
// segment exactly, and cleanup is blocked while the re-armed boost is active.
func (suite *IntegrationTestSuite) TestCleanupBoost_C7_ExtensionResetsCursorAndPaysSecondSegment() {
	creator := sdk.AccAddress("creator")
	addrs := sortedSampleAddresses(2)
	sr, _ := suite.seedRewardAndParticipant("000000000001", "100")
	for _, addr := range addrs {
		suite.seedIndexedParticipant(sr.RewardId, addr, "100", sr.DistributedStake)
	}
	boost := suite.setBoostWithAccumulator(sr.RewardId, "000000000001", "uboost", "2", true)

	//first half: sweep addrs[0] only — 100 x 2 = 200
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sdk.MustAccAddressFromBech32(addrs[0]), sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(200)))).
		Return(nil).
		Times(1)
	response, err := suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 1))
	suite.Require().NoError(err)
	suite.Require().False(response.Completed)
	stored, found := suite.k.GetBoost(suite.ctx, sr.RewardId, boost.Id)
	suite.Require().True(found)
	suite.Require().Equal(addrs[0], stored.CleanupCursor)

	//extend by 5 days: escrow 5 x 1000 uboost, and the cursor must reset
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(10000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(5000)))).
		Return(nil).
		Times(1)
	_, err = suite.msgServer.UpdateBoost(suite.ctx, &types.MsgUpdateBoost{
		Creator: creator.String(), RewardId: sr.RewardId, BoostId: boost.Id, Days: "5",
	})
	suite.Require().NoError(err)
	stored, found = suite.k.GetBoost(suite.ctx, sr.RewardId, boost.Id)
	suite.Require().True(found)
	suite.Require().Equal("", stored.CleanupCursor)

	//the re-armed boost is active again: cleanup is blocked until it finishes
	_, err = suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 0))
	suite.Require().ErrorIs(err, types.ErrBoostNotFinished)

	//run the second segment to finished (accumulator 2 -> 3)
	stored.Payouts = stored.Duration
	stored.DistributedStake = "3"
	suite.k.SetBoost(suite.ctx, stored)

	//full cleanup: addrs[0] gets exactly the second segment (100 x (3 - 2)),
	//addrs[1] the full window (100 x 3)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sdk.MustAccAddressFromBech32(addrs[0]), sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(100)))).
		Return(nil).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sdk.MustAccAddressFromBech32(addrs[1]), sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(300)))).
		Return(nil).
		Times(1)
	response, err = suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 0))
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(2), response.Processed)
	suite.Require().True(response.Completed)
	_, found = suite.k.GetBoost(suite.ctx, sr.RewardId, boost.Id)
	suite.Require().False(found)
}

// TestCleanupBoost_EmptyIndexNoOpCompletes: nothing to sweep is an idempotent
// no-op success that still deletes the finished record; a repeat call errors
// like any other missing boost.
func (suite *IntegrationTestSuite) TestCleanupBoost_EmptyIndexNoOpCompletes() {
	suite.seedStakingReward("000000000001", 30, 30)
	boost := suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", true)

	response, err := suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), "000000000001", boost.Id, 0))
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(0), response.Processed)
	suite.Require().True(response.Completed)
	_, found := suite.k.GetBoost(suite.ctx, "000000000001", boost.Id)
	suite.Require().False(found)

	_, err = suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), "000000000001", boost.Id, 0))
	suite.Require().ErrorIs(err, types.ErrInvalidBoostId)
}

// TestBoostCleanupStatus_Query: remaining is correct before, during and at
// the end of a sweep, and the deleted record is a clean NotFound.
func (suite *IntegrationTestSuite) TestBoostCleanupStatus_Query() {
	addrs := sortedSampleAddresses(3)
	sr, _ := suite.seedRewardAndParticipant("000000000001", "100")
	for _, addr := range addrs {
		suite.seedIndexedParticipant(sr.RewardId, addr, "100", sr.DistributedStake)
		suite.bank.EXPECT().
			SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sdk.MustAccAddressFromBech32(addr), sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(200)))).
			Return(nil).
			Times(1)
	}
	boost := suite.setBoostWithAccumulator(sr.RewardId, "000000000001", "uboost", "2", true)

	status, err := suite.k.BoostCleanupStatus(suite.ctx, &types.QueryBoostCleanupStatusRequest{RewardId: sr.RewardId, BoostId: boost.Id})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(3), status.Remaining)
	suite.Require().Equal("", status.Cursor)

	_, err = suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 1))
	suite.Require().NoError(err)
	status, err = suite.k.BoostCleanupStatus(suite.ctx, &types.QueryBoostCleanupStatusRequest{RewardId: sr.RewardId, BoostId: boost.Id})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(2), status.Remaining)
	suite.Require().Equal(addrs[0], status.Cursor)

	//an exact-limit batch covers the tail: everything swept, record still
	//stored, remaining is zero
	response, err := suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 2))
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(2), response.Processed)
	suite.Require().False(response.Completed)
	status, err = suite.k.BoostCleanupStatus(suite.ctx, &types.QueryBoostCleanupStatusRequest{RewardId: sr.RewardId, BoostId: boost.Id})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(0), status.Remaining)
	suite.Require().Equal(addrs[2], status.Cursor)

	//completion deletes the record: the query NotFounds like any other
	//missing boost
	_, err = suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 0))
	suite.Require().NoError(err)
	_, err = suite.k.BoostCleanupStatus(suite.ctx, &types.QueryBoostCleanupStatusRequest{RewardId: sr.RewardId, BoostId: boost.Id})
	suite.Require().Error(err)
}

// TestCleanupBoost_FreesBoostSlot: creation at the cap fails, succeeds
// immediately after a cleanup completes (the cap counts existing records).
func (suite *IntegrationTestSuite) TestCleanupBoost_FreesBoostSlot() {
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 1)
	creator := sdk.AccAddress("creator")
	sr, _ := suite.seedRewardAndParticipant("000000000001", "100")
	suite.k.SetBoostsCounter(suite.ctx, 5)
	boost := suite.setBoostWithAccumulator(sr.RewardId, "000000000001", "uboost", "2", true)

	suite.bank.EXPECT().HasSupply(suite.ctx, "uboost").Return(true).AnyTimes()

	createMsg := &types.MsgCreateBoost{
		Creator: creator.String(), RewardId: sr.RewardId, Denom: "uboost", DailyAmount: "100", Days: "5",
	}
	_, err := suite.msgServer.CreateBoost(suite.ctx, createMsg)
	suite.Require().ErrorIs(err, types.ErrBoostCapReached)

	response, err := suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), sr.RewardId, boost.Id, 0))
	suite.Require().NoError(err)
	suite.Require().True(response.Completed)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(10000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(500)))).
		Return(nil).
		Times(1)
	created, err := suite.msgServer.CreateBoost(suite.ctx, createMsg)
	suite.Require().NoError(err)
	//the freed slot is reused but the id is fresh — orphans can never be
	//misread by the new boost
	suite.Require().Equal("000000000006", created.BoostId)
}

// TestCleanupBoost_OrphanedEntriesRemovedAtNextSettle: entries referencing a
// deleted boost are dropped by the owner's next settle (a top-up join here).
func (suite *IntegrationTestSuite) TestCleanupBoost_OrphanedEntriesRemovedAtNextSettle() {
	creator := sdk.AccAddress("creator")
	sr, participant := suite.seedRewardAndParticipant("000000000001", "100")
	//orphan: no boost 000000000009 exists
	suite.k.SetBoostParticipant(suite.ctx, types.BoostParticipant{
		Address: participant.Address, RewardId: sr.RewardId, BoostId: "000000000009", JoinedAt: "2",
	})

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(50)))).
		Return(nil).
		Times(1)
	_, err := suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{
		Creator: participant.Address, RewardId: sr.RewardId, Amount: "50",
	})
	suite.Require().NoError(err)

	_, found := suite.k.GetBoostParticipant(suite.ctx, participant.Address, sr.RewardId, "000000000009")
	suite.Require().False(found)
}

// TestExportGenesis_ExcludesOrphansAndKeepsCursor: orphaned entries are
// filtered from the export, a mid-cleanup boost exports its cursor, and the
// exported state passes genesis validation.
func (suite *IntegrationTestSuite) TestExportGenesis_ExcludesOrphansAndKeepsCursor() {
	addr := sample.AccAddress()
	suite.seedStakingReward("000000000001", 30, 30)
	suite.k.SetBoostsCounter(suite.ctx, 1)
	boost := suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", true)
	boost.CleanupCursor = addr
	suite.k.SetBoost(suite.ctx, boost)

	live := types.BoostParticipant{Address: addr, RewardId: "000000000001", BoostId: boost.Id, JoinedAt: "2"}
	suite.k.SetBoostParticipant(suite.ctx, live)
	//orphan: its boost was cleaned up
	suite.k.SetBoostParticipant(suite.ctx, types.BoostParticipant{
		Address: addr, RewardId: "000000000001", BoostId: "000000000099", JoinedAt: "2",
	})

	exported := rewards.ExportGenesis(suite.ctx, *suite.k)
	suite.Require().Equal([]types.BoostParticipant{live}, exported.BoostParticipantList)
	suite.Require().Len(exported.BoostList, 1)
	suite.Require().Equal(addr, exported.BoostList[0].CleanupCursor)
	suite.Require().NoError(exported.Validate())
}

// TestBoostSecurity_EscrowConservationWithCleanup extends the I5 suite: a
// full lifecycle driven through the public entry points that includes a
// partial cleanup sweep, a mid-sweep extension (re-arm) and a final full
// sweep — everything escrowed leaves again except truncation dust.
func (suite *IntegrationTestSuite) TestBoostSecurity_EscrowConservationWithCleanup() {
	ledger := recordBankFlows(suite.bank, suite.epoch)
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)

	rewardId := "000000000001"
	suite.seedLifecycleReward(rewardId, 10)

	participants := []sdk.AccAddress{sdk.AccAddress("cleanup-addr-00"), sdk.AccAddress("cleanup-addr-01")}
	stakes := []int64{100, 300}
	for i, p := range participants {
		suite.joinStaking(p, rewardId, stakes[i])
	}

	booster := sdk.AccAddress("booster")
	_, err := suite.msgServer.CreateBoost(suite.ctx, &types.MsgCreateBoost{
		Creator: booster.String(), RewardId: rewardId, Denom: "uboost", DailyAmount: "1000", Days: "3",
	})
	suite.Require().NoError(err)
	boostId := "000000000001"

	for day := 0; day < 3; day++ {
		suite.distributeOneDay()
	}
	stored, found := suite.k.GetBoost(suite.ctx, rewardId, boostId)
	suite.Require().True(found)
	suite.Require().Equal(stored.Duration, stored.Payouts)

	//partial sweep, then a mid-sweep re-arm: cursor resets and further
	//cleanup is blocked while the boost is active again
	response, err := suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), rewardId, boostId, 1))
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(1), response.Processed)
	suite.Require().False(response.Completed)

	_, err = suite.msgServer.UpdateBoost(suite.ctx, &types.MsgUpdateBoost{
		Creator: booster.String(), RewardId: rewardId, BoostId: boostId, Days: "2",
	})
	suite.Require().NoError(err)
	_, err = suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), rewardId, boostId, 0))
	suite.Require().ErrorIs(err, types.ErrBoostNotFinished)

	for day := 0; day < 2; day++ {
		suite.distributeOneDay()
	}

	response, err = suite.msgServer.CleanupBoost(suite.ctx, types.NewMsgCleanupBoost(sample.AccAddress(), rewardId, boostId, 0))
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(2), response.Processed)
	suite.Require().True(response.Completed)

	for _, p := range participants {
		suite.exitStaking(p, rewardId)
	}

	//every piece of boost state is gone: the record via cleanup, the stamped
	//entries via the exits' prefix delete
	suite.Require().Empty(suite.k.GetAllBoosts(suite.ctx))
	suite.Require().Empty(suite.k.GetAllBoostParticipant(suite.ctx))

	//I5: escrow captured exactly (3 + 2 days x 1000), and everything left
	//again except truncation dust (3 sweep settles truncate < 1 unit each)
	escrowed := ledger.in.AmountOf("uboost")
	suite.Require().True(escrowed.Equal(math.NewInt(5000)), fmt.Sprintf("escrowed %s", escrowed))
	residue := escrowed.Sub(ledger.out.AmountOf("uboost"))
	suite.Require().True(residue.GTE(math.ZeroInt()), fmt.Sprintf("residue %s negative", residue))
	suite.Require().True(residue.LTE(math.NewInt(3)), fmt.Sprintf("residue %s exceeds dust bound", residue))

	//stakes conserve exactly
	suite.Require().True(ledger.in.AmountOf("ubze").Equal(ledger.out.AmountOf("ubze")))
}
