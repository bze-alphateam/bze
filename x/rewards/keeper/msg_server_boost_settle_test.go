package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// seedRewardAndParticipant stores a staking reward and a participant whose
// base pending is zero (JoinedAt == DistributedStake), so claim outcomes are
// driven purely by boost state unless a test moves the base accumulator.
func (suite *IntegrationTestSuite) seedRewardAndParticipant(rewardId, amount string) (types.StakingReward, types.StakingRewardParticipant) {
	sr := types.StakingReward{
		RewardId:         rewardId,
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         30,
		Payouts:          5,
		StakedAmount:     amount,
		DistributedStake: "1",
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	participant := types.StakingRewardParticipant{
		Address:  sdk.AccAddress("creator").String(),
		RewardId: rewardId,
		Amount:   amount,
		JoinedAt: sr.DistributedStake,
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	return sr, participant
}

// setBoostWithAccumulator stores a boost with the given accumulator value,
// standing in for the distribution advance that ships in a later ticket.
func (suite *IntegrationTestSuite) setBoostWithAccumulator(rewardId, boostId, denom, sBoost string, finished bool) types.Boost {
	boost := newBoost(rewardId, boostId, denom)
	boost.DistributedStake = sBoost
	if finished {
		boost.Payouts = boost.Duration
	}
	suite.k.SetBoost(suite.ctx, boost)

	return boost
}

// TestBoostSettle_ClaimBoostOnlyAbsentEntry covers A11 (boost-only claim
// succeeds) and the absent-entry rule: no BoostParticipant record means the
// user staked since before the boost, so S0 = 0.
func (suite *IntegrationTestSuite) TestBoostSettle_ClaimBoostOnlyAbsentEntry() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "100")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", false)

	//pending = 100 x (2 - 0) = 200
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(200)))).
		Return(nil).
		Times(1)

	msg := &types.MsgClaimStakingRewards{Creator: creator.String(), RewardId: "000000000001"}
	response, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	//the base paid nothing; the claim succeeded on the boost payout alone
	suite.Require().Equal("0", response.Amount)

	entry, found := suite.k.GetBoostParticipant(suite.ctx, creator.String(), "000000000001", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal("2", entry.JoinedAt)
}

// TestBoostSettle_ClaimPresentEntry settles from a stored entry baseline.
func (suite *IntegrationTestSuite) TestBoostSettle_ClaimPresentEntry() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "100")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", false)
	suite.k.SetBoostParticipant(suite.ctx, types.BoostParticipant{
		Address: creator.String(), RewardId: "000000000001", BoostId: "000000000001", JoinedAt: "1.5",
	})

	//pending = 100 x (2 - 1.5) = 50
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(50)))).
		Return(nil).
		Times(1)

	msg := &types.MsgClaimStakingRewards{Creator: creator.String(), RewardId: "000000000001"}
	_, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().NoError(err)

	entry, found := suite.k.GetBoostParticipant(suite.ctx, creator.String(), "000000000001", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal("2", entry.JoinedAt)
}

// TestBoostSettle_ClaimNoPendingAnywhereErrors: base pending zero and every
// boost fully settled — the claim must still error (no free success).
func (suite *IntegrationTestSuite) TestBoostSettle_ClaimNoPendingAnywhereErrors() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "100")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", false)
	suite.k.SetBoostParticipant(suite.ctx, types.BoostParticipant{
		Address: creator.String(), RewardId: "000000000001", BoostId: "000000000001", JoinedAt: "2",
	})

	msg := &types.MsgClaimStakingRewards{Creator: creator.String(), RewardId: "000000000001"}
	response, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim)
}

// TestBoostSettle_ClaimIdempotent: a paid claim followed by an immediate
// repeat claim pays nothing the second time.
func (suite *IntegrationTestSuite) TestBoostSettle_ClaimIdempotent() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "100")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", false)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(200)))).
		Return(nil).
		Times(1)

	msg := &types.MsgClaimStakingRewards{Creator: creator.String(), RewardId: "000000000001"}
	_, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().NoError(err)

	//the repeat claim finds nothing pending; an unexpected bank send would
	//panic the mock controller
	_, err = suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim)
}

