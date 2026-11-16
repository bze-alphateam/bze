package app_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

const (
	rollbackBase  = "urbbase"
	rollbackQuote = "urbquote"

	// the two makers at rollbackPrice receive 33.3 quote each for 100 base: every fill leaves dust
	rollbackPrice     = "0.333"
	rollbackHighPrice = "0.5"
	rollbackMakerSell = 100

	executedEventType = "bze.tradebin.OrderExecutedEvent"
)

// rollbackBook is a market with three resting sells of rollbackMakerSell base from distinct makers, two at
// rollbackPrice and one at rollbackHighPrice, placed through the real router with the real bank keeper.
// With a trading reward the market is fillGasBase/native and has an active reward with the maximum slots
// and a full leaderboard; without one both denoms are non-native, so a trader's native balance is only ever
// used for protocol fees.
type rollbackBook struct {
	bApp     *app.App
	ctx      sdk.Context
	params   v2types.Params
	market   tradebintypes.Market
	marketId string
	makers   []sdk.AccAddress
	taker    sdk.AccAddress
	rewardId string
	probe    *fillHookProbe
}

func newRollbackBook(t *testing.T, withReward bool) *rollbackBook {
	t.Helper()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()
	bApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)
	ctx := bApp.NewUncachedContext(false, cmtproto.Header{Time: time.Unix(1_700_000_000, 0)})

	require.NoError(t, bApp.BankKeeper.SetParams(ctx, banktypes.DefaultParams()))
	params := v2types.DefaultParams()
	require.NoError(t, bApp.TradebinKeeper.SetParams(ctx, params))

	b := &rollbackBook{
		bApp:   bApp,
		ctx:    ctx,
		params: params,
		market: tradebintypes.Market{Base: rollbackBase, Quote: rollbackQuote, Creator: sdk.AccAddress("market_creator______").String()},
		taker:  sdk.AccAddress("rollback_taker______"),
	}
	if withReward {
		b.market.Base, b.market.Quote = fillGasBase, params.NativeDenom
	}
	bApp.TradebinKeeper.SetMarket(ctx, b.market)
	b.marketId = b.market.Base + "/" + b.market.Quote

	deliver := func(ctx sdk.Context, msg sdk.Msg) {
		_, err := bApp.MsgServiceRouter().Handler(msg)(ctx, msg)
		require.NoError(t, err)
	}
	if withReward {
		b.rewardId = activeTradingRewardWithFullLeaderboard(t, bApp, ctx, b.marketId, b.fund(t), deliver)
	}

	for i, price := range []string{rollbackPrice, rollbackPrice, rollbackHighPrice} {
		maker := sdk.AccAddress(fmt.Sprintf("rollback_maker%06d", i))
		b.fund(t)(maker, sdk.NewCoins(sdk.NewInt64Coin(b.market.Base, rollbackMakerSell), params.MarketMakerFee))
		deliver(ctx, &tradebintypes.MsgCreateOrder{
			Creator: maker.String(), MarketId: b.marketId, OrderType: tradebintypes.OrderTypeSell,
			Amount: fmt.Sprintf("%d", rollbackMakerSell), Price: price,
		})
		b.makers = append(b.makers, maker)
	}
	require.Len(t, bApp.TradebinKeeper.GetAllOrder(ctx), 3)
	requireEscrowConserved(t, bApp, ctx, nil)

	b.probe = installFillHookProbe(bApp)

	return b
}

func (b *rollbackBook) fund(t *testing.T) func(sdk.AccAddress, sdk.Coins) {
	return func(addr sdk.AccAddress, coins sdk.Coins) {
		require.NoError(t, b.bApp.BankKeeper.MintCoins(b.ctx, minttypes.ModuleName, coins))
		require.NoError(t, b.bApp.BankKeeper.SendCoinsFromModuleToAccount(b.ctx, minttypes.ModuleName, addr, coins))
	}
}

