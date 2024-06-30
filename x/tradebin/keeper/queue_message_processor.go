package keeper

import (
	"fmt"
	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"
)

type ProcessingKeeper interface {
	//queue messages
	IterateAllQueueMessages(ctx sdk.Context, msgHandler func(ctx sdk.Context, message types.QueueMessage))
	RemoveQueueMessage(ctx sdk.Context, messageId string)
	ResetQueueMessageCounter(ctx sdk.Context)

	//orders
	GetPriceOrderByPrice(ctx sdk.Context, marketId, orderType, price string) (list []types.OrderReference)

	GetOrder(ctx sdk.Context, marketId, orderType, orderId string) (order types.Order, found bool)
	RemoveOrder(ctx sdk.Context, order types.Order)
	NewOrder(ctx sdk.Context, order types.Order) types.Order
	SaveOrder(ctx sdk.Context, order types.Order) types.Order

	GetAggregatedOrder(ctx sdk.Context, marketId, orderType, price string) (order types.AggregatedOrder, found bool)
	SetAggregatedOrder(ctx sdk.Context, order types.AggregatedOrder)
	RemoveAggregatedOrder(ctx sdk.Context, order types.AggregatedOrder)

	SetHistoryOrder(ctx sdk.Context, order types.HistoryOrder, index string)

	//market
	GetMarketById(ctx sdk.Context, marketId string) (val types.Market, found bool)

	//calculator
	GetOrderCoinsWithDust(ctx sdk.Context, orderCoinsArgs types.OrderCoinsArguments) (types.OrderCoins, error)
	StoreProcessedUserDust(ctx sdk.Context, userDust *types.UserDust, userDustDec *sdk.Dec)

	Logger(ctx sdk.Context) log.Logger

	GetOnOrderFillHooks() []types.OnMarketOrderFill
}

type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

type ProcessingEngine struct {
	msgsToDelete []string

	k    ProcessingKeeper
	bank BankKeeper

	logger log.Logger
}

func NewProcessingEngine(k ProcessingKeeper, bank BankKeeper, logger log.Logger) (*ProcessingEngine, error) {
	if k == nil || bank == nil || logger == nil {
		return nil, fmt.Errorf("invalid dependencies provided to ProcessingEngine")
	}

	return &ProcessingEngine{
		k:      k,
		bank:   bank,
		logger: logger,
	}, nil
}

func (pe *ProcessingEngine) ProcessQueueMessages(ctx sdk.Context) {
	logger := pe.logger.With(
		"method", "ProcessQueueMessages",
	)
	pe.k.IterateAllQueueMessages(ctx, pe.getMessageHandler())

	if len(pe.msgsToDelete) == 0 {
		logger.Info("no queue message to process")
		return
	}

	logger.Info("preparing to delete processed queue messages", "number_of_messages", len(pe.msgsToDelete))
	for _, msgId := range pe.msgsToDelete {
		pe.k.RemoveQueueMessage(ctx, msgId)
	}

	pe.k.ResetQueueMessageCounter(ctx)
	logger.Info("queue message counter reset")
}

func (pe *ProcessingEngine) getMessageHandler() func(ctx sdk.Context, message types.QueueMessage) {
	return func(ctx sdk.Context, message types.QueueMessage) {
		var wrappingFn func(ctx sdk.Context) error

		switch message.MessageType {
		case types.OrderTypeCancel:
			wrappingFn = func(ctx sdk.Context) error {
				return pe.cancelOrder(ctx, message)
			}
		default:
			wrappingFn = func(ctx sdk.Context) error {
				return pe.addOrder(ctx, message)
			}
		}

		err := bzeutils.ApplyFuncIfNoError(ctx, wrappingFn)
		if err != nil {
			//leave the message on queue until we discover what the issue was.
			pe.logger.Error("error on handling queue message", "message", message)
			return
		}

		pe.msgsToDelete = append(pe.msgsToDelete, message.MessageId)
	}
}

