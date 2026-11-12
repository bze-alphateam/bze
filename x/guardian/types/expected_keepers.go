package types

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"

	tradebinv2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

// RateLimitKeeper is the part of the ibc-apps rate-limiting keeper x/guardian
// uses. It is limited to reading a rate limit and storing it back: the keeper's
// UpdateRateLimit and ResetRateLimit reset the flow of the window, so they are
// not part of it on purpose. The blacklist functions are the module's only
// entry points to its blacklist (it has no message for it).
type RateLimitKeeper interface {
	GetRateLimit(ctx sdk.Context, denom, channelId string) (ratelimittypes.RateLimit, bool)
	SetRateLimit(ctx sdk.Context, rateLimit ratelimittypes.RateLimit)
	AddDenomToBlacklist(ctx sdk.Context, denom string)
	RemoveDenomFromBlacklist(ctx sdk.Context, denom string)
	IsDenomBlacklisted(ctx sdk.Context, denom string) bool
	GetAllBlacklistedDenoms(ctx sdk.Context) []string
}

// BankKeeper is the part of the bank keeper x/guardian uses: it only checks
// that a denom exists on chain before blocking it.
type BankKeeper interface {
	HasSupply(ctx context.Context, denom string) bool
}

// CircuitKeeper is the part of the x/circuit keeper x/guardian uses: it reads
// the list of disabled message types and adds to it. It has no way to take a
// type off the list: only governance re-enables a message, through x/circuit's
// own MsgResetCircuitBreaker.
type CircuitKeeper interface {
	IsDisabled(ctx context.Context, msgTypeURL string) (bool, error)
	Disable(ctx context.Context, msgTypeURL string) error
}

// MsgRouter tells whether a type URL is a message the app routes to a Msg
// service, so only real message types can be disabled.
type MsgRouter interface {
	IsRegistered(msgTypeURL string) bool
}

// TradebinKeeper is the part of the tradebin keeper x/guardian uses to halt
// denoms on the DEX: the same store functions tradebin's own MsgHaltDenoms
// handler calls, and its params for the native denom. Lifting a halt is not
// part of it on purpose: only governance does that, through tradebin.
type TradebinKeeper interface {
	GetParams(ctx context.Context) tradebinv2types.Params
	IsDenomHalted(ctx sdk.Context, denom string) bool
	SetHaltedDenom(ctx sdk.Context, denom string)
}

// UpgradeKeeper is the part of the x/upgrade keeper x/guardian uses: it only
// schedules the plan that halts the chain.
type UpgradeKeeper interface {
	ScheduleUpgrade(ctx context.Context, plan upgradetypes.Plan) error
}

// ChannelKeeper is the part of the IBC channel keeper x/guardian uses: it only
// checks that a transfer channel exists before closing it.
type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, portID, channelID string) (channeltypes.Channel, bool)
}
