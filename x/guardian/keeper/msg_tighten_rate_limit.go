package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// TightenRateLimit lowers the quotas of an existing rate limit. The stored
// rate limit is written back with only the two percents changed: the window
// length, the path and the flow of the current window stay as they are. The
// ratelimit keeper's UpdateRateLimit is deliberately not used, it zeroes the
// flow, which would hand out a fresh window.
func (k msgServer) TightenRateLimit(goCtx context.Context, msg *types.MsgTightenRateLimit) (*types.MsgTightenRateLimitResponse, error) {
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

	current, found := k.rateLimitKeeper.GetRateLimit(ctx, msg.Denom, msg.ChannelId)
	if !found || current.Quota == nil {
		return nil, errorsmod.Wrapf(types.ErrRateLimitNotFound, "denom %s, channel %s", msg.Denom, msg.ChannelId)
	}

	if msg.MaxPercentSend.GT(current.Quota.MaxPercentSend) {
		return nil, errorsmod.Wrapf(types.ErrQuotaRaise, "max_percent_send %s is above the current %s", msg.MaxPercentSend, current.Quota.MaxPercentSend)
	}

	if msg.MaxPercentRecv.GT(current.Quota.MaxPercentRecv) {
		return nil, errorsmod.Wrapf(types.ErrQuotaRaise, "max_percent_recv %s is above the current %s", msg.MaxPercentRecv, current.Quota.MaxPercentRecv)
	}

	// copy the quota: the stored object is never edited in place
	quota := *current.Quota
	quota.MaxPercentSend = msg.MaxPercentSend
	quota.MaxPercentRecv = msg.MaxPercentRecv

	updated := current
	updated.Quota = &quota
	k.rateLimitKeeper.SetRateLimit(ctx, updated)

	if err := ctx.EventManager().EmitTypedEvent(&types.TightenRateLimitEvent{
		Signer:    msg.Signer,
		Denom:     msg.Denom,
		ChannelId: msg.ChannelId,
		OldSend:   current.Quota.MaxPercentSend.String(),
		OldRecv:   current.Quota.MaxPercentRecv.String(),
		NewSend:   msg.MaxPercentSend.String(),
		NewRecv:   msg.MaxPercentRecv.String(),
	}); err != nil {
		return nil, err
	}

	return &types.MsgTightenRateLimitResponse{}, nil
}

// authorize accepts the guardian and the module authority.
func (k Keeper) authorize(ctx sdk.Context, signer string) error {
	if k.isAuthority(signer) {
		return nil
	}

	if guardian := k.GetParams(ctx).Guardian; guardian != "" && sameAddress(signer, guardian) {
		return nil
	}

	return errorsmod.Wrapf(types.ErrUnauthorized, "%s", signer)
}

// isAuthority reports whether the address is the module authority.
func (k Keeper) isAuthority(address string) bool {
	return sameAddress(address, k.authority)
}

// sameAddress compares the account bytes behind two bech32 strings, not the
// strings. An address that does not decode is never equal to anything.
func sameAddress(a, b string) bool {
	left, err := sdk.AccAddressFromBech32(a)
	if err != nil {
		return false
	}

	right, err := sdk.AccAddressFromBech32(b)
	if err != nil {
		return false
	}

	return left.Equals(right)
}
