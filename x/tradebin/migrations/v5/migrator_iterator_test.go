package v5_test

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v5 "github.com/bze-alphateam/bze/x/tradebin/migrations/v5"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// iteratorTrackingStore is a KVStore whose iterators report whether one of them is still open
type iteratorTrackingStore struct {
	storetypes.KVStore
	open int
}

func (s *iteratorTrackingStore) Iterator(start, end []byte) storetypes.Iterator {
	s.open++
	return &trackedIterator{Iterator: s.KVStore.Iterator(start, end), store: s}
}

func (s *iteratorTrackingStore) ReverseIterator(start, end []byte) storetypes.Iterator {
	s.open++
	return &trackedIterator{Iterator: s.KVStore.ReverseIterator(start, end), store: s}
}

type trackedIterator struct {
	storetypes.Iterator
	store  *iteratorTrackingStore
	closed bool
}

func (it *trackedIterator) Close() error {
	if !it.closed {
		it.closed = true
		it.store.open--
	}

	return it.Iterator.Close()
}

// The refunds write user dust and send coins only after the queue iterator is closed.
func TestMigrate_NoRefundWhileTheQueueIteratorIsOpen(t *testing.T) {
	mt := setup(t)
	store := &iteratorTrackingStore{KVStore: mt.store}
	first := sdk.AccAddress("first_______________")
	second := sdk.AccAddress("second______________")
	for i, owner := range []sdk.AccAddress{first, second} {
		m := oldQueueMessage{messageId: orderIds[i], marketId: marketId, messageType: types.OrderTypeBuy, amount: "10", price: "1", orderType: types.OrderTypeBuy, owner: owner.String()}
		store.Set(m.key(), m.encode())
	}

	noOpenIterator := func(what string) {
		require.Zero(t, store.open, "%s while a store iterator is open", what)
	}
	refund := sdk.NewInt64Coin("ubze", 10)
	mt.k.EXPECT().GetMarketById(gomock.Any(), marketId).Return(testMarket, true).Times(2)
	mt.k.EXPECT().GetOrderCoinsWithDust(gomock.Any(), gomock.Any()).Return(types.OrderCoins{Coin: refund, Dust: math.LegacyZeroDec()}, nil).Times(2)
	gomock.InOrder(
		mt.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, first, sdk.NewCoins(refund)).
			Do(func(_, _, _, _ any) { noOpenIterator("bank send") }).Return(nil),
		mt.k.EXPECT().StoreProcessedUserDust(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(_, _, _ any) { noOpenIterator("dust write") }),
		mt.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, second, sdk.NewCoins(refund)).
			Do(func(_, _, _, _ any) { noOpenIterator("bank send") }).Return(nil),
		mt.k.EXPECT().StoreProcessedUserDust(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(_, _, _ any) { noOpenIterator("dust write") }),
	)

	require.NoError(t, v5.Migrate(mt.ctx, store, mt.k, mt.bank))

	require.Zero(t, store.open, "every iterator is closed")
	requireNoQueueKeys(t, mt.store)
	events := mt.refundedEvents(t)
	require.Len(t, events, 2)
	require.Equal(t, first.String(), events[0].Owner)
	require.Equal(t, second.String(), events[1].Owner)
}

var orderIds = []string{"000000000000000000000001", "000000000000000000000002"}