// createBuy crosses both makers at rollbackPrice: two fills, 66.6 quote escrowed.
func (b *rollbackBook) createBuy() sdk.Msg {
	return &tradebintypes.MsgCreateOrder{
		Creator: b.taker.String(), MarketId: b.marketId, OrderType: tradebintypes.OrderTypeBuy, Amount: "200", Price: rollbackPrice,
	}
}

// fillSells is a two-entry fill at fractional and round prices: entry 1 fills both makers at rollbackPrice
// (66.6 quote, escrowed as 67 with 0.4 dust kept for the taker), entry 2 the maker at rollbackHighPrice (50
// quote). The taker fee is charged after both.
func (b *rollbackBook) fillSells() sdk.Msg {
	return &tradebintypes.MsgFillOrders{
		Creator: b.taker.String(), MarketId: b.marketId, OrderType: tradebintypes.OrderTypeSell,
		Orders: []*tradebintypes.FillOrderItem{{Price: rollbackPrice, Amount: "200"}, {Price: rollbackHighPrice, Amount: "100"}},
	}
}

// fillHookProbe wraps the trading reward order fill hook the app registers and records the tx gas consumed
// when each call enters and leaves it. Reading the gas meter consumes no gas, so the message's gas is the
// same as with the app's own wiring.
type fillHookProbe struct {
	enter, exit []uint64
}

func installFillHookProbe(bApp *app.App) *fillHookProbe {
	p := &fillHookProbe{}
	hook := bApp.RewardsKeeper.GetOnOrderFillHook()
	bApp.TradebinKeeper.SetOnOrderFillHooks([]tradebintypes.OnMarketOrderFill{
		func(ctx sdk.Context, marketId, amountTraded, userAddress string) {
			p.enter = append(p.enter, ctx.GasMeter().GasConsumed())
			hook(ctx, marketId, amountTraded, userAddress)
			p.exit = append(p.exit, ctx.GasMeter().GasConsumed())
		},
	})

	return p
}

func (p *fillHookProbe) reset() {
	p.enter, p.exit = nil, nil
}

// runAsTxWithGas runs msg like runAsTx on a gas meter limited to gasLimit and turns an out of gas panic into
// sdkerrors.ErrOutOfGas, the way baseapp runs a transaction. The cache is written only on success. The
// returned context is the transaction's own, with an infinite gas meter, so the test can read what the
// message did before it failed.
func runAsTxWithGas(bApp *app.App, ctx sdk.Context, gasLimit uint64, msg sdk.Msg) (txCtx sdk.Context, res *sdk.Result, err error) {
	cacheCtx, write := ctx.CacheContext()
	txCtx = cacheCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	defer func() {
		if r := recover(); r != nil {
			oog, ok := r.(storetypes.ErrorOutOfGas)
			if !ok {
				panic(r)
			}
			res, err = nil, errorsmod.Wrap(sdkerrors.ErrOutOfGas, oog.Descriptor)
		}
	}()

	res, err = bApp.MsgServiceRouter().Handler(msg)(cacheCtx.WithGasMeter(storetypes.NewGasMeter(gasLimit)), msg)
	if err == nil {
		write()
	}

	return txCtx, res, err
}

// dryRun runs msg on a cache that is thrown away, with an infinite gas meter, and returns its gas and result.
func dryRun(t *testing.T, b *rollbackBook, msg sdk.Msg) (uint64, *sdk.Result) {
	t.Helper()
	cacheCtx, _ := b.ctx.CacheContext()
	cacheCtx = cacheCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	res, err := b.bApp.MsgServiceRouter().Handler(msg)(cacheCtx, msg)
	require.NoError(t, err)

	return cacheCtx.GasMeter().GasConsumed(), res
}

