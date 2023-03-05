package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/cointrunk/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) AcceptedDomain(goCtx context.Context, req *types.QueryAcceptedDomainRequest) (*types.QueryAcceptedDomainResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var acceptedDomains []types.AcceptedDomain
	store := ctx.KVStore(k.storeKey)
	adStore := prefix.NewStore(store, types.KeyPrefix(types.AcceptedDomainKeyPrefix))

	pageRes, err := query.Paginate(adStore, req.Pagination, func(key []byte, value []byte) error {
		var acceptedDomain types.AcceptedDomain
		if err := k.cdc.Unmarshal(value, &acceptedDomain); err != nil {
			return err
		}
		acceptedDomains = append(acceptedDomains, acceptedDomain)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAcceptedDomainResponse{AcceptedDomain: acceptedDomains, Pagination: pageRes}, nil
}
