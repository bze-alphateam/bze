package types

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
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

// UpgradeKeeper is the part of the x/upgrade keeper x/guardian uses: it only
// schedules the plan that halts the chain.
type UpgradeKeeper interface {
	ScheduleUpgrade(ctx context.Context, plan upgradetypes.Plan) error
}
