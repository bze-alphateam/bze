package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) ChangeAdmin(goCtx context.Context, msg *types.MsgChangeAdmin) (*types.MsgChangeAdminResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dAuth, err := k.Keeper.GetDenomAuthority(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	if msg.NewAdmin != "" {
		//validate new admin address only if one was provided
		_, err = sdk.AccAddressFromBech32(msg.NewAdmin)
		if err != nil {
			return nil, err
		}
	}

	if msg.Creator != dAuth.GetAdmin() {
		return nil, types.ErrUnauthorized
	}

	dAuth.Admin = msg.NewAdmin
	err = k.Keeper.SetDenomAuthority(ctx, msg.Denom, dAuth)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.TypeMsgChangeAdmin,
			sdk.NewAttribute(types.AttributeDenom, msg.GetDenom()),
			sdk.NewAttribute(types.AttributeNewAdmin, msg.NewAdmin),
		),
	})

	return &types.MsgChangeAdminResponse{}, nil
}
