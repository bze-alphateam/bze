package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/log"
)

// ProcessingKeeper is the state the order book engine reads and writes.
type ProcessingKeeper interface {
	//orders
	IteratePriceOrders(ctx sdk.Context, marketId, orderType, price string, fn func(ref types.OrderReference) (stop bool))

	GetOrder(ctx sdk.Context, marketId, orderType, orderId string) (order types.Order, found bool)
	RemoveOrder(ctx sdk.Context, order types.Order)
	AssignOrderId(ctx sdk.Context, order types.Order) types.Order
	SaveOrder(ctx sdk.Context, order types.Order) types.Order

	GetAggregatedOrder(ctx sdk.Context, marketId, orderType, price string) (order types.AggregatedOrder, found bool)
	SetAggregatedOrder(ctx sdk.Context, order types.AggregatedOrder)
	RemoveAggregatedOrder(ctx sdk.Context, order types.AggregatedOrder)

	SetHistoryOrder(ctx sdk.Context, order types.HistoryOrder, index string)

	//market
	GetMarketById(ctx sdk.Context, marketId string) (val types.Market, found bool)

	//calculator
	GetOrderCoinsWithDust(ctx sdk.Context, orderCoinsArgs types.OrderCoinsArguments) (types.OrderCoins, error)
	StoreProcessedUserDust(ctx sdk.Context, userDust *types.UserDust, userDustDec *math.LegacyDec)

	Logger() log.Logger

	GetOnOrderFillHooks() []types.OnMarketOrderFill
}

type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// ProcessingEngine is the order book matching engine. The message servers call its verbs inside the
// transaction, so all the work it does is metered by the transaction's gas meter and an error reverts
// the whole transaction.
type ProcessingEngine struct {
	k    ProcessingKeeper
	bank BankKeeper

	// maxFills is the maximum number of resting orders one order examines (filled or skipped)
	maxFills uint64

	logger log.Logger
}

func NewProcessingEngine(k ProcessingKeeper, bank BankKeeper, maxFills uint64, logger log.Logger) (*ProcessingEngine, error) {
	if k == nil || bank == nil || logger == nil {
		return nil, fmt.Errorf("invalid dependencies provided to ProcessingEngine")
	}

	if maxFills < 1 {
		return nil, fmt.Errorf("max fills per order must be at least 1")
	}

	return &ProcessingEngine{
		k:        k,
		bank:     bank,
		maxFills: maxFills,
		logger:   logger,
	}, nil
}

// PlaceOrder executes a limit order: it is matched against the opposite side at its price and the
// remainder rests on the book as a new maker order when the opposite side at that price was emptied
// and the remainder is at least the minimum amount, otherwise the remainder is refunded.
// At most maxFills resting orders are examined (filled or skipped); when that stops the matching the
// remainder is refunded.
// The order coins must already be escrowed in the module account. The order gets its id here,
// before matching.
func (pe *ProcessingEngine) PlaceOrder(ctx sdk.Context, order types.Order) error {
	order = pe.k.AssignOrderId(ctx, order)

	return pe.addOrder(ctx, order)
}

// FillOrder executes an immediate-or-cancel order: it is matched against the opposite side at its
// price and whatever is not executed is refunded. It never rests on the book.
// At most maxFills resting orders are examined (filled or skipped).
// The order coins must already be escrowed in the module account. The order gets its id here,
// before matching.
func (pe *ProcessingEngine) FillOrder(ctx sdk.Context, order types.Order) error {
	order = pe.k.AssignOrderId(ctx, order)

	return pe.fillOrder(ctx, order)
}

// CancelOrder removes a resting order from the book and refunds its remaining amount to its owner.
// The caller loads the order and checks its ownership.
func (pe *ProcessingEngine) CancelOrder(ctx sdk.Context, order types.Order) error {
	return pe.cancelOrder(ctx, order)
}

func (pe *ProcessingEngine) cancelOrder(ctx sdk.Context, order types.Order) error {
	logger := pe.logger.With(
		"order", order,
		"func", "cancelOrder",
	)
	logger.Info("cancelling order")

	pe.k.RemoveOrder(ctx, order)

	market, _ := pe.k.GetMarketById(ctx, order.MarketId)
	orderAmountInt, ok := math.NewIntFromString(order.Amount)
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

	err = pe.removeOrderFromAggregate(ctx, &order)
	if err != nil {
		return err
	}
	pe.k.StoreProcessedUserDust(ctx, orderCoins.UserDust, &orderCoins.Dust)
	pe.emitOrderCanceledEvent(ctx, &order, order.Id)
	logger.Info("order cancelled")

	return nil
}

