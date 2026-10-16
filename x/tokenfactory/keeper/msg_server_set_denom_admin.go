package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// SetDenomAdmin is the governance message that assigns a new admin to a factory denom, whatever its
// current admin is, including a renounced denom whose admin is empty.
func (k msgServer) SetDenomAdmin(goCtx context.Context, msg *types.MsgSetDenomAdmin) (*types.MsgSetDenomAdminResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	dAuth, err := k.Keeper.GetDenomAuthority(ctx, msg.Denom)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrDenomDoesNotExist, "no authority record for denom %s", msg.Denom)
	}

	previousAdmin := dAuth.GetAdmin()
	dAuth.Admin = msg.NewAdmin
	err = k.Keeper.SetDenomAuthority(ctx, msg.Denom, dAuth)
	if err != nil {
		return nil, err
	}

	err = ctx.EventManager().EmitTypedEvent(&types.DenomAdminChangeEvent{
		Admin:    previousAdmin,
		NewAdmin: msg.NewAdmin,
		Denom:    msg.Denom,
	})

	if err != nil {
		k.Logger().Error("failed to emit admin changed event", "error", err)
	}

	return &types.MsgSetDenomAdminResponse{}, nil
}