// bookState is the state a tradebin message can change, read through the keepers.
type bookState struct {
	Balances    map[string]string
	Orders      []tradebintypes.Order
	Aggregates  []tradebintypes.AggregatedOrder
	OwnerIndex  []tradebintypes.OrderReference
	History     []tradebintypes.HistoryOrder
	Dust        []tradebintypes.UserDust
	Candidates  []rewardstypes.TradingRewardCandidate
	Leaderboard rewardstypes.TradingRewardLeaderboard
	Rewards     []rewardstypes.TradingReward
}

func captureBookState(b *rollbackBook, ctx sdk.Context) bookState {
	s := bookState{Balances: map[string]string{}}
	accounts := append([]sdk.AccAddress{b.taker, authtypes.NewModuleAddress(tradebintypes.ModuleName)}, b.makers...)
	for _, addr := range accounts {
		s.Balances[addr.String()] = b.bApp.BankKeeper.GetAllBalances(ctx, addr).String()
	}
	k := b.bApp.TradebinKeeper
	s.Orders = k.GetAllOrder(ctx)
	s.Aggregates = k.GetAllAggregatedOrder(ctx)
	s.OwnerIndex = k.GetAllOwnerPriceOrder(ctx)
	s.History = k.GetAllHistoryOrder(ctx)
	s.Dust = k.GetAllUserDust(ctx)
	s.Candidates = b.bApp.RewardsKeeper.GetAllTradingRewardCandidate(ctx)
	s.Leaderboard, _ = b.bApp.RewardsKeeper.GetTradingRewardLeaderboard(ctx, b.rewardId)
	s.Rewards = b.bApp.RewardsKeeper.GetAllActiveTradingReward(ctx)

	return s
}

// storeSnapshot copies every key and value of every KV store of the app.
func storeSnapshot(bApp *app.App, ctx sdk.Context) map[string]map[string][]byte {
	snapshot := map[string]map[string][]byte{}
	for _, key := range bApp.GetStoreKeys() {
		kvKey, ok := key.(*storetypes.KVStoreKey)
		if !ok {
			continue
		}

		entries := map[string][]byte{}
		iterator := ctx.KVStore(kvKey).Iterator(nil, nil)
		for ; iterator.Valid(); iterator.Next() {
			entries[string(iterator.Key())] = bytes.Clone(iterator.Value())
		}
		iterator.Close()
		snapshot[kvKey.Name()] = entries
	}

	return snapshot
}

// requireSameStores fails with every key written, changed or deleted between the two snapshots.
func requireSameStores(t *testing.T, before, after map[string]map[string][]byte) {
	t.Helper()
	var changed []string
	for name, entries := range after {
		for key, value := range entries {
			old, found := before[name][key]
			if !found || !bytes.Equal(old, value) {
				changed = append(changed, fmt.Sprintf("%s: written %x", name, key))
			}
		}
	}
	for name, entries := range before {
		for key := range entries {
			if _, found := after[name][key]; !found {
				changed = append(changed, fmt.Sprintf("%s: deleted %x", name, key))
			}
		}
	}
	require.Empty(t, changed, "store keys left behind by the failed transaction")
}

