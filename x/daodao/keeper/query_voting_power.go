package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// VotingPower returns an address's CURRENT voting power within a DAO and
// the DAO's total. For historical / snapshotted reads at a proposal's
// creation height, callers should use Epic 3's proposal queries.
func (k Keeper) VotingPower(ctx context.Context, req *types.QueryVotingPowerRequest) (*types.QueryVotingPowerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.DaoId == 0 {
		return nil, status.Error(codes.InvalidArgument, "dao_id must be non-zero")
	}
	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}

	dao, found := k.GetDao(ctx, req.DaoId)
	if !found {
		return nil, status.Errorf(codes.NotFound, "dao id=%d not found", req.DaoId)
	}
	backend, err := k.backendFor(dao)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	power, err := backend.Power(ctx, dao, addr)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	total, err := backend.TotalPower(ctx, dao)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryVotingPowerResponse{Power: power, Total: total}, nil
}

// TotalVotingPower returns the DAO's CURRENT total voting power.
func (k Keeper) TotalVotingPower(ctx context.Context, req *types.QueryTotalVotingPowerRequest) (*types.QueryTotalVotingPowerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.DaoId == 0 {
		return nil, status.Error(codes.InvalidArgument, "dao_id must be non-zero")
	}
	dao, found := k.GetDao(ctx, req.DaoId)
	if !found {
		return nil, status.Errorf(codes.NotFound, "dao id=%d not found", req.DaoId)
	}
	backend, err := k.backendFor(dao)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	total, err := backend.TotalPower(ctx, dao)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryTotalVotingPowerResponse{Total: total}, nil
}