func (pe *ProcessingEngine) addOrder(ctx sdk.Context, message types.Order) error {
	logger := pe.logger.With(
		"order", message,
		"func", "addOrder",
	)
	logger.Info("adding new order on market")

	zeroInt := math.ZeroInt()
	msgAmountInt, ok := math.NewIntFromString(message.Amount)
	if !ok {

		return fmt.Errorf("could not convert order amount")
	}

	//find opposite orders if they exist
	agg, found := pe.k.GetAggregatedOrder(ctx, message.MarketId, types.TheOtherOrderType(message.OrderType), message.Price)
	//no pending orders to fill. save the order and finish
	if !found {
		order := pe.saveOrder(ctx, message, message.Amount)
		logger.Info("no orders to fill. saving as new order", "order", *order)

		return pe.addOrderToAggregate(ctx, order)
	}

	//should always exist
	market, _ := pe.k.GetMarketById(ctx, message.MarketId)
	minAmount, _ := CalculateMinAmount(message.Price)
	msgOwnerAddr, _ := sdk.AccAddressFromBech32(message.Owner)

	remaining, maxFillsReached, err := pe.fillAggregatedOrder(ctx, &agg, &message, msgAmountInt, &market)
	if err != nil {
		return fmt.Errorf("could not fill aggregated order: %v", err)
	}

	if remaining.Equal(zeroInt) {
		logger.Info("message was completely filled")
		return nil
	}

	// orders are still resting at this price: the remainder can not rest on the other side
	if maxFillsReached {
		logger.Info("max fills reached, returning the remainder")

		return pe.refundMessageFunds(ctx, &message, *remaining, &market, msgOwnerAddr, types.RefundReasonMaxFillsReached)
	}

	aggAmountInt, ok := math.NewIntFromString(agg.Amount)
	if !ok {
		return fmt.Errorf("could not convert agg amount")
	}

	//if min amount condition is met and all orders were filled we can proceed to place the order
	if remaining.GTE(minAmount) && aggAmountInt.IsZero() {
		order := pe.saveOrder(ctx, message, message.Amount)

		logger.Info("order remainder has been saved on the book", "order", *order)
		return pe.addOrderToAggregate(ctx, order)
	}

	logger.Info("message remaining amount is too low, returning dust")

	return pe.refundMessageFunds(ctx, &message, *remaining, &market, msgOwnerAddr, types.RefundReasonBelowMinAmount)
}

func (pe *ProcessingEngine) fundUsersAccounts(ctx sdk.Context, order *types.Order, market *types.Market, amount math.Int, taker sdk.AccAddress) error {
	logger := pe.logger.With(
		"func", "fundUsersAccounts",
		"order", order,
	)
	orderOwnerAddr, _ := sdk.AccAddressFromBech32(order.Owner)
	orderOwnerCoinsReq := types.OrderCoinsArguments{
		OrderType:    types.TheOtherOrderType(order.OrderType),
		OrderPrice:   order.Price,
		OrderAmount:  amount,
		Market:       market,
		UserAddress:  order.Owner,
		UserReceives: true,
	}
	logger.Debug("created order owner coins request", "orderOwnerCoinsReq", orderOwnerCoinsReq)

	coinsForOrderOwner, err := pe.k.GetOrderCoinsWithDust(ctx, orderOwnerCoinsReq)
	logger.Debug("created coins for order owner", "coinsForOrderOwner", coinsForOrderOwner)
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
	logger.Debug("created msg owner coins request", "msgOwnerCoinsReq", msgOwnerCoinsReq)

	coinsForMsgOwner, err := pe.k.GetOrderCoinsWithDust(ctx, msgOwnerCoinsReq)
	logger.Debug("created coins for msg owner", "coinsForMsgOwner", coinsForMsgOwner)
	if err != nil {
		return fmt.Errorf("could not get order coins: %v", err)
	}

	if coinsForOrderOwner.Coin.IsPositive() {
		err = pe.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, orderOwnerAddr, sdk.NewCoins(coinsForOrderOwner.Coin))
		if err != nil {
			return fmt.Errorf("error 3 when funding user accounts: %v", err)
		}
		logger.Debug("funded order owner", "coinsForOrderOwner", coinsForOrderOwner)
	} else {
		logger.Info("will not send dust to order order owner because the amount is 0 (zero) but will save dust", "coinsForOrderOwner", coinsForOrderOwner)
	}
	pe.k.StoreProcessedUserDust(ctx, coinsForOrderOwner.UserDust, &coinsForOrderOwner.Dust)

	if coinsForMsgOwner.Coin.IsPositive() {
		err = pe.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, taker, sdk.NewCoins(coinsForMsgOwner.Coin))
		if err != nil {
			return fmt.Errorf("error 4 when funding user accounts: %v", err)
		}
		logger.Debug("funded msg owner", "coinsForMsgOwner", coinsForMsgOwner)
	} else {
		logger.Info("will not send dust to message order owner because the amount is 0 (zero) but will save dust", "coinsForMsgOwner", coinsForMsgOwner)
	}

	pe.k.StoreProcessedUserDust(ctx, coinsForMsgOwner.UserDust, &coinsForMsgOwner.Dust)

	pe.logger.Debug("funded users accounts", "amount", amount.String(), "order_id", order.Id)
	return nil
}

