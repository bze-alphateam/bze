package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Boost(goCtx context.Context, req *types.QueryBoostRequest) (*types.QueryBoostResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetBoost(ctx, req.RewardId, req.BoostId)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryBoostResponse{Boost: val}, nil
}

func (k Keeper) RewardBoosts(goCtx context.Context, req *types.QueryRewardBoostsRequest) (*types.QueryRewardBoostsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryRewardBoostsResponse{List: k.GetRewardBoosts(ctx, req.RewardId)}, nil
}

func (k Keeper) AllBoosts(goCtx context.Context, req *types.QueryAllBoostsRequest) (*types.QueryAllBoostsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var boosts []types.Boost
	ctx := sdk.UnwrapSDKContext(goCtx)

	boostStore := k.getPrefixedStore(ctx, types.KeyPrefix(types.BoostKeyPrefix))

	pageRes, err := query.Paginate(boostStore, req.Pagination, func(key []byte, value []byte) error {
		var boost types.Boost
		if err := k.cdc.Unmarshal(value, &boost); err != nil {
			return err
		}

		boosts = append(boosts, boost)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllBoostsResponse{List: boosts, Pagination: pageRes}, nil
}

// MaxCleanupStatusRemaining caps the counting work a single
// BoostCleanupStatus query performs. The index grows with (cheap) JoinStaking
// txs, so an uncapped count would let anyone turn the public query into O(n)
// node work. A response with Remaining == MaxCleanupStatusRemaining means
// "this many or more".
const MaxCleanupStatusRemaining uint64 = 10_000

// BoostCleanupStatus reports a boost's cleanup progress: the persisted cursor
// and how many participant index entries remain after it (capped at
// MaxCleanupStatusRemaining — the cap value means "this many or more"). The
// count lives here and not in the tx response because computing it on-chain
// would cost O(remaining) gas on every cleanup call.
func (k Keeper) BoostCleanupStatus(goCtx context.Context, req *types.QueryBoostCleanupStatusRequest) (*types.QueryBoostCleanupStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	boost, found := k.GetBoost(ctx, req.RewardId, req.BoostId)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryBoostCleanupStatusResponse{
		Remaining: k.CountStakingRewardParticipantIndexEntries(ctx, req.RewardId, boost.CleanupCursor, MaxCleanupStatusRemaining),
		Cursor:    boost.CleanupCursor,
	}, nil
}
