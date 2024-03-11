package keeper

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	GetOrderCoins(orderType, orderPrice string, orderAmount sdk.Int, market *types.Market) (sdk.Coin, error)
}

type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

type ProcessingEngine struct {
	msgsToDelete []string

	k    ProcessingKeeper
	bank BankKeeper
}

func NewProcessingEngine(k ProcessingKeeper, bank BankKeeper) (*ProcessingEngine, error) {
	if k == nil || bank == nil {
		return nil, fmt.Errorf("invalid dependencies provided to ProcessingEngine")
	}

	return &ProcessingEngine{
		k:    k,
		bank: bank,
	}, nil
}

func (pe *ProcessingEngine) ProcessQueueMessages(ctx sdk.Context) {
	pe.k.IterateAllQueueMessages(ctx, pe.getMessageHandler())

	if len(pe.msgsToDelete) == 0 {
		ctx.Logger().Info("[ProcessQueueMessages] no queue message to process")
		return
	}

	ctx.Logger().Info(fmt.Sprintf("[ProcessQueueMessages] preparing to delete %d queue messages", len(pe.msgsToDelete)))
	for _, msgId := range pe.msgsToDelete {
		pe.k.RemoveQueueMessage(ctx, msgId)
	}

	pe.k.ResetQueueMessageCounter(ctx)
	ctx.Logger().Info("[ProcessQueueMessages] queue message counter reset")
}

func (pe *ProcessingEngine) getMessageHandler() func(ctx sdk.Context, message types.QueueMessage) {
	return func(ctx sdk.Context, message types.QueueMessage) {
		switch message.MessageType {
		case types.OrderTypeCancel:
			pe.cancelOrder(ctx, message)
		default:
			pe.addOrder(ctx, message)
		}

		pe.msgsToDelete = append(pe.msgsToDelete, message.MessageId)
	}
}

func (pe *ProcessingEngine) cancelOrder(ctx sdk.Context, message types.QueueMessage) {
	ctx.Logger().Info(fmt.Sprintf("[cancelOrder] cancelling order with id: %s", message.OrderId))
	order, found := pe.k.GetOrder(ctx, message.MarketId, message.OrderType, message.OrderId)
	if !found {
		return
	}
	pe.k.RemoveOrder(ctx, order)

	market, _ := pe.k.GetMarketById(ctx, order.MarketId)
	orderAmountInt, ok := sdk.NewIntFromString(order.Amount)
	if !ok {
		//should never happen
		ctx.Logger().Error("[addOrder] could not convert order amount")
		return
	}

	coin, err := pe.k.GetOrderCoins(order.OrderType, order.Price, orderAmountInt, &market)
	if err != nil {
		ctx.Logger().Error("[ProcessingEngine][cancelOrder] could not get order coins: %v", err)
		return
	}

	accAddr, err := sdk.AccAddressFromBech32(order.Owner)
	if err != nil {
		ctx.Logger().
			Error(fmt.Sprintf("[ProcessingEngine][cancelOrder] error on getting account address for order owner: %v", err))
		return
	}

	err = pe.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, accAddr, sdk.NewCoins(coin))
	if err != nil {
		ctx.Logger().
			Error(fmt.Sprintf("[ProcessingEngine][cancelOrder] error on sending funds to order owner: %v", err))
		return
	}

	pe.removeOrderFromAggregate(ctx, &order)
	ctx.Logger().Info(fmt.Sprintf("[cancelOrder] order %s cancelled", message.OrderId))

	pe.emitOrderCanceledEvent(ctx, &order)
}

