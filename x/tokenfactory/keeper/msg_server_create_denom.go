package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateDenom(goCtx context.Context, msg *types.MsgCreateDenom) (*types.MsgCreateDenomResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	denom, err := k.validateCreateDenom(ctx, msg.Creator, msg.Subdenom)
	if err != nil {
		return nil, err
	}

	err = k.chargeForCreateDenom(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	err = k.createDenomAfterValidation(ctx, msg.Creator, denom)

	return &types.MsgCreateDenomResponse{NewDenom: denom}, err
}