// TestBoostSettle_TruncationGuardAbsentEntry covers A8 for a participant with
// NO entry: a zero-truncated settle must leave the entry absent so the dust
// staker keeps accruing from S0 = 0, and pay once pending reaches a unit.
func (suite *IntegrationTestSuite) TestBoostSettle_TruncationGuardAbsentEntry() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "1")
	boost := suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "0.4", false)

	//pending = 1 x 0.4 truncated = 0: claim errors, entry must stay absent
	msg := &types.MsgClaimStakingRewards{Creator: creator.String(), RewardId: "000000000001"}
	_, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim)

	_, found := suite.k.GetBoostParticipant(suite.ctx, creator.String(), "000000000001", "000000000001")
	suite.Require().False(found)

	//the accumulator advances; the untouched baseline keeps the accrual
	boost.DistributedStake = "1.2"
	suite.k.SetBoost(suite.ctx, boost)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(1)))).
		Return(nil).
		Times(1)

	_, err = suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().NoError(err)

	entry, found := suite.k.GetBoostParticipant(suite.ctx, creator.String(), "000000000001", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal("1.2", entry.JoinedAt)
}

// TestBoostSettle_TruncationGuardPresentEntry covers A8 for a stored entry:
// the baseline must not advance while payouts truncate to zero.
func (suite *IntegrationTestSuite) TestBoostSettle_TruncationGuardPresentEntry() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "1")
	boost := suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "1.9", false)
	suite.k.SetBoostParticipant(suite.ctx, types.BoostParticipant{
		Address: creator.String(), RewardId: "000000000001", BoostId: "000000000001", JoinedAt: "1.2",
	})

	//pending = 1 x 0.7 truncated = 0: entry must keep its baseline
	msg := &types.MsgClaimStakingRewards{Creator: creator.String(), RewardId: "000000000001"}
	_, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().Error(err)

	entry, _ := suite.k.GetBoostParticipant(suite.ctx, creator.String(), "000000000001", "000000000001")
	suite.Require().Equal("1.2", entry.JoinedAt)

	//pending = 1 x (2.3 - 1.2) = 1.1 -> pays 1, baseline advances
	boost.DistributedStake = "2.3"
	suite.k.SetBoost(suite.ctx, boost)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(1)))).
		Return(nil).
		Times(1)

	_, err = suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().NoError(err)

	entry, _ = suite.k.GetBoostParticipant(suite.ctx, creator.String(), "000000000001", "000000000001")
	suite.Require().Equal("2.3", entry.JoinedAt)
}

// TestBoostSettle_FinishedBoostSettled: a finished boost (payouts == duration)
// still pays its outstanding entitlement at claim.
func (suite *IntegrationTestSuite) TestBoostSettle_FinishedBoostSettled() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "100")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", true)
	suite.k.SetBoostParticipant(suite.ctx, types.BoostParticipant{
		Address: creator.String(), RewardId: "000000000001", BoostId: "000000000001", JoinedAt: "1",
	})

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(100)))).
		Return(nil).
		Times(1)

	msg := &types.MsgClaimStakingRewards{Creator: creator.String(), RewardId: "000000000001"}
	_, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().NoError(err)
}

// TestBoostSettle_MultipleBoostsSameDenom: two same-denom boosts settle
// independently and the claim pays both.
func (suite *IntegrationTestSuite) TestBoostSettle_MultipleBoostsSameDenom() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "10")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "1", false)
	suite.setBoostWithAccumulator("000000000001", "000000000002", "uboost", "2", false)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(10)))).
		Return(nil).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(20)))).
		Return(nil).
		Times(1)

	msg := &types.MsgClaimStakingRewards{Creator: creator.String(), RewardId: "000000000001"}
	_, err := suite.msgServer.ClaimStakingRewards(suite.ctx, msg)
	suite.Require().NoError(err)

	first, _ := suite.k.GetBoostParticipant(suite.ctx, creator.String(), "000000000001", "000000000001")
	suite.Require().Equal("1", first.JoinedAt)
	second, _ := suite.k.GetBoostParticipant(suite.ctx, creator.String(), "000000000001", "000000000002")
	suite.Require().Equal("2", second.JoinedAt)
}

