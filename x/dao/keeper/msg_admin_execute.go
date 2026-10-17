package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k msgServer) AdminExecute(goCtx context.Context, msg *types.MsgAdminExecute) (*types.MsgAdminExecuteResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, found := k.GetDao(ctx, msg.DaoId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao %d", msg.DaoId)
	}

	// a self-owned DAO executes through its proposals; refusing it here also
	// keeps a nested MsgAdminExecute (signed by the DAO) from passing
	if dao.Admin == dao.Address {
		return nil, errorsmod.Wrapf(types.ErrNotDaoAdmin, "dao %d is self-owned", dao.Id)
	}

	admin := sdk.MustAccAddressFromBech32(msg.Admin)
	if admin.String() != dao.Admin {
		return nil, errorsmod.Wrapf(types.ErrNotDaoAdmin, "dao %d is administered by %s", dao.Id, dao.Admin)
	}

	params := k.GetParams(ctx)
	if uint64(len(msg.Msgs)) > params.MaxMsgsPerProposal {
		return nil, errorsmod.Wrapf(types.ErrInvalidMsgs, "at most %d messages, got %d", params.MaxMsgsPerProposal, len(msg.Msgs))
	}

	size := 0
	for _, m := range msg.Msgs {
		size += m.Size()
	}
	if uint64(size) > params.MaxProposalBytes {
		return nil, errorsmod.Wrapf(types.ErrInvalidMsgs, "messages take %d bytes, at most %d allowed", size, params.MaxProposalBytes)
	}

	msgs, err := msg.UnpackMsgs()
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidMsgs, err.Error())
	}

	// on ctx itself: any failure reverts the whole transaction
	if err := k.executeAsDao(ctx, dao, msgs); err != nil {
		return nil, err
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.EventAdminExecuted{
		Id:       dao.Id,
		Admin:    dao.Admin,
		MsgCount: uint64(len(msgs)),
	}); err != nil {
		return nil, err
	}

	return &types.MsgAdminExecuteResponse{}, nil
}
