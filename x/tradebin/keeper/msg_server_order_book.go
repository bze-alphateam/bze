package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateMarket(goCtx context.Context, msg *types.MsgCreateMarket) (*types.MsgCreateMarketResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgCreateMarketResponse{}, nil
}

func (k msgServer) CreateOrder(goCtx context.Context, msg *types.MsgCreateOrder) (*types.MsgCreateOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgCreateOrderResponse{}, nil
}

func (k msgServer) CancelOrder(goCtx context.Context, msg *types.MsgCancelOrder) (*types.MsgCancelOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgCancelOrderResponse{}, nil
}

func (k msgServer) FillOrders(goCtx context.Context, msg *types.MsgFillOrders) (*types.MsgFillOrdersResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgFillOrdersResponse{}, nil
}
