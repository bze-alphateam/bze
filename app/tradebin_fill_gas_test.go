package app_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	epochstypes "github.com/bze-alphateam/bze/x/epochs/types"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

const (
	fillGasBase = "ufillgas"

	// the largest leaderboard a trading reward can have (MsgCreateTradingReward refuses more slots)
	maxTradingRewardSlots = 10
)

// takerGasForFills builds the real app, rests fills sell orders of 2 units at price 1, each from a
// distinct owner, and returns the gas of one MsgCreateOrder that fills all of them. The message runs
// through the real message router with the real bank keeper: taker fee, escrow, both settlement sends
// per fill, store writes, history, events and the order fill hooks. With withReward the market has an
// active trading reward with the maximum slots and a leaderboard already full of traders who out-trade
// the taker, so every fill updates the taker's candidate, appends the taker to a full leaderboard,
// sorts it, trims it and writes it back (the hook is keyed by the taker, once per fill).
// The ante handler (signatures, tx size, fee deduction) is not included.
func takerGasForFills(t *testing.T, fills int, withReward bool) uint64 {
	t.Helper()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()
	bApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)
	ctx := bApp.NewUncachedContext(false, cmtproto.Header{Time: time.Unix(1_700_000_000, 0)})

	require.NoError(t, bApp.BankKeeper.SetParams(ctx, banktypes.DefaultParams()))
	params := v2types.DefaultParams()
	params.MaxFillsPerOrder = uint64(fills)
	require.NoError(t, bApp.TradebinKeeper.SetParams(ctx, params))
	market := tradebintypes.Market{Base: fillGasBase, Quote: params.NativeDenom, Creator: sdk.AccAddress("market_creator______").String()}
	bApp.TradebinKeeper.SetMarket(ctx, market)
	marketId := market.Base + "/" + market.Quote

	fund := func(addr sdk.AccAddress, coins sdk.Coins) {
		require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
		require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, addr, coins))
	}
	deliver := func(ctx sdk.Context, msg sdk.Msg) {
		_, err := bApp.MsgServiceRouter().Handler(msg)(ctx, msg)
		require.NoError(t, err)
	}

	var rewardId string
	if withReward {
		rewardId = activeTradingRewardWithFullLeaderboard(t, bApp, ctx, marketId, fund, deliver)
	}

	for i := 0; i < fills; i++ {
		maker := sdk.AccAddress(fmt.Sprintf("maker%015d", i))
		fund(maker, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 2), params.MarketMakerFee))
		deliver(ctx, &tradebintypes.MsgCreateOrder{Creator: maker.String(), MarketId: marketId, OrderType: tradebintypes.OrderTypeSell, Amount: "2", Price: "1"})
	}

	taker := sdk.AccAddress("taker_______________")
	fund(taker, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, int64(2*fills)).Add(params.MarketTakerFee)))

	// only the taker message runs on the fresh gas meter
	gasCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	deliver(gasCtx, &tradebintypes.MsgCreateOrder{Creator: taker.String(), MarketId: marketId, OrderType: tradebintypes.OrderTypeBuy, Amount: fmt.Sprintf("%d", 2*fills), Price: "1"})

	// every maker was filled in that same message
	require.Empty(t, bApp.TradebinKeeper.GetAllOrder(ctx))
	require.Equal(t, sdk.NewInt64Coin(market.Base, int64(2*fills)), bApp.BankKeeper.GetBalance(ctx, taker, market.Base))

	if withReward {
		// the hook ran for every fill: the taker's candidate holds the whole traded amount, while the
		// leaderboard stayed full of the bigger traders and the taker was trimmed out on every fill
		candidate, found := bApp.RewardsKeeper.GetTradingRewardCandidate(ctx, rewardId, taker.String())
		require.True(t, found)
		require.Equal(t, fmt.Sprintf("%d", 2*fills), candidate.Amount)
		leaderboard, found := bApp.RewardsKeeper.GetTradingRewardLeaderboard(ctx, rewardId)
		require.True(t, found)
		require.Len(t, leaderboard.List, maxTradingRewardSlots)
		for _, entry := range leaderboard.List {
			require.NotEqual(t, taker.String(), entry.Address)
		}
	}

	return gasCtx.GasMeter().GasConsumed()
}

