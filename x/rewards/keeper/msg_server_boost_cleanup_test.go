package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) boostFinalizeEventCount() (count int) {
	for _, ev := range suite.ctx.EventManager().Events() {
		if ev.Type == "bze.rewards.BoostFinalizeEvent" {
			count++
		}
	}
	return
}

func (suite *IntegrationTestSuite) seedCleanupReward(rewardId string) {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId: rewardId, PrizeAmount: "1000", PrizeDenom: "ubze",
		StakingDenom: "ubze", Duration: 5, Payouts: 5, MinStake: 1, Lock: 7,
		StakedAmount: "1000", DistributedStake: "0",
	})
}

// seedCleanupParticipant registers a participant with the given stake and
// reverse-index entry. joined_at matches the reward's DistributedStake so the
// base reward owes nothing.
func (suite *IntegrationTestSuite) seedCleanupParticipant(rewardId string, addr sdk.AccAddress, amount string) {
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: addr.String(), RewardId: rewardId, Amount: amount, JoinedAt: "0",
	})
	suite.k.SetBoostParticipantIndex(suite.ctx, rewardId, addr.String())
}

func (suite *IntegrationTestSuite) TestCleanupBoost_SweepPaysAgainstFinalSBoost() {
	rewardId := "000000000001"
	suite.seedCleanupReward(rewardId)

	// finalizing boost: s_final = 2
	suite.k.SetBoost(suite.ctx, boostRecord(rewardId, "uvdl", 7, "2", 0))

	// three participants covering the semantics table:
	// matching snapshot (S0=0.5), stale-uid snapshot (S0 treated as 0), no snapshot
	addr1 := sdk.AccAddress("cleanup-addr-1")
	addr2 := sdk.AccAddress("cleanup-addr-2")
	addr3 := sdk.AccAddress("cleanup-addr-3")
	suite.seedCleanupParticipant(rewardId, addr1, "100")
	p1, _ := suite.k.GetStakingRewardParticipant(suite.ctx, addr1.String(), rewardId)
	p1.BoostSnapshots = map[string]*types.BoostSnapshot{"uvdl": {Uid: 7, S0: "0.5"}}
	suite.k.SetStakingRewardParticipant(suite.ctx, p1)

	suite.seedCleanupParticipant(rewardId, addr2, "40")
	p2, _ := suite.k.GetStakingRewardParticipant(suite.ctx, addr2.String(), rewardId)
	p2.BoostSnapshots = map[string]*types.BoostSnapshot{"uvdl": {Uid: 3, S0: "1.9"}} // stale uid
	suite.k.SetStakingRewardParticipant(suite.ctx, p2)

	suite.seedCleanupParticipant(rewardId, addr3, "10")

	// exact payouts: 100×(2−0.5)=150, 40×(2−0)=80, 10×2=20
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin("uvdl", math.NewInt(150))),
	).Return(nil).Times(1)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin("uvdl", math.NewInt(80))),
	).Return(nil).Times(1)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, addr3, sdk.NewCoins(sdk.NewCoin("uvdl", math.NewInt(20))),
	).Return(nil).Times(1)

	resp, err := suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: addr1.String(), RewardId: rewardId, Limit: 200,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(3), resp.Processed)
	suite.Require().True(resp.Completed)

	// completion deletes the record and emits the finalize event
	_, found := suite.k.GetBoost(suite.ctx, rewardId, "uvdl")
	suite.Require().False(found)
	suite.Require().Equal(1, suite.boostFinalizeEventCount())

	// snapshots stamped to (uid, s_final) — entries never deleted while swept
	for _, addr := range []sdk.AccAddress{addr1, addr2, addr3} {
		p, found := suite.k.GetStakingRewardParticipant(suite.ctx, addr.String(), rewardId)
		suite.Require().True(found)
		suite.Require().Equal(uint64(7), p.BoostSnapshots["uvdl"].Uid)
		suite.Require().Equal("2", p.BoostSnapshots["uvdl"].S0)
	}
}