func (pe *ProcessingEngine) cancelOrder(ctx sdk.Context, message types.QueueMessage) error {
	logger := pe.logger.With(
		"message", message,
		"func", "cancelOrder",
	)
	logger.Info("cancelling order")

	order, found := pe.k.GetOrder(ctx, message.MarketId, message.OrderType, message.OrderId)
	if !found {
		logger.Error("could not find order")
		return nil
	}
	pe.k.RemoveOrder(ctx, order)

	market, _ := pe.k.GetMarketById(ctx, order.MarketId)
	orderAmountInt, ok := sdk.NewIntFromString(order.Amount)
	if !ok {

		return fmt.Errorf("could not convert order amount")
	}

	accAddr, err := sdk.AccAddressFromBech32(order.Owner)
	if err != nil {

		return fmt.Errorf("error on getting account address for order owner: %v", err)
	}

	coinReq := types.OrderCoinsArguments{
		OrderType:    order.OrderType,
		OrderPrice:   order.Price,
		OrderAmount:  orderAmountInt,
		Market:       &market,
		UserAddress:  order.Owner,
		UserReceives: true,
	}

	orderCoins, err := pe.k.GetOrderCoinsWithDust(ctx, coinReq)
	if err != nil {
		return fmt.Errorf("could not get order coins: %v", err)
	}

	if orderCoins.Coin.IsPositive() {
		err = pe.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, accAddr, sdk.NewCoins(orderCoins.Coin))
		if err != nil {

			return fmt.Errorf("error on sending funds to order owner: %v", err)
		}
	} else {
		logger.Info("will not send funds to order owner because the amount is too low but will save dust", "orderCoins", orderCoins)
	}

	pe.removeOrderFromAggregate(ctx, &order)
	pe.k.StoreProcessedUserDust(ctx, orderCoins.UserDust, &orderCoins.Dust)
	pe.emitOrderCanceledEvent(ctx, &order)
	logger.Info("order cancelled")

	return nil
}

func (pe *ProcessingEngine) addOrder(ctx sdk.Context, message types.QueueMessage) error {
	logger := pe.logger.With(
		"message", message,
		"func", "addOrder",
	)
	logger.Info("adding new order on market")

	zeroInt := sdk.ZeroInt()
	msgAmountInt, ok := sdk.NewIntFromString(message.Amount)
	if !ok {

		return fmt.Errorf("could not convert queue message amount")
	}

	//find opposite orders if they exist
	agg, found := pe.k.GetAggregatedOrder(ctx, message.MarketId, types.TheOtherOrderType(message.OrderType), message.Price)
	//no pending orders to fill. save the order and finish
	if !found {
		order := pe.saveOrder(ctx, message, message.Amount)
		pe.addOrderToAggregate(ctx, order)
		logger.Info("no orders to fill. saving as new order", "order", *order)

		return nil
	}

	aggAmountInt, ok := sdk.NewIntFromString(agg.Amount)
	if !ok {

		return fmt.Errorf("could not convert agg amount")
	}

	oppositeOrderRefs := pe.k.GetPriceOrderByPrice(ctx, message.MarketId, types.TheOtherOrderType(message.OrderType), message.Price)
	logger.Info("found orders to fill", "number_of_orders", len(oppositeOrderRefs))
	//should always exist
	market, _ := pe.k.GetMarketById(ctx, message.MarketId)
	minAmount := CalculateMinAmount(message.Price)
	msgOwnerAddr, _ := sdk.AccAddressFromBech32(message.Owner)
	for _, orderRef := range oppositeOrderRefs {
		//stop when all message amount was spent
		if !msgAmountInt.IsPositive() {
			break
		}

		orderToFill, _ := pe.k.GetOrder(ctx, orderRef.MarketId, orderRef.OrderType, orderRef.Id)
		orderAmountInt, ok := sdk.NewIntFromString(orderToFill.Amount)
		if !ok {

			return fmt.Errorf("could not convert order to fill amount")
		}

		//find how much to send to the found order's owner
		amountToExecute := pe.getExecutedAmount(msgAmountInt, orderAmountInt, minAmount)
		if amountToExecute.IsZero() {
			break
		}

		msgAmountInt = msgAmountInt.Sub(amountToExecute)
		message.Amount = msgAmountInt.String()

		orderAmountInt = orderAmountInt.Sub(amountToExecute)
		orderToFill.Amount = orderAmountInt.String()

		aggAmountInt = aggAmountInt.Sub(amountToExecute)
		agg.Amount = aggAmountInt.String()

		err := pe.fundUsersAccounts(ctx, &orderToFill, &market, amountToExecute, msgOwnerAddr)
		if err != nil {

			return err
		}

		if orderAmountInt.GT(zeroInt) {
			pe.k.SaveOrder(ctx, orderToFill)
		} else {
			pe.k.RemoveOrder(ctx, orderToFill)
		}

		pe.addHistoryOrder(ctx, &orderToFill, amountToExecute, &message)
		pe.emitOrderExecutedEvent(ctx, &orderToFill, amountToExecute.String(), message.Owner)
	}

	if aggAmountInt.GT(zeroInt) {
		pe.k.SetAggregatedOrder(ctx, agg)
		logger.Info("aggregated order updated")
	} else {
		pe.k.RemoveAggregatedOrder(ctx, agg)
		logger.Info("aggregated order removed")
	}

	if msgAmountInt.Equal(zeroInt) {
		logger.Info("message was completely filled")
		return nil
	}

	//if min amount condition is met and all orders were filled we can proceed to place the order
	if msgAmountInt.GTE(minAmount) && aggAmountInt.IsZero() {
		logger.Info("message with has a remaining amount")
		order := pe.saveOrder(ctx, message, message.Amount)
		pe.addOrderToAggregate(ctx, order)

		logger.Info("message has been saved as order", "order", *order)
		return nil
	}

	logger.Info("message remaining amount is too low, returning dust")
	//we have a remaining amount smaller than min amount. We should send it back to the msg owner
	coinReq := types.OrderCoinsArguments{
		OrderType:    message.OrderType,
		OrderPrice:   message.Price,
		OrderAmount:  msgAmountInt,
		Market:       &market,
		UserAddress:  message.Owner,
		UserReceives: true,
	}

	orderCoins, err := pe.k.GetOrderCoinsWithDust(ctx, coinReq)
	if err != nil {
		return fmt.Errorf("could not get order coins: %v", err)
	}

	if orderCoins.Coin.IsPositive() {
		err = pe.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, msgOwnerAddr, sdk.NewCoins(orderCoins.Coin))
		if err != nil {

			return fmt.Errorf("error when returning funds to message owner: %v", err)
		}
	} else {
		logger.Info("will not send dust to order message owner because the amount is 0 (zero) but will save dust", "orderCoins", orderCoins)
	}
	pe.k.StoreProcessedUserDust(ctx, orderCoins.UserDust, &orderCoins.Dust)

	return nil
}

