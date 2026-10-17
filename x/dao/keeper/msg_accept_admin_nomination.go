package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// AcceptAdminNomination makes the pending nominee the DAO's admin. A DAO
// nominee signs through its own proposal or admin, which proves it can act.
// No live-electorate check: the new admin is external and can fix the
// electorate itself.
func (k msgServer) AcceptAdminNomination(goCtx context.Context, msg *types.MsgAcceptAdminNomination) (*types.MsgAcceptAdminNominationResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, found := k.GetDao(ctx, msg.DaoId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao %d", msg.DaoId)
	}

	if dao.PendingAdmin == "" {
		return nil, errorsmod.Wrapf(types.ErrNoPendingNomination, "dao %d", dao.Id)
	}

	nominee := sdk.MustAccAddressFromBech32(msg.Nominee)
	if nominee.String() != dao.PendingAdmin {
		return nil, errorsmod.Wrapf(types.ErrNotNominee, "dao %d nominated %s", dao.Id, dao.PendingAdmin)
	}

	oldAdmin := dao.Admin
	dao = k.setAdmin(ctx, dao, nominee)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventAdminChanged{
		Id:       dao.Id,
		OldAdmin: oldAdmin,
		NewAdmin: dao.Admin,
	}); err != nil {
		return nil, err
	}

	return &types.MsgAcceptAdminNominationResponse{}, nil
}