func (pe *ProcessingEngine) addHistoryOrder(ctx sdk.Context, order *types.Order, amount math.Int, message *types.Order) {
	history := types.HistoryOrder{
		MarketId:   order.MarketId,
		OrderType:  order.OrderType,
		Amount:     amount.String(),
		Price:      order.Price,
		ExecutedAt: ctx.BlockTime().Unix(),
		Maker:      order.Owner,
		Taker:      message.Owner,
	}

	// Index format: {takerOrderId}{makerOrderId}
	index := fmt.Sprintf("%s%s", message.Id, order.Id)
	pe.k.SetHistoryOrder(ctx, history, index)
}

func (pe *ProcessingEngine) getExecutedAmount(messageAmount, orderAmount, minAmount math.Int) math.Int {
	//if the amount of the message is too low nothing to execute
	if messageAmount.LT(minAmount) {
		return math.ZeroInt()
	}

	//if the entire amount of the order is filled return it as the executed amount
	if messageAmount.GTE(orderAmount) {
		return orderAmount
	}

	orderRemaining := orderAmount.Sub(messageAmount)
	//if orderRemaining >= minAmount {
	// check if the order amount resulted after match with message amount is still greater than min amount for future
	// trades.
	if orderRemaining.GTE(minAmount) {
		return messageAmount
	}

	//order amount remains too low if we fill the entire message amount. keep min the minimum amount required for this
	// order to be filled later.
	executedAmount := orderAmount.Sub(minAmount)
	if executedAmount.GTE(minAmount) {
		return executedAmount
	}

	return math.ZeroInt()
}

func (pe *ProcessingEngine) saveOrder(ctx sdk.Context, message types.Order, amount string) *types.Order {
	order := message
	order.Amount = amount

	order = pe.k.SaveOrder(ctx, order)

	pe.emitOrderSavedEvent(ctx, &order, message.Id)

	return &order
}

func (pe *ProcessingEngine) removeOrderFromAggregate(ctx sdk.Context, order *types.Order) error {
	agg, found := pe.k.GetAggregatedOrder(ctx, order.MarketId, order.OrderType, order.Price)
	if !found {
		return nil
	}

	aggAmountInt, ok := math.NewIntFromString(agg.Amount)
	if !ok {
		//should never happen
		return fmt.Errorf("could not convert agg amount")
	}

	orderAmountInt, ok := math.NewIntFromString(order.Amount)
	if !ok {
		//should never happen
		return fmt.Errorf("could not convert order amount")
	}

	aggAmountInt = aggAmountInt.Sub(orderAmountInt)

	if aggAmountInt.GT(math.ZeroInt()) {
		agg.Amount = aggAmountInt.String()
		pe.k.SetAggregatedOrder(ctx, agg)
	} else {
		pe.k.RemoveAggregatedOrder(ctx, agg)
	}

	return nil
}

