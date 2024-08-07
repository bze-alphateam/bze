package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateOrder(goCtx context.Context, msg *types.MsgCreateOrder) (*types.MsgCreateOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

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
