package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/rewards/types"
)

type drJoinRecord struct {
	denom, address string
	amount         math.Int
}

type drIncreaseRecord struct {
	denom, address        string
	amountAdded, newTotal math.Int
}

type drExitRecord struct {
	denom, address string
	unstaked       math.Int
}

// recordingDenomHooks records every Denom Reward hook call and returns err from each.
type recordingDenomHooks struct {
	joins     []drJoinRecord
	increases []drIncreaseRecord
	exits     []drExitRecord
	err       error
}

func (r *recordingDenomHooks) AfterDenomRewardJoin(_ sdk.Context, stakingDenom, address string, amount math.Int) error {
	r.joins = append(r.joins, drJoinRecord{denom: stakingDenom, address: address, amount: amount})
	return r.err
}

func (r *recordingDenomHooks) AfterDenomRewardIncrease(_ sdk.Context, stakingDenom, address string, amountAdded, newTotal math.Int) error {
	r.increases = append(r.increases, drIncreaseRecord{denom: stakingDenom, address: address, amountAdded: amountAdded, newTotal: newTotal})
	return r.err
}

func (r *recordingDenomHooks) AfterDenomRewardExit(_ sdk.Context, stakingDenom, address string, unstakedAmount math.Int) error {
	r.exits = append(r.exits, drExitRecord{denom: stakingDenom, address: address, unstaked: unstakedAmount})
	return r.err
}

// registerDenomHooks registers recording hooks on the suite keeper, which the
// msg server shares, and a Denom Reward on ubze with no lock and no min stake.
func (suite *IntegrationTestSuite) registerDenomHooks() *recordingDenomHooks {
	hooks := &recordingDenomHooks{}
	suite.k.SetDenomRewardHooks(hooks)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 0, StakedAmount: math.ZeroInt()})

	return hooks
}

func (suite *IntegrationTestSuite) TestDenomRewardHooks_FreshJoinFiresJoinHook() {
	joiner := sdk.AccAddress("dr-hook-joiner-1")
	hooks := suite.registerDenomHooks()

	suite.Require().NoError(suite.joinDr(joiner, "ubze", 500))

	suite.Require().Equal([]drJoinRecord{{denom: "ubze", address: joiner.String(), amount: math.NewInt(500)}}, hooks.joins)
	suite.Require().Empty(hooks.increases)
	suite.Require().Empty(hooks.exits)
}

func (suite *IntegrationTestSuite) TestDenomRewardHooks_TopUpFiresIncreaseHook() {
	joiner := sdk.AccAddress("dr-hook-joiner-2")
	hooks := suite.registerDenomHooks()

	suite.Require().NoError(suite.joinDr(joiner, "ubze", 500))
	suite.Require().NoError(suite.joinDr(joiner, "ubze", 200))

	suite.Require().Len(hooks.joins, 1)
	suite.Require().Equal([]drIncreaseRecord{{
		denom: "ubze", address: joiner.String(), amountAdded: math.NewInt(200), newTotal: math.NewInt(700),
	}}, hooks.increases)
	suite.Require().Empty(hooks.exits)
}

// The Exit hook fires after the participant is gone, with the whole amount.
func (suite *IntegrationTestSuite) TestDenomRewardHooks_ExitFiresExitHook() {
	exiter := sdk.AccAddress("dr-hook-exiter-1")
	hooks := suite.registerDenomHooks()
	suite.Require().NoError(suite.joinDr(exiter, "ubze", 500))

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, exiter, sdk.NewCoins(sdk.NewInt64Coin("ubze", 500))).
		Return(nil).Times(1)
	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().NoError(err)

	suite.Require().Equal([]drExitRecord{{denom: "ubze", address: exiter.String(), unstaked: math.NewInt(500)}}, hooks.exits)
	_, found := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", exiter.String())
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestDenomRewardHooks_JoinHookErrorFailsJoin() {
	joiner := sdk.AccAddress("dr-hook-joiner-3")
	hooks := suite.registerDenomHooks()
	hooks.err = errMockFailure

	suite.Require().ErrorIs(suite.joinDr(joiner, "ubze", 500), errMockFailure)
	suite.Require().Len(hooks.joins, 1)
}

func (suite *IntegrationTestSuite) TestDenomRewardHooks_IncreaseHookErrorFailsTopUp() {
	joiner := sdk.AccAddress("dr-hook-joiner-4")
	hooks := suite.registerDenomHooks()
	suite.Require().NoError(suite.joinDr(joiner, "ubze", 500))
	hooks.err = errMockFailure

	suite.Require().ErrorIs(suite.joinDr(joiner, "ubze", 200), errMockFailure)
	suite.Require().Len(hooks.increases, 1)
}

func (suite *IntegrationTestSuite) TestDenomRewardHooks_ExitHookErrorFailsExit() {
	exiter := sdk.AccAddress("dr-hook-exiter-2")
	hooks := suite.registerDenomHooks()
	suite.Require().NoError(suite.joinDr(exiter, "ubze", 500))
	hooks.err = errMockFailure

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, exiter, sdk.NewCoins(sdk.NewInt64Coin("ubze", 500))).
		Return(nil).Times(1)
	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})

	suite.Require().ErrorIs(err, errMockFailure)
	suite.Require().Len(hooks.exits, 1)
}

// A refused join (below min stake) fires nothing.
func (suite *IntegrationTestSuite) TestDenomRewardHooks_RefusedJoinFiresNothing() {
	joiner := sdk.AccAddress("dr-hook-joiner-5")
	hooks := suite.registerDenomHooks()
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", MinStake: 1_000, StakedAmount: math.ZeroInt()})

	suite.bank.EXPECT().SpendableCoins(suite.ctx, joiner).Return(sdk.NewCoins(sdk.NewInt64Coin("ubze", 500))).Times(1)
	_, err := suite.msgServer.JoinDenomReward(suite.ctx, types.NewMsgJoinDenomReward(joiner.String(), "ubze", math.NewInt(500)))

	suite.Require().Error(err)
	suite.Require().Empty(hooks.joins)
	suite.Require().Empty(hooks.increases)
}

func (suite *IntegrationTestSuite) TestDenomRewardHooks_SetTwicePanics() {
	suite.registerDenomHooks()

	suite.Require().Panics(func() {
		suite.k.SetDenomRewardHooks(&recordingDenomHooks{})
	})
}

// Without hooks the handlers behave as before.
func (suite *IntegrationTestSuite) TestDenomRewardHooks_NoHooksIsNoOp() {
	joiner := sdk.AccAddress("dr-hook-joiner-6")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", StakedAmount: math.ZeroInt()})

	suite.Require().NoError(suite.joinDr(joiner, "ubze", 500))
	suite.Require().NoError(suite.joinDr(joiner, "ubze", 100))

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, joiner, sdk.NewCoins(sdk.NewInt64Coin("ubze", 600))).
		Return(nil).Times(1)
	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: joiner.String(), Denom: "ubze"})
	suite.Require().NoError(err)
}
