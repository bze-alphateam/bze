package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/burner/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) FundBurner(goCtx context.Context, msg *types.MsgFundBurner) (*types.MsgFundBurnerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	amount, err := sdk.ParseCoinsNormalized(msg.Amount)
	if err != nil {
		return nil, err
	}

	creatorAccount, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAccount, types.ModuleName, amount)
	if err != nil {
		return nil, err
	}

	err = ctx.EventManager().EmitTypedEvent(&types.FundBurnerEvent{From: msg.Creator, Amount: amount.String()})
	if err != nil {
		ctx.Logger().Error("failed to emit fund burner event", "error", err)
	}

	return &types.MsgFundBurnerResponse{}, nil
}
