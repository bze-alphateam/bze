package v5

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// OrderKeeper is the tradebin state the migration needs to refund a queued message.
type OrderKeeper interface {
	GetMarketById(ctx sdk.Context, marketId string) (val types.Market, found bool)
	GetOrderCoinsWithDust(ctx sdk.Context, orderCoinsArgs types.OrderCoinsArguments) (types.OrderCoins, error)
	StoreProcessedUserDust(ctx sdk.Context, userDust *types.UserDust, userDustDec *math.LegacyDec)
}

// BankKeeper sends the refunds out of the module account.
type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// Migrate removes the order book queue: from consensus version 5 orders execute inside their
// transaction. Every message still queued is refunded exactly once instead of being executed:
// the owner gets the escrowed coins back and an OrderMessageRefundedEvent with reason "upgrade" is
// emitted. A queued cancel refunds nothing, its order stays on the book. Then every key under the
// queue prefixes (messages, counter, pending cancels) is deleted.
//
// store is the module's root store.
func Migrate(ctx sdk.Context, store storetypes.KVStore, k OrderKeeper, bank BankKeeper) error {
	qmKeys, err := refundQueueMessages(ctx, store, k, bank)
	if err != nil {
		return err
	}

	keys := qmKeys
	keys = append(keys, prefixKeys(store, []byte(QueueMessageCounterPrefix))...)
	keys = append(keys, prefixKeys(store, []byte(PendingCancelPrefix))...)
	// keys are deleted only after every iterator is closed
	for _, key := range keys {
		store.Delete(key)
	}

	return nil
}

// refundQueueMessages refunds every queued message in store order and returns their keys.
func refundQueueMessages(ctx sdk.Context, store storetypes.KVStore, k OrderKeeper, bank BankKeeper) ([][]byte, error) {
	iterator := storetypes.KVStorePrefixIterator(store, []byte(QueueMessagePrefix))
	defer iterator.Close()

	var keys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())

		qm, err := decodeQueueMessage(iterator.Value())
		if err != nil {
			return nil, fmt.Errorf("queue message %q: %w", iterator.Key(), err)
		}

		if qm.MessageType == messageTypeCancel {
			continue
		}

		err = refundQueueMessage(ctx, &qm, k, bank)
		if err != nil {
			return nil, fmt.Errorf("could not refund queue message %s: %w", qm.MessageId, err)
		}
	}

	return keys, nil
}

// refundQueueMessage returns the coins escrowed for the message to its owner. Fill messages were
// queued with the inverted order type already, so the stored order type is the escrowed side for
// every message type.
func refundQueueMessage(ctx sdk.Context, qm *queueMessage, k OrderKeeper, bank BankKeeper) error {
	amount, ok := math.NewIntFromString(qm.Amount)
	if !ok {
		return fmt.Errorf("could not convert amount %q", qm.Amount)
	}

	owner, err := sdk.AccAddressFromBech32(qm.Owner)
	if err != nil {
		return fmt.Errorf("invalid owner: %w", err)
	}

	market, found := k.GetMarketById(ctx, qm.MarketId)
	if !found {
		return fmt.Errorf("market %s not found", qm.MarketId)
	}

	orderCoins, err := k.GetOrderCoinsWithDust(ctx, types.OrderCoinsArguments{
		OrderType:    qm.OrderType,
		OrderPrice:   qm.Price,
		OrderAmount:  amount,
		Market:       &market,
		UserAddress:  qm.Owner,
		UserReceives: true,
	})
	if err != nil {
		return fmt.Errorf("could not get order coins: %w", err)
	}

	refund := ""
	if orderCoins.Coin.IsPositive() {
		err = bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, owner, sdk.NewCoins(orderCoins.Coin))
		if err != nil {
			return fmt.Errorf("could not send refund: %w", err)
		}
		refund = orderCoins.Coin.String()
	}
	k.StoreProcessedUserDust(ctx, orderCoins.UserDust, &orderCoins.Dust)

	return ctx.EventManager().EmitTypedEvent(&types.OrderMessageRefundedEvent{
		OrderId:   qm.MessageId,
		MarketId:  qm.MarketId,
		OrderType: qm.OrderType,
		Owner:     qm.Owner,
		Amount:    amount.String(),
		Refund:    refund,
		Reason:    types.RefundReasonUpgrade,
	})
}

func prefixKeys(store storetypes.KVStore, prefix []byte) [][]byte {
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var keys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
	}

	return keys
}
