package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type joinRecord struct {
	rewardId, address, denom string
	amount                   math.Int
}

type increaseRecord struct {
	rewardId, address, denom string
	amountAdded, newTotal    math.Int
}

type exitRecord struct {
	rewardId, address, denom string
	unstaked                 math.Int
}

// recordingStakingHooks records every hook call so tests can assert exactly
// which hooks fired and with what arguments.
type recordingStakingHooks struct {
	joins     []joinRecord
	increases []increaseRecord
	exits     []exitRecord
	removals  []string
	err       error
}

func (r *recordingStakingHooks) AfterStakingRewardJoin(_ sdk.Context, rewardId, address string, amount math.Int, stakingDenom string) error {
	r.joins = append(r.joins, joinRecord{rewardId: rewardId, address: address, amount: amount, denom: stakingDenom})
	return r.err
}

func (r *recordingStakingHooks) AfterStakingRewardIncrease(_ sdk.Context, rewardId, address string, amountAdded, newTotal math.Int, stakingDenom string) error {
	r.increases = append(r.increases, increaseRecord{rewardId: rewardId, address: address, amountAdded: amountAdded, newTotal: newTotal, denom: stakingDenom})
	return r.err
}

func (r *recordingStakingHooks) AfterStakingRewardExit(_ sdk.Context, rewardId, address string, unstakedAmount math.Int, stakingDenom string) error {
	r.exits = append(r.exits, exitRecord{rewardId: rewardId, address: address, unstaked: unstakedAmount, denom: stakingDenom})
	return r.err
}

func (r *recordingStakingHooks) BeforeStakingRewardRemoval(_ sdk.Context, rewardId string) error {
	r.removals = append(r.removals, rewardId)
	return r.err
}

// registerHooks registers the recording hooks on the suite keeper and rebuilds
// the msg server so its embedded keeper copy carries the hooks.
func (suite *IntegrationTestSuite) registerHooks() *recordingStakingHooks {
	hooks := &recordingStakingHooks{}
	suite.k.SetHooks(hooks)
	suite.msgServer = keeper.NewMsgServerImpl(*suite.k)

	return hooks
}

func (suite *IntegrationTestSuite) TestStakingRewardHooks_FreshJoinFiresJoinHook() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()

	stakingReward := types.StakingReward{
		RewardId:         "hook-join-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "0",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	msg := &types.MsgJoinStaking{
		Creator:  creator.String(),
		RewardId: "hook-join-reward",
		Amount:   "500",
	}

	_, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().Len(hooks.joins, 1)
	suite.Require().Empty(hooks.increases)
	suite.Require().Empty(hooks.exits)
	suite.Require().Equal("hook-join-reward", hooks.joins[0].rewardId)
	suite.Require().Equal(creator.String(), hooks.joins[0].address)
	suite.Require().Equal(math.NewInt(500), hooks.joins[0].amount)
	suite.Require().Equal("ubze", hooks.joins[0].denom)
}

func (suite *IntegrationTestSuite) TestStakingRewardHooks_RepeatJoinFiresIncreaseHook() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()

	stakingReward := types.StakingReward{
		RewardId:         "hook-increase-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "500",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	participant := types.StakingRewardParticipant{
		Address:  creator.String(),
		RewardId: "hook-increase-reward",
		Amount:   "500",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(300)))).
		Return(nil).
		Times(1)

	msg := &types.MsgJoinStaking{
		Creator:  creator.String(),
		RewardId: "hook-increase-reward",
		Amount:   "300",
	}

	_, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().Len(hooks.increases, 1)
	suite.Require().Empty(hooks.joins)
	suite.Require().Empty(hooks.exits)
	suite.Require().Equal("hook-increase-reward", hooks.increases[0].rewardId)
	suite.Require().Equal(creator.String(), hooks.increases[0].address)
	suite.Require().Equal(math.NewInt(300), hooks.increases[0].amountAdded)
	suite.Require().Equal(math.NewInt(800), hooks.increases[0].newTotal)
	suite.Require().Equal("ubze", hooks.increases[0].denom)
}