func (pe *ProcessingEngine) addOrderToAggregate(ctx sdk.Context, order *types.Order) error {
	agg, found := pe.k.GetAggregatedOrder(ctx, order.MarketId, order.OrderType, order.Price)
	if !found {
		agg = types.AggregatedOrder{
			MarketId:  order.MarketId,
			OrderType: order.OrderType,
			Amount:    "0", // on purpose. it's added below
			Price:     order.Price,
		}
	}

	aggAmountInt, ok := math.NewIntFromString(agg.Amount)
	if !ok {
		//should never happen
		return fmt.Errorf("could not convert agg amount")
	}

	orderAmountInt, ok := math.NewIntFromString(order.Amount)
	if !ok {
		//should never happen
		return fmt.Errorf("could not convert order amount")
	}

	agg.Amount = aggAmountInt.Add(orderAmountInt).String()

	pe.k.SetAggregatedOrder(ctx, agg)

	return nil
}

func (pe *ProcessingEngine) emitOrderExecutedEvent(ctx sdk.Context, order *types.Order, amount, userAddress, takerOrderId string) {
	err := ctx.EventManager().EmitTypedEvent(
		&types.OrderExecutedEvent{
			Id:        order.Id,
			MarketId:  order.MarketId,
			OrderType: order.OrderType,
			Amount:    amount,
			Price:     order.Price,
			Taker:     userAddress,
			Maker:     order.Owner,
			OrderId:   takerOrderId,
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
			pe.k.Logger().Error(err.Error())
		}
	}
}

func (pe *ProcessingEngine) emitOrderCanceledEvent(ctx sdk.Context, order *types.Order, orderId string) {
	err := ctx.EventManager().EmitTypedEvent(
		&types.OrderCanceledEvent{
			Id:        order.Id,
			MarketId:  order.MarketId,
			OrderType: order.OrderType,
			Amount:    order.Amount,
			Price:     order.Price,
			Owner:     order.Owner,
			OrderId:   orderId,
		},
	)

	if err != nil {
		pe.logger.Error(err.Error())
	}
}

func (pe *ProcessingEngine) emitOrderSavedEvent(ctx sdk.Context, order *types.Order, orderId string) {
	err := ctx.EventManager().EmitTypedEvent(
		&types.OrderSavedEvent{
			Id:        order.Id,
			MarketId:  order.MarketId,
			OrderType: order.OrderType,
			Amount:    order.Amount,
			Price:     order.Price,
			Owner:     order.Owner,
			OrderId:   orderId,
		},
	)

	if err != nil {
		pe.logger.Error(err.Error())
	}
}

func (pe *ProcessingEngine) emitOrderMessageRefundedEvent(ctx sdk.Context, message *types.Order, amount math.Int, refund sdk.Coin, reason string) {
	refundStr := ""
	if refund.IsPositive() {
		refundStr = refund.String()
	}

	err := ctx.EventManager().EmitTypedEvent(
		&types.OrderMessageRefundedEvent{
			OrderId:   message.Id,
			MarketId:  message.MarketId,
			OrderType: message.OrderType,
			Owner:     message.Owner,
			Amount:    amount.String(),
			Refund:    refundStr,
			Reason:    reason,
		},
	)

	if err != nil {
		pe.logger.Error(err.Error())
	}
}

func (pe *ProcessingEngine) fillOrder(ctx sdk.Context, message types.Order) error {
	logger := pe.logger.With(
		"order", message,
		"func", "fillOrder",
	)
	logger.Info("filling order on market")

	msgAmountInt, ok := math.NewIntFromString(message.Amount)
	if !ok {

		return fmt.Errorf("could not convert order amount")
	}

	//should always exist
	market, _ := pe.k.GetMarketById(ctx, message.MarketId)
	msgOwnerAddr, _ := sdk.AccAddressFromBech32(message.Owner)

	//find opposite orders if they exist
	agg, found := pe.k.GetAggregatedOrder(ctx, message.MarketId, types.TheOtherOrderType(message.OrderType), message.Price)
	if !found {
		//no pending orders to fill. refund the amount for this order
		return pe.refundMessageFunds(ctx, &message, msgAmountInt, &market, msgOwnerAddr, types.RefundReasonNoMatch)
	}

	remainingAmount, maxFillsReached, err := pe.fillAggregatedOrder(ctx, &agg, &message, msgAmountInt, &market)
	if err != nil {
		return err
	}

	if !remainingAmount.IsPositive() {
		logger.Info("message was completely filled")
		return nil
	}

	//we haven't filled the entire message amount so we have to refund the msg owner's coins
	reason := types.RefundReasonUnfilledRemainder
	if maxFillsReached {
		reason = types.RefundReasonMaxFillsReached
	}

	return pe.refundMessageFunds(ctx, &message, *remainingAmount, &market, msgOwnerAddr, reason)
}

