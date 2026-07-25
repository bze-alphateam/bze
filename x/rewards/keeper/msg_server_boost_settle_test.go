package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// boostRecord builds a boost with the given accumulator and remaining days.
func boostRecord(rewardId, denom string, uid uint64, sBoost string, daysLeft uint32) types.Boost {
	return types.Boost{
		RewardId: rewardId, Denom: denom, Uid: uid,
		DailyAmount: "1000", DaysLeft: daysLeft, SBoost: sBoost,
	}
}

// TestBoostSettle_A2_TopUpPaysPreTopUpAmount is the A2 regression: a top-up must
// settle boosts on the PRE-top-up stake. If settle ran after the amount grew, a
// whale that stakes dust then tops up 1000x would drain the whole accrual window
// at the new amount. Here the participant holds 10, then tops up by 90; the boost
// (s_boost=1) must pay 10 (= 10 * 1), never 100.
func (suite *IntegrationTestSuite) TestBoostSettle_A2_TopUpPaysPreTopUpAmount() {
	creator := sdk.AccAddress("creator")
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId: "000000000001", PrizeAmount: "1000", PrizeDenom: "ubze",
		StakingDenom: "ubze", Duration: 5, Payouts: 0, MinStake: 1, Lock: 7,
		StakedAmount: "10", DistributedStake: "0",
	})
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: creator.String(), RewardId: "000000000001", Amount: "10", JoinedAt: "0",
	})
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "ubze", 1, "1", 10))

	suite.bank.EXPECT().SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000000)))).Times(1)
	// boost payout must be at the PRE-top-up amount (10), not 100
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10))),
	).Return(nil).Times(1)
	// the top-up capture of 90
	suite.bank.EXPECT().SendCoinsFromAccountToModule(
		suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(90))),
	).Return(nil).Times(1)

	_, err := suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{
		Creator: creator.String(), RewardId: "000000000001", Amount: "90",
	})
	suite.Require().NoError(err)

	p, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().True(found)
	suite.Require().Equal("100", p.Amount)
	// snapshot reset to current s_boost so the added stake earns nothing retroactively
	suite.Require().Equal(uint64(1), p.BoostSnapshots["ubze"].Uid)
	suite.Require().Equal("1", p.BoostSnapshots["ubze"].S0)
	// reverse index recorded
	suite.Require().True(suite.k.HasBoostParticipantIndex(suite.ctx, "000000000001", creator.String()))
}

// TestBoostSettle_A3_JoinDuringFinalizationWritesSnapshot is the A3 regression:
// joining while a boost is finalizing (days_left == 0) must write a snapshot
// (uid, s_final) so the joiner earns nothing from it. Omitting finalizing records
// would let the sweep pay amount * s_final for zero staking time.
func (suite *IntegrationTestSuite) TestBoostSettle_A3_JoinDuringFinalizationWritesSnapshot() {
	creator := sdk.AccAddress("creator")
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId: "000000000001", PrizeAmount: "1000", PrizeDenom: "ubze",
		StakingDenom: "ubze", Duration: 5, Payouts: 0, MinStake: 1, Lock: 7,
		StakedAmount: "0", DistributedStake: "0",
	})
	// finalizing boost: days_left == 0, s_boost = 5
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "ubze", 1, "5", 0))

	suite.bank.EXPECT().SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000000)))).Times(1)
	// only the join capture — the new joiner receives NO boost payout
	suite.bank.EXPECT().SendCoinsFromAccountToModule(
		suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100))),
	).Return(nil).Times(1)

	_, err := suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{
		Creator: creator.String(), RewardId: "000000000001", Amount: "100",
	})
	suite.Require().NoError(err)

	p, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(uint64(1), p.BoostSnapshots["ubze"].Uid)
	suite.Require().Equal("5", p.BoostSnapshots["ubze"].S0) // s_final -> earns nothing
}

// TestBoostSettle_A7_ExitPaysPendingAndRemovesIndex is the A7 regression: exit
// settles boosts before the participant record is removed (pendings paid, not
// forfeited) and drops the reverse-index entry.
func (suite *IntegrationTestSuite) TestBoostSettle_A7_ExitPaysPendingAndRemovesIndex() {
	creator := sdk.AccAddress("creator")
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId: "000000000001", PrizeAmount: "1000", PrizeDenom: "ubze",
		StakingDenom: "ubze", Duration: 5, Payouts: 0, MinStake: 1, Lock: 0,
		StakedAmount: "10", DistributedStake: "0",
	})
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: creator.String(), RewardId: "000000000001", Amount: "10", JoinedAt: "0",
	})
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "ubze", 1, "2", 10))
	suite.k.SetBoostParticipantIndex(suite.ctx, "000000000001", creator.String())

	// staked amount unlocked (lock = 0)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10))),
	).Return(nil).Times(1)
	// boost pending paid at exit (10 * 2)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(20))),
	).Return(nil).Times(1)

	_, err := suite.msgServer.ExitStaking(suite.ctx, &types.MsgExitStaking{
		Creator: creator.String(), RewardId: "000000000001",
	})
	suite.Require().NoError(err)

	_, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().False(found)
	suite.Require().False(suite.k.HasBoostParticipantIndex(suite.ctx, "000000000001", creator.String()))
}

// TestBoostSettle_ClaimSettlesAndIsIdempotent: a claim pays boost pendings even
// when the base reward has nothing to pay, and a repeat claim pays zero (A4-style
// idempotency — the snapshot advanced to the current s_boost on the first claim).
func (suite *IntegrationTestSuite) TestBoostSettle_ClaimSettlesAndIsIdempotent() {
	creator := sdk.AccAddress("creator")
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId: "000000000001", PrizeAmount: "1000", PrizeDenom: "ubze",
		StakingDenom: "ubze", Duration: 5, Payouts: 0, MinStake: 1, Lock: 7,
		StakedAmount: "10", DistributedStake: "0",
	})
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: creator.String(), RewardId: "000000000001", Amount: "10", JoinedAt: "0",
	})
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "ubze", 1, "2", 10))

	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(20))),
	).Return(nil).Times(1)

	// first claim: base pays nothing, boost pays 20
	_, err := suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{
		Creator: creator.String(), RewardId: "000000000001",
	})
	suite.Require().NoError(err)

	p, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().True(found)
	suite.Require().Equal("2", p.BoostSnapshots["ubze"].S0)

	// second claim: nothing left to pay -> ErrNoRewardsToClaim, no further send
	_, err = suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{
		Creator: creator.String(), RewardId: "000000000001",
	})
	suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim)
}

// TestBoostSettle_A10_HooksNotApplicable documents that the A10 acceptance
// criterion (BZE-64 StakingRewardHooks preservation) cannot be tested against
// this base: BZE-64 is NOT merged into main / the BZE-73 foundation branch — no
// StakingRewardHooks / SetHooks / AfterJoin exists in x/rewards. The boost
// insertions therefore cannot break or reorder hooks that are not present. If
// BZE-64 lands before this ships, add the recording-mock hook test then. Flagged
// to Stefan in the Jira comment.
func TestBoostSettle_A10_HooksNotApplicable(t *testing.T) {
	t.Skip("BZE-64 StakingRewardHooks not present in this base; A10 hook-preservation test is N/A (see Jira)")
}
