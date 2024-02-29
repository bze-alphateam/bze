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
		return nil, types.ErrMarketNotFound.Wrapf("market id: %s", msg.MarketId)
	}

	//TODO: check the user owns the actual order
	//TODO: check order is part of the selected market

	qm := types.QueueMessage{
		MarketId:    msg.MarketId,
		MessageType: types.OrderTypeCancel,
		OrderId:     msg.OrderId,
	}

	k.SetQueueMessage(ctx, qm)

	return &types.MsgCancelOrderResponse{}, nil
}
