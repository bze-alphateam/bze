package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k msgServer) NominateAdmin(goCtx context.Context, msg *types.MsgNominateAdmin) (*types.MsgNominateAdminResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, found := k.GetDao(ctx, msg.DaoId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao %d", msg.DaoId)
	}

	// a person signs directly, a self-owned DAO through its proposals, a parent
	// DAO through its own proposals or admin
	authority := sdk.MustAccAddressFromBech32(msg.Authority)
	if authority.String() != dao.Admin {
		return nil, errorsmod.Wrapf(types.ErrNotDaoAdmin, "dao %d is administered by %s", dao.Id, dao.Admin)
	}

	if dao.PendingAdmin != "" {
		return nil, errorsmod.Wrapf(types.ErrNominationPending, "dao %d already nominated %s", dao.Id, dao.PendingAdmin)
	}

	if msg.Nominee == "" {
		if err := k.makeSelfOwned(ctx, dao); err != nil {
			return nil, err
		}

		return &types.MsgNominateAdminResponse{}, nil
	}

	nominee := sdk.MustAccAddressFromBech32(msg.Nominee).String()
	if nominee == dao.Admin {
		return nil, errorsmod.Wrapf(types.ErrInvalidNominee, "%s is already the admin of dao %d", nominee, dao.Id)
	}

	// handing the DAO to itself has no accept step: it goes through an empty nominee
	if nominee == dao.Address {
		return nil, errorsmod.Wrapf(types.ErrInvalidNominee, "the nominee is dao %d itself: nominate nobody to make it self-owned", dao.Id)
	}

	dao.PendingAdmin = nominee
	k.setDao(ctx, dao)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventAdminNominated{
		Id:      dao.Id,
		Admin:   dao.Admin,
		Nominee: nominee,
	}); err != nil {
		return nil, err
	}

	return &types.MsgNominateAdminResponse{}, nil
}

// makeSelfOwned hands the DAO to its own voters at once, which needs voters
// able to pass proposals (D11).
func (k msgServer) makeSelfOwned(ctx sdk.Context, dao types.Dao) error {
	if dao.Admin == dao.Address {
		return errorsmod.Wrapf(types.ErrInvalidNominee, "dao %d is already self-owned", dao.Id)
	}

	if !k.isLiveElectorate(ctx, dao) {
		return errorsmod.Wrapf(types.ErrElectorateNotLive, "dao %d has no voters to hand it to", dao.Id)
	}

	oldAdmin := dao.Admin
	dao = k.setAdmin(ctx, dao, sdk.MustAccAddressFromBech32(dao.Address))

	return ctx.EventManager().EmitTypedEvent(&types.EventAdminChanged{
		Id:       dao.Id,
		OldAdmin: oldAdmin,
		NewAdmin: dao.Admin,
	})
}
