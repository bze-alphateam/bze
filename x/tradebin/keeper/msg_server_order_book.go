package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
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

	minAmt, err := CalculateMinAmount(msg.Price)
	if err != nil {
		return nil, types.ErrInvalidOrderPrice.Wrapf("could not calculate minimum amount: %v", err)
	}

	amtInt, err := types.ParseOrderAmount(msg.Amount)
	if err != nil {
		return nil, err
	}
	if minAmt.GT(amtInt) {
		return nil, types.ErrInvalidOrderAmount.Wrapf("amount should be bigger than: %s", minAmt.String())
	}

	market, found := k.GetMarketById(ctx, msg.MarketId)
	if !found {
		return nil, types.ErrMarketNotFound.Wrapf("market id: %s", msg.MarketId)
	}

	//no new orders on a halted market; cancelling stays possible
	if k.isMarketHalted(ctx, &market) {
		return nil, types.ErrDenomHalted.Wrapf("market %s is halted", msg.MarketId)
	}

	//a new owner joining a crowded price level pays extra gas, before any fee or escrow
	params := k.GetParams(ctx)
	k.consumePriceLevelGas(ctx, msg, params)

	accAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	//calculate needed funds for this order
	ocReq := types.OrderCoinsArguments{
		OrderType:    msg.OrderType,
		OrderPrice:   msg.Price,
		OrderAmount:  amtInt,
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
	// the escrow dust is stored before the engine runs: the engine may update the same user's dust
	k.StoreProcessedUserDust(ctx, orderCoins.UserDust, &orderCoins.Dust)

	engine, err := k.orderBookEngine(params.MaxFillsPerOrder)
	if err != nil {
		return nil, err
	}

	order := types.Order{
		MarketId:  msg.MarketId,
		OrderType: msg.OrderType,
		Amount:    amtInt.String(),
		Price:     msg.Price,
		Owner:     msg.Creator,
	}

	err = engine.PlaceOrder(ctx, order)
	if err != nil {
		return nil, err
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

	engine, err := k.orderBookEngine(k.GetParams(ctx).MaxFillsPerOrder)
	if err != nil {
		return nil, err
	}

	err = engine.CancelOrder(ctx, order)
	if err != nil {
		return nil, err
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

	//no fills on a halted market
	if k.isMarketHalted(ctx, &market) {
		return nil, types.ErrDenomHalted.Wrapf("market %s is halted", msg.MarketId)
	}

	params := k.GetParams(ctx)
	engine, err := k.orderBookEngine(params.MaxFillsPerOrder)
	if err != nil {
		return nil, err
	}

	ctx.GasMeter().ConsumeGas(params.FillOrdersExtraGas, "fill_orders")
	totalCoins := sdk.NewCoins()
	for key, fo := range msg.Orders {
		minAmt, err := CalculateMinAmount(fo.Price)
		if err != nil {
			return nil, types.ErrInvalidOrdersToFill.Wrapf("could not calculate minimum amount: %v", err)
		}

		amtInt, err := types.ParseOrderAmount(fo.Amount)
		if err != nil {
			return nil, err
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

		//capture user funds for this entry before executing it: settlement pays out of the module account
		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, accAddr, types.ModuleName, sdk.NewCoins(orderCoins.Coin))
		if err != nil {
			return nil, err
		}
		// the escrow dust is stored before the engine runs: the engine may update the same user's dust
		k.StoreProcessedUserDust(ctx, orderCoins.UserDust, &orderCoins.Dust)

		//orderType is the opposite of the orders we want to fill: if we fill "buy" it means we "sell", and vice versa.
		order := types.Order{
			MarketId:  msg.MarketId,
			OrderType: ocReq.OrderType, // already inversed when ocReq was built a few lines above
			Amount:    amtInt.String(),
			Price:     fo.Price,
			Owner:     msg.Creator,
		}

		err = engine.FillOrder(ctx, order)
		if err != nil {
			return nil, err
		}

		totalCoins = totalCoins.Add(orderCoins.Coin)
		//take extra gas for each order to fill
		//increase the gas based on the number of orders to fill
		ctx.GasMeter().ConsumeGas(params.FillOrdersExtraGas*uint64(key), "fill_orders")
	}

	if totalCoins.IsZero() {
		return nil, types.ErrInvalidOrdersToFill
	}

	//the taker fee is captured once per message, after every entry was executed
	_, err = k.captureTradingFees(ctx, accAddr, true)
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

func (k msgServer) captureMsgFees(ctx sdk.Context, msg *types.MsgCreateOrder, sender sdk.AccAddress) (sdk.Coin, error) {
	//used to decide if it's market taker or market maker
	_, found := k.GetAggregatedOrder(ctx, msg.MarketId, types.TheOtherOrderType(msg.OrderType), msg.Price)

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

	captured, err := k.CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx, sender, sdk.NewCoins(fee), types.ModuleName)
	if err != nil {
		return fee, err
	}

	destModule := txfeecollectormoduletypes.CpFeeCollector
	if destination == v2types.FeeDestinationBurnerModule {
		destModule = burnermoduletypes.ModuleName
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
//   - if order type is "sell":
//   - price is higher than the first "buy" order found
//
// TODO: implement a proper price "oracle" for bid and ask. We could store them on each market
func (k msgServer) checkPrice(ctx sdk.Context, msg *types.MsgCreateOrder) error {
	currentPrice, err := math.LegacyNewDecFromStr(msg.Price)
	if err != nil {
		//should never happen! Message should be validated before this function is called
		return fmt.Errorf("could not parse current price: %s", msg.Price)
	}

	err = k.checkPriceInOrderBook(ctx, msg, &currentPrice)
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

		sPrice, err := math.LegacyNewDecFromStr(sells[0].Price)
		if err != nil {

			return fmt.Errorf("could not parse sell price: %s", sells[0].Price)
		}

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

		bPrice, err := math.LegacyNewDecFromStr(buys[0].Price)
		if err != nil {
			return fmt.Errorf("could not parse buy price: %s", buys[0].Price)
		}

		if currentPrice.LT(bPrice) {
			return fmt.Errorf("selling price is invalid. A better price is available: %s", bPrice.String())
		}
	}

	return nil
}

// orderBookEngine builds the matching engine the order book message servers execute orders with.
func (k msgServer) orderBookEngine(maxFills uint64) (*ProcessingEngine, error) {
	return NewProcessingEngine(k.Keeper, k.bankKeeper, maxFills, k.Logger().With("processor", "order_book_processor"))
}
