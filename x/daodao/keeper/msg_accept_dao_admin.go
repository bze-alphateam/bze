package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// AcceptDaoAdmin completes a two-step admin handoff. Must be signed by the
// pending_admin recorded on the DAO. On success: admin = new_admin,
// pending_admin = "".
func (k msgServer) AcceptDaoAdmin(goCtx context.Context, msg *types.MsgAcceptDaoAdmin) (*types.MsgAcceptDaoAdminResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, found := k.GetDao(ctx, msg.DaoId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "id=%d", msg.DaoId)
	}

	if dao.PendingAdmin == "" {
		return nil, errorsmod.Wrap(types.ErrNoPendingAdmin, "no nomination in flight")
	}

	// Compare addresses by parsed bech32 to be tolerant of formatting nits.
	pending, err := sdk.AccAddressFromBech32(dao.PendingAdmin)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "stored pending_admin: %s", err.Error())
	}
	signer, err := sdk.AccAddressFromBech32(msg.NewAdmin)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "new_admin: %s", err.Error())
	}
	if !signer.Equals(pending) {
		return nil, errorsmod.Wrapf(types.ErrPendingAdminMismatch,
			"signer %s does not match pending_admin %s", msg.NewAdmin, dao.PendingAdmin)
	}

	dao.Admin = msg.NewAdmin
	dao.PendingAdmin = ""
	k.SetDao(ctx, dao)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeAcceptDaoAdmin,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", dao.Id)),
		sdk.NewAttribute(types.AttributeKeyAdmin, msg.NewAdmin),
	))

	return &types.MsgAcceptDaoAdminResponse{}, nil
}
