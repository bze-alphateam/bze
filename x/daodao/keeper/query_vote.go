package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Vote returns a single voter's vote on a proposal. NOT_FOUND if the voter
// hasn't voted (or doesn't exist as an account at all).
func (k Keeper) Vote(goCtx context.Context, req *types.QueryVoteRequest) (*types.QueryVoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	voterAddr, err := sdk.AccAddressFromBech32(req.Voter)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}
	v, ok := k.GetVote(ctx, req.DaoId, req.ProposalId, voterAddr)
	if !ok {
		return nil, status.Errorf(codes.NotFound,
			"no vote from %s on dao=%d proposal=%d", req.Voter, req.DaoId, req.ProposalId)
	}
	return &types.QueryVoteResponse{Vote: v}, nil
}
