package keeper

import (
	"context"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/tendermint/tendermint/crypto"

	"github.com/cosmonaut/bzedgev5/x/scavenge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) SubmitScavenge(goCtx context.Context, msg *types.MsgSubmitScavenge) (*types.MsgSubmitScavengeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var scavenge = types.Scavenge{
		Index:        msg.SolutionHash,
		Description:  msg.Description,
		SolutionHash: msg.SolutionHash,
		Reward:       msg.Reward,
	}

	_, isFound := k.GetScavenge(ctx, scavenge.SolutionHash)
	if isFound {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "Scavenge with the same solution already exists")
	}

	scavenger, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}

	reward, err := sdk.ParseCoinsNormalized(scavenge.Reward)
	if err != nil {
		panic(err)
	}

	moduleAccountAddress := sdk.AccAddress(crypto.AddressHash([]byte(types.ModuleName)))
	sdkErr := k.bankKeeper.SendCoins(ctx, scavenger, moduleAccountAddress, reward)
	if sdkErr != nil {
		return nil, sdkErr
	}

	k.SetScavenge(ctx, scavenge)

	return &types.MsgSubmitScavengeResponse{}, nil
}
