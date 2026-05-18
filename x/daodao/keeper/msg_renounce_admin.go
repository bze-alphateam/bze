package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// RenounceAdmin implements MsgRenounceAdmin.
//
// Order of operations:
//  1. ValidateBasic (signer + dao_id).
//  2. assertAdmin — only the current admin may renounce.
//  3. Reject if the DAO is already self-administered.
//  4. Flip admin to dao.account_address; clear any pending nomination
//     (a renounce while a handoff is in flight cancels the handoff —
//     simpler than juggling two terminal states).
//  5. Emit event.
//
// After this, direct admin-gated msgs from the previous human admin
// fail (signer no longer matches dao.admin). Proposals — which dispatch
// AS the DAO — continue to work because their signer equals
// dao.account_address == dao.admin.
func (k msgServer) RenounceAdmin(goCtx context.Context, msg *types.MsgRenounceAdmin) (*types.MsgRenounceAdminResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, err := k.assertAdmin(ctx, msg.DaoId, msg.Authority)
	if err != nil {
		return nil, err
	}

	if dao.Admin == dao.AccountAddress {
		return nil, errorsmod.Wrapf(types.ErrAlreadySelfAdmin,
			"dao %d admin is already its own account", dao.Id)
	}

	previousAdmin := dao.Admin
	dao.Admin = dao.AccountAddress
	dao.PendingAdmin = ""
	k.SetDao(ctx, dao)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRenounceAdmin,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", dao.Id)),
		sdk.NewAttribute(types.AttributeKeyAdmin, previousAdmin),
		sdk.NewAttribute(types.AttributeKeyAccountAddress, dao.AccountAddress),
	))

	return &types.MsgRenounceAdminResponse{}, nil
}