func (suite *IntegrationTestSuite) TestStakingRewardHooks_ExitWithoutLockFiresExitHook() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()

	stakingReward := types.StakingReward{
		RewardId:         "hook-exit-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             0,
		StakedAmount:     "500",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	participant := types.StakingRewardParticipant{
		Address:  creator.String(),
		RewardId: "hook-exit-reward",
		Amount:   "500",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	//lock = 0 -> funds are sent back immediately
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	msg := &types.MsgExitStaking{
		Creator:  creator.String(),
		RewardId: "hook-exit-reward",
	}

	_, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().Len(hooks.exits, 1)
	suite.Require().Empty(hooks.joins)
	suite.Require().Empty(hooks.increases)
	suite.Require().Equal("hook-exit-reward", hooks.exits[0].rewardId)
	suite.Require().Equal(creator.String(), hooks.exits[0].address)
	suite.Require().Equal(math.NewInt(500), hooks.exits[0].unstaked)
	suite.Require().Equal("ubze", hooks.exits[0].denom)
}

func (suite *IntegrationTestSuite) TestStakingRewardHooks_ExitWithLockFiresExitHook() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()

	stakingReward := types.StakingReward{
		RewardId:         "hook-exit-lock-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "500",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	participant := types.StakingRewardParticipant{
		Address:  creator.String(),
		RewardId: "hook-exit-lock-reward",
		Amount:   "500",
		JoinedAt: "0",
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(100), nil).
		Times(1)

	msg := &types.MsgExitStaking{
		Creator:  creator.String(),
		RewardId: "hook-exit-lock-reward",
	}

	_, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	//participation ends at exit time even though the funds unlock later
	suite.Require().Len(hooks.exits, 1)
	suite.Require().Equal(math.NewInt(500), hooks.exits[0].unstaked)
}

func (suite *IntegrationTestSuite) TestStakingRewardHooks_HookErrorFailsJoin() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()
	hooks.err = fmt.Errorf("consumer rejected the join")

	stakingReward := types.StakingReward{
		RewardId:         "hook-error-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "0",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	msg := &types.MsgJoinStaking{
		Creator:  creator.String(),
		RewardId: "hook-error-reward",
		Amount:   "500",
	}

	response, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "consumer rejected the join")
}

func (suite *IntegrationTestSuite) TestStakingRewardHooks_FailedJoinFiresNoHooks() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()

	stakingReward := types.StakingReward{
		RewardId:         "hook-minstake-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         1000,
		Lock:             7,
		StakedAmount:     "0",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)

	msg := &types.MsgJoinStaking{
		Creator:  creator.String(),
		RewardId: "hook-minstake-reward",
		Amount:   "500", //below min stake
	}

	_, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().Error(err)

	suite.Require().Empty(hooks.joins)
	suite.Require().Empty(hooks.increases)
	suite.Require().Empty(hooks.exits)
}

func (suite *IntegrationTestSuite) TestStakingRewardHooks_SetHooksTwicePanics() {
	suite.registerHooks()

	suite.Require().Panics(func() {
		suite.k.SetHooks(&recordingStakingHooks{})
	})
}

func (suite *IntegrationTestSuite) TestStakingRewardHooks_NoHooksRegisteredIsNoOp() {
	creator := sdk.AccAddress("creator")
	//no SetHooks call: handlers must behave exactly as before

	stakingReward := types.StakingReward{
		RewardId:         "no-hooks-reward",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "0",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, stakingReward)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	msg := &types.MsgJoinStaking{
		Creator:  creator.String(),
		RewardId: "no-hooks-reward",
		Amount:   "500",
	}

	_, err := suite.msgServer.JoinStaking(suite.ctx, msg)
	suite.Require().NoError(err)
}

// seedFinishedRewardWithLastParticipant stores a finished reward (all payouts
// executed) whose entire remaining stake belongs to one participant — the
// exact state in which the next exit deletes the reward record.
func (suite *IntegrationTestSuite) seedFinishedRewardWithLastParticipant(rewardId string, creator sdk.AccAddress) {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         rewardId,
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          5,
		MinStake:         100,
		Lock:             0,
		StakedAmount:     "500",
		DistributedStake: "0",
	})
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address:  creator.String(),
		RewardId: rewardId,
		Amount:   "500",
		JoinedAt: "0",
	})
}

