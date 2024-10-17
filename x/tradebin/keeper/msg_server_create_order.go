package keeper

import (
	"context"
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

func (k msgServer) CreateOrder(goCtx context.Context, msg *types.MsgCreateOrder) (*types.MsgCreateOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.checkPrice(ctx, msg)
	if err != nil {
		return nil, types.ErrInvalidOrderPrice.Wrapf("check price failed: %v", err)
	}

	minAmt := CalculateMinAmount(msg.Price)
	amtInt, ok := sdk.NewIntFromString(msg.Amount)
	if !ok {
		return nil, types.ErrInvalidOrderAmount.Wrapf("amount could not be converted to Int")
	}
	if minAmt.GT(amtInt) {
		return nil, types.ErrInvalidOrderAmount.Wrapf("amount should be bigger than: %s", minAmt.String())
	}

	market, found := k.GetMarketById(ctx, msg.MarketId)
	if !found {
		return nil, types.ErrMarketNotFound.Wrapf("market id: %s", msg.MarketId)
	}

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

	_, err = k.captureOrderFees(ctx, msg, accAddr)
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
		k.Logger(ctx).Error(err.Error())
	}

	return &types.MsgCreateOrderResponse{}, nil
}

func (k msgServer) captureOrderFees(ctx sdk.Context, msg *types.MsgCreateOrder, sender sdk.AccAddress) (coin sdk.Coin, err error) {
	//used to decide if it's market taker or market maker
	_, found := k.GetAggregatedOrder(ctx, msg.MarketId, types.TheOtherOrderType(msg.OrderType), msg.Price)
	params := k.GetParams(ctx)
	var fee string
	var destination string
	if found {
		//is market taker
		fee = params.GetMarketTakerFee()
		destination = params.GetTakerFeeDestination()
	} else {
		//is market maker
		fee = params.GetMarketMakerFee()
		destination = params.GetMakerFeeDestination()
	}

	coin, err = sdk.ParseCoinNormalized(fee)
	if err != nil {
		k.Logger(ctx).
			With("err", err).
			With("param_fee", fee).
			Error("could not parse fee coin. trading fee not captured")

		//do not return error!! if we have a wrong fee parameter we don't want to stall the trading process
		return coin, nil
	}

	if !coin.IsPositive() {
		k.Logger(ctx).
			With("param_fee", fee).
			Debug("not capturing order create fee because it is not a positive value")

		return
	}

	if destination == types.FeeDestinationBurnerModule {
		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.FeeDestinationBurnerModule, sdk.NewCoins(coin))

		return
	}

	err = k.distrKeeper.FundCommunityPool(ctx, sdk.NewCoins(coin), sender)

	return
}

// checkPrice - validates the price of a message in order to make sure orders prices don't get messed up.
// Ensures users are not allowed to submit an order with a price lower/higher than first buy/sell
// in descending/ascending order by price.
//
// A price is valid if:
// - has an opposite order that can match and the order is filled immediately (market taker)
// OR
//   - if order type is "buy":
//   - price is lower than the first "sell" order found
//     AND
//   - price is lower than ALL queue messages of type "sell"
//   - if order type is "sell":
//   - price is higher than the first "buy" order found
//     AND
//   - price is higher than ALL queue messages of type "buy"
func (k msgServer) checkPrice(ctx sdk.Context, msg *types.MsgCreateOrder) error {
	oppositeType := types.TheOtherOrderType(msg.OrderType)
	//if order can be filled then the price is valid
	_, found := k.GetAggregatedOrder(ctx, msg.MarketId, oppositeType, msg.Price)
	if found {

		return nil
	}

	currentPrice, err := sdk.NewDecFromStr(msg.Price)
	if err != nil {
		//should never happen! Message should be validated before this function is called
		return fmt.Errorf("could not parse current price: %s", msg.Price)
	}

	err = k.checkPriceInQueueMessages(ctx, msg, &currentPrice)
	if err != nil {
		return err
	}

	err = k.checkPriceInOrderBook(ctx, msg, &currentPrice)
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) checkPriceInOrderBook(ctx sdk.Context, msg *types.MsgCreateOrder, currentPrice *sdk.Dec) error {
	oppositeType := types.TheOtherOrderType(msg.OrderType)
	if msg.OrderType == types.OrderTypeBuy {
		sells, _, err := k.getMarketAggregatedOrdersPaginated(ctx, msg.MarketId, oppositeType, &query.PageRequest{Limit: 1, Reverse: false})
		if err != nil {

			return fmt.Errorf("could not get buy orders pagination query: %w", err)
		}

		if len(sells) == 0 {

			return nil
		}

		sPrice, err := sdk.NewDecFromStr(sells[0].Price)
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

		bPrice, err := sdk.NewDecFromStr(buys[0].Price)
		if err != nil {
			return fmt.Errorf("could not parse sell price: %s", buys[0].Price)
		}

		if currentPrice.LT(bPrice) {
			return fmt.Errorf("selling price is invalid. A better price is available: %s", bPrice.String())
		}
	}

	return nil
}

func (k Keeper) checkPriceInQueueMessages(ctx sdk.Context, msg *types.MsgCreateOrder, currentPrice *sdk.Dec) error {
	oppositeType := types.TheOtherOrderType(msg.OrderType)
	queueMessages := k.GetAllQueueMessage(ctx)
	msgsPrice := sdk.ZeroDec()
	for _, queueMessage := range queueMessages {
		//check against MessageType because we have "cancel" type besides "buy" and "sell"
		if queueMessage.MarketId != msg.MarketId || queueMessage.MessageType != oppositeType {
			continue
		}

		p, err := sdk.NewDecFromStr(queueMessage.Price)
		if err != nil {
			k.Logger(ctx).With("func", "checkPrice").Error(fmt.Sprintf("could not parse message price: %s", err.Error()))
			continue
		}

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

func (k msgServer) emitOrderCreateMessageEvent(ctx sdk.Context, qm *types.QueueMessage) error {
	return ctx.EventManager().EmitTypedEvent(
		&types.OrderCreateMessageEvent{
			Creator:   qm.Owner,
			MarketId:  qm.MarketId,
			OrderType: qm.OrderType,
			Amount:    qm.Amount,
			Price:     qm.Price,
		},
	)
}