// requireEscrowConserved checks, per denom, that the tradebin module account holds exactly what it owes:
// the escrow of every resting order (amount × price quote for a buy, amount base for a sell), every user's
// stored dust and extra (escrow held for something else, such as queued messages). The test markets have no
// liquidity pool, so the module holds nothing else.
func requireEscrowConserved(t *testing.T, bApp *app.App, ctx sdk.Context, extra map[string]math.LegacyDec) {
	t.Helper()
	owed := map[string]math.LegacyDec{}
	add := func(denom string, amount math.LegacyDec) {
		if current, found := owed[denom]; found {
			amount = amount.Add(current)
		}
		owed[denom] = amount
	}
	for denom, amount := range extra {
		add(denom, amount)
	}

	k := bApp.TradebinKeeper
	for _, order := range k.GetAllOrder(ctx) {
		market, found := k.GetMarketById(ctx, order.MarketId)
		require.True(t, found)
		amount := math.LegacyMustNewDecFromStr(order.Amount)
		if order.OrderType == tradebintypes.OrderTypeBuy {
			add(market.Quote, amount.Mul(math.LegacyMustNewDecFromStr(order.Price)))
		} else {
			add(market.Base, amount)
		}
	}
	for _, dust := range k.GetAllUserDust(ctx) {
		add(dust.Denom, math.LegacyMustNewDecFromStr(dust.Amount))
	}

	balances := bApp.BankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(tradebintypes.ModuleName))
	for _, coin := range balances {
		if _, found := owed[coin.Denom]; !found {
			owed[coin.Denom] = math.LegacyZeroDec()
		}
	}
	for denom, amount := range owed {
		require.Equal(t, amount.String(), math.LegacyNewDecFromInt(balances.AmountOf(denom)).String(), "tradebin module balance of %s", denom)
	}
}

func countEvents(events []abci.Event, eventType string) int {
	n := 0
	for _, e := range events {
		if e.Type == eventType {
			n++
		}
	}

	return n
}

// requireRolledBack checks that the failed message left nothing behind, through the keepers and store by store.
func requireRolledBack(t *testing.T, b *rollbackBook, before bookState, snapshot map[string]map[string][]byte) {
	t.Helper()
	require.Equal(t, before, captureBookState(b, b.ctx))
	requireSameStores(t, snapshot, storeSnapshot(b.bApp, b.ctx))
	require.Empty(t, b.ctx.EventManager().Events())
	requireEscrowConserved(t, b.bApp, b.ctx, nil)
}

// requireReplayFills runs msg again with enough gas and funds and checks what it settles: the failed run left
// no state, in a store or in memory, that changes the outcome.
func requireReplayFills(t *testing.T, b *rollbackBook, msg sdk.Msg, makersPaid []int64, bought int64) {
	t.Helper()
	b.probe.reset()
	quoteBefore := b.bApp.BankKeeper.GetBalance(b.ctx, b.taker, b.market.Quote).Amount
	res, err := b.bApp.MsgServiceRouter().Handler(msg)(b.ctx, msg)
	require.NoError(t, err)
	require.Equal(t, len(makersPaid), countEvents(res.Events, executedEventType))
	require.Len(t, b.probe.exit, len(makersPaid))

	paid := int64(0)
	for i, amount := range makersPaid {
		require.Equal(t, amount, b.bApp.BankKeeper.GetBalance(b.ctx, b.makers[i], b.market.Quote).Amount.Int64(), "maker %d", i)
		paid += amount
	}
	require.Equal(t, bought, b.bApp.BankKeeper.GetBalance(b.ctx, b.taker, b.market.Base).Amount.Int64())
	spent := quoteBefore.Sub(b.bApp.BankKeeper.GetBalance(b.ctx, b.taker, b.market.Quote).Amount)
	if b.market.Quote == b.params.NativeDenom {
		spent = spent.Sub(b.params.MarketTakerFee.Amount)
	}
	// the makers were paid less than the taker escrowed by the dust kept for them
	require.True(t, spent.GT(math.NewInt(paid)), "taker spent %s for %d paid to makers", spent, paid)
	requireEscrowConserved(t, b.bApp, b.ctx, nil)
	if b.rewardId != "" {
		candidate, found := b.bApp.RewardsKeeper.GetTradingRewardCandidate(b.ctx, b.rewardId, b.taker.String())
		require.True(t, found)
		require.Equal(t, fmt.Sprintf("%d", bought), candidate.Amount)
	}
}

