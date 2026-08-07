package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// seedStrandedReward stores a finished (all payouts executed), emptied
// (zero staked amount, hence zero participants) staking reward — the exact
// state a removal veto at final exit leaves behind.
func (suite *IntegrationTestSuite) seedStrandedReward(rewardId string) {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         rewardId,
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          5,
		MinStake:         100,
		Lock:             0,
		StakedAmount:     "0",
		DistributedStake: "0",
	})
}

func (suite *IntegrationTestSuite) finishEventEmitted() bool {
	for _, event := range suite.ctx.EventManager().Events() {
		if event.Type == "bze.rewards.StakingRewardFinishEvent" {
			return true
		}
	}

	return false
}

func (suite *IntegrationTestSuite) TestDeleteStakingReward_NoHooksSet() {
	suite.seedStrandedReward("remove-no-hooks")

	msg := types.NewMsgDeleteStakingReward(sdk.AccAddress("janitor").String(), "remove-no-hooks")
	_, err := suite.msgServer.DeleteStakingReward(suite.ctx, msg)
	suite.Require().NoError(err)

	_, found := suite.k.GetStakingReward(suite.ctx, "remove-no-hooks")
	suite.Require().False(found)
	suite.Require().True(suite.finishEventEmitted())
}

func (suite *IntegrationTestSuite) TestDeleteStakingReward_HookObservesRemoval() {
	hooks := suite.registerHooks()
	suite.seedStrandedReward("remove-hook-ok")

	msg := types.NewMsgDeleteStakingReward(sdk.AccAddress("janitor").String(), "remove-hook-ok")
	_, err := suite.msgServer.DeleteStakingReward(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().Equal([]string{"remove-hook-ok"}, hooks.removals)
	_, found := suite.k.GetStakingReward(suite.ctx, "remove-hook-ok")
	suite.Require().False(found)
	suite.Require().True(suite.finishEventEmitted())
}

// TestDeleteStakingReward_HookVetoFailsMsg: unlike ExitStaking (where a veto
// suppresses only the deletion), here the veto fails the whole message —
// an explicit error is more informative for the janitor caller.
func (suite *IntegrationTestSuite) TestDeleteStakingReward_HookVetoFailsMsg() {
	hooks := suite.registerHooks()
	hooks.removalErr = fmt.Errorf("reward is pinned")
	suite.seedStrandedReward("remove-hook-veto")

	msg := types.NewMsgDeleteStakingReward(sdk.AccAddress("janitor").String(), "remove-hook-veto")
	response, err := suite.msgServer.DeleteStakingReward(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "reward is pinned")

	suite.Require().Equal([]string{"remove-hook-veto"}, hooks.removals)
	_, found := suite.k.GetStakingReward(suite.ctx, "remove-hook-veto")
	suite.Require().True(found)
	suite.Require().False(suite.finishEventEmitted())
}

func (suite *IntegrationTestSuite) TestDeleteStakingReward_NotFinished() {
	hooks := suite.registerHooks()
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "remove-not-finished",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             0,
		StakedAmount:     "0",
		DistributedStake: "0",
	})

	msg := types.NewMsgDeleteStakingReward(sdk.AccAddress("janitor").String(), "remove-not-finished")
	_, err := suite.msgServer.DeleteStakingReward(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrStakingRewardNotFinished)

	suite.Require().Empty(hooks.removals)
	_, found := suite.k.GetStakingReward(suite.ctx, "remove-not-finished")
	suite.Require().True(found)
}

func (suite *IntegrationTestSuite) TestDeleteStakingReward_NotEmpty() {
	staker := sdk.AccAddress("staker")
	hooks := suite.registerHooks()
	suite.seedFinishedRewardWithLastParticipant("remove-not-empty", staker)

	msg := types.NewMsgDeleteStakingReward(sdk.AccAddress("janitor").String(), "remove-not-empty")
	_, err := suite.msgServer.DeleteStakingReward(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrStakingRewardNotEmpty)

	suite.Require().Empty(hooks.removals)
	reward, found := suite.k.GetStakingReward(suite.ctx, "remove-not-empty")
	suite.Require().True(found)
	suite.Require().Equal("500", reward.StakedAmount)
	participant, found := suite.k.GetStakingRewardParticipant(suite.ctx, staker.String(), "remove-not-empty")
	suite.Require().True(found)
	suite.Require().Equal("500", participant.Amount)
}

func (suite *IntegrationTestSuite) TestDeleteStakingReward_RewardNotFound() {
	msg := types.NewMsgDeleteStakingReward(sdk.AccAddress("janitor").String(), "no-such-reward")
	_, err := suite.msgServer.DeleteStakingReward(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidRewardId)
}

// TestDeleteStakingReward_StrandThenClean: end-to-end regression of the state
// this message exists for — a removal veto at final exit strands the record,
// the veto lifts, and the janitor message deletes it, finally emitting the
// finish event the veto path withheld.
func (suite *IntegrationTestSuite) TestDeleteStakingReward_StrandThenClean() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()
	hooks.removalErr = fmt.Errorf("reward is pinned")
	suite.seedFinishedRewardWithLastParticipant("strand-then-clean", creator)

	//lock = 0 -> the stake is returned immediately despite the veto
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	_, err := suite.msgServer.ExitStaking(suite.ctx, &types.MsgExitStaking{Creator: creator.String(), RewardId: "strand-then-clean"})
	suite.Require().NoError(err)

	//the record survived the veto with zero stake and no finish event
	reward, found := suite.k.GetStakingReward(suite.ctx, "strand-then-clean")
	suite.Require().True(found)
	suite.Require().Equal("0", reward.StakedAmount)
	suite.Require().False(suite.finishEventEmitted())

	//the subscriber stops objecting (e.g. the reward got unpinned) and anyone cleans up
	hooks.removalErr = nil
	msg := types.NewMsgDeleteStakingReward(sdk.AccAddress("janitor").String(), "strand-then-clean")
	_, err = suite.msgServer.DeleteStakingReward(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().Equal([]string{"strand-then-clean", "strand-then-clean"}, hooks.removals)
	_, found = suite.k.GetStakingReward(suite.ctx, "strand-then-clean")
	suite.Require().False(found)
	suite.Require().True(suite.finishEventEmitted())
}
