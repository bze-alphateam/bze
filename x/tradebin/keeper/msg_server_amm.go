package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateLiquidityPool(goCtx context.Context, msg *types.MsgCreateLiquidityPool) (*types.MsgCreateLiquidityPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgCreateLiquidityPoolResponse{}, nil
}

func (k msgServer) RemoveLiquidity(goCtx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgRemoveLiquidityResponse{}, nil
}

func (k msgServer) AddLiquidity(goCtx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgAddLiquidityResponse{}, nil
}

func (k msgServer) MultiSwap(goCtx context.Context, msg *types.MsgMultiSwap) (*types.MsgMultiSwapResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgMultiSwapResponse{}, nil
}
