package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	txfeecollectormoduletypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateMarket(goCtx context.Context, msg *types.MsgCreateMarket) (*types.MsgCreateMarketResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	_, found := k.GetMarket(ctx, msg.Base, msg.Quote)
	if found {
		return nil, types.ErrMarketAlreadyExists
	}

	creatorAcc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	//check aliases too: user can try to create a market that exists
	_, found = k.GetMarketAlias(ctx, msg.Base, msg.Quote)
	if found {
		return nil, types.ErrMarketAlreadyExists
	}

	err = k.validateMarketAssets(ctx, msg.Base, msg.Quote)
	if err != nil {
		return nil, err
	}

	err = k.payMarketCreateFee(ctx, creatorAcc)
	if err != nil {
		return nil, err
	}

	market := types.Market{
		Base:    msg.Base,
		Quote:   msg.Quote,
		Creator: msg.Creator,
	}
	k.SetMarket(ctx, market)

	err = k.emitMarketCreatedEvent(ctx, &market)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgCreateMarketResponse{}, nil
}

func (k msgServer) CreateOrder(goCtx context.Context, msg *types.MsgCreateOrder) (*types.MsgCreateOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.checkPrice(ctx, msg)
	if err != nil {
		return nil, types.ErrInvalidOrderPrice.Wrapf("check price failed: %v", err)
	}

	minAmt, err := CalculateMinAmountFromDecPrice(msg.Price)
	if err != nil {
		return nil, types.ErrInvalidOrderPrice.Wrapf("could not calculate minimum amount: %v", err)
	}

	if minAmt.GT(msg.Amount) {
		return nil, types.ErrInvalidOrderAmount.Wrapf("amount should be bigger than: %s", minAmt.String())
	}

	market, found := k.GetMarketById(ctx, msg.MarketId)
	if !found {
		return nil, types.ErrMarketNotFound.Wrapf("market id: %s", msg.MarketId)
	}

	// Apply dynamic gas cost based on queue size to prevent spam attacks
	// Formula: max(queue_size - window, 0) * queueExtraGas
	// This makes it progressively more expensive to submit orders when the queue is full
	params := k.GetParams(ctx)
	queueCounter := k.GetQueueMessageCounter(ctx)
	if queueCounter > params.OrderBookExtraGasWindow {
		// The queue surcharge is extraGas = (queueCounter - window) * queueExtraGas with defaults window=100 and queueExtraGas=25,000. Baseline per trade is 100,000 gas. So for n trades in the block:
		//
		//  - First 100 trades: each 100,000 → 10,000,000 gas.
		//  - For the next m = n-100 trades: each costs 100,000 + (i*25,000) for i=1..m.
		//    Total = 10,000,000 + 100,000*m + 25,000 * m(m+1)/2
		//    = 10,000,000 + 112,500*m + 12,500*m^2.
		//
		//  Solving 12,500*m^2 + 112,500*m + 10,000,000 ≤ 1,000,000,000 gives m ≈ 276. (m=277 pushes the block slightly over the 1,000,000,000 limit - the limit at the time of this writing.),
		//
		//  So with those assumptions, the block can fit about 376 trades (100 before the surcharge kicks in, plus ~276 more with increasing gas).
		extraGas := (queueCounter - params.OrderBookExtraGasWindow) * params.OrderBookQueueExtraGas
		ctx.GasMeter().ConsumeGas(extraGas, "queue spam protection")
	}

	accAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	//calculate needed funds for this order
	ocReq := types.OrderCoinsArguments{
		OrderType:    msg.OrderType,
		OrderPrice:   msg.Price.String(),
		OrderAmount:  msg.Amount,
		Market:       &market,
		UserAddress:  msg.Creator,
		UserReceives: false,
	}

	orderCoins, err := k.GetOrderCoinsWithDust(ctx, ocReq)
	if err != nil {
		return nil, err
	}

	_, err = k.captureMsgFees(ctx, msg, accAddr)
	if err != nil {
		return nil, err
	}

	//capture user funds for this order
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, accAddr, types.ModuleName, sdk.NewCoins(orderCoins.Coin))
	if err != nil {
		return nil, err
	}

	qm := types.QueueMessage{
		MarketId:    msg.MarketId,
		OrderType:   msg.OrderType,
		MessageType: msg.OrderType,
		Amount:      msg.Amount,
		Price:       msg.Price,
		Owner:       msg.Creator,
	}

	k.SetQueueMessage(ctx, qm)
	k.StoreProcessedUserDust(ctx, orderCoins.UserDust, &orderCoins.Dust)

	err = k.emitOrderCreateMessageEvent(ctx, &qm)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgCreateOrderResponse{}, nil
}