// activeTradingRewardWithFullLeaderboard creates and activates a trading reward on marketId with the
// maximum slots, then fills its leaderboard with maxTradingRewardSlots distinct traders, each buying far
// more than any measured taker. It returns the reward id.
func activeTradingRewardWithFullLeaderboard(
	t *testing.T,
	bApp *app.App,
	ctx sdk.Context,
	marketId string,
	fund func(sdk.AccAddress, sdk.Coins),
	deliver func(sdk.Context, sdk.Msg),
) string {
	t.Helper()

	// trading reward expirations are counted in epochs
	for _, epoch := range epochstypes.DefaultGenesis().Epochs {
		require.NoError(t, bApp.EpochKeeper.AddEpochInfo(ctx, epoch))
	}

	rewardsParams := rewardstypes.DefaultParams()
	rewardsParams.CreateTradingRewardFee = sdk.NewInt64Coin(rewardsParams.CreateTradingRewardFee.Denom, 0)
	require.NoError(t, bApp.RewardsKeeper.SetParams(ctx, rewardsParams))
	tradebinParams := bApp.TradebinKeeper.GetParams(ctx)

	creator := sdk.AccAddress("reward_creator______")
	fund(creator, sdk.NewCoins(sdk.NewInt64Coin(tradebinParams.NativeDenom, 1_000*maxTradingRewardSlots)))
	createMsg := func(slots int) *rewardstypes.MsgCreateTradingReward {
		return &rewardstypes.MsgCreateTradingReward{
			Creator:     creator.String(),
			PrizeAmount: "1000",
			PrizeDenom:  tradebinParams.NativeDenom,
			Duration:    "30",
			MarketId:    marketId,
			Slots:       fmt.Sprintf("%d", slots),
		}
	}

	// maxTradingRewardSlots really is the ceiling
	tooMany := createMsg(maxTradingRewardSlots + 1)
	_, err := bApp.MsgServiceRouter().Handler(tooMany)(ctx, tooMany)
	require.ErrorIs(t, err, rewardstypes.ErrInvalidSlots)

	create := createMsg(maxTradingRewardSlots)
	res, err := bApp.MsgServiceRouter().Handler(create)(ctx, create)
	require.NoError(t, err)
	var created rewardstypes.MsgCreateTradingRewardResponse
	require.NoError(t, created.Unmarshal(res.MsgResponses[0].Value))
	deliver(ctx, &rewardstypes.MsgActivateTradingReward{Creator: bApp.RewardsKeeper.GetAuthority(), RewardId: created.RewardId})

	for i := 0; i < maxTradingRewardSlots; i++ {
		amount := int64(1_000_000 + i)
		seller := sdk.AccAddress(fmt.Sprintf("leader_maker%08d", i))
		fund(seller, sdk.NewCoins(sdk.NewInt64Coin(fillGasBase, amount), tradebinParams.MarketMakerFee))
		deliver(ctx, &tradebintypes.MsgCreateOrder{Creator: seller.String(), MarketId: marketId, OrderType: tradebintypes.OrderTypeSell, Amount: fmt.Sprintf("%d", amount), Price: "1"})
		leader := sdk.AccAddress(fmt.Sprintf("leader%014d", i))
		fund(leader, sdk.NewCoins(sdk.NewInt64Coin(tradebinParams.NativeDenom, amount).Add(tradebinParams.MarketTakerFee)))
		deliver(ctx, &tradebintypes.MsgCreateOrder{Creator: leader.String(), MarketId: marketId, OrderType: tradebintypes.OrderTypeBuy, Amount: fmt.Sprintf("%d", amount), Price: "1"})
	}

	leaderboard, found := bApp.RewardsKeeper.GetTradingRewardLeaderboard(ctx, created.RewardId)
	require.True(t, found)
	require.Len(t, leaderboard.List, maxTradingRewardSlots)

	return created.RewardId
}

// TestTradebin_TakerGasPerFill logs the message gas of a taker filling 1, 10, 100 and 500 resting orders
// with the real bank keeper.
func TestTradebin_TakerGasPerFill(t *testing.T) {
	gas := map[int]uint64{}
	for _, fills := range []int{1, 10, 100, 500} {
		gas[fills] = takerGasForFills(t, fills, false)
		t.Logf("gas consumed by one MsgCreateOrder filling %d resting orders: %d", fills, gas[fills])
	}
	t.Logf("marginal gas per fill (1 -> 500 fills): %d", (gas[500]-gas[1])/499)

	// well below 10% of the 1,000,000,000 block gas limit
	require.Less(t, gas[100], uint64(100_000_000))
	require.Less(t, gas[500], uint64(100_000_000))
}

// TestTradebin_TakerGasPerFillWithTradingReward logs the same message gas on a market with an active
// trading reward at the maximum slots and a full leaderboard: the worst case of the order fill hook.
func TestTradebin_TakerGasPerFillWithTradingReward(t *testing.T) {
	gas := map[int]uint64{}
	for _, fills := range []int{1, 10, 100, 500} {
		gas[fills] = takerGasForFills(t, fills, true)
		t.Logf("gas consumed by one MsgCreateOrder filling %d resting orders with an active trading reward: %d", fills, gas[fills])
	}
	t.Logf("marginal gas per fill with an active trading reward (1 -> 500 fills): %d", (gas[500]-gas[1])/499)

	// well below 10% of the 1,000,000,000 block gas limit
	require.Less(t, gas[100], uint64(100_000_000))
	require.Less(t, gas[500], uint64(100_000_000))
}
