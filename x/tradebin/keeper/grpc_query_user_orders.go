package keeper

import (
	"context"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) UserOrders(goCtx context.Context, req *types.QueryUserOrdersRequest) (*types.QueryUserOrdersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	_, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	var orders []types.OrderReference
	ctx := sdk.UnwrapSDKContext(goCtx)
	userOrderStore := k.getUserOrderByAddressStore(ctx, req.Address)

	pageRes, err := query.Paginate(userOrderStore, req.Pagination, func(key []byte, value []byte) error {
		var order types.OrderReference
		if err := k.cdc.Unmarshal(value, &order); err != nil {
			return err
		}

		orders = append(orders, order)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryUserOrdersResponse{List: orders, Pagination: pageRes}, nil
}
