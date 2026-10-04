package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// HaltDenoms is the governance message that halts denoms on the DEX. The native denom is refused before
// anything is written (halting it would halt every market and pool). Idempotent: a denom that is already
// halted stays halted and emits no event, so a batch proposal never fails because one of its entries was
// halted by an earlier proposal while this one was being voted.
func (k msgServer) HaltDenoms(goCtx context.Context, msg *types.MsgHaltDenoms) (*types.MsgHaltDenomsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	if err := msg.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	nativeDenom := k.getNativeDenom(ctx)
	for _, denom := range msg.Denoms {
		if denom == nativeDenom {
			return nil, errorsmod.Wrapf(types.ErrInvalidDenom, "the native denom %s cannot be halted", denom)
		}
	}

	for _, denom := range msg.Denoms {
		if k.IsDenomHalted(ctx, denom) {
			continue
		}

		k.SetHaltedDenom(ctx, denom)
		if err := ctx.EventManager().EmitTypedEvent(&types.DenomHaltedEvent{Denom: denom}); err != nil {
			k.Logger().Error(err.Error())
		}
	}

	return &types.MsgHaltDenomsResponse{}, nil
}

// UnhaltDenoms is the governance message that lifts the halt on denoms: resting orders resume matching
// and pools resume swapping, nothing else to do. Idempotent: a denom that is not halted is left as is and
// emits no event.
func (k msgServer) UnhaltDenoms(goCtx context.Context, msg *types.MsgUnhaltDenoms) (*types.MsgUnhaltDenomsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	if err := msg.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	for _, denom := range msg.Denoms {
		if !k.IsDenomHalted(ctx, denom) {
			continue
		}

		k.RemoveHaltedDenom(ctx, denom)
		if err := ctx.EventManager().EmitTypedEvent(&types.DenomUnhaltedEvent{Denom: denom}); err != nil {
			k.Logger().Error(err.Error())
		}
	}

	return &types.MsgUnhaltDenomsResponse{}, nil
}