// TestBoostSettle_JoinTopUpOrdering covers A2: the top-up join settles boosts
// at the PRE-top-up amount; only the window after the join accrues at the new
// amount.
func (suite *IntegrationTestSuite) TestBoostSettle_JoinTopUpOrdering() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "10")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "3", false)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)
	//boost settle at the pre-top-up amount: 10 x 3 = 30, NOT 1000 x 3
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(30)))).
		Return(nil).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(990)))).
		Return(nil).
		Times(1)

	msg := &types.MsgJoinStaking{Creator: creator.String(), RewardId: "000000000001", Amount: "990"}
	_, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	participant, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().True(found)
	suite.Require().Equal("1000", participant.Amount)

	//the stamp restarts the boost window at the current accumulator
	entry, found := suite.k.GetBoostParticipant(suite.ctx, creator.String(), "000000000001", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal("3", entry.JoinedAt)
}

// TestBoostSettle_JoinStampsFinishedBoost covers A3: a user joining while a
// finished boost is still stored gets an entry at its final accumulator and
// therefore earns nothing from it.
func (suite *IntegrationTestSuite) TestBoostSettle_JoinStampsFinishedBoost() {
	joiner := sdk.AccAddress("joiner.")
	sr, _ := suite.seedRewardAndParticipant("000000000001", "100")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "5", true)

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

	entry, found := suite.k.GetBoostParticipant(suite.ctx, joiner.String(), "000000000001", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal("5", entry.JoinedAt)

	//nothing is claimable from the finished boost: the claim errors and an
	//unexpected bank send would panic the mock controller
	claimMsg := &types.MsgClaimStakingRewards{Creator: joiner.String(), RewardId: sr.RewardId}
	_, err = suite.msgServer.ClaimStakingRewards(suite.ctx, claimMsg)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim)
}

// TestBoostSettle_ExitPaysAndDeletes covers A7: the exit tx pays the boost
// pending before the participant record is removed and deletes the user's
// boost entries.
func (suite *IntegrationTestSuite) TestBoostSettle_ExitPaysAndDeletes() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "500")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", false)

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

	msg := &types.MsgExitStaking{Creator: creator.String(), RewardId: "000000000001"}
	_, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	_, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "000000000001")
	suite.Require().False(found)
	suite.Require().Empty(suite.k.GetAllBoostParticipant(suite.ctx))
}

// TestBoostSettle_ExitReJoinClaim covers A13: a full exit pays everything and
// deletes the entries; the re-joiner is stamped fresh and a later claim pays
// only the post-re-join accrual.
func (suite *IntegrationTestSuite) TestBoostSettle_ExitReJoinClaim() {
	creator := sdk.AccAddress("creator")
	sr, _ := suite.seedRewardAndParticipant("000000000001", "100")
	boost := suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", false)

	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "hour").Return(int64(100), nil).AnyTimes()
	//exit: boost pending 100 x 2 = 200 + the unlocked stake
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(200)))).
		Return(nil).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100)))).
		Return(nil).
		Times(1)

	_, err := suite.msgServer.ExitStaking(suite.ctx, &types.MsgExitStaking{Creator: creator.String(), RewardId: sr.RewardId})
	suite.Require().NoError(err)
	suite.Require().Empty(suite.k.GetAllBoostParticipant(suite.ctx))

	//re-join: stamped at the current accumulator ("2")
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100)))).
		Return(nil).
		Times(1)
	_, err = suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{Creator: creator.String(), RewardId: sr.RewardId, Amount: "100"})
	suite.Require().NoError(err)

	entry, found := suite.k.GetBoostParticipant(suite.ctx, creator.String(), "000000000001", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal("2", entry.JoinedAt)

	//the accumulator advances; the claim pays ONLY the post-re-join window:
	//100 x (3 - 2) = 100, not 100 x 3
	boost.DistributedStake = "3"
	suite.k.SetBoost(suite.ctx, boost)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(100)))).
		Return(nil).
		Times(1)

	_, err = suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{Creator: creator.String(), RewardId: sr.RewardId})
	suite.Require().NoError(err)
}

