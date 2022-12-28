package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
		return nil, err
	}

	_ = ctx

	return &types.MsgFundBurnerResponse{}, nil
}
