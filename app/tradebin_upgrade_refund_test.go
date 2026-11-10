package app_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/bze-alphateam/bze/app"
	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	v5 "github.com/bze-alphateam/bze/x/tradebin/migrations/v5"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

// message types of the pre-v8.3.0 order book queue (a create order used its order type)
const (
	queuedFillBuy  = "fill_buy"
	queuedFillSell = "fill_sell"
	queuedCancel   = "cancel"
)

// queuedMessage is a message the pre-v8.3.0 binary queued for EndBlock, stored as its
// bze.tradebin.QueueMessage proto encoding under the queue prefix.
type queuedMessage struct {
	id, messageType, orderType, amount, price, orderId string
	owner                                              sdk.AccAddress
}

func (m queuedMessage) encode(marketId string) []byte {
	var b []byte
	for _, f := range []struct {
		num protowire.Number
		v   string
	}{{1, m.id}, {2, marketId}, {3, m.messageType}, {4, m.amount}, {5, m.price}, {7, m.orderId}, {8, m.orderType}, {9, m.owner.String()}} {
		if f.v == "" {
			continue
		}
		b = protowire.AppendTag(b, f.num, protowire.BytesType)
		b = protowire.AppendString(b, f.v)
	}
	b = protowire.AppendTag(b, 6, protowire.VarintType)

	return protowire.AppendVarint(b, 1_699_999_000)
}

