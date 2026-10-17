package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// WithdrawAdminNomination cancels a pending handover before the nominee
// accepts it. The admin stays in charge and can nominate again.
func (k msgServer) WithdrawAdminNomination(goCtx context.Context, msg *types.MsgWithdrawAdminNomination) (*types.MsgWithdrawAdminNominationResponse, error) {
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

	if dao.PendingAdmin == "" {
		return nil, errorsmod.Wrapf(types.ErrNoPendingNomination, "dao %d", dao.Id)
	}

	nominee := dao.PendingAdmin
	dao.PendingAdmin = ""
	k.setDao(ctx, dao)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventAdminNominationWithdrawn{
		Id:      dao.Id,
		Nominee: nominee,
	}); err != nil {
		return nil, err
	}

	return &types.MsgWithdrawAdminNominationResponse{}, nil
}
