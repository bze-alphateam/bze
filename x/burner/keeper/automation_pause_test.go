package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/burner/types"
)

// While paused, a pending periodic burn reads no balance and burns, locks or pools nothing (no account, bank or trade
// expectation: a call fails the test). The queue stays pending and no event is emitted.
func (suite *IntegrationTestSuite) TestPeriodicBurn_PausedDoesNothing() {
	suite.paused = true
	suite.k.EnqueuePeriodicBurn(suite.ctx)

	for i := 0; i < 3; i++ {
		suite.Require().NoError(suite.k.ProcessPeriodicBurnQueue(suite.ctx))
	}

	queue, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
	suite.Require().Empty(suite.ctx.EventManager().Events())
	suite.Require().Empty(suite.k.GetAllBurnedCoins(suite.ctx))
}

// The epoch hook still marks the burn pending while paused (it moves no coin), so a burning epoch that ended during
// the pause is not lost.
func (suite *IntegrationTestSuite) TestPeriodicBurn_HookEnqueuesWhilePaused() {
	suite.paused = true

	suite.Require().NoError(suite.k.GetBurnerPeriodicBurnHook().AfterEpochEnd(suite.ctx, "week", 4))

	queue, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().True(queue.Pending)
}

// After the resume, the burn left pending during the pause burns the whole balance, coins that arrived during the
// pause included.
func (suite *IntegrationTestSuite) TestPeriodicBurn_ResumeBurnsAccumulatedCoins() {
	suite.paused = true
	suite.Require().NoError(suite.k.GetBurnerPeriodicBurnHook().AfterEpochEnd(suite.ctx, "week", 4))
	suite.Require().NoError(suite.k.ProcessPeriodicBurnQueue(suite.ctx))
	suite.Require().NoError(suite.k.ProcessPeriodicBurnQueue(suite.ctx))

	suite.paused = false
	addr := sdk.AccAddress("moduleacc")
	accumulated := sdk.NewCoins(sdk.NewInt64Coin("ubze", 3000), sdk.NewInt64Coin("utoken", 700))
	suite.burnerModuleAcc(addr)
	suite.bank.EXPECT().GetBalancesPage(gomock.Any(), addr, "", types.MaxDenomsBurnPerBlock+1).Return(accumulated, "").Times(1)
	suite.expectNativeBurn(accumulated)

	suite.Require().NoError(suite.k.ProcessPeriodicBurnQueue(suite.ctx))

	_, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found)
	suite.Require().Len(suite.k.GetAllBurnedCoins(suite.ctx), 1)
}

// Not paused, the periodic burn runs as before.
func (suite *IntegrationTestSuite) TestPeriodicBurn_NotPausedBurns() {
	suite.k.EnqueuePeriodicBurn(suite.ctx)
	addr := sdk.AccAddress("moduleacc")
	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1000))
	suite.burnerModuleAcc(addr)
	suite.bank.EXPECT().GetBalancesPage(gomock.Any(), addr, "", types.MaxDenomsBurnPerBlock+1).Return(coins, "").Times(1)
	suite.expectNativeBurn(coins)

	suite.Require().NoError(suite.k.ProcessPeriodicBurnQueue(suite.ctx))

	_, found := suite.k.GetPeriodicBurnQueue(suite.ctx)
	suite.Require().False(found)
}

// Raffle cleanup is user facing and is not part of the pause: a finished raffle's pot is still burned.
func (suite *IntegrationTestSuite) TestRaffleCleanup_RunsWhilePaused() {
	suite.paused = true
	epochNumber := uint64(100)
	denom := "utoken"
	suite.k.SetRaffleDeleteHook(suite.ctx, types.RaffleDeleteHook{Denom: denom, EndAt: epochNumber})
	suite.k.SetRaffle(suite.ctx, types.Raffle{Denom: denom, Pot: "1000"})
	suite.k.EnqueueRaffleCleanup(suite.ctx, epochNumber)

	addr := sdk.AccAddress("raffleacc")
	raffleAcc := authtypes.ModuleAccount{BaseAccount: &authtypes.BaseAccount{Address: addr.String()}, Name: types.RaffleModuleName}
	pot := sdk.NewInt64Coin(denom, 1000)
	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.RaffleModuleName).Return(&raffleAcc).Times(1)
	suite.bank.EXPECT().GetBalance(gomock.Any(), addr, denom).Return(pot).Times(1)
	suite.trade.EXPECT().IsNativeDenom(gomock.Any(), denom).Return(true).Times(1)
	suite.bank.EXPECT().BurnCoins(gomock.Any(), types.RaffleModuleName, sdk.NewCoins(pot)).Return(nil).Times(1)

	suite.Require().NoError(suite.k.ProcessRaffleCleanupQueue(suite.ctx))

	_, found := suite.k.GetRaffle(suite.ctx, denom)
	suite.Require().False(found)
}

// A keeper built without a guardian (nil) is never paused.
func TestKeeper_NilGuardianIsNeverPaused(t *testing.T) {
	k, ctx := keepertest.BurnerKeeper(t, nil, nil, nil, nil, nil)

	require.False(t, k.IsAutomationPaused(ctx))
}
