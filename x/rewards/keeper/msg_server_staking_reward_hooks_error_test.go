package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// The After* hooks run after the module's own state writes and a non-nil error aborts the whole
// message. TestStakingRewardHooks_HookErrorFailsJoin pins that for a fresh join; the two tests here
// pin the same contract for the remaining participation changes — a top-up and an exit — so "a
// listener module can block a top-up or an exit" is a recorded decision, not an accident.
//
// The message runs on a cache context that is deliberately never written back, the same way
// baseapp discards a failed tx's writes: the assertions on the parent context prove that an
// aborted message leaves no partial state behind.

func (suite *IntegrationTestSuite) TestStakingRewardHooks_IncreaseHookErrorFailsTopUp() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()
	hooks.err = fmt.Errorf("consumer rejected the increase")

	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "hook-increase-error",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          2,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "500",
		DistributedStake: "0",
	})
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
		Address:  creator.String(),
		RewardId: "hook-increase-error",
		Amount:   "500",
		JoinedAt: "0",
	})

	// the top-up escrows 300 more before the hook fires; nothing is pending so no payout is sent
	suite.bank.EXPECT().
		SpendableCoins(gomock.Any(), creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(300)))).
		Return(nil).
		Times(1)

	txCtx, _ := suite.ctx.CacheContext()
	response, err := suite.msgServer.JoinStaking(txCtx, &types.MsgJoinStaking{
		Creator:  creator.String(),
		RewardId: "hook-increase-error",
		Amount:   "300",
	})
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "consumer rejected the increase")

	// the increase hook did fire (with the post-top-up total) and nothing else did
	suite.Require().Len(hooks.increases, 1)
	suite.Require().Empty(hooks.joins)
	suite.Require().Empty(hooks.exits)
	suite.Require().Equal(math.NewInt(300), hooks.increases[0].amountAdded)
	suite.Require().Equal(math.NewInt(800), hooks.increases[0].newTotal)

	// the aborted tx left the position and the reward exactly as they were
	participant, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "hook-increase-error")
	suite.Require().True(found)
	suite.Require().Equal("500", participant.Amount)
	reward, found := suite.k.GetStakingReward(suite.ctx, "hook-increase-error")
	suite.Require().True(found)
	suite.Require().Equal("500", reward.StakedAmount)
}

func (suite *IntegrationTestSuite) TestStakingRewardHooks_ExitHookErrorFailsExit() {
	creator := sdk.AccAddress("creator")
	hooks := suite.registerHooks()
	hooks.err = fmt.Errorf("consumer rejected the exit")

	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "hook-exit-error",
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
		RewardId: "hook-exit-error",
		Amount:   "500",
		JoinedAt: "0",
	})

	// lock 0: the stake is sent back before the hook fires (and rolled back with the tx)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).
		Times(1)

	txCtx, _ := suite.ctx.CacheContext()
	response, err := suite.msgServer.ExitStaking(txCtx, &types.MsgExitStaking{
		Creator:  creator.String(),
		RewardId: "hook-exit-error",
	})
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "consumer rejected the exit")

	suite.Require().Len(hooks.exits, 1)
	suite.Require().Empty(hooks.joins)
	suite.Require().Empty(hooks.increases)
	suite.Require().Equal(math.NewInt(500), hooks.exits[0].unstaked)

	// the position survives the aborted exit: the user is still staked, nothing was removed
	participant, found := suite.k.GetStakingRewardParticipant(suite.ctx, creator.String(), "hook-exit-error")
	suite.Require().True(found)
	suite.Require().Equal("500", participant.Amount)
	reward, found := suite.k.GetStakingReward(suite.ctx, "hook-exit-error")
	suite.Require().True(found)
	suite.Require().Equal("500", reward.StakedAmount)
	suite.Require().Empty(suite.k.GetAllPendingUnlockParticipant(suite.ctx))
}