// A taker message that runs out of gas after it settled real fills (and, with a trading reward, inside the
// reward hook of a later fill) is reverted whole with the real bank keeper: balances, escrow dust, history,
// orders, the owner-price index and the owner counts, the reward candidate and leaderboard, every store key
// and every event. The limit comes from a dry run of the same message, so it lands exactly where the case
// says.
func TestTradebin_OutOfGasAfterFillsRollsBack(t *testing.T) {
	// between(k) runs out after fill k-1 left its hook and before fill k enters it: in fill k's settlement
	between := func(k int) func(*fillHookProbe, uint64) uint64 {
		return func(p *fillHookProbe, _ uint64) uint64 {
			require.Greater(t, p.enter[k], p.exit[k-1]+1)
			return p.exit[k-1] + (p.enter[k]-p.exit[k-1])/2
		}
	}
	// inside(k) runs out in the middle of fill k's trading reward hook
	inside := func(k int) func(*fillHookProbe, uint64) uint64 {
		return func(p *fillHookProbe, _ uint64) uint64 {
			require.Greater(t, p.exit[k], p.enter[k]+1)
			return p.enter[k] + (p.exit[k]-p.enter[k])/2
		}
	}
	lastGasUnit := func(_ *fillHookProbe, total uint64) uint64 { return total - 1 }

	for _, c := range []struct {
		name       string
		withReward bool
		msg        func(*rollbackBook) sdk.Msg
		gasLimit   func(*fillHookProbe, uint64) uint64
		// fill hooks entered and left before the gas ran out; every entered fill was settled
		entered, exited int
		makersPaid      []int64
		bought          int64
	}{
		{
			name: "create order, in the second fill", msg: (*rollbackBook).createBuy, gasLimit: between(1),
			entered: 1, exited: 1, makersPaid: []int64{33, 33}, bought: 200,
		},
		{
			name: "create order, inside the reward hook of the second fill", withReward: true, msg: (*rollbackBook).createBuy, gasLimit: inside(1),
			entered: 2, exited: 1, makersPaid: []int64{33, 33}, bought: 200,
		},
		{
			name: "fill orders, in the second entry", msg: (*rollbackBook).fillSells, gasLimit: between(2),
			entered: 2, exited: 2, makersPaid: []int64{33, 33, 50}, bought: 300,
		},
		{
			name: "fill orders, inside the reward hook of the second entry", withReward: true, msg: (*rollbackBook).fillSells, gasLimit: inside(2),
			entered: 3, exited: 2, makersPaid: []int64{33, 33, 50}, bought: 300,
		},
		{
			name: "fill orders, at the last gas unit", msg: (*rollbackBook).fillSells, gasLimit: lastGasUnit,
			entered: 3, exited: 3, makersPaid: []int64{33, 33, 50}, bought: 300,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			b := newRollbackBook(t, c.withReward)
			b.fund(t)(b.taker, sdk.NewCoins(sdk.NewInt64Coin(b.market.Quote, 1_000)).Add(b.params.MarketTakerFee))
			msg := c.msg(b)

			total, res := dryRun(t, b, msg)
			require.Equal(t, len(c.makersPaid), countEvents(res.Events, executedEventType))
			require.Len(t, b.probe.exit, len(c.makersPaid))
			limit := c.gasLimit(b.probe, total)

			// the parent context starts the transaction with no events, so it shows any event that leaks out
			b.ctx = b.ctx.WithEventManager(sdk.NewEventManager())
			before := captureBookState(b, b.ctx)
			snapshot := storeSnapshot(b.bApp, b.ctx)
			b.probe.reset()
			txCtx, res, err := runAsTxWithGas(b.bApp, b.ctx, limit, msg)
			require.ErrorIs(t, err, sdkerrors.ErrOutOfGas)
			require.Nil(t, res)
			require.Len(t, b.probe.enter, c.entered)
			require.Len(t, b.probe.exit, c.exited)

			// the message settled real fills before it ran out of gas
			for i := 0; i < c.entered; i++ {
				require.Equal(t, c.makersPaid[i], b.bApp.BankKeeper.GetBalance(txCtx, b.makers[i], b.market.Quote).Amount.Int64(), "maker %d paid in the tx", i)
			}
			require.GreaterOrEqual(t, len(b.bApp.TradebinKeeper.GetAllHistoryOrder(txCtx)), len(before.History)+c.entered)
			require.NotEmpty(t, b.bApp.TradebinKeeper.GetAllUserDust(txCtx))
			if c.withReward {
				_, found := b.bApp.RewardsKeeper.GetTradingRewardCandidate(txCtx, b.rewardId, b.taker.String())
				require.True(t, found, "an earlier fill's reward hook wrote the taker's candidate in the tx")
			}

			// and none of it is left
			requireRolledBack(t, b, before, snapshot)
			requireReplayFills(t, b, msg, c.makersPaid, c.bought)
		})
	}
}

