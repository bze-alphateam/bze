package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"
)

// activateTradingReward stores a pending trading reward and activates it through the governance message.
func (suite *IntegrationTestSuite) activateTradingReward(tr types.TradingReward) {
	tr.ExpireAt = 1000 // the pending expiry, replaced by the activation
	suite.k.SetPendingTradingReward(suite.ctx, tr)
	suite.k.SetPendingTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{RewardId: tr.RewardId, ExpireAt: tr.ExpireAt})

	_, err := suite.msgServer.ActivateTradingReward(suite.ctx, &types.MsgActivateTradingReward{
		Creator:  authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		RewardId: tr.RewardId,
	})
	suite.Require().NoError(err)
}

func countdownTestReward() types.TradingReward {
	return types.TradingReward{
		RewardId:    "countdown-reward",
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    "market-1",
		Slots:       3,
	}
}

// The first trade on an activated reward starts its countdown; later trades leave it alone.
func (suite *IntegrationTestSuite) TestTradingRewardCountdown_FirstTradeStartsIt() {
	tr := countdownTestReward()
	suite.activateTradingReward(tr)

	// exactly one epoch read: the first trade starts the clock, the second one does not touch it
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(gomock.Any(), "hour").
		Return(int64(100), nil).
		Times(1)

	hook := suite.k.GetOnOrderFillHook()
	hook(suite.ctx, tr.MarketId, "500", "bze1first")

	wantExpireAt := uint32(100 + 30*24)
	active, found := suite.k.GetActiveTradingReward(suite.ctx, tr.RewardId)
	suite.Require().True(found)
	suite.Require().Equal(wantExpireAt, active.ExpireAt)

	expirations := suite.k.GetAllActiveTradingRewardExpiration(suite.ctx)
	suite.Require().Len(expirations, 1)
	suite.Require().Equal(types.TradingRewardExpiration{RewardId: tr.RewardId, ExpireAt: wantExpireAt}, expirations[0])

	ev, found := suite.findTypedEvent(proto.MessageName(&types.TradingRewardStartEvent{}))
	suite.Require().True(found, "start event emitted")
	suite.requireEventAttr(ev, "reward_id", tr.RewardId)
	suite.requireEventAttr(ev, "expire_at", fmt.Sprint(wantExpireAt))

	hook(suite.ctx, tr.MarketId, "700", "bze1second")

	active, _ = suite.k.GetActiveTradingReward(suite.ctx, tr.RewardId)
	suite.Require().Equal(wantExpireAt, active.ExpireAt)
	suite.Require().Len(suite.k.GetAllActiveTradingRewardExpiration(suite.ctx), 1)

	leaderboard, found := suite.k.GetTradingRewardLeaderboard(suite.ctx, tr.RewardId)
	suite.Require().True(found)
	suite.Require().Len(leaderboard.List, 2)
}

// A trade that cannot start the clock (hour epoch catching up) still counts for the leaderboard, and the next
// trade starts the clock instead.
func (suite *IntegrationTestSuite) TestTradingRewardCountdown_EpochCatchingUpRetriesOnNextTrade() {
	tr := countdownTestReward()
	suite.activateTradingReward(tr)

	gomock.InOrder(
		suite.epoch.EXPECT().
			SafeGetEpochCountByIdentifier(gomock.Any(), "hour").
			Return(int64(0), fmt.Errorf("epoch hour is catching up")),
		suite.epoch.EXPECT().
			SafeGetEpochCountByIdentifier(gomock.Any(), "hour").
			Return(int64(200), nil),
	)

	hook := suite.k.GetOnOrderFillHook()
	hook(suite.ctx, tr.MarketId, "500", "bze1first")

	active, _ := suite.k.GetActiveTradingReward(suite.ctx, tr.RewardId)
	suite.Require().Zero(active.ExpireAt)
	suite.Require().Empty(suite.k.GetAllActiveTradingRewardExpiration(suite.ctx))
	candidate, found := suite.k.GetTradingRewardCandidate(suite.ctx, tr.RewardId, "bze1first")
	suite.Require().True(found, "the trade still counts")
	suite.Require().Equal("500", candidate.Amount)

	hook(suite.ctx, tr.MarketId, "100", "bze1first")

	active, _ = suite.k.GetActiveTradingReward(suite.ctx, tr.RewardId)
	suite.Require().Equal(uint32(200+30*24), active.ExpireAt)
	suite.Require().Len(suite.k.GetAllActiveTradingRewardExpiration(suite.ctx), 1)
}

