package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/guardian/types"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
)

// HaltTradingDenoms halts denoms on the DEX with the same effect as
// tradebin's governance message MsgHaltDenoms: the same validation, the same
// store writes and the same per-denom DenomHaltedEvent. The native denom is
// refused before anything is written (halting it would halt every market and
// pool). Idempotent: a denom that is already halted stays halted and emits no
// tradebin event.
//
// The guardian or the governance authority can sign it. Lifting a halt is
// left to governance, through tradebin's MsgUnhaltDenoms.
func (k msgServer) HaltTradingDenoms(goCtx context.Context, msg *types.MsgHaltTradingDenoms) (*types.MsgHaltTradingDenomsResponse, error) {
	// the signer first, as tradebin's MsgHaltDenoms handler does: any other
	// account is refused as unauthorized whatever the message carries
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.authorize(ctx, msg.Signer); err != nil {
		return nil, err
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.tradebinKeeper == nil {
		return nil, types.ErrTradebinKeeperNotSet
	}

	nativeDenom := k.tradebinKeeper.GetParams(ctx).NativeDenom
	for _, denom := range msg.Denoms {
		if denom == nativeDenom {
			return nil, errorsmod.Wrapf(types.ErrInvalidDenom, "the native denom %s cannot be halted", denom)
		}
	}

	halted := make([]string, 0, len(msg.Denoms))
	for _, denom := range msg.Denoms {
		if k.tradebinKeeper.IsDenomHalted(ctx, denom) {
			continue
		}

		k.tradebinKeeper.SetHaltedDenom(ctx, denom)
		halted = append(halted, denom)
		if err := ctx.EventManager().EmitTypedEvent(&tradebintypes.DenomHaltedEvent{Denom: denom}); err != nil {
			k.Logger().Error(err.Error())
		}
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.TradingDenomsHaltedEvent{
		Signer: msg.Signer,
		Denoms: halted,
	}); err != nil {
		return nil, err
	}

	return &types.MsgHaltTradingDenomsResponse{}, nil
}