func (suite *IntegrationTestSuite) TestCleanupBoost_PartialSweepAcrossCalls() {
	rewardId := "000000000001"
	suite.seedCleanupReward(rewardId)
	suite.k.SetBoost(suite.ctx, boostRecord(rewardId, "uvdl", 1, "1", 0))

	addr1 := sdk.AccAddress("partial-addr-1")
	addr2 := sdk.AccAddress("partial-addr-2")
	addr3 := sdk.AccAddress("partial-addr-3")
	suite.seedCleanupParticipant(rewardId, addr1, "10")
	suite.seedCleanupParticipant(rewardId, addr2, "20")
	suite.seedCleanupParticipant(rewardId, addr3, "30")

	// each participant is paid exactly once across all cranks (.Times(1))
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin("uvdl", math.NewInt(10))),
	).Return(nil).Times(1)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin("uvdl", math.NewInt(20))),
	).Return(nil).Times(1)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, addr3, sdk.NewCoins(sdk.NewCoin("uvdl", math.NewInt(30))),
	).Return(nil).Times(1)

	// crank 1: limit 1 → one participant, record survives, cursor persisted
	resp, err := suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: addr1.String(), RewardId: rewardId, Limit: 1,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(1), resp.Processed)
	suite.Require().False(resp.Completed)

	boost, found := suite.k.GetBoost(suite.ctx, rewardId, "uvdl")
	suite.Require().True(found)
	suite.Require().NotEmpty(boost.FinalizeCursor)

	// crank 2: sweeps the remaining two, deletes the record
	resp, err = suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: addr1.String(), RewardId: rewardId, Limit: 200,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(2), resp.Processed)
	suite.Require().True(resp.Completed)

	_, found = suite.k.GetBoost(suite.ctx, rewardId, "uvdl")
	suite.Require().False(found)
	suite.Require().Equal(1, suite.boostFinalizeEventCount())

	// crank 3 (A9): nothing left — no-op success, nobody is paid again
	resp, err = suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: addr1.String(), RewardId: rewardId, Limit: 200,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(0), resp.Processed)
	suite.Require().True(resp.Completed)
}

func (suite *IntegrationTestSuite) TestCleanupBoost_A4_SweptUserClaimsAgainPaysZero() {
	rewardId := "000000000001"
	suite.seedCleanupReward(rewardId)
	suite.k.SetBoost(suite.ctx, boostRecord(rewardId, "uvdl", 1, "3", 0))

	addr1 := sdk.AccAddress("a4-addr-1")
	addr2 := sdk.AccAddress("a4-addr-2")
	suite.seedCleanupParticipant(rewardId, addr1, "100")
	suite.seedCleanupParticipant(rewardId, addr2, "50")

	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin("uvdl", math.NewInt(300))),
	).Return(nil).Times(1)

	// sweep only addr1; the record must survive (addr2 still pending)
	resp, err := suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: addr1.String(), RewardId: rewardId, Limit: 1,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(1), resp.Processed)
	_, found := suite.k.GetBoost(suite.ctx, rewardId, "uvdl")
	suite.Require().True(found)

	// swept user claims while the record still exists: snapshot (uid, s_final)
	// makes the boost owe 0, base owes 0 → ErrNoRewardsToClaim and, crucially,
	// no bank transfer (an unexpected transfer would panic the mock).
	_, err = suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{
		Creator: addr1.String(), RewardId: rewardId,
	})
	suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim)
}

func (suite *IntegrationTestSuite) TestCleanupBoost_A9_ActiveBoostUntouched() {
	rewardId := "000000000001"
	suite.seedCleanupReward(rewardId)
	suite.k.SetBoost(suite.ctx, boostRecord(rewardId, "uvdl", 1, "1.5", 3)) // active

	addr := sdk.AccAddress("active-addr-1")
	suite.seedCleanupParticipant(rewardId, addr, "100")

	resp, err := suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: addr.String(), RewardId: rewardId, Limit: 200,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(0), resp.Processed)
	suite.Require().True(resp.Completed)

	// record untouched: still there, cursor untouched, no payout, no event
	boost, found := suite.k.GetBoost(suite.ctx, rewardId, "uvdl")
	suite.Require().True(found)
	suite.Require().Equal(uint32(3), boost.DaysLeft)
	suite.Require().Equal("", boost.FinalizeCursor)
	suite.Require().Equal(0, suite.boostFinalizeEventCount())
}

