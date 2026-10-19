package keeper

import (
	"context"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k Keeper) Daos(goCtx context.Context, req *types.QueryDaosRequest) (*types.QueryDaosResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var daos []types.Dao
	store := k.getPrefixedStore(goCtx, types.KeyPrefix(types.DaoKeyPrefix))
	pageRes, err := query.Paginate(store, req.Pagination, func(_ []byte, value []byte) error {
		var dao types.Dao
		if err := k.cdc.Unmarshal(value, &dao); err != nil {
			return err
		}

		daos = append(daos, dao)

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDaosResponse{Daos: daos, Pagination: pageRes}, nil
}

func (k Keeper) DaosByAdmin(goCtx context.Context, req *types.QueryDaosByAdminRequest) (*types.QueryDaosByAdminResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	admin, err := sdk.AccAddressFromBech32(req.Admin)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid admin address: %s", err)
	}

	// the index keys below the admin are the dao ids
	var daos []types.Dao
	index := k.getPrefixedStore(goCtx, append(types.KeyPrefix(types.DaoAdminKeyPrefix), types.DaoAdminPrefix(admin)...))
	pageRes, err := query.Paginate(index, req.Pagination, func(key []byte, _ []byte) error {
		dao, found := k.GetDao(goCtx, sdk.BigEndianToUint64(key))
		if !found {
			return status.Errorf(codes.Internal, "admin index names missing dao %d", sdk.BigEndianToUint64(key))
		}

		daos = append(daos, dao)

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDaosByAdminResponse{Daos: daos, Pagination: pageRes}, nil
}

func (k Keeper) Proposals(goCtx context.Context, req *types.QueryProposalsRequest) (*types.QueryProposalsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// the index keys below the dao are the proposal ids
	var proposals []types.Proposal
	index := k.getPrefixedStore(goCtx, append(types.KeyPrefix(types.DaoProposalKeyPrefix), types.DaoProposalsPrefix(req.DaoId)...))
	pageRes, err := query.FilteredPaginate(index, req.Pagination, func(key []byte, _ []byte, accumulate bool) (bool, error) {
		proposal, found := k.GetProposal(goCtx, sdk.BigEndianToUint64(key))
		if !found {
			return false, status.Errorf(codes.Internal, "proposal index names missing proposal %d", sdk.BigEndianToUint64(key))
		}

		if len(req.Statuses) > 0 && !slices.Contains(req.Statuses, proposal.Status) {
			return false, nil
		}

		if accumulate {
			proposals = append(proposals, proposal)
		}

		return true, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryProposalsResponse{Proposals: proposals, Pagination: pageRes}, nil
}

func (k Keeper) Ballots(goCtx context.Context, req *types.QueryBallotsRequest) (*types.QueryBallotsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var ballots []types.Ballot
	store := k.getPrefixedStore(goCtx, append(types.KeyPrefix(types.BallotKeyPrefix), types.ProposalBallotsPrefix(req.ProposalId)...))
	pageRes, err := query.Paginate(store, req.Pagination, func(_ []byte, value []byte) error {
		var ballot types.Ballot
		if err := k.cdc.Unmarshal(value, &ballot); err != nil {
			return err
		}

		ballots = append(ballots, ballot)

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryBallotsResponse{Ballots: ballots, Pagination: pageRes}, nil
}
