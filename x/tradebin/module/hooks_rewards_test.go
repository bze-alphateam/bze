package tradebin_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	rewardskeeper "github.com/bze-alphateam/bze/x/rewards/keeper"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/testutil"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

// newRewardsWiredFixture is the app's hook wiring on one multistore: a real tradebin keeper whose
// order-fill hook is the real rewards keeper's GetOnOrderFillHook (app.go setTradebinHooks).
func newRewardsWiredFixture(t *testing.T) (wiringFixture, rewardskeeper.Keeper) {
	t.Helper()
	tradebinKey := storetypes.NewKVStoreKey(types.StoreKey)
	rewardsKey := storetypes.NewKVStoreKey(rewardstypes.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(tradebinKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(rewardsKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	ctrl := gomock.NewController(t)
	bank := testutil.NewMockBankKeeper(ctrl)
	acc := testutil.NewMockAccountKeeper(ctrl)

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(tradebinKey), log.NewNopLogger(), authority, acc, bank)
	require.NoError(t, k.SetParams(ctx, v2types.DefaultParams()))

	rk := rewardskeeper.NewKeeper(cdc, runtime.NewKVStoreService(rewardsKey), log.NewNopLogger(), authority, nil, nil, &k, nil)
	require.NoError(t, rk.SetParams(ctx, rewardstypes.DefaultParams()))
	k.SetOnOrderFillHooks([]types.OnMarketOrderFill{rk.GetOnOrderFillHook()})

	return wiringFixture{k: &k, ctx: ctx, bank: bank, acc: acc}, rk
}

func activateTradingReward(ctx sdk.Context, rk rewardskeeper.Keeper, rewardId, marketId string) {
	rk.SetActiveTradingReward(ctx, rewardstypes.TradingReward{
		RewardId:    rewardId,
		PrizeAmount: "1000",
		PrizeDenom:  "ubze",
		Duration:    30,
		MarketId:    marketId,
		Slots:       3,
		ExpireAt:    1000,
	})
	rk.SetMarketIdRewardId(ctx, rewardstypes.MarketIdTradingRewardId{RewardId: rewardId, MarketId: marketId})
}

func requireTraded(t *testing.T, ctx sdk.Context, rk rewardskeeper.Keeper, rewardId string, trader sdk.AccAddress, amount math.Int) {
	t.Helper()
	candidate, found := rk.GetTradingRewardCandidate(ctx, rewardId, trader.String())
	require.True(t, found, "the swap must make the trader a candidate of reward %s", rewardId)
	require.Equal(t, amount.String(), candidate.Amount)

	leaderboard, found := rk.GetTradingRewardLeaderboard(ctx, rewardId)
	require.True(t, found)
	require.Len(t, leaderboard.List, 1)
	require.Equal(t, trader.String(), leaderboard.List[0].Address)
	require.Equal(t, amount.String(), leaderboard.List[0].Amount)
}

// BZE-157: an active trading reward on the order book market ubze/stake is credited by a swap
// through the ubze_stake pool, with the volume in the market's base (the ubze leg).
func TestTradingReward_AmmSwap_CountsTowardPairMarketReward(t *testing.T) {
	f, rk := newRewardsWiredFixture(t)
	f.k.SetMarket(f.ctx, types.Market{Base: "ubze", Quote: "stake", Creator: "creator"})
	activateTradingReward(f.ctx, rk, "reward-1", "ubze/stake")

	trader := f.swapThroughRouter(t, f.registerServices(t))

	requireTraded(t, f.ctx, rk, "reward-1", trader, math.NewInt(1000))
}

// Same with the market created as stake/ubze: the volume is the swap's stake leg.
func TestTradingReward_AmmSwap_CountsTowardReversedPairMarketReward(t *testing.T) {
	f, rk := newRewardsWiredFixture(t)
	f.k.SetMarket(f.ctx, types.Market{Base: "stake", Quote: "ubze", Creator: "creator"})
	activateTradingReward(f.ctx, rk, "reward-1", "stake/ubze")

	trader := f.swapThroughRouter(t, f.registerServices(t))

	requireTraded(t, f.ctx, rk, "reward-1", trader, f.swapOutput(t))
}