// The v8.3.0 upgrade refunds every queued order book message with the real bank keeper. Queued buys and a
// queued fill at fractional prices escrowed whole coins and kept the rest as their owner's dust; their
// refund pays the escrow back with that dust, so each owner ends with exactly the coins they escrowed and no
// dust. The module account holds exactly the resting orders' escrow plus the dust still owed, before the
// upgrade (with the queued escrow on top) and after it; the supply does not move.
func TestTradebin_UpgradeRefundsFractionalQueuedOrders(t *testing.T) {
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()
	bApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)
	ctx := bApp.NewUncachedContext(false, cmtproto.Header{Time: time.Unix(1_700_000_000, 0)})
	require.NoError(t, bApp.BankKeeper.SetParams(ctx, banktypes.DefaultParams()))
	bApp.IBCKeeper.ClientKeeper.SetParams(ctx, clienttypes.DefaultParams())
	params := v2types.DefaultParams()
	require.NoError(t, bApp.TradebinKeeper.SetParams(ctx, params))

	k := bApp.TradebinKeeper
	market := tradebintypes.Market{Base: rollbackBase, Quote: rollbackQuote, Creator: sdk.AccAddress("market_creator______").String()}
	k.SetMarket(ctx, market)
	marketId := market.Base + "/" + market.Quote

	fund := func(addr sdk.AccAddress, coins sdk.Coins) {
		require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
		require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, addr, coins))
	}
	deliver := func(msg sdk.Msg) {
		_, err := bApp.MsgServiceRouter().Handler(msg)(ctx, msg)
		require.NoError(t, err)
	}

	// the book rests a fractional buy (3.3 quote escrowed as 4, 0.7 dust) and a sell
	restBuyer := sdk.AccAddress("upgrade_rest_buyer__")
	restSeller := sdk.AccAddress("upgrade_rest_seller_")
	fund(restBuyer, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 4), params.MarketMakerFee))
	fund(restSeller, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 20), params.MarketMakerFee))
	deliver(&tradebintypes.MsgCreateOrder{Creator: restBuyer.String(), MarketId: marketId, OrderType: tradebintypes.OrderTypeBuy, Amount: "11", Price: "0.3"})
	deliver(&tradebintypes.MsgCreateOrder{Creator: restSeller.String(), MarketId: marketId, OrderType: tradebintypes.OrderTypeSell, Amount: "20", Price: "0.9"})
	resting := k.GetAllOrder(ctx)
	require.Len(t, resting, 2)
	var sellId string
	for _, o := range resting {
		if o.OrderType == tradebintypes.OrderTypeSell {
			sellId = o.Id
		}
	}

	// queued messages, escrowed the way the old binary did when it queued them: order coins with the payer's
	// dust, dust stored, coins sent to the module
	queuer := sdk.AccAddress("upgrade_queued_buyer") // two buys: 3.33 escrowed as 4 (0.67 dust), 3.3 as 3 (0.37 dust)
	filler := sdk.AccAddress("upgrade_queued_fill_") // a fill of sells: 8.1 escrowed as 9 (0.9 dust), 9.1 as 9 (0.8 dust)
	seller := sdk.AccAddress("upgrade_queued_sell_") // a fill of buys: 10 base
	initial := map[string]sdk.Coins{
		queuer.String(): sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 7)),
		filler.String(): sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 18)),
		seller.String(): sdk.NewCoins(sdk.NewInt64Coin(market.Base, 10)),
	}
	for owner, coins := range initial {
		fund(sdk.MustAccAddressFromBech32(owner), coins)
	}

	queued := []queuedMessage{
		{messageType: tradebintypes.OrderTypeBuy, orderType: tradebintypes.OrderTypeBuy, amount: "10", price: "0.333", owner: queuer},
		{messageType: tradebintypes.OrderTypeBuy, orderType: tradebintypes.OrderTypeBuy, amount: "11", price: "0.3", owner: queuer},
		{messageType: queuedFillSell, orderType: tradebintypes.OrderTypeBuy, amount: "9", price: "0.9", owner: filler},
		{messageType: queuedFillSell, orderType: tradebintypes.OrderTypeBuy, amount: "13", price: "0.7", owner: filler},
		{messageType: queuedFillBuy, orderType: tradebintypes.OrderTypeSell, amount: "10", price: "0.3", owner: seller},
		// a queued cancel escrowed nothing and refunds nothing: its order stays on the book
		{messageType: queuedCancel, orderType: tradebintypes.OrderTypeSell, amount: "20", price: "0.9", orderId: sellId, owner: restSeller},
	}
	queuedEscrow := map[string]math.LegacyDec{market.Quote: math.LegacyZeroDec(), market.Base: math.LegacyZeroDec()}
	store := ctx.KVStore(bApp.GetKey(tradebintypes.StoreKey))
	for i := range queued {
		qm := &queued[i]
		qm.id = fmt.Sprintf("%024d", 100+i)
		store.Set([]byte(v5.QueueMessagePrefix+marketId+"/"+qm.id+"/"), qm.encode(marketId))
		if qm.messageType == queuedCancel {
			store.Set([]byte(v5.PendingCancelPrefix+marketId+"/"+qm.orderId+"/"), []byte{1})
			continue
		}

		amount, ok := math.NewIntFromString(qm.amount)
		require.True(t, ok)
		orderCoins, err := k.GetOrderCoinsWithDust(ctx, tradebintypes.OrderCoinsArguments{
			OrderType: qm.orderType, OrderPrice: qm.price, OrderAmount: amount, Market: &market, UserAddress: qm.owner.String(),
		})
		require.NoError(t, err)
		require.NoError(t, bApp.BankKeeper.SendCoinsFromAccountToModule(ctx, qm.owner, tradebintypes.ModuleName, sdk.NewCoins(orderCoins.Coin)))
		k.StoreProcessedUserDust(ctx, orderCoins.UserDust, &orderCoins.Dust)

		if qm.orderType == tradebintypes.OrderTypeBuy {
			queuedEscrow[market.Quote] = queuedEscrow[market.Quote].Add(math.LegacyNewDecFromInt(amount).Mul(math.LegacyMustNewDecFromStr(qm.price)))
		} else {
			queuedEscrow[market.Base] = queuedEscrow[market.Base].Add(math.LegacyNewDecFromInt(amount))
		}
	}
	store.Set([]byte(v5.QueueMessageCounterPrefix), []byte("106"))

	// the escrow took every coin the queued owners had, the fractions are their dust
	for owner := range initial {
		require.True(t, bApp.BankKeeper.GetAllBalances(ctx, sdk.MustAccAddressFromBech32(owner)).IsZero(), owner)
	}
	requireDust(t, bApp, ctx, queuer, market.Quote, "0.370000000000000000")
	requireDust(t, bApp, ctx, filler, market.Quote, "0.800000000000000000")
	requireDust(t, bApp, ctx, restBuyer, market.Quote, "0.700000000000000000")
	moduleAddr := authtypes.NewModuleAddress(tradebintypes.ModuleName)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 30), sdk.NewInt64Coin(market.Quote, 29)).String(), bApp.BankKeeper.GetAllBalances(ctx, moduleAddr).String())
	requireEscrowConserved(t, bApp, ctx, queuedEscrow)
	supplyBefore := sdk.NewCoins(bApp.BankKeeper.GetSupply(ctx, market.Base), bApp.BankKeeper.GetSupply(ctx, market.Quote))

	// the upgrade, with x/tradebin at consensus version 4
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	vm := bApp.ModuleManager.GetVersionMap()
	vm[tradebintypes.ModuleName] = 4
	newVM, err := v830.CreateUpgradeHandler(bApp.Configurator(), bApp.ModuleManager, bApp.GuardianKeeper, bApp.RatelimitKeeper, bApp.BankKeeper, bApp.TradebinKeeper, bApp.BankKeeper, bApp.RewardsKeeper, ibcStoreService(bApp))(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, vm,
	)
	require.NoError(t, err)
	require.Equal(t, uint64(5), newVM[tradebintypes.ModuleName])

	// every queued owner has exactly the coins they escrowed back, and no dust
	for owner, coins := range initial {
		addr := sdk.MustAccAddressFromBech32(owner)
		require.Equal(t, coins.String(), bApp.BankKeeper.GetAllBalances(ctx, addr).String(), owner)
		require.Empty(t, k.GetUserDustByOwner(ctx, owner), owner)
	}
	// the resting orders and their owners' dust are untouched, the cancel did nothing
	require.Equal(t, resting, k.GetAllOrder(ctx))
	requireDust(t, bApp, ctx, restBuyer, market.Quote, "0.700000000000000000")
	for _, agg := range k.GetAllAggregatedOrder(ctx) {
		require.Equal(t, uint64(1), agg.Owners, "%s %s", agg.OrderType, agg.Price)
	}
	require.Len(t, k.GetAllOwnerPriceOrder(ctx), 2)

	// the module holds the resting escrow and the dust owed: 3.3 + 0.7 quote, 20 base
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 20), sdk.NewInt64Coin(market.Quote, 4)).String(), bApp.BankKeeper.GetAllBalances(ctx, moduleAddr).String())
	requireEscrowConserved(t, bApp, ctx, nil)
	require.Equal(t, supplyBefore.String(), sdk.NewCoins(bApp.BankKeeper.GetSupply(ctx, market.Base), bApp.BankKeeper.GetSupply(ctx, market.Quote)).String())

	// the queue store is gone
	iterator := storetypes.KVStorePrefixIterator(ctx.KVStore(bApp.GetKey(tradebintypes.StoreKey)), []byte("Qm/"))
	require.False(t, iterator.Valid())
	require.NoError(t, iterator.Close())

	// one refund event per escrowed message, in queue order
	var refunds []*tradebintypes.OrderMessageRefundedEvent
	for _, e := range ctx.EventManager().ABCIEvents() {
		if e.Type != "bze.tradebin.OrderMessageRefundedEvent" {
			continue
		}
		msg, err := sdk.ParseTypedEvent(e)
		require.NoError(t, err)
		refunds = append(refunds, msg.(*tradebintypes.OrderMessageRefundedEvent))
	}
	wantRefunds := []string{"3" + market.Quote, "4" + market.Quote, "8" + market.Quote, "10" + market.Quote, "10" + market.Base}
	require.Len(t, refunds, len(wantRefunds))
	for i, refund := range refunds {
		require.Equal(t, queued[i].id, refund.OrderId)
		require.Equal(t, queued[i].owner.String(), refund.Owner)
		require.Equal(t, queued[i].amount, refund.Amount)
		require.Equal(t, wantRefunds[i], refund.Refund)
		require.Equal(t, tradebintypes.RefundReasonUpgrade, refund.Reason)
	}
}

func requireDust(t *testing.T, bApp *app.App, ctx sdk.Context, owner sdk.AccAddress, denom, amount string) {
	t.Helper()
	dust, found := bApp.TradebinKeeper.GetUserDust(ctx, owner.String(), denom)
	require.True(t, found, "dust of %s", owner)
	require.Equal(t, amount, dust.Amount)
}
