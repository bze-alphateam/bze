package v5_test

import (
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v5 "github.com/bze-alphateam/bze/x/tradebin/migrations/v5"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// The queue was filled by binaries that parsed amounts with math.NewIntFromString (base 0), so a queued amount
// string can be in any form that parser accepts. The refund is computed on the value that was escrowed: "0x64"
// is 100 and "010" is 8 (octal). The migration does not fail on these forms and the queue is cleared.
func TestMigrate_NonCanonicalQueuedAmountsRefundTheEscrowedValue(t *testing.T) {
	mt := setup(t)
	owner := sdk.AccAddress("owner_______________")
	cases := []struct {
		id, amount string
		value      int64
	}{
		{id: "000000000000000000000001", amount: "0x64", value: 100},
		{id: "000000000000000000000002", amount: "010", value: 8},
	}

	mt.k.EXPECT().GetMarketById(gomock.Any(), marketId).Return(testMarket, true).Times(len(cases))
	var calls []any
	for _, c := range cases {
		m := oldQueueMessage{messageId: c.id, marketId: marketId, messageType: types.OrderTypeBuy, amount: c.amount, price: "2", orderType: types.OrderTypeBuy, owner: owner.String()}
		mt.store.Set(m.key(), m.encode())

		refund := sdk.NewInt64Coin("ubze", 2*c.value)
		dust := math.LegacyZeroDec()
		calls = append(calls,
			mt.k.EXPECT().GetOrderCoinsWithDust(gomock.Any(), types.OrderCoinsArguments{
				OrderType: types.OrderTypeBuy, OrderPrice: "2", OrderAmount: math.NewInt(c.value), Market: &testMarket, UserAddress: owner.String(), UserReceives: true,
			}).Return(types.OrderCoins{Coin: refund, Dust: dust}, nil),
			mt.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(refund)).Return(nil),
			mt.k.EXPECT().StoreProcessedUserDust(gomock.Any(), gomock.Nil(), gomock.Any()),
		)
	}
	gomock.InOrder(calls...)

	require.NotPanics(t, func() {
		require.NoError(t, v5.Migrate(mt.ctx, mt.store, mt.k, mt.bank))
	})

	requireNoQueueKeys(t, mt.store)
	events := mt.refundedEvents(t)
	require.Len(t, events, len(cases))
	for i, c := range cases {
		require.Equal(t, c.id, events[i].OrderId)
		require.Equal(t, math.NewInt(c.value).String(), events[i].Amount)
		require.Equal(t, sdk.NewInt64Coin("ubze", 2*c.value).String(), events[i].Refund)
	}
}

// Queued messages were never added to an aggregate, so two buys of ~2^255 at one price could sit in the queue
// together. Each is refunded on its own: no sum of the two is ever made, nothing panics and the queue is cleared.
func TestMigrate_TwoHugeQueuedBuysAtOnePriceAreRefundedSeparately(t *testing.T) {
	mt := setup(t)
	huge := math.NewIntFromBigInt(new(big.Int).Lsh(big.NewInt(1), 255))
	owners := []sdk.AccAddress{sdk.AccAddress("owner1______________"), sdk.AccAddress("owner2______________")}

	mt.k.EXPECT().GetMarketById(gomock.Any(), marketId).Return(testMarket, true).Times(len(owners))
	var calls []any
	for i, owner := range owners {
		m := oldQueueMessage{messageId: fmt.Sprintf("%024d", i+1), marketId: marketId, messageType: types.OrderTypeBuy, amount: huge.String(), price: "1", orderType: types.OrderTypeBuy, owner: owner.String()}
		mt.store.Set(m.key(), m.encode())

		refund := sdk.NewCoin("ubze", huge)
		calls = append(calls,
			mt.k.EXPECT().GetOrderCoinsWithDust(gomock.Any(), types.OrderCoinsArguments{
				OrderType: types.OrderTypeBuy, OrderPrice: "1", OrderAmount: huge, Market: &testMarket, UserAddress: owner.String(), UserReceives: true,
			}).Return(types.OrderCoins{Coin: refund, Dust: math.LegacyZeroDec()}, nil),
			mt.bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, owner, sdk.NewCoins(refund)).Return(nil),
			mt.k.EXPECT().StoreProcessedUserDust(gomock.Any(), gomock.Nil(), gomock.Any()),
		)
	}
	gomock.InOrder(calls...)

	require.NotPanics(t, func() {
		require.NoError(t, v5.Migrate(mt.ctx, mt.store, mt.k, mt.bank))
	})

	requireNoQueueKeys(t, mt.store)
	events := mt.refundedEvents(t)
	require.Len(t, events, len(owners))
	for i, owner := range owners {
		require.Equal(t, owner.String(), events[i].Owner)
		require.Equal(t, huge.String(), events[i].Amount)
		require.Equal(t, sdk.NewCoin("ubze", huge).String(), events[i].Refund)
	}
}
