package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// UpdateDaoMetadata replaces the DAO's metadata. Admin-gated.
func (k msgServer) UpdateDaoMetadata(goCtx context.Context, msg *types.MsgUpdateDaoMetadata) (*types.MsgUpdateDaoMetadataResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, err := k.assertAdmin(ctx, msg.DaoId, msg.Authority)
	if err != nil {
		return nil, err
	}

	dao.Metadata = msg.Metadata
	k.SetDao(ctx, dao)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateDaoMetadata,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", dao.Id)),
		sdk.NewAttribute(types.AttributeKeyAdmin, msg.Authority),
	))

	return &types.MsgUpdateDaoMetadataResponse{}, nil
}