func (pe *ProcessingEngine) fundUsersAccounts(ctx sdk.Context, order *types.Order, market *types.Market, amount sdk.Int, taker sdk.AccAddress) error {
	orderOwnerAddr, _ := sdk.AccAddressFromBech32(order.Owner)
	orderOwnerCoinsReq := types.OrderCoinsArguments{
		OrderType:    types.TheOtherOrderType(order.OrderType),
		OrderPrice:   order.Price,
		OrderAmount:  amount,
		Market:       market,
		UserAddress:  order.Owner,
		UserReceives: true,
	}

	coinsForOrderOwner, err := pe.k.GetOrderCoinsWithDust(ctx, orderOwnerCoinsReq)
	if err != nil {
		return fmt.Errorf("could not get order coins: %v", err)
	}

	msgOwnerCoinsReq := types.OrderCoinsArguments{
		OrderType:    order.OrderType,
		OrderPrice:   order.Price,
		OrderAmount:  amount,
		Market:       market,
		UserAddress:  taker.String(),
		UserReceives: true,
	}

	coinsForMsgOwner, err := pe.k.GetOrderCoinsWithDust(ctx, msgOwnerCoinsReq)
	if err != nil {
		return fmt.Errorf("could not get order coins: %v", err)
	}

	if coinsForOrderOwner.Coin.IsPositive() {
		err = pe.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, orderOwnerAddr, sdk.NewCoins(coinsForOrderOwner.Coin))
		if err != nil {
			return fmt.Errorf("error 3 when funding user accounts: %v", err)
		}
	} else {
		ctx.Logger().Info("will not send dust to order order owner because the amount is 0 (zero) but will save dust", "coinsForOrderOwner", coinsForOrderOwner)
	}
	pe.k.StoreProcessedUserDust(ctx, coinsForOrderOwner.UserDust, &coinsForOrderOwner.Dust)

	if coinsForMsgOwner.Coin.IsPositive() {
		err = pe.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, taker, sdk.NewCoins(coinsForMsgOwner.Coin))
		if err != nil {
			return fmt.Errorf("error 4 when funding user accounts: %v", err)
		}
	} else {
		ctx.Logger().Info("will not send dust to message order owner because the amount is 0 (zero) but will save dust", "coinsForMsgOwner", coinsForMsgOwner)
	}

	pe.k.StoreProcessedUserDust(ctx, coinsForMsgOwner.UserDust, &coinsForMsgOwner.Dust)

	pe.logger.Debug("funded users accounts", "amount", amount.String(), "order_id", order.Id)
	return nil
}

func (pe *ProcessingEngine) addHistoryOrder(ctx sdk.Context, order *types.Order, amount sdk.Int, message *types.QueueMessage) {
	history := types.HistoryOrder{
		MarketId:   order.MarketId,
		OrderType:  order.OrderType,
		Amount:     amount.String(),
		Price:      order.Price,
		ExecutedAt: ctx.BlockTime().Unix(),
		Maker:      order.Owner,
		Taker:      message.Owner,
	}

	pe.k.SetHistoryOrder(ctx, history, fmt.Sprintf("%s%s", order.Id, message.MessageId[len(message.MessageId)-5:]))
}