// refundMessageFunds returns refundAmount (base denom) of the taker order to its owner and emits an
// OrderMessageRefundedEvent with the given reason in the same context as the bank send.
func (pe *ProcessingEngine) refundMessageFunds(ctx sdk.Context, message *types.Order, refundAmount math.Int, market *types.Market, msgOwnerAddr sdk.AccAddress, reason string) error {
	coinReq := types.OrderCoinsArguments{
		OrderType:    message.OrderType,
		OrderPrice:   message.Price,
		OrderAmount:  refundAmount,
		Market:       market,
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
		pe.logger.Info("will not send dust to order message owner because the amount is 0 (zero) but will save dust", "orderCoins", orderCoins)
	}
	pe.k.StoreProcessedUserDust(ctx, orderCoins.UserDust, &orderCoins.Dust)
	pe.emitOrderMessageRefundedEvent(ctx, message, refundAmount, orderCoins.Coin, reason)

	return nil
}

// fillAggregatedOrder fills the resting orders of agg's price level in FIFO order with the message amount
// and returns the amount left. It stops when the amount is spent, when it drops below the minimum
// amount or when maxFills resting orders were examined (filled or skipped); maxFillsReached is true
// only in the last case, when resting orders are left at that price.
//
// The level is read first and the fills are settled after the iterator is closed: nothing is written
// to the store while it is open.
func (pe *ProcessingEngine) fillAggregatedOrder(ctx sdk.Context, agg *types.AggregatedOrder, message *types.Order, msgAmountInt math.Int, market *types.Market) (_ *math.Int, maxFillsReached bool, _ error) {
	msgOwnerAddr, _ := sdk.AccAddressFromBech32(message.Owner)
	zeroInt := math.ZeroInt()
	aggAmountInt, ok := math.NewIntFromString(agg.Amount)
	if !ok {
		return nil, false, fmt.Errorf("could not convert agg amount")
	}

	minAmount, err := CalculateMinAmount(message.Price)
	if err != nil {
		return nil, false, fmt.Errorf("could not calculate min amount: %v", err)
	}

	fills, err := pe.matchPriceLevel(ctx, message, &msgAmountInt, &aggAmountInt, minAmount)
	if err != nil {
		return nil, false, err
	}
	maxFillsReached = fills.maxFillsReached
	pe.logger.Info("found orders to fill", "number_of_orders", len(fills.fills))

	for _, fill := range fills.fills {
		orderToFill := fill.order
		amountToExecute := fill.amount

		err := pe.fundUsersAccounts(ctx, &orderToFill, market, amountToExecute, msgOwnerAddr)
		if err != nil {

			return nil, false, err
		}

		if fill.remaining.GT(zeroInt) {
			pe.logger.Debug("order partially filled")
			pe.k.SaveOrder(ctx, orderToFill)
		} else {
			pe.logger.Debug("order fully filled")
			pe.k.RemoveOrder(ctx, orderToFill)
		}

		pe.addHistoryOrder(ctx, &orderToFill, amountToExecute, message)
		pe.emitOrderExecutedEvent(ctx, &orderToFill, amountToExecute.String(), message.Owner, message.Id)
	}

	if len(fills.fills) > 0 {
		message.Amount = msgAmountInt.String()
		agg.Amount = aggAmountInt.String()
	}

	if aggAmountInt.GT(zeroInt) {
		pe.k.SetAggregatedOrder(ctx, *agg)
		pe.logger.Info("aggregated order updated")
	} else {
		pe.k.RemoveAggregatedOrder(ctx, *agg)
		pe.logger.Info("aggregated order removed")
	}

	return &msgAmountInt, maxFillsReached, nil
}
