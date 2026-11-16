package keeper

import (
	"context"
	"math"
	"slices"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k Keeper) Ballot(goCtx context.Context, req *types.QueryBallotRequest) (*types.QueryBallotResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	voter, err := sdk.AccAddressFromBech32(req.Voter)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid voter address: %s", err)
	}

	ballot, found := k.GetBallot(goCtx, req.ProposalId, voter)
	if !found {
		return nil, status.Errorf(codes.NotFound, "no ballot of %s on proposal %d", req.Voter, req.ProposalId)
	}

	return &types.QueryBallotResponse{Ballot: ballot}, nil
}

func (k Keeper) VotingPower(goCtx context.Context, req *types.QueryVotingPowerRequest) (*types.QueryVotingPowerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid address: %s", err)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	dao, found := k.GetDao(ctx, req.DaoId)
	if !found {
		return nil, status.Errorf(codes.NotFound, "dao %d not found", req.DaoId)
	}

	res := &types.QueryVotingPowerResponse{Current: k.currentWeight(ctx, dao, addr)}

	// the weight a vote would count now, the one Vote computes; bounded by
	// max_open_proposals
	for _, proposalId := range k.openProposalIds(ctx, dao.Id) {
		proposal, found := k.GetProposal(ctx, proposalId)
		if !found {
			continue
		}

		weight, shares := k.voterWeight(ctx, dao, proposal, addr)
		res.Proposals = append(res.Proposals, types.ProposalVotingPower{
			ProposalId:     proposalId,
			EligibleWeight: weight,
			Shares:         shares,
		})
	}

	return res, nil
}

// currentWeight is the address's weight in the DAO now: 1 for a member of a
// static list, or its stake in the DAO's current sources. Join records are
// ignored: they only matter against a proposal's submit height.
func (k Keeper) currentWeight(ctx sdk.Context, dao types.Dao, addr sdk.AccAddress) sdkmath.Int {
	if dao.Type == types.DaoType_DAO_TYPE_STATIC_LIST {
		if slices.Contains(dao.Members, addr.String()) {
			return sdkmath.OneInt()
		}

		return sdkmath.ZeroInt()
	}

	// the DAO's own stake never votes on its proposals
	if addr.String() == dao.Address {
		return sdkmath.ZeroInt()
	}

	// no join record reaches the highest height, so all stake counts
	weight, _ := k.eligibleWeight(ctx, dao.Electorate, addr, math.MaxInt64)

	return weight
}

func (k Keeper) JoinRecord(goCtx context.Context, req *types.QueryJoinRecordRequest) (*types.QueryJoinRecordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Source == "" {
		return nil, status.Error(codes.InvalidArgument, "source is empty")
	}

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid address: %s", err)
	}

	record, found := k.GetJoinRecord(goCtx, []byte(req.Source), addr)
	if !found {
		return nil, status.Errorf(codes.NotFound, "no join record of %s on %s", req.Address, req.Source)
	}

	return &types.QueryJoinRecordResponse{Height: record.Height}, nil
}
