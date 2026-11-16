package v5_test

import (
	"errors"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/protowire"

	v5 "github.com/bze-alphateam/bze/x/tradebin/migrations/v5"
	tbtestutil "github.com/bze-alphateam/bze/x/tradebin/testutil"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

const marketId = "ustake/ubze"

var testMarket = types.Market{Base: "ustake", Quote: "ubze", Creator: "creator"}

// oldQueueMessage mirrors the removed bze.tradebin.QueueMessage proto message.
type oldQueueMessage struct {
	messageId   string
	marketId    string
	messageType string
	amount      string
	price       string
	createdAt   uint64
	orderId     string
	orderType   string
	owner       string
}

// encode produces the protobuf encoding the old binary stored under Qm/value/.
func (m oldQueueMessage) encode() []byte {
	var b []byte
	appendString := func(num protowire.Number, v string) {
		if v == "" {
			return
		}
		b = protowire.AppendTag(b, num, protowire.BytesType)
		b = protowire.AppendString(b, v)
	}
	appendString(1, m.messageId)
	appendString(2, m.marketId)
	appendString(3, m.messageType)
	appendString(4, m.amount)
	appendString(5, m.price)
	if m.createdAt > 0 {
		b = protowire.AppendTag(b, 6, protowire.VarintType)
		b = protowire.AppendVarint(b, m.createdAt)
	}
	appendString(7, m.orderId)
	appendString(8, m.orderType)
	appendString(9, m.owner)

	return b
}

func (m oldQueueMessage) key() []byte {
	return []byte(v5.QueueMessagePrefix + m.marketId + "/" + m.messageId + "/")
}

type migrationTest struct {
	ctx   sdk.Context
	store storetypes.KVStore
	k     *tbtestutil.MockV5OrderKeeper
	bank  *tbtestutil.MockV5BankKeeper
}

func setup(t *testing.T) migrationTest {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)
	ctrl := gomock.NewController(t)

	return migrationTest{
		ctx:   ctx,
		store: ctx.KVStore(storeKey),
		k:     tbtestutil.NewMockV5OrderKeeper(ctrl),
		bank:  tbtestutil.NewMockV5BankKeeper(ctrl),
	}
}

func (mt migrationTest) refundedEvents(t *testing.T) []*types.OrderMessageRefundedEvent {
	var list []*types.OrderMessageRefundedEvent
	for _, ev := range mt.ctx.EventManager().Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		require.NoError(t, err)
		if r, ok := msg.(*types.OrderMessageRefundedEvent); ok {
			list = append(list, r)
		}
	}

	return list
}

func requireNoQueueKeys(t *testing.T, store storetypes.KVStore) {
	it := storetypes.KVStorePrefixIterator(store, []byte("Qm/"))
	defer it.Close()
	require.False(t, it.Valid(), "a key under Qm/ survived the migration")
}

func TestMigrate_RefundsQueuedOrdersOnceAndRemovesTheQueue(t *testing.T) {
	mt := setup(t)
	buyer := sdk.AccAddress("buyer_______________")
	filler := sdk.AccAddress("filler______________")
	canceller := sdk.AccAddress("canceller___________")

	buy := oldQueueMessage{messageId: "000000000000000000000001", marketId: marketId, messageType: types.OrderTypeBuy, amount: "1000", price: "2", createdAt: 7, orderType: types.OrderTypeBuy, owner: buyer.String()}
	// a fill of buy orders was queued as a sell: order_type already holds the escrowed side
	fill := oldQueueMessage{messageId: "000000000000000000000002", marketId: marketId, messageType: "fill_buy", amount: "500", price: "3", orderType: types.OrderTypeSell, owner: filler.String()}
	cancel := oldQueueMessage{messageId: "000000000000000000000003", marketId: marketId, messageType: "cancel", orderId: "000000000000000000000042", orderType: types.OrderTypeSell, owner: canceller.String()}
	for _, m := range []oldQueueMessage{buy, fill, cancel} {
		mt.store.Set(m.key(), m.encode())
	}
	mt.store.Set([]byte("Qm/counter/cnt/"), []byte{0, 0, 0, 0, 0, 0, 0, 3})
	mt.store.Set([]byte("Qm/pc/"+marketId+"/sell/000000000000000000000042/"), []byte{1})
	// a resting order is not queue state and must survive
	orderKey := []byte("Order/value/" + marketId + "/sell/000000000000000000000042/")
	mt.store.Set(orderKey, []byte("order"))

	buyRefund := sdk.NewInt64Coin("ubze", 2000)
	buyDust := math.LegacyMustNewDecFromStr("0.5")
	buyUserDust := &types.UserDust{Owner: buyer.String(), Denom: "ubze", Amount: "0.5"}
	fillRefund := sdk.NewInt64Coin("ustake", 500)

	mt.k.EXPECT().GetMarketById(gomock.Any(), marketId).Return(testMarket, true).Times(2)
	gomock.InOrder(
		mt.k.EXPECT().GetOrderCoinsWithDust(gomock.Any(), types.OrderCoinsArguments{
			OrderType: types.OrderTypeBuy, OrderPrice: "2", OrderAmount: math.NewInt(1000), Market: &testMarket, UserAddress: buyer.String(), UserReceives: true,
		}).Return(types.OrderCoins{Coin: buyRefund, Dust: buyDust, UserDust: buyUserDust}, nil),
		mt.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, buyer, sdk.NewCoins(buyRefund)).Return(nil),
		mt.k.EXPECT().StoreProcessedUserDust(gomock.Any(), buyUserDust, &buyDust),
		mt.k.EXPECT().GetOrderCoinsWithDust(gomock.Any(), types.OrderCoinsArguments{
			OrderType: types.OrderTypeSell, OrderPrice: "3", OrderAmount: math.NewInt(500), Market: &testMarket, UserAddress: filler.String(), UserReceives: true,
		}).Return(types.OrderCoins{Coin: fillRefund, Dust: math.LegacyZeroDec()}, nil),
		mt.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, filler, sdk.NewCoins(fillRefund)).Return(nil),
		mt.k.EXPECT().StoreProcessedUserDust(gomock.Any(), gomock.Nil(), gomock.Any()),
	)

	require.NoError(t, v5.Migrate(mt.ctx, mt.store, mt.k, mt.bank))

	requireNoQueueKeys(t, mt.store)
	require.Equal(t, []byte("order"), mt.store.Get(orderKey))

	events := mt.refundedEvents(t)
	require.Len(t, events, 2, "one refund per escrowed message, none for the cancel")
	require.Equal(t, &types.OrderMessageRefundedEvent{
		OrderId: buy.messageId, MarketId: marketId, OrderType: types.OrderTypeBuy, Owner: buyer.String(),
		Amount: "1000", Refund: buyRefund.String(), Reason: types.RefundReasonUpgrade,
	}, events[0])
	require.Equal(t, &types.OrderMessageRefundedEvent{
		OrderId: fill.messageId, MarketId: marketId, OrderType: types.OrderTypeSell, Owner: filler.String(),
		Amount: "500", Refund: fillRefund.String(), Reason: types.RefundReasonUpgrade,
	}, events[1])
}

