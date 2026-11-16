package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// DisableMinting turns minting off for good on a factory denom. Admin only; no message turns it back on.
func (k msgServer) DisableMinting(goCtx context.Context, msg *types.MsgDisableMinting) (*types.MsgDisableMintingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dAuth, err := k.GetDenomAuthority(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	if msg.Creator != dAuth.GetAdmin() {
		return nil, types.ErrUnauthorized
	}

	if !dAuth.GetMintable() {
		return nil, types.ErrMintingDisabled.Wrapf("denom: %s", msg.Denom)
	}

	dAuth.Mintable = false
	err = k.SetDenomAuthority(ctx, msg.Denom, dAuth)
	if err != nil {
		return nil, err
	}

	k.emitMintingDisabledEvent(ctx, msg.Denom, msg.Creator)

	return &types.MsgDisableMintingResponse{}, nil
}

func (k Keeper) emitMintingDisabledEvent(ctx sdk.Context, denom, admin string) {
	err := ctx.EventManager().EmitTypedEvent(&types.DenomMintingDisabledEvent{
		Denom: denom,
		Admin: admin,
	})

	if err != nil {
		k.Logger().Error("failed to emit minting disabled event", "error", err)
	}
}