func (k msgServer) CancelOrder(goCtx context.Context, msg *types.MsgCancelOrder) (*types.MsgCancelOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	_, found := k.GetMarketById(ctx, msg.MarketId)
	if !found {
		return nil, types.ErrMarketNotFound
	}

	order, found := k.GetOrder(ctx, msg.MarketId, msg.OrderType, msg.OrderId)
	if !found {
		return nil, types.ErrOrderNotFound
	}

	if order.Owner != msg.Creator {
		return nil, types.ErrUnauthorizedOrder
	}

	if k.HasPendingCancel(ctx, msg.MarketId, msg.OrderType, msg.OrderId) {
		return nil, fmt.Errorf("cancel already pending for order %s", msg.OrderId)
	}

	k.SetPendingCancel(ctx, msg.MarketId, msg.OrderType, msg.OrderId)

	qm := types.QueueMessage{
		MarketId:    msg.MarketId,
		MessageType: types.MessageTypeCancel,
		OrderId:     msg.OrderId,
		OrderType:   msg.OrderType,
		Owner:       msg.Creator,
	}

	k.SetQueueMessage(ctx, qm)

	err := k.emitOrderCancelMessageEvent(ctx, &order)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgCancelOrderResponse{}, nil
}

func (k msgServer) FillOrders(goCtx context.Context, msg *types.MsgFillOrders) (*types.MsgFillOrdersResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	accAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	market, found := k.GetMarketById(ctx, msg.MarketId)
	if !found {
		return nil, types.ErrMarketNotFound.Wrapf("market id: %s", msg.MarketId)
	}

	params := k.GetParams(ctx)
	ctx.GasMeter().ConsumeGas(params.FillOrdersExtraGas, "fill_orders")
	totalCoins := sdk.NewCoins()
	for key, fo := range msg.Orders {
		minAmt, err := CalculateMinAmount(fo.Price)
		if err != nil {
			return nil, types.ErrInvalidOrdersToFill.Wrapf("could not calculate minimum amount: %v", err)
		}

		amtInt, ok := math.NewIntFromString(fo.Amount)
		if !ok {
			return nil, types.ErrInvalidOrderAmount.Wrapf("amount could not be converted to Int")
		}
		if minAmt.GT(amtInt) {
			ctx.Logger().Debug("order amount is smaller than minimum", "order_to_fill", fo)

			continue
		}

		//calculate needed funds for this order
		//inverse the order type because the message specifies what kind of orders it wants to fill
		//so if user says he wants to fill buy orders, we need to act like he's selling, and vice versa
		ocReq := types.OrderCoinsArguments{
			OrderType:    types.TheOtherOrderType(msg.OrderType),
			OrderPrice:   fo.Price,
			OrderAmount:  amtInt,
			Market:       &market,
			UserAddress:  msg.Creator,
			UserReceives: false,
		}

		orderCoins, err := k.GetOrderCoinsWithDust(ctx, ocReq)
		if err != nil {
			return nil, err
		}
		foPriceDec, err := math.LegacyNewDecFromStr(fo.Price)
		if err != nil {
			return nil, types.ErrInvalidOrderPrice.Wrapf("could not parse fill order price: %v", err)
		}

		//messages of type "buy" are added on queue as messageType = "fill_buy"
		//messages of type "sell" are added on queue as messageType = "fill_sell"
		//orderType is the opposite of the orders we want to fill: if we fill "buy" it means we "sell", and vice versa.
		qm := types.QueueMessage{
			MarketId:    msg.MarketId,
			OrderType:   ocReq.OrderType, // already inversed when ocReq was built a few lines above
			MessageType: types.OrderTypeToMessageTypeFill(msg.OrderType),
			Amount:      amtInt,
			Price:       foPriceDec,
			Owner:       msg.Creator,
		}

		k.SetQueueMessage(ctx, qm)
		k.StoreProcessedUserDust(ctx, orderCoins.UserDust, &orderCoins.Dust)

		totalCoins = totalCoins.Add(orderCoins.Coin)
		//take extra gas for each order to fill
		//increase the gas based on the number of orders to fill
		ctx.GasMeter().ConsumeGas(params.FillOrdersExtraGas*uint64(key), "fill_orders")
	}

	if totalCoins.IsZero() {
		return nil, types.ErrInvalidOrdersToFill
	}

	_, err = k.captureTradingFees(ctx, accAddr, true)
	if err != nil {
		return nil, err
	}

	//capture user funds for this order
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, accAddr, types.ModuleName, totalCoins)
	if err != nil {
		return nil, err
	}

	return &types.MsgFillOrdersResponse{}, nil
}

