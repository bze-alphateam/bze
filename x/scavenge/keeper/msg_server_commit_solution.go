package keeper

import (
	"context"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmonaut/bzedgev5/x/scavenge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CommitSolution(goCtx context.Context, msg *types.MsgCommitSolution) (*types.MsgCommitSolutionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var commit = types.Commit{
		Index:                 msg.SolutionScavengerHash,
		SolutionHash:          msg.SolutionHash,
		SolutionScavengerHash: msg.SolutionScavengerHash,
	}

	_, isFound := k.GetCommit(ctx, commit.SolutionScavengerHash)
	if isFound {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "Commit with this hash already exists")
	}
	k.SetCommit(ctx, commit)

	return &types.MsgCommitSolutionResponse{}, nil
}
