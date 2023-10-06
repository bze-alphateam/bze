package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/cometbft/cometbft/crypto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/scavenge/types"
)

func (k msgServer) SubmitScavenge(goCtx context.Context, msg *types.MsgSubmitScavenge) (*types.MsgSubmitScavengeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var indexBytes = []byte(msg.SolutionHash + msg.Description)
	var indexHash = sha256.Sum256(indexBytes)
	var indexString = hex.EncodeToString(indexHash[:])

	var scavenge = types.Scavenge{
		Index:        indexString,
		Description:  msg.Description,
		SolutionHash: msg.SolutionHash,
		Reward:       msg.Reward,
	}

	_, isFound := k.GetScavenge(ctx, scavenge.Index)
	if isFound {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "Scavenge already exists")
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