func (k msgServer) payMarketCreateFee(ctx sdk.Context, payer sdk.AccAddress) error {
	if payer == nil {
		return fmt.Errorf("could not get payer address")
	}

	createMarketFee := k.CreateMarketFee(ctx)
	if !createMarketFee.IsPositive() {
		return nil
	}

	coinsCaptured, err := k.CaptureAndSwapUserFee(ctx, payer, sdk.NewCoins(createMarketFee), types.ModuleName)
	if err != nil {
		return err
	}

	sendErr := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, txfeecollectormoduletypes.CpFeeCollector, coinsCaptured)
	if sendErr != nil {
		return sendErr
	}

	return nil
}

func (k msgServer) emitMarketCreatedEvent(ctx sdk.Context, market *types.Market) error {
	return ctx.EventManager().EmitTypedEvent(
		&types.MarketCreatedEvent{
			Creator: market.Creator,
			Base:    market.Base,
			Quote:   market.Quote,
		},
	)
}

func (k msgServer) emitOrderCancelMessageEvent(ctx sdk.Context, order *types.Order) error {
	return ctx.EventManager().EmitTypedEvent(
		&types.OrderCancelMessageEvent{
			Creator:   order.Owner,
			MarketId:  order.MarketId,
			OrderId:   order.Id,
			OrderType: order.OrderType,
		},
	)
}

func (k msgServer) emitOrderCreateMessageEvent(ctx sdk.Context, qm *types.QueueMessage) error {
	return ctx.EventManager().EmitTypedEvent(
		&types.OrderCreateMessageEvent{
			Creator:   qm.Owner,
			MarketId:  qm.MarketId,
			OrderType: qm.OrderType,
			Amount:    qm.Amount.String(),
			Price:     qm.Price.String(),
		},
	)
}

func (k msgServer) captureMsgFees(ctx sdk.Context, msg *types.MsgCreateOrder, sender sdk.AccAddress) (sdk.Coin, error) {
	//used to decide if it's market taker or market maker
	_, found := k.GetAggregatedOrder(ctx, msg.MarketId, types.TheOtherOrderType(msg.OrderType), msg.Price.String())

	return k.captureTradingFees(ctx, sender, found)
}

func (k msgServer) captureTradingFees(ctx sdk.Context, sender sdk.AccAddress, isTaker bool) (coin sdk.Coin, err error) {
	params := k.GetParams(ctx)
	var fee sdk.Coin
	var destination string
	if isTaker {
		//is market taker
		fee = params.MarketTakerFee
		destination = params.TakerFeeDestination
	} else {
		//is market maker
		fee = params.MarketMakerFee
		destination = params.MakerFeeDestination
	}

	if !fee.IsPositive() {
		k.Logger().
			With("param_fee", fee).
			Debug("not capturing order create fee because it is not a positive value")

		return fee, nil
	}

	captured, err := k.CaptureAndSwapUserFee(ctx, sender, sdk.NewCoins(fee), types.ModuleName)
	if err != nil {
		return fee, err
	}

	destModule := txfeecollectormoduletypes.CpFeeCollector
	if destination == v2types.FeeDestinationBurnerModule {
		destModule = txfeecollectormoduletypes.BurnerFeeCollector
	}

	err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, destModule, captured)

	return fee, err
}