func TestMigrate_RefundRoundingToZeroStoresDustOnly(t *testing.T) {
	mt := setup(t)
	owner := sdk.AccAddress("owner_______________")
	m := oldQueueMessage{messageId: "000000000000000000000001", marketId: marketId, messageType: types.OrderTypeSell, amount: "1", price: "0.1", orderType: types.OrderTypeSell, owner: owner.String()}
	mt.store.Set(m.key(), m.encode())

	dust := math.LegacyMustNewDecFromStr("0.1")
	userDust := &types.UserDust{Owner: owner.String(), Denom: "ubze", Amount: "0.1"}
	mt.k.EXPECT().GetMarketById(gomock.Any(), marketId).Return(testMarket, true)
	mt.k.EXPECT().GetOrderCoinsWithDust(gomock.Any(), gomock.Any()).Return(types.OrderCoins{Coin: sdk.NewInt64Coin("ubze", 0), Dust: dust, UserDust: userDust}, nil)
	mt.k.EXPECT().StoreProcessedUserDust(gomock.Any(), userDust, &dust)
	// no bank send: the mock fails the test on an unexpected call

	require.NoError(t, v5.Migrate(mt.ctx, mt.store, mt.k, mt.bank))

	requireNoQueueKeys(t, mt.store)
	events := mt.refundedEvents(t)
	require.Len(t, events, 1)
	require.Equal(t, "", events[0].Refund)
	require.Equal(t, types.RefundReasonUpgrade, events[0].Reason)
}

func TestMigrate_EmptyQueueIsANoOp(t *testing.T) {
	mt := setup(t)
	mt.store.Set([]byte("Qm/counter/cnt/"), []byte{0, 0, 0, 0, 0, 0, 0, 0})

	require.NoError(t, v5.Migrate(mt.ctx, mt.store, mt.k, mt.bank))

	requireNoQueueKeys(t, mt.store)
	require.Empty(t, mt.refundedEvents(t))
}

func TestMigrate_Errors(t *testing.T) {
	owner := sdk.AccAddress("owner_______________")
	valid := oldQueueMessage{messageId: "000000000000000000000001", marketId: marketId, messageType: types.OrderTypeBuy, amount: "10", price: "1", orderType: types.OrderTypeBuy, owner: owner.String()}

	tests := []struct {
		name   string
		msg    oldQueueMessage
		raw    []byte
		expect func(mt migrationTest)
	}{
		{name: "undecodable value", msg: valid, raw: []byte{0x0a, 0x05, 'a'}},
		{name: "invalid amount", msg: func() oldQueueMessage { m := valid; m.amount = "ten"; return m }()},
		{name: "invalid owner", msg: func() oldQueueMessage { m := valid; m.owner = "nobody"; return m }()},
		{
			name: "market not found", msg: valid,
			expect: func(mt migrationTest) {
				mt.k.EXPECT().GetMarketById(gomock.Any(), marketId).Return(types.Market{}, false)
			},
		},
		{
			name: "order coins error", msg: valid,
			expect: func(mt migrationTest) {
				mt.k.EXPECT().GetMarketById(gomock.Any(), marketId).Return(testMarket, true)
				mt.k.EXPECT().GetOrderCoinsWithDust(gomock.Any(), gomock.Any()).Return(types.OrderCoins{}, errors.New("boom"))
			},
		},
		{
			name: "bank error", msg: valid,
			expect: func(mt migrationTest) {
				mt.k.EXPECT().GetMarketById(gomock.Any(), marketId).Return(testMarket, true)
				mt.k.EXPECT().GetOrderCoinsWithDust(gomock.Any(), gomock.Any()).Return(types.OrderCoins{Coin: sdk.NewInt64Coin("ubze", 10), Dust: math.LegacyZeroDec()}, nil)
				mt.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, gomock.Any()).Return(errors.New("boom"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mt := setup(t)
			value := tc.msg.encode()
			if tc.raw != nil {
				value = tc.raw
			}
			mt.store.Set(tc.msg.key(), value)
			if tc.expect != nil {
				tc.expect(mt)
			}

			require.Error(t, v5.Migrate(mt.ctx, mt.store, mt.k, mt.bank))
		})
	}
}
