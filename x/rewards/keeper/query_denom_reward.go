package keeper

import (
	"context"
	"strings"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) DenomReward(goCtx context.Context, req *types.QueryDenomRewardRequest) (*types.QueryDenomRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetDenomReward(ctx, req.Denom)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryDenomRewardResponse{DenomReward: val}, nil
}

func (k Keeper) DenomRewardAll(goCtx context.Context, req *types.QueryDenomRewardAllRequest) (*types.QueryDenomRewardAllResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var list []types.DenomReward
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.DenomRewardKeyPrefix))

	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		var denomReward types.DenomReward
		if err := k.cdc.Unmarshal(value, &denomReward); err != nil {
			return err
		}

		list = append(list, denomReward)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDenomRewardAllResponse{List: list, Pagination: pageRes}, nil
}

// DenomRewardPrizes returns every prize accumulator of a denom reward. The list is
// bounded by the max_prize_denoms_per_dr cap, so it is deliberately unpaginated.
func (k Keeper) DenomRewardPrizes(goCtx context.Context, req *types.QueryDenomRewardPrizesRequest) (*types.QueryDenomRewardPrizesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryDenomRewardPrizesResponse{List: k.GetAllDenomRewardPrizes(ctx, req.Denom)}, nil
}

func (k Keeper) DenomRewardSchedules(goCtx context.Context, req *types.QueryDenomRewardSchedulesRequest) (*types.QueryDenomRewardSchedulesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var list []types.DenomRewardSchedule
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := k.getPrefixedStore(ctx, types.DenomRewardSchedulePrefix(req.Denom))

	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		var schedule types.DenomRewardSchedule
		if err := k.cdc.Unmarshal(value, &schedule); err != nil {
			return err
		}

		list = append(list, schedule)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDenomRewardSchedulesResponse{List: list, Pagination: pageRes}, nil
}

// DenomRewardParticipant returns a participant's position together with the pending
// (claimable) coins per prize denom. The pending computation is a read-only mirror of
// settleDenomParticipant: pending = amount × (S − index) truncated to whole units, a
// missing index reading as zero (the lazy-zero rule). Nothing is written — no payout,
// no index stamping — so the response is exactly what a claim would pay right now.
func (k Keeper) DenomRewardParticipant(goCtx context.Context, req *types.QueryDenomRewardParticipantRequest) (*types.QueryDenomRewardParticipantResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	participant, found := k.GetDenomRewardParticipant(ctx, req.Denom, req.Address)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	deposited, err := math.LegacyNewDecFromStr(participant.Amount)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pending := sdk.NewCoins()
	var iterErr error
	k.IterateDenomRewardPrizes(ctx, req.Denom, func(ctx sdk.Context, prize types.DenomRewardPrize) (stop bool) {
		s, sErr := math.LegacyNewDecFromStr(prize.DistributedStake)
		if sErr != nil {
			iterErr = sErr
			return true
		}

		index := math.LegacyZeroDec()
		if stored, idxFound := k.GetDenomRewardParticipantIndex(ctx, req.Address, req.Denom, prize.PrizeDenom); idxFound {
			index, sErr = math.LegacyNewDecFromStr(stored.Index)
			if sErr != nil {
				iterErr = sErr
				return true
			}
		}

		reward := deposited.Mul(s.Sub(index)).TruncateInt()
		if reward.IsPositive() {
			pending = pending.Add(sdk.NewCoin(prize.PrizeDenom, reward))
		}

		return false
	})

	if iterErr != nil {
		return nil, status.Error(codes.Internal, iterErr.Error())
	}

	return &types.QueryDenomRewardParticipantResponse{Participant: participant, Pending: pending}, nil
}

// DenomRewardParticipations lists every denom reward position of an address by paginating
// the address-first marker index and resolving each marker to its participant record.
func (k Keeper) DenomRewardParticipations(goCtx context.Context, req *types.QueryDenomRewardParticipationsRequest) (*types.QueryDenomRewardParticipationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var list []types.DenomRewardParticipant
	ctx := sdk.UnwrapSDKContext(goCtx)

	markerStore := k.getPrefixedStore(ctx, types.DenomRewardParticipantMarkerPrefix(req.Address))

	pageRes, err := query.Paginate(markerStore, req.Pagination, func(key []byte, value []byte) error {
		// the marker key (prefix stripped) is "{staking_denom}/"; denoms never contain "/".
		stakingDenom := strings.TrimSuffix(string(key), "/")
		participant, found := k.GetDenomRewardParticipant(ctx, stakingDenom, req.Address)
		if !found {
			// markers are written and deleted in the same call as the participant record,
			// so this cannot happen on a consistent store; skip rather than fail the page.
			return nil
		}

		list = append(list, participant)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDenomRewardParticipationsResponse{List: list, Pagination: pageRes}, nil
}