// An activated reward nobody trades on is never paid or cleaned up: it keeps its escrow and keeps its market, so
// a second reward on the same market cannot be activated.
func (suite *IntegrationTestSuite) TestTradingRewardCountdown_NoTradesWaitsForever() {
	tr := countdownTestReward()
	suite.activateTradingReward(tr)

	// no bank and no epoch expectations: any payout, burn or countdown start fails the test
	hook := suite.k.GetTradingRewardsDistributionHook()
	for epoch := int64(1); epoch <= 2000; epoch++ {
		suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "hour", epoch))
	}

	active, found := suite.k.GetActiveTradingReward(suite.ctx, tr.RewardId)
	suite.Require().True(found)
	suite.Require().Zero(active.ExpireAt)
	mapping, found := suite.k.GetMarketIdRewardId(suite.ctx, tr.MarketId)
	suite.Require().True(found)
	suite.Require().Equal(tr.RewardId, mapping.RewardId)

	second := countdownTestReward()
	second.RewardId = "second-reward"
	suite.k.SetPendingTradingReward(suite.ctx, second)
	_, err := suite.msgServer.ActivateTradingReward(suite.ctx, &types.MsgActivateTradingReward{
		Creator:  authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		RewardId: second.RewardId,
	})
	suite.Require().ErrorIs(err, types.ErrRewardAlreadyExists)
}

// With fewer traders than slots, each trader gets one slot's prize and the prize of the unfilled slots is burnt.
func (suite *IntegrationTestSuite) TestTradingRewardCountdown_FewerTradersBurnsUnfilledSlots() {
	epochNumber := int64(500)
	tr := countdownTestReward()
	tr.ExpireAt = uint32(epochNumber)
	winner := sdk.AccAddress("winner1")

	suite.k.SetActiveTradingReward(suite.ctx, tr)
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{RewardId: tr.RewardId, ExpireAt: tr.ExpireAt})
	suite.k.SetMarketIdRewardId(suite.ctx, types.MarketIdTradingRewardId{RewardId: tr.RewardId, MarketId: tr.MarketId})
	suite.k.SetTradingRewardLeaderboard(suite.ctx, types.TradingRewardLeaderboard{
		RewardId: tr.RewardId,
		List:     []types.TradingRewardLeaderboardEntry{{Amount: "5000", Address: winner.String(), CreatedAt: 1}},
	})

	unfilled := sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(2000)))
	gomock.InOrder(
		suite.bank.EXPECT().
			SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, winner, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000)))).
			Return(nil),
		suite.bank.EXPECT().
			SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnermoduletypes.ModuleName, unfilled).
			Return(nil),
	)

	suite.Require().NoError(suite.k.GetTradingRewardsDistributionHook().AfterEpochEnd(suite.ctx, "hour", epochNumber))

	_, found := suite.k.GetMarketIdRewardId(suite.ctx, tr.MarketId)
	suite.Require().False(found, "market freed")
	ev, found := suite.findTypedEvent(proto.MessageName(&types.TradingRewardDistributionEvent{}))
	suite.Require().True(found)
	suite.requireEventAttr(ev, "unpaid_burnt", unfilled.String())
}

// A reward reaching its payout with no leaderboard can only come from state written before the countdown waited
// for the first trade. It is finished with no winners: the whole escrow is burnt and the market is freed.
func (suite *IntegrationTestSuite) TestTradingRewardCountdown_MissingLeaderboardFinishesTheReward() {
	epochNumber := int64(500)
	tr := countdownTestReward()
	tr.ExpireAt = uint32(epochNumber)

	suite.k.SetActiveTradingReward(suite.ctx, tr)
	suite.k.SetActiveTradingRewardExpiration(suite.ctx, types.TradingRewardExpiration{RewardId: tr.RewardId, ExpireAt: tr.ExpireAt})
	suite.k.SetMarketIdRewardId(suite.ctx, types.MarketIdTradingRewardId{RewardId: tr.RewardId, MarketId: tr.MarketId})

	escrow := sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(3000)))
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, burnermoduletypes.ModuleName, escrow).
		Return(nil).
		Times(1)

	hook := suite.k.GetTradingRewardsDistributionHook()
	suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "hour", epochNumber))

	_, found := suite.k.GetMarketIdRewardId(suite.ctx, tr.MarketId)
	suite.Require().False(found, "market freed")
	suite.Require().Len(suite.k.GetAllActiveTradingRewardExpirationByExpireAt(suite.ctx, uint32(epochNumber+168)), 1, "kept one week for display")

	// after the display week the reward is gone and a new one can be activated on the market
	suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "hour", epochNumber+168))
	_, found = suite.k.GetActiveTradingReward(suite.ctx, tr.RewardId)
	suite.Require().False(found)

	next := countdownTestReward()
	next.RewardId = "next-reward"
	suite.activateTradingReward(next)
	mapping, found := suite.k.GetMarketIdRewardId(suite.ctx, tr.MarketId)
	suite.Require().True(found)
	suite.Require().Equal(next.RewardId, mapping.RewardId)
}
