package v830

import (
	"errors"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
)

const (
	rateLimitWindowHours = 24

	// usdcNobleDenom is USDC from Noble (USDC.n) on BZE, reachable over channel-3 only.
	usdcNobleDenom = "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"
)

// RateLimitKeeper is the part of the ratelimit keeper the upgrade needs. The keeper
// does not check the message authority (the msg server does), so the handler fills
// it with the gov module address and calls the keeper directly.
type RateLimitKeeper interface {
	AddRateLimit(ctx sdk.Context, msg *ratelimittypes.MsgAddRateLimit) error
}

// InitialRateLimits is the mainnet IBC policy set by the v8.3.0 upgrade: 24h windows,
// percents of the denom's total supply on BZE. Authority is filled by the handler.
//
// ubze: 2% per channel per direction per day on every channel active in the chain
// registry. No other IBC voucher has a quota yet: their supply on BZE is tiny and any
// percentage would refuse a legitimate deposit; governance adds quotas later and the
// guardian can block a denom outright.
func InitialRateLimits() []ratelimittypes.MsgAddRateLimit {
	ubzeChannels := []string{
		"channel-0",  // Osmosis
		"channel-2",  // Archway
		"channel-3",  // Noble
		"channel-6",  // AtomOne
		"channel-10", // Lumen
		"channel-11", // Cosmos Hub
		"channel-12", // Dungeon
		"channel-13", // Injective
	}

	limits := make([]ratelimittypes.MsgAddRateLimit, 0, len(ubzeChannels)+1)
	for _, channel := range ubzeChannels {
		limits = append(limits, newRateLimit("ubze", channel, 2, 2))
	}

	// USDC.n: exits open, inbound capped at 0% of supply. The limit is on NET flow,
	// so inbound is refused except up to what already left in the same window; the
	// blocked_ibc_inbound filter still refuses all inbound.
	limits = append(limits, newRateLimit(usdcNobleDenom, "channel-3", 100, 0))

	return limits
}

func newRateLimit(denom, channel string, send, recv int64) ratelimittypes.MsgAddRateLimit {
	return ratelimittypes.MsgAddRateLimit{
		Denom:             denom,
		ChannelOrClientId: channel,
		MaxPercentSend:    sdkmath.NewInt(send),
		MaxPercentRecv:    sdkmath.NewInt(recv),
		DurationHours:     rateLimitWindowHours,
	}
}

// addInitialRateLimits adds the initial quotas. A path that does not exist on this
// network (channel missing, denom without supply, quota already set) is logged and
// skipped; any other error aborts the upgrade.
func addInitialRateLimits(ctx sdk.Context, keeper RateLimitKeeper) error {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	for _, limit := range InitialRateLimits() {
		limit.Authority = authority
		err := keeper.AddRateLimit(ctx, &limit)
		switch {
		case err == nil:
		case errors.Is(err, ratelimittypes.ErrChannelNotFound),
			errors.Is(err, ratelimittypes.ErrZeroChannelValue),
			errors.Is(err, ratelimittypes.ErrRateLimitAlreadyExists):
			ctx.Logger().Info("skipping initial rate limit", "denom", limit.Denom, "channel", limit.ChannelOrClientId, "reason", err.Error())
		default:
			return err
		}
	}

	return nil
}