func (suite *IntegrationTestSuite) TestCleanupBoost_MixedActiveAndFinalizing() {
	rewardId := "000000000001"
	suite.seedCleanupReward(rewardId)
	suite.k.SetBoost(suite.ctx, boostRecord(rewardId, "uactive", 1, "1", 2)) // active
	suite.k.SetBoost(suite.ctx, boostRecord(rewardId, "udone", 2, "2", 0))   // finalizing

	addr := sdk.AccAddress("mixed-addr-1")
	suite.seedCleanupParticipant(rewardId, addr, "10")

	// only the finalizing boost pays: 10 × 2 = 20 udone, nothing in uactive
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("udone", math.NewInt(20))),
	).Return(nil).Times(1)

	resp, err := suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: addr.String(), RewardId: rewardId, Limit: 200,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(1), resp.Processed)
	suite.Require().True(resp.Completed)

	_, found := suite.k.GetBoost(suite.ctx, rewardId, "udone")
	suite.Require().False(found)
	active, found := suite.k.GetBoost(suite.ctx, rewardId, "uactive")
	suite.Require().True(found)
	suite.Require().Equal(uint32(2), active.DaysLeft)
}

func (suite *IntegrationTestSuite) TestCleanupBoost_DeletedRewardOnlyDeletes() {
	rewardId := "999999999999" // no such reward
	suite.k.SetBoost(suite.ctx, boostRecord(rewardId, "uvdl", 1, "2", 0))
	suite.k.SetBoost(suite.ctx, boostRecord(rewardId, "stone", 2, "1", 4)) // even active orphans go

	// orphaned index entries must not trigger payouts (participants provably gone)
	suite.k.SetBoostParticipantIndex(suite.ctx, rewardId, sdk.AccAddress("gone-addr-1").String())

	resp, err := suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: sdk.AccAddress("cranker").String(), RewardId: rewardId, Limit: 200,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(2), resp.Processed)
	suite.Require().True(resp.Completed)

	// store empty afterward, one finalize event per record, no bank calls made
	suite.Require().Empty(suite.k.GetRewardBoosts(suite.ctx, rewardId))
	suite.Require().Equal(2, suite.boostFinalizeEventCount())
}

func (suite *IntegrationTestSuite) TestCleanupBoost_IdempotentNoBoosts() {
	rewardId := "000000000001"
	suite.seedCleanupReward(rewardId)

	resp, err := suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: sdk.AccAddress("cranker").String(), RewardId: rewardId, Limit: 0, // clamped to 1
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(0), resp.Processed)
	suite.Require().True(resp.Completed)
}

func (suite *IntegrationTestSuite) TestCleanupBoost_ExitedParticipantSkipped() {
	rewardId := "000000000001"
	suite.seedCleanupReward(rewardId)
	suite.k.SetBoost(suite.ctx, boostRecord(rewardId, "uvdl", 1, "1", 0))

	// index entry without a participant record (defensive: exit normally removes
	// the index entry too) — must be skipped without payout and without error
	ghost := sdk.AccAddress("ghost-addr-1")
	suite.k.SetBoostParticipantIndex(suite.ctx, rewardId, ghost.String())

	addr := sdk.AccAddress("real-addr-1")
	suite.seedCleanupParticipant(rewardId, addr, "10")

	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("uvdl", math.NewInt(10))),
	).Return(nil).Times(1)

	resp, err := suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: addr.String(), RewardId: rewardId, Limit: 200,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(2), resp.Processed) // ghost counted as processed
	suite.Require().True(resp.Completed)

	_, found := suite.k.GetBoost(suite.ctx, rewardId, "uvdl")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestCleanupBoost_TruncationDust() {
	rewardId := "000000000001"
	suite.seedCleanupReward(rewardId)
	// s_final = 0.4: a 1-unit staker accrues 0.4 units — truncates to 0.
	// The sweep stamps the snapshot anyway (dust by design, unlike settleBoosts).
	suite.k.SetBoost(suite.ctx, boostRecord(rewardId, "uvdl", 1, "0.4", 0))

	addr := sdk.AccAddress("dust-addr-1")
	suite.seedCleanupParticipant(rewardId, addr, "1")

	resp, err := suite.msgServer.CleanupBoost(suite.ctx, &types.MsgCleanupBoost{
		Creator: addr.String(), RewardId: rewardId, Limit: 200,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint32(1), resp.Processed)
	suite.Require().True(resp.Completed)

	// no payout expected (mock would panic), snapshot stamped unconditionally
	p, found := suite.k.GetStakingRewardParticipant(suite.ctx, addr.String(), rewardId)
	suite.Require().True(found)
	suite.Require().Equal(uint64(1), p.BoostSnapshots["uvdl"].Uid)
	suite.Require().Equal("0.4", p.BoostSnapshots["uvdl"].S0)

	_, found = suite.k.GetBoost(suite.ctx, rewardId, "uvdl")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestCleanupBoost_NilMessage() {
	_, err := suite.msgServer.CleanupBoost(suite.ctx, nil)
	suite.Require().Error(err)
}
