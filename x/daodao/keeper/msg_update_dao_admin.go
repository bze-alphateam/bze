package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// UpdateDaoAdmin nominates a new admin. The nominee must call AcceptDaoAdmin
// to actually complete the transfer. Until accepted, the current admin
// retains authority — pending_admin is informational only.
//
// Re-issuing UpdateDaoAdmin while a previous nomination is in flight simply
// overwrites pending_admin (the prior nominee can no longer accept).
func (k msgServer) UpdateDaoAdmin(goCtx context.Context, msg *types.MsgUpdateDaoAdmin) (*types.MsgUpdateDaoAdminResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, err := k.assertAdmin(ctx, msg.DaoId, msg.Authority)
	if err != nil {
		return nil, err
	}

	dao.PendingAdmin = msg.NewAdmin
	k.SetDao(ctx, dao)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateDaoAdmin,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", dao.Id)),
		sdk.NewAttribute(types.AttributeKeyAdmin, msg.Authority),
		sdk.NewAttribute(types.AttributeKeyNewAdmin, msg.NewAdmin),
	))

	return &types.MsgUpdateDaoAdminResponse{}, nil
}
