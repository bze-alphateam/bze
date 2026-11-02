package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
)

// RateLimitKeeper is the part of the ibc-apps rate-limiting keeper x/guardian
// uses. It is limited to reading a rate limit and storing it back: the keeper's
// UpdateRateLimit and ResetRateLimit reset the flow of the window, so they are
// not part of it on purpose.
type RateLimitKeeper interface {
	GetRateLimit(ctx sdk.Context, denom, channelId string) (ratelimittypes.RateLimit, bool)
	SetRateLimit(ctx sdk.Context, rateLimit ratelimittypes.RateLimit)
}