// checkPrice - validates the price of a message in order to make sure orders prices don't get messed up.
// Ensures users are not allowed to submit an order with a price lower/higher than first buy/sell
// in descending/ascending order by price.
//
// A price is valid if:
//   - if order type is "buy":
//   - price is lower than the first "sell" order found
//     AND
//   - price is lower than ALL queue messages of type "sell"
//   - if order type is "sell":
//   - price is higher than the first "buy" order found
//     AND
//   - price is higher than ALL queue messages of type "buy"
//
// TODO: implement a proper price "oracle" for bid and ask. We could store them on each market
func (k msgServer) checkPrice(ctx sdk.Context, msg *types.MsgCreateOrder) error {
	err := k.checkPriceInQueueMessages(ctx, msg, &msg.Price)
	if err != nil {
		return err
	}

	err = k.checkPriceInOrderBook(ctx, msg, &msg.Price)
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) checkPriceInOrderBook(ctx sdk.Context, msg *types.MsgCreateOrder, currentPrice *math.LegacyDec) error {
	oppositeType := types.TheOtherOrderType(msg.OrderType)
	if msg.OrderType == types.OrderTypeBuy {
		sells, _, err := k.getMarketAggregatedOrdersPaginated(ctx, msg.MarketId, oppositeType, &query.PageRequest{Limit: 1, Reverse: false})
		if err != nil {

			return fmt.Errorf("could not get sell orders pagination query: %w", err)
		}

		if len(sells) == 0 {

			return nil
		}

		sPrice := sells[0].Price

		if currentPrice.GT(sPrice) {

			return fmt.Errorf("buying price is invalid. A better price is available: %s", sPrice.String())
		}

	} else if msg.OrderType == types.OrderTypeSell {
		buys, _, err := k.getMarketAggregatedOrdersPaginated(ctx, msg.MarketId, oppositeType, &query.PageRequest{Limit: 1, Reverse: true})
		if err != nil {

			return fmt.Errorf("could not get buy orders pagination query: %w", err)
		}

		if len(buys) == 0 {
			return nil
		}

		bPrice := buys[0].Price

		if currentPrice.LT(bPrice) {
			return fmt.Errorf("selling price is invalid. A better price is available: %s", bPrice.String())
		}
	}

	return nil
}

func (k Keeper) checkPriceInQueueMessages(ctx sdk.Context, msg *types.MsgCreateOrder, currentPrice *math.LegacyDec) error {
	oppositeType := types.TheOtherOrderType(msg.OrderType)
	// Use market-filtered lookup to get only messages for this market
	// This is O(M) where M is messages for this market, instead of O(N) for all messages
	queueMessages := k.GetQueueMessagesByMarket(ctx, msg.MarketId)
	params := k.GetParams(ctx)
	msgsPrice := math.LegacyZeroDec()
	for _, queueMessage := range queueMessages {
		// Consume extra gas for each queue message scan
		ctx.GasMeter().ConsumeGas(params.OrderBookQueueMessageScanExtraGas, "queue_message_scan")

		// Filter by message type (only check opposite type orders)
		if queueMessage.MessageType != oppositeType {
			continue
		}

		p := queueMessage.Price

		if msgsPrice.IsZero() {
			msgsPrice = p
			continue
		}

		if oppositeType == types.OrderTypeBuy && p.GT(msgsPrice) {
			//save the biggest price for "buy" type
			msgsPrice = p
		} else if oppositeType == types.OrderTypeSell && p.LT(msgsPrice) {
			//save the lowest price for "sell" type
			msgsPrice = p
		}
	}

	//if we found opposite type messages
	if msgsPrice.IsPositive() {
		if oppositeType == types.OrderTypeBuy && currentPrice.LT(msgsPrice) {

			return fmt.Errorf("price is outdated. Better price is available: %s", msgsPrice.String())
		} else if oppositeType == types.OrderTypeSell && currentPrice.GT(msgsPrice) {

			return fmt.Errorf("price is outdated. Better price is available: %s", msgsPrice.String())
		}
	}

	return nil
}
