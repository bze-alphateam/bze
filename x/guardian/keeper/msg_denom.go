package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// BlockDenom puts the denom on the rate limit module's blacklist. The module
// refuses every transfer of a blacklisted denom, sent or received, on every
// channel, whether or not a rate limit exists for it. Existing rate limits are
// not touched.
func (k msgServer) BlockDenom(goCtx context.Context, msg *types.MsgBlockDenom) (*types.MsgBlockDenomResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.authorize(ctx, msg.Signer); err != nil {
		return nil, err
	}

	if k.rateLimitKeeper == nil {
		return nil, types.ErrRateLimitKeeperNotSet
	}

	if k.rateLimitKeeper.IsDenomBlacklisted(ctx, msg.Denom) {
		return nil, errorsmod.Wrapf(types.ErrAlreadyBlocked, "denom %s", msg.Denom)
	}

	k.rateLimitKeeper.AddDenomToBlacklist(ctx, msg.Denom)

	if err := ctx.EventManager().EmitTypedEvent(&types.DenomBlockedEvent{
		Signer: msg.Signer,
		Denom:  msg.Denom,
	}); err != nil {
		return nil, err
	}

	return &types.MsgBlockDenomResponse{}, nil
}

// UnblockDenom takes the denom off the blacklist. Only the governance
// authority can sign it: the guardian can stop a denom, never reopen it.
func (k msgServer) UnblockDenom(goCtx context.Context, msg *types.MsgUnblockDenom) (*types.MsgUnblockDenomResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if !k.isAuthority(msg.Authority) {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "only the governance authority can unblock, got %s", msg.Authority)
	}

	if k.rateLimitKeeper == nil {
		return nil, types.ErrRateLimitKeeperNotSet
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if !k.rateLimitKeeper.IsDenomBlacklisted(ctx, msg.Denom) {
		return nil, errorsmod.Wrapf(types.ErrNotBlocked, "denom %s", msg.Denom)
	}

	k.rateLimitKeeper.RemoveDenomFromBlacklist(ctx, msg.Denom)

	if err := ctx.EventManager().EmitTypedEvent(&types.DenomUnblockedEvent{
		Authority: msg.Authority,
		Denom:     msg.Denom,
	}); err != nil {
		return nil, err
	}

	return &types.MsgUnblockDenomResponse{}, nil
}
