package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k msgServer) UpdateMetadata(goCtx context.Context, msg *types.MsgUpdateMetadata) (*types.MsgUpdateMetadataResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, found := k.GetDao(ctx, msg.DaoId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao %d", msg.DaoId)
	}

	// only the DAO account: its voters through a proposal, or its admin
	// through MsgAdminExecute, never the admin signing directly
	authority := sdk.MustAccAddressFromBech32(msg.Authority)
	if authority.String() != dao.Address {
		return nil, errorsmod.Wrapf(types.ErrUnauthorizedSigner, "expected dao %d's address %s, got %s", dao.Id, dao.Address, authority)
	}

	dao.Name = msg.Name
	dao.Description = msg.Description
	k.setDao(ctx, dao)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventDaoMetadataUpdated{Id: dao.Id}); err != nil {
		return nil, err
	}

	return &types.MsgUpdateMetadataResponse{}, nil
}
