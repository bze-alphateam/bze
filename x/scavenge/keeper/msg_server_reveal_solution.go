package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/tendermint/tendermint/crypto"

	"github.com/bze-alphateam/bze/x/scavenge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) RevealSolution(goCtx context.Context, msg *types.MsgRevealSolution) (*types.MsgRevealSolutionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var solutionScavengerBytes = []byte(msg.ScavengeIndex + msg.Solution + msg.Creator)
	var solutionScavengerHash = sha256.Sum256(solutionScavengerBytes)
	var solutionScavengerHashString = hex.EncodeToString(solutionScavengerHash[:])
	_, isFound := k.GetCommit(ctx, solutionScavengerHashString)
	if !isFound {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "Commit with this hash does not exist")
	}

	var scavenge = types.Scavenge{}
	scavenge, isFound = k.GetScavenge(ctx, msg.ScavengeIndex)
	if !isFound {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "Scavenge with this solution hash doesnt exist")
	}

	_, err := sdk.AccAddressFromBech32(scavenge.Scavenger)
	if err == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "Scavenge already solved")
	}

	var solutionHash = sha256.Sum256([]byte(msg.Solution))
	var solutionHashString = hex.EncodeToString(solutionHash[:])
	if solutionHashString != scavenge.SolutionHash {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "This is NOT the correct solution")
	}

	scavenge.Scavenger = msg.Creator
	scavenge.Solution = msg.Solution

	moduleAccount := sdk.AccAddress(crypto.AddressHash([]byte(types.ModuleName)))
	scavengerAddress, err := sdk.AccAddressFromBech32(scavenge.Scavenger)
	if err != nil {
		panic(err)
	}

	reward, err := sdk.ParseCoinsNormalized(scavenge.Reward)
	if err != nil {
		panic(err)
	}

	sdkErr := k.bankKeeper.SendCoins(ctx, moduleAccount, scavengerAddress, reward)
	if sdkErr != nil {
		return nil, sdkErr
	}

	k.SetScavenge(ctx, scavenge)

	return &types.MsgRevealSolutionResponse{}, nil
}