func (pe *ProcessingEngine) addOrder(ctx sdk.Context, message types.QueueMessage) {
	ctx.Logger().Info(fmt.Sprintf("[addOrder] adding new order on market: %s", message.MarketId))
	zeroInt := sdk.ZeroInt()
	msgAmountInt, ok := sdk.NewIntFromString(message.Amount)
	if !ok {
		//should never happen
		ctx.Logger().Error("[addOrder] could not convert queue message amount")
		return
	}

	//find opposite orders if they exist
	agg, found := pe.k.GetAggregatedOrder(ctx, message.MarketId, types.TheOtherOrderType(message.OrderType), message.Price)
	//no pending orders to fill. save the order and finish
	if !found {
		order := pe.saveOrder(ctx, message, message.Amount)
		pe.addOrderToAggregate(ctx, order)

		ctx.Logger().Info(fmt.Sprintf("[addOrder] No orders to fill found. new order %s saved ", order.Id))
		return
	}

	aggAmountInt, ok := sdk.NewIntFromString(agg.Amount)
	if !ok {
		//should never happen
		ctx.Logger().Error("[addOrder] could not convert agg amount")
		return
	}

	oppositeOrderRefs := pe.k.GetPriceOrderByPrice(ctx, message.MarketId, types.TheOtherOrderType(message.OrderType), message.Price)
	ctx.Logger().Info(fmt.Sprintf("[addOrder] found orders to fill: %d ", len(oppositeOrderRefs)))
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
			//should never happen
			ctx.Logger().Error("[addOrder] could not convert order amount")
			continue
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

		err := pe.fundUsersAccounts(ctx, orderToFill, market, amountToExecute, msgOwnerAddr)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("[addOrder] %v", err))
			return
		}

		if orderAmountInt.GT(zeroInt) {
			pe.k.SaveOrder(ctx, orderToFill)
		} else {
			pe.k.RemoveOrder(ctx, orderToFill)
		}

		pe.addHistoryOrder(ctx, orderToFill, amountToExecute, message.Owner)
		pe.emitOrderExecutedEvent(ctx, &orderToFill, amountToExecute.String())
	}

	ctx.Logger().Info("[addOrder] finished filling orders.")
	if msgAmountInt.Equal(zeroInt) {
		ctx.Logger().Info(fmt.Sprintf("[addOrder] message with id %s was completely filled", message.MessageId))
		if aggAmountInt.GT(zeroInt) {
			pe.k.SetAggregatedOrder(ctx, agg)
			ctx.Logger().Info("[addOrder] aggregated order updated")
		} else {
			pe.k.RemoveAggregatedOrder(ctx, agg)
			ctx.Logger().Info("[addOrder] aggregated order removed")
		}
		return
	}
	//if this code is reached then all orders are filled, and we have a remaining amount in the message to deal with
	pe.k.RemoveAggregatedOrder(ctx, agg)

	//if min amount condition is met we can place an order with the remaining message amount
	if msgAmountInt.GTE(minAmount) {
		ctx.Logger().Info(fmt.Sprintf("[addOrder] message with id %s has a remaining amount", message.MessageId))
		order := pe.saveOrder(ctx, message, message.Amount)
		pe.addOrderToAggregate(ctx, order)

		ctx.Logger().Info(fmt.Sprintf("[addOrder] message with id %s has been placed as order %s", message.MessageId, order.Id))
		return
	}

	ctx.Logger().Info(fmt.Sprintf("[addOrder] message with id %s remaining amount is too low, returning dust.", message.MessageId))
	//we have a remaining amount smaller than min amount. We should send it back to the msg owner
	coinsForMsgOwner, err := pe.k.GetOrderCoins(message.OrderType, message.Price, msgAmountInt, &market)
	if err != nil {
		ctx.Logger().Error(fmt.Sprintf("error when creating funds to return to message owner: %v", err.Error()))
		return
	}

	err = pe.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, msgOwnerAddr, sdk.NewCoins(coinsForMsgOwner))
	if err != nil {
		ctx.Logger().Error(fmt.Sprintf("error when returning funds to message owner: %v", err.Error()))
		return
	}
}

func (pe *ProcessingEngine) fundUsersAccounts(ctx sdk.Context, order types.Order, market types.Market, amount sdk.Int, taker sdk.AccAddress) error {
	orderOwnerAddr, _ := sdk.AccAddressFromBech32(order.Owner)
	coinsForOrderOwner, err := pe.k.GetOrderCoins(types.TheOtherOrderType(order.OrderType), order.Price, amount, &market)
	if err != nil {
		return fmt.Errorf("error 1 when funding user accounts: %v", err)
	}

	coinsForMsgOwner, err := pe.k.GetOrderCoins(order.OrderType, order.Price, amount, &market)
	if err != nil {
		return fmt.Errorf("error 2 when funding user accounts: %v", err)
	}

	err = pe.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, orderOwnerAddr, sdk.NewCoins(coinsForOrderOwner))
	if err != nil {
		return fmt.Errorf("error 3 when funding user accounts: %v", err)
	}

	err = pe.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, taker, sdk.NewCoins(coinsForMsgOwner))
	if err != nil {
		return fmt.Errorf("error 4 when funding user accounts: %v", err)
	}

	return nil
}

func (pe *ProcessingEngine) addHistoryOrder(ctx sdk.Context, order types.Order, amount sdk.Int, taker string) {
	history := types.HistoryOrder{
		MarketId:   order.MarketId,
		OrderType:  order.OrderType,
		Amount:     amount.String(),
		Price:      order.Price,
		ExecutedAt: ctx.BlockTime().Unix(),
		Maker:      order.Owner,
		Taker:      taker,
	}

	pe.k.SetHistoryOrder(ctx, history, order.Id)
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
		ctx.Logger().Error("[addOrder] could not convert agg amount")
		return
	}

	orderAmountInt, ok := sdk.NewIntFromString(order.Amount)
	if !ok {
		//should never happen
		ctx.Logger().Error("[addOrder] could not convert order amount")
		return
	}

	aggAmountInt = aggAmountInt.Sub(orderAmountInt)

	if aggAmountInt.GT(sdk.ZeroInt()) {
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
		ctx.Logger().Error("[addOrder] could not convert agg amount")
		return
	}

	orderAmountInt, ok := sdk.NewIntFromString(order.Amount)
	if !ok {
		//should never happen
		ctx.Logger().Error("[addOrder] could not convert order amount")
		return
	}

	agg.Amount = aggAmountInt.Add(orderAmountInt).String()

	pe.k.SetAggregatedOrder(ctx, agg)
}

func (pe *ProcessingEngine) emitOrderExecutedEvent(ctx sdk.Context, order *types.Order, amount string) {
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
		ctx.Logger().Error(err.Error())
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
		ctx.Logger().Error(err.Error())
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
		ctx.Logger().Error(err.Error())
	}
}
