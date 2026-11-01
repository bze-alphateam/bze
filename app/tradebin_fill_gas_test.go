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
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

const fillGasBase = "ufillgas"

// takerGasForFills builds the real app, rests fills sell orders of 2 units at price 1, each from a
// distinct owner, and returns the gas of one MsgCreateOrder that fills all of them. The message runs
// through the real message router with the real bank keeper: taker fee, escrow, both settlement sends
// per fill, store writes, history, events and the order fill hooks (no trading reward on the market).
// The ante handler (signatures, tx size, fee deduction) is not included.
func takerGasForFills(t *testing.T, fills int) uint64 {
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

	return gasCtx.GasMeter().GasConsumed()
}

// TestTradebin_TakerGasPerFill logs the message gas of a taker filling 1, 10, 100 and 500 resting orders
// with the real bank keeper.
func TestTradebin_TakerGasPerFill(t *testing.T) {
	gas := map[int]uint64{}
	for _, fills := range []int{1, 10, 100, 500} {
		gas[fills] = takerGasForFills(t, fills)
		t.Logf("gas consumed by one MsgCreateOrder filling %d resting orders: %d", fills, gas[fills])
	}
	t.Logf("marginal gas per fill (1 -> 500 fills): %d", (gas[500]-gas[1])/499)

	// well below 10% of the 1,000,000,000 block gas limit
	require.Less(t, gas[100], uint64(100_000_000))
	require.Less(t, gas[500], uint64(100_000_000))
}