// TestStakingRewardHooks_RemovalVetoBlocksFinalExit: a subscriber error from
// BeforeStakingRewardRemoval fails the final exit before any mutation — the
// reward, the participant and the unlock queue are untouched and no coins
// move (the mock would panic on an unexpected transfer).
func (suite *IntegrationTestSuite) TestStakingRewardHooks_RemovalVetoBlocksFinalExit() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()
	hooks.err = fmt.Errorf("reward is pinned")
	suite.seedFinishedRewardWithLastParticipant("hook-removal-veto", creator)

	msg := &types.MsgExitStaking{Creator: creator.String(), RewardId: "hook-removal-veto"}
	_, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().ErrorContains(err, "reward is pinned")

	//the veto fired for this reward and nothing else did
	suite.Require().Equal([]string{"hook-removal-veto"}, hooks.removals)
	suite.Require().Empty(hooks.exits)

	//all SR state is intact
	reward, found := suite.k.GetStakingReward(suite.ctx, "hook-removal-veto")
	suite.Require().True(found)
	suite.Require().Equal("500", reward.StakedAmount)
	participant, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "hook-removal-veto")
	suite.Require().True(found)
	suite.Require().Equal("500", participant.Amount)
	suite.Require().Empty(suite.k.GetAllPendingUnlockParticipant(suite.ctx))
}

// TestStakingRewardHooks_RemovalAllowedOnFinalExit: a nil-returning
// subscriber observes the removal, the exit succeeds, the reward is removed,
// the finish event is emitted and AfterStakingRewardExit still fires.
func (suite *IntegrationTestSuite) TestStakingRewardHooks_RemovalAllowedOnFinalExit() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()
	suite.seedFinishedRewardWithLastParticipant("hook-removal-ok", creator)

	//lock = 0 -> the stake is returned immediately
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	msg := &types.MsgExitStaking{Creator: creator.String(), RewardId: "hook-removal-ok"}
	_, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().Equal([]string{"hook-removal-ok"}, hooks.removals)
	suite.Require().Len(hooks.exits, 1)
	suite.Require().Equal("hook-removal-ok", hooks.exits[0].rewardId)

	_, found := suite.k.GetStakingReward(suite.ctx, "hook-removal-ok")
	suite.Require().False(found)

	finishEmitted := false
	for _, event := range suite.ctx.EventManager().Events() {
		if event.Type == "bze.rewards.StakingRewardFinishEvent" {
			finishEmitted = true
		}
	}
	suite.Require().True(finishEmitted)
}

// TestStakingRewardHooks_NonFinalExitSkipsRemovalHook: an exit that leaves
// stake behind (reward not finished) must not consult the removal hook.
func (suite *IntegrationTestSuite) TestStakingRewardHooks_NonFinalExitSkipsRemovalHook() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()

	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "hook-no-removal",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             0,
		StakedAmount:     "500",
		DistributedStake: "0",
	})
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address:  creator.String(),
		RewardId: "hook-no-removal",
		Amount:   "500",
		JoinedAt: "0",
	})

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	msg := &types.MsgExitStaking{Creator: creator.String(), RewardId: "hook-no-removal"}
	_, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().Empty(hooks.removals)
	suite.Require().Len(hooks.exits, 1)
}

// TestStakingRewardHooks_FinalExitRemovalWithoutHooksSet: SetHooks never
// called — the final exit removes the finished reward exactly as before, no
// panic.
func (suite *IntegrationTestSuite) TestStakingRewardHooks_FinalExitRemovalWithoutHooksSet() {
	creator := sdk.AccAddress("creator")
	suite.seedFinishedRewardWithLastParticipant("no-hooks-removal", creator)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	msg := &types.MsgExitStaking{Creator: creator.String(), RewardId: "no-hooks-removal"}
	_, err := suite.msgServer.ExitStaking(suite.ctx, msg)
	suite.Require().NoError(err)

	_, found := suite.k.GetStakingReward(suite.ctx, "no-hooks-removal")
	suite.Require().False(found)
}
