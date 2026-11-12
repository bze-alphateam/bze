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
// electorate itself. The nomination is void once any DAO on the admin chain
// above changed its admin after it was made: whoever controlled the DAO then
// may no longer control it.
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

	if err := k.checkNominationControl(ctx, dao); err != nil {
		return nil, err
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

// checkNominationControl refuses the DAO's pending nomination when a DAO on its
// admin chain changed its admin after the nomination was made. The DAO's own
// admin change clears the nomination already; the walk covers the DAOs that
// control it through their admin, up to maxControllerDepth, which is above the
// deepest chain a nomination can be signed through (MaxDaoExecutionDepth).
func (k msgServer) checkNominationControl(ctx sdk.Context, dao types.Dao) error {
	chain, _, _, err := k.adminChain(ctx, dao)
	if err != nil {
		return err
	}

	for _, link := range chain {
		if link.AdminSeq > dao.PendingAdminSeq {
			return errorsmod.Wrapf(types.ErrStaleNomination, "dao %d changed its admin after dao %d nominated %s", link.Id, dao.Id, dao.PendingAdmin)
		}
	}

	return nil
}
