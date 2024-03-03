package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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

	qm := types.QueueMessage{
		MarketId:    msg.MarketId,
		MessageType: types.OrderTypeCancel,
		OrderId:     msg.OrderId,
		OrderType:   msg.OrderType,
		Owner:       msg.Creator,
	}

	k.SetQueueMessage(ctx, qm)

	return &types.MsgCancelOrderResponse{}, nil
}