func (pe *ProcessingEngine) getExecutedAmount(messageAmount, orderAmount, minAmount sdk.Int) sdk.Int {
	//if the amount of the message is too low nothing to execute
	if messageAmount.LT(minAmount) {
		return sdk.ZeroInt()
	}

	//if the entire amount of the order is filled return it as the executed amount
	if messageAmount.GTE(orderAmount) {
		return orderAmount
	}

	orderRemaining := orderAmount.Sub(messageAmount)
	//if orderRemaining >= minAmount {
	if orderRemaining.GTE(minAmount) {
		return messageAmount
	}

	//order amount remains too low. keep min amount for it
	executedAmount := orderAmount.Sub(minAmount)
	if executedAmount.GTE(minAmount) {
		return executedAmount
	}

	return sdk.ZeroInt()
}

func (pe *ProcessingEngine) saveOrder(ctx sdk.Context, message types.QueueMessage, amount string) *types.Order {
	order := types.Order{
		MarketId:  message.MarketId,
		OrderType: message.OrderType,
		Amount:    amount,
		Price:     message.Price,
		Owner:     message.Owner,
	}

	order = pe.k.NewOrder(ctx, order)

	pe.emitOrderSavedEvent(ctx, &order)

	return &order
}

func (pe *ProcessingEngine) removeOrderFromAggregate(ctx sdk.Context, order *types.Order) {
	agg, found := pe.k.GetAggregatedOrder(ctx, order.MarketId, order.OrderType, order.Price)
	if !found {
		return
	}

	aggAmountInt, ok := sdk.NewIntFromString(agg.Amount)
	if !ok {
		//should never happen
		pe.logger.Error("could not convert agg amount", "method", "removeOrderFromAggregate")
		return
	}

	orderAmountInt, ok := sdk.NewIntFromString(order.Amount)
	if !ok {
		//should never happen
		pe.logger.Error("could not convert order amount", "method", "removeOrderFromAggregate")
		return
	}

	aggAmountInt = aggAmountInt.Sub(orderAmountInt)

	if aggAmountInt.GT(sdk.ZeroInt()) {
		agg.Amount = aggAmountInt.String()
		pe.k.SetAggregatedOrder(ctx, agg)
	} else {
		pe.k.RemoveAggregatedOrder(ctx, agg)
	}
}

func (pe *ProcessingEngine) addOrderToAggregate(ctx sdk.Context, order *types.Order) {
	agg, found := pe.k.GetAggregatedOrder(ctx, order.MarketId, order.OrderType, order.Price)
	if !found {
		agg = types.AggregatedOrder{
			MarketId:  order.MarketId,
			OrderType: order.OrderType,
			Amount:    "0", // on purpose. it's added below
			Price:     order.Price,
		}
	}

	aggAmountInt, ok := sdk.NewIntFromString(agg.Amount)
	if !ok {
		//should never happen
		pe.logger.Error("could not convert agg amount", "method", "addOrderToAggregate")
		return
	}

	orderAmountInt, ok := sdk.NewIntFromString(order.Amount)
	if !ok {
		//should never happen
		pe.logger.Error("could not convert order amount", "method", "addOrderToAggregate")
		return
	}

	agg.Amount = aggAmountInt.Add(orderAmountInt).String()

	pe.k.SetAggregatedOrder(ctx, agg)
}

func (pe *ProcessingEngine) emitOrderExecutedEvent(ctx sdk.Context, order *types.Order, amount, userAddress string) {
	err := ctx.EventManager().EmitTypedEvent(
		&types.OrderExecutedEvent{
			Id:        order.Id,
			MarketId:  order.MarketId,
			OrderType: order.OrderType,
			Amount:    amount,
			Price:     order.Price,
		},
	)

	if err != nil {
		pe.logger.Error(err.Error())
	}

	//call hooks for the filled order
	for _, h := range pe.k.GetOnOrderFillHooks() {
		wrappedFn := func(ctx sdk.Context) error {
			h(ctx, order.MarketId, amount, userAddress)

			return nil
		}

		err = bzeutils.ApplyFuncIfNoError(ctx, wrappedFn)
		if err != nil {
			pe.k.Logger(ctx).Error(err.Error())
		}
	}
}

func (pe *ProcessingEngine) emitOrderCanceledEvent(ctx sdk.Context, order *types.Order) {
	err := ctx.EventManager().EmitTypedEvent(
		&types.OrderCanceledEvent{
			Id:        order.Id,
			MarketId:  order.MarketId,
			OrderType: order.OrderType,
			Amount:    order.Amount,
			Price:     order.Price,
		},
	)

	if err != nil {
		pe.logger.Error(err.Error())
	}
}

func (pe *ProcessingEngine) emitOrderSavedEvent(ctx sdk.Context, order *types.Order) {
	err := ctx.EventManager().EmitTypedEvent(
		&types.OrderSavedEvent{
			Id:        order.Id,
			MarketId:  order.MarketId,
			OrderType: order.OrderType,
			Amount:    order.Amount,
			Price:     order.Price,
		},
	)

	if err != nil {
		pe.logger.Error(err.Error())
	}
}
