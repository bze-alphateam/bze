package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The settle routine is unexported, so these unit tests drive it through the
// ClaimStakingRewards handler with a base reward that pays nothing
// (DistributedStake == JoinedAt), which isolates the boost settle behaviour.

// seedZeroBaseReward stores a reward whose base claim always pays zero.
func (suite *IntegrationTestSuite) seedZeroBaseReward(id string) {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId: id, PrizeAmount: "1000", PrizeDenom: "ubze", StakingDenom: "ubze",
		Duration: 5, Payouts: 0, MinStake: 1, Lock: 7, StakedAmount: "100", DistributedStake: "0",
	})
}

func (suite *IntegrationTestSuite) claim(creator sdk.AccAddress, rewardId string) error {
	_, err := suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{
		Creator: creator.String(), RewardId: rewardId,
	})
	return err
}

func (suite *IntegrationTestSuite) expectBoostPayout(creator sdk.AccAddress, denom string, amount int64) {
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(amount))),
	).Return(nil).Times(1)
}

// Semantics row 1: an absent snapshot entry means S0 = 0.
func (suite *IntegrationTestSuite) TestSettle_AbsentEntry_S0Zero() {
	creator := sdk.AccAddress("creator")
	suite.seedZeroBaseReward("000000000001")
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: creator.String(), RewardId: "000000000001", Amount: "3", JoinedAt: "0",
	})
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "ubze", 1, "2", 10))
	suite.expectBoostPayout(creator, "ubze", 6) // 3 * (2 - 0)

	suite.Require().NoError(suite.claim(creator, "000000000001"))
	p, _ := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().Equal(uint64(1), p.BoostSnapshots["ubze"].Uid)
	suite.Require().Equal("2", p.BoostSnapshots["ubze"].S0)
}

// Semantics row 2: a matching-uid entry uses the stored S0.
func (suite *IntegrationTestSuite) TestSettle_MatchingUid_StoredS0() {
	creator := sdk.AccAddress("creator")
	suite.seedZeroBaseReward("000000000001")
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: creator.String(), RewardId: "000000000001", Amount: "3", JoinedAt: "0",
		BoostSnapshots: map[string]*types.BoostSnapshot{"ubze": {Uid: 5, S0: "1"}},
	})
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "ubze", 5, "2", 10))
	suite.expectBoostPayout(creator, "ubze", 3) // 3 * (2 - 1)

	suite.Require().NoError(suite.claim(creator, "000000000001"))
	p, _ := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().Equal("2", p.BoostSnapshots["ubze"].S0)
}

// Semantics row 3: a stale-uid entry is treated as S0 = 0 and overwritten. Using
// the stale stored S0 (10 > s_boost) would make pending negative — exploit A1.
func (suite *IntegrationTestSuite) TestSettle_StaleUid_S0ZeroAndOverwrite() {
	creator := sdk.AccAddress("creator")
	suite.seedZeroBaseReward("000000000001")
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: creator.String(), RewardId: "000000000001", Amount: "3", JoinedAt: "0",
		BoostSnapshots: map[string]*types.BoostSnapshot{"ubze": {Uid: 4, S0: "10"}},
	})
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "ubze", 5, "2", 10))
	suite.expectBoostPayout(creator, "ubze", 6) // 3 * (2 - 0), stale entry ignored

	suite.Require().NoError(suite.claim(creator, "000000000001"))
	p, _ := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().Equal(uint64(5), p.BoostSnapshots["ubze"].Uid)
	suite.Require().Equal("2", p.BoostSnapshots["ubze"].S0)
}

