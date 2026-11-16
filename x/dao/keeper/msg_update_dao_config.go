package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k msgServer) UpdateDaoConfig(goCtx context.Context, msg *types.MsgUpdateDaoConfig) (*types.MsgUpdateDaoConfigResponse, error) {
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

	params := k.GetParams(ctx)
	if err := msg.Config.ValidatePeriods(params.MinVotingPeriod, params.MaxVotingPeriod); err != nil {
		return nil, err
	}

	// open proposals keep their pinned_config: only the DAO record changes
	fields := dao.Config.ChangedFields(msg.Config)
	dao.Config = msg.Config
	k.setDao(ctx, dao)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventDaoConfigUpdated{Id: dao.Id, Fields: fields}); err != nil {
		return nil, err
	}

	return &types.MsgUpdateDaoConfigResponse{}, nil
}