// TestBoostSettle_RewardRemovalDeletesBoosts covers I4/I3: the last exit of a
// finished reward removes it AND all its boost records, leaving nothing
// outstanding in either boost store.
func (suite *IntegrationTestSuite) TestBoostSettle_RewardRemovalDeletesBoosts() {
	creator := sdk.AccAddress("creator")
	sr, _ := suite.seedRewardAndParticipant("000000000001", "500")
	sr.Duration = 5
	sr.Payouts = 5
	suite.k.SetStakingReward(suite.ctx, sr)
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", true)
	suite.setBoostWithAccumulator("000000000001", "000000000002", "ubze", "0", true)

	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "hour").Return(int64(100), nil).AnyTimes()
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(1000)))).
		Return(nil).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	_, err := suite.msgServer.ExitStaking(suite.ctx, &types.MsgExitStaking{Creator: creator.String(), RewardId: sr.RewardId})
	suite.Require().NoError(err)

	_, found := suite.k.GetStakingReward(suite.ctx, sr.RewardId)
	suite.Require().False(found)
	suite.Require().Empty(suite.k.GetRewardBoosts(suite.ctx, sr.RewardId))
	suite.Require().Empty(suite.k.GetAllBoosts(suite.ctx))
	suite.Require().Empty(suite.k.GetAllBoostParticipant(suite.ctx))
}

// callbackStakingHooks lets a test run assertions at the exact moment a hook
// fires, proving the hook sees the final state.
type callbackStakingHooks struct {
	joins, increases, exits int
	onJoin                  func(rewardId, address string, amount math.Int, denom string)
	onIncrease              func(rewardId, address string, amountAdded, newTotal math.Int, denom string)
	onExit                  func(rewardId, address string, unstaked math.Int, denom string)
}

func (c *callbackStakingHooks) AfterStakingRewardJoin(_ sdk.Context, rewardId, address string, amount math.Int, denom string) error {
	c.joins++
	if c.onJoin != nil {
		c.onJoin(rewardId, address, amount, denom)
	}
	return nil
}

func (c *callbackStakingHooks) AfterStakingRewardIncrease(_ sdk.Context, rewardId, address string, amountAdded, newTotal math.Int, denom string) error {
	c.increases++
	if c.onIncrease != nil {
		c.onIncrease(rewardId, address, amountAdded, newTotal, denom)
	}
	return nil
}

func (c *callbackStakingHooks) AfterStakingRewardExit(_ sdk.Context, rewardId, address string, unstaked math.Int, denom string) error {
	c.exits++
	if c.onExit != nil {
		c.onExit(rewardId, address, unstaked, denom)
	}
	return nil
}

// TestBoostSettle_HookPreservationOnJoin covers A10 for the join path: with
// boosts present, the increase hook still fires exactly once, with the same
// arguments as before, and only after every state write (base participant,
// reward and boost stamp all committed).
func (suite *IntegrationTestSuite) TestBoostSettle_HookPreservationOnJoin() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "10")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "3", false)

	hooks := &callbackStakingHooks{}
	hooks.onIncrease = func(rewardId, address string, amountAdded, newTotal math.Int, denom string) {
		suite.Require().Equal("000000000001", rewardId)
		suite.Require().Equal(creator.String(), address)
		suite.Require().Equal(math.NewInt(990), amountAdded)
		suite.Require().Equal(math.NewInt(1000), newTotal)
		suite.Require().Equal("ubze", denom)

		//all state writes happened before the hook fired
		participant, found := suite.k.GetStakingRewardParticipant(suite.ctx, address, rewardId)
		suite.Require().True(found)
		suite.Require().Equal("1000", participant.Amount)
		entry, found := suite.k.GetBoostParticipant(suite.ctx, address, rewardId, "000000000001")
		suite.Require().True(found)
		suite.Require().Equal("3", entry.JoinedAt)
	}
	suite.k.SetHooks(hooks)
	suite.msgServer = keeper.NewMsgServerImpl(*suite.k)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(30)))).
		Return(nil).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(990)))).
		Return(nil).
		Times(1)

	_, err := suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{Creator: creator.String(), RewardId: "000000000001", Amount: "990"})
	suite.Require().NoError(err)

	suite.Require().Equal(1, hooks.increases)
	suite.Require().Equal(0, hooks.joins)
	suite.Require().Equal(0, hooks.exits)
}