// Truncation guard (A8): a dust staker whose payout truncates to zero keeps its
// accrual (snapshot not advanced) and is paid once pending reaches one unit.
func (suite *IntegrationTestSuite) TestSettle_TruncationGuard_AccruesThenPays() {
	creator := sdk.AccAddress("creator")
	suite.seedZeroBaseReward("000000000001")
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: creator.String(), RewardId: "000000000001", Amount: "1", JoinedAt: "0",
	})
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "ubze", 1, "0.5", 10))

	// 1 * 0.5 = 0.5 truncates to 0 -> no boost payout, base pays 0 -> nothing to claim
	suite.Require().ErrorIs(suite.claim(creator, "000000000001"), types.ErrNoRewardsToClaim)
	p, _ := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().Nil(p.BoostSnapshots["ubze"]) // accrual preserved, not erased

	// s_boost advances to 1.5 (out-of-scope distribution simulated in store)
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "ubze", 1, "1.5", 10))
	suite.expectBoostPayout(creator, "ubze", 1) // 1 * 1.5 -> 1

	suite.Require().NoError(suite.claim(creator, "000000000001"))
	p, _ = suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().Equal("1.5", p.BoostSnapshots["ubze"].S0)
}

// Stale-entry cleanup: a snapshot entry whose denom no longer has a record is
// dropped at the owner's next settle; a live denom is retained.
func (suite *IntegrationTestSuite) TestSettle_DropsOrphanSnapshotEntries() {
	creator := sdk.AccAddress("creator")
	suite.seedZeroBaseReward("000000000001")
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: creator.String(), RewardId: "000000000001", Amount: "3", JoinedAt: "0",
		BoostSnapshots: map[string]*types.BoostSnapshot{
			"ubze":   {Uid: 1, S0: "0"},
			"orphan": {Uid: 9, S0: "3"},
		},
	})
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "ubze", 1, "2", 10))
	suite.expectBoostPayout(creator, "ubze", 6)

	suite.Require().NoError(suite.claim(creator, "000000000001"))
	p, _ := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().Contains(p.BoostSnapshots, "ubze")
	suite.Require().NotContains(p.BoostSnapshots, "orphan")
}

// When the reward has no boost records, all snapshot entries are orphans of a
// finalized run and are cleared. Base is made to pay so the participant persists.
func (suite *IntegrationTestSuite) TestSettle_NoRecords_ClearsAllSnapshots() {
	creator := sdk.AccAddress("creator")
	// base pays: DistributedStake (1) > JoinedAt (0), amount 10 -> base pending 10
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId: "000000000001", PrizeAmount: "1000", PrizeDenom: "ubze", StakingDenom: "ubze",
		Duration: 5, Payouts: 0, MinStake: 1, Lock: 7, StakedAmount: "100", DistributedStake: "1",
	})
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: creator.String(), RewardId: "000000000001", Amount: "10", JoinedAt: "0",
		BoostSnapshots: map[string]*types.BoostSnapshot{"gone": {Uid: 1, S0: "5"}},
	})
	// base claim payout of 10 (no boost records exist)
	suite.bank.EXPECT().SendCoinsFromModuleToAccount(
		suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10))),
	).Return(nil).Times(1)

	suite.Require().NoError(suite.claim(creator, "000000000001"))
	p, _ := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().Empty(p.BoostSnapshots)
}

// Multiple boosts of one reward are each settled in a single call.
func (suite *IntegrationTestSuite) TestSettle_MultipleRecords() {
	creator := sdk.AccAddress("creator")
	suite.seedZeroBaseReward("000000000001")
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address: creator.String(), RewardId: "000000000001", Amount: "2", JoinedAt: "0",
	})
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "ubze", 1, "2", 10))
	suite.k.SetBoost(suite.ctx, boostRecord("000000000001", "uother", 2, "3", 10))
	suite.expectBoostPayout(creator, "ubze", 4)   // 2 * 2
	suite.expectBoostPayout(creator, "uother", 6) // 2 * 3

	suite.Require().NoError(suite.claim(creator, "000000000001"))
	p, _ := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().Equal("2", p.BoostSnapshots["ubze"].S0)
	suite.Require().Equal("3", p.BoostSnapshots["uother"].S0)
}
