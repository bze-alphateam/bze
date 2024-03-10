package keeper

import (
	"context"
	"fmt"
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
		return nil, types.ErrInvalidOrderAmount.Wrapf("amount should be bigger than: %d", minAmt)
	}

	market, found := k.GetMarketById(ctx, msg.MarketId)
	if !found {
		return nil, types.ErrMarketNotFound.Wrapf("market id: %s", msg.MarketId)
	}

	//calculate needed funds for this order
	coin, err := k.GetOrderCoins(msg.OrderType, msg.Price, amtInt, &market)
	if err != nil {
		return nil, err
	}

	accAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	_ = k.captureOrderFees(ctx, msg, accAddr)
	//capture user funds for this order
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, accAddr, types.ModuleName, sdk.NewCoins(coin))
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

	_ = ctx

	return &types.MsgCreateOrderResponse{}, nil
}

func (k msgServer) captureOrderFees(ctx sdk.Context, msg *types.MsgCreateOrder, sender sdk.AccAddress) (coin sdk.Coin) {
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

	coin, err := sdk.ParseCoinNormalized(fee)
	if err != nil {
		ctx.Logger().Error(fmt.Sprintf("[MsgCreateOrder][captureOrderFees] could not parse fees: %v", err))
		return
	}

	if !coin.IsPositive() {
		ctx.Logger().Debug("[MsgCreateOrder][captureOrderFees] not capturing order create fee because if is not a positive value")
		return
	}

	if destination == types.FeeDestinationBurnerModule {
		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.FeeDestinationBurnerModule, sdk.NewCoins(coin))
		if err == nil {
			//successfully captured the funds
			return
		}

		ctx.Logger().Error(fmt.Sprintf("[MsgCreateOrder][captureOrderFees] could not send fee to burner: %v", err))
	}

	err = k.distrKeeper.FundCommunityPool(ctx, sdk.NewCoins(coin), sender)
	if err != nil {
		ctx.Logger().Error(fmt.Sprintf("[MsgCreateOrder][captureOrderFees] could not fund community pool: %v", err))
	}

	return
}