// TestBoostSettle_HookPreservationOnFreshJoin covers A10 for the fresh-join
// path: with a boost present, the join hook (not the increase hook) fires
// exactly once, with the same arguments as before, and only after every state
// write — the boost stamp included, so a hook consumer never observes a
// joiner without their boost entries.
func (suite *IntegrationTestSuite) TestBoostSettle_HookPreservationOnFreshJoin() {
	joiner := sdk.AccAddress("joiner.")
	suite.seedRewardAndParticipant("000000000001", "10")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "3", false)

	hooks := &callbackStakingHooks{}
	hooks.onJoin = func(rewardId, address string, amount math.Int, denom string) {
		suite.Require().Equal("000000000001", rewardId)
		suite.Require().Equal(joiner.String(), address)
		suite.Require().Equal(math.NewInt(500), amount)
		suite.Require().Equal("ubze", denom)

		//all state writes happened before the hook fired, the boost stamp included
		participant, found := suite.k.GetStakingRewardParticipant(suite.ctx, address, rewardId)
		suite.Require().True(found)
		suite.Require().Equal("500", participant.Amount)
		entry, found := suite.k.GetBoostParticipant(suite.ctx, address, rewardId, "000000000001")
		suite.Require().True(found)
		suite.Require().Equal("3", entry.JoinedAt)
	}
	suite.k.SetHooks(hooks)
	suite.msgServer = keeper.NewMsgServerImpl(*suite.k)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, joiner).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, joiner, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	_, err := suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{Creator: joiner.String(), RewardId: "000000000001", Amount: "500"})
	suite.Require().NoError(err)

	//the fresh joiner earns from the stamp onward only: no boost payment was
	//expected and none happened (the mock would panic on an unexpected send)
	suite.Require().Equal(1, hooks.joins)
	suite.Require().Equal(0, hooks.increases)
	suite.Require().Equal(0, hooks.exits)
}

// TestBoostSettle_HookPreservationOnExit covers A10 for the exit path: the
// exit hook fires exactly once, after the participant record AND the boost
// entries are gone.
func (suite *IntegrationTestSuite) TestBoostSettle_HookPreservationOnExit() {
	creator := sdk.AccAddress("creator")
	suite.seedRewardAndParticipant("000000000001", "500")
	suite.setBoostWithAccumulator("000000000001", "000000000001", "uboost", "2", false)

	hooks := &callbackStakingHooks{}
	hooks.onExit = func(rewardId, address string, unstaked math.Int, denom string) {
		suite.Require().Equal("000000000001", rewardId)
		suite.Require().Equal(creator.String(), address)
		suite.Require().Equal(math.NewInt(500), unstaked)
		suite.Require().Equal("ubze", denom)

		//the hook observes the final state: participant and entries removed
		_, found := suite.k.GetStakingRewardParticipant(suite.ctx, address, rewardId)
		suite.Require().False(found)
		suite.Require().Empty(suite.k.GetAllBoostParticipant(suite.ctx))
	}
	suite.k.SetHooks(hooks)
	suite.msgServer = keeper.NewMsgServerImpl(*suite.k)

	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "hour").Return(int64(100), nil).AnyTimes()
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(1000)))).
		Return(nil).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	_, err := suite.msgServer.ExitStaking(suite.ctx, &types.MsgExitStaking{Creator: creator.String(), RewardId: "000000000001"})
	suite.Require().NoError(err)

	suite.Require().Equal(1, hooks.exits)
	suite.Require().Equal(0, hooks.joins)
	suite.Require().Equal(0, hooks.increases)
}