// MsgFillOrders settles every entry before it charges the taker fee, and escrows each entry right before it
// executes it. When the fee or a later entry's escrow fails after real fills were settled, the message
// returns the error and every change it made is reverted with the real bank keeper.
func TestTradebin_FillOrdersFailureAfterSettlementRollsBack(t *testing.T) {
	for _, c := range []struct {
		name string
		// funds is the taker's balance: enough for the settled entries, short of what fails
		funds func(*rollbackBook) sdk.Coins
		// topUp is what the replay needs on top of funds
		topUp func(*rollbackBook) sdk.Coins
		// settled is what each maker was paid in the tx before it failed, fills the fills settled
		settled    []int64
		fills      int
		makersPaid []int64
		bought     int64
	}{
		{
			name: "taker fee after both entries",
			// 67 escrows entry 1 (66.6, 0.4 dust kept), 50 entry 2; no native denom for the fee
			funds:      func(b *rollbackBook) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin(b.market.Quote, 117)) },
			topUp:      func(b *rollbackBook) sdk.Coins { return sdk.NewCoins(b.params.MarketTakerFee) },
			settled:    []int64{33, 33, 50},
			fills:      3,
			makersPaid: []int64{33, 33, 50},
			bought:     300,
		},
		{
			name: "escrow of the second entry",
			funds: func(b *rollbackBook) sdk.Coins {
				return sdk.NewCoins(sdk.NewInt64Coin(b.market.Quote, 67), b.params.MarketTakerFee)
			},
			topUp:      func(b *rollbackBook) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin(b.market.Quote, 50)) },
			settled:    []int64{33, 33, 0},
			fills:      2,
			makersPaid: []int64{33, 33, 50},
			bought:     300,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			b := newRollbackBook(t, false)
			b.fund(t)(b.taker, c.funds(b))
			msg := b.fillSells()

			// the parent context starts the transaction with no events, so it shows any event that leaks out
			b.ctx = b.ctx.WithEventManager(sdk.NewEventManager())
			before := captureBookState(b, b.ctx)
			snapshot := storeSnapshot(b.bApp, b.ctx)
			txCtx, res, err := runAsTxWithGas(b.bApp, b.ctx, storetypes.Gas(1)<<62, msg)
			require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)
			require.Nil(t, res)

			// the entries before the failure were settled in the tx
			for i, amount := range c.settled {
				require.Equal(t, amount, b.bApp.BankKeeper.GetBalance(txCtx, b.makers[i], b.market.Quote).Amount.Int64(), "maker %d paid in the tx", i)
			}
			require.Len(t, b.probe.exit, c.fills)
			require.NotEmpty(t, b.bApp.TradebinKeeper.GetAllUserDust(txCtx))

			// and none of it is left
			requireRolledBack(t, b, before, snapshot)

			b.fund(t)(b.taker, c.topUp(b))
			requireReplayFills(t, b, msg, c.makersPaid, c.bought)
			require.True(t, b.bApp.BankKeeper.GetAllBalances(b.ctx, b.taker).AmountOf(b.market.Quote).IsZero(), "the replay used exactly the funds")
		})
	}
}
