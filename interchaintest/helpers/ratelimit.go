package helpers

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
)

// RateLimit is an IBC rate limit as `q ratelimit rate-limit <channel> --denom <denom>`
// prints it: the path, the quota in percent of the channel value and the flow
// of the current window.
type RateLimit struct {
	Path struct {
		Denom             string `json:"denom"`
		ChannelOrClientID string `json:"channel_or_client_id"`
	} `json:"path"`
	Quota struct {
		MaxPercentSend sdkmath.Int `json:"max_percent_send"`
		MaxPercentRecv sdkmath.Int `json:"max_percent_recv"`
		DurationHours  Int64       `json:"duration_hours"`
	} `json:"quota"`
	Flow struct {
		Inflow       sdkmath.Int `json:"inflow"`
		Outflow      sdkmath.Int `json:"outflow"`
		ChannelValue sdkmath.Int `json:"channel_value"`
	} `json:"flow"`
}

// Threshold returns the most the net flow of a window may reach for a quota of
// percent of the channel value: value × percent / 100, integer division, as
// the rate limit module computes it. A flow above it is refused, a flow equal
// to it is accepted.
func Threshold(value sdkmath.Int, percent int64) sdkmath.Int {
	return value.MulRaw(percent).QuoRaw(100)
}

// NetOutflow returns outflow − inflow, the amount the send quota is checked
// against (negative when more came in than went out).
func (r RateLimit) NetOutflow() sdkmath.Int {
	return intOrZero(r.Flow.Outflow).Sub(intOrZero(r.Flow.Inflow))
}

// FlowIs reports whether the window's inflow and outflow are the given values.
func (r RateLimit) FlowIs(inflow, outflow sdkmath.Int) bool {
	return intOrZero(r.Flow.Inflow).Equal(inflow) && intOrZero(r.Flow.Outflow).Equal(outflow)
}

func intOrZero(i sdkmath.Int) sdkmath.Int {
	if i.IsNil() {
		return sdkmath.ZeroInt()
	}
	return i
}

// QueryRateLimit returns the rate limit of (denom, channel). It is an error
// when there is none.
func QueryRateLimit(ctx context.Context, chain *cosmos.CosmosChain, channel, denom string) (RateLimit, error) {
	return Query[RateLimit](ctx, chain, "ratelimit", "rate-limit", channel, "--denom", denom)
}

// QueryRateLimits returns every rate limit.
func QueryRateLimits(ctx context.Context, chain *cosmos.CosmosChain) ([]RateLimit, error) {
	res, err := Query[struct {
		RateLimits []RateLimit `json:"rate_limits"`
	}](ctx, chain, "ratelimit", "list-rate-limits")
	return res.RateLimits, err
}

// QueryBlockedDenoms returns the denoms on the rate limit blacklist, through
// x/guardian's passthrough query.
func QueryBlockedDenoms(ctx context.Context, chain *cosmos.CosmosChain) ([]string, error) {
	res, err := Query[struct {
		Denoms []string `json:"denoms"`
	}](ctx, chain, "guardian", "blocked-denoms")
	return res.Denoms, err
}

// QueryGuardian returns the guardian address of x/guardian ("" when unset).
func QueryGuardian(ctx context.Context, chain *cosmos.CosmosChain) (string, error) {
	res, err := Query[struct {
		Params struct {
			Guardian string `json:"guardian"`
		} `json:"params"`
	}](ctx, chain, "guardian", "params")
	return res.Params.Guardian, err
}

// TrySendIbc sends coin like SendIbc but with a fixed gas limit, so a message
// the chain refuses (an exceeded quota, a blocked denom) is broadcast, included
// and charged its fee, and comes back as a TxResult with its code and
// codespace instead of failing the simulation. Unlike SendIbc it does not
// require the tx to be accepted nor a send_packet event.
func TrySendIbc(ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, channel, receiver string, coin sdk.Coin, gas uint64, flags ...string) (TxResult, error) {
	args := append([]string{"ibc-transfer", "transfer", transferPort, channel, receiver, coin.String()}, flags...)
	return ExecTxWith(ctx, chain, user, []TxOption{WithGas(gas)}, args...)
}

// FixedGasFee returns the fee of a tx sent with WithGas(gas) at the chain's
// gas price, rounded up as the CLI does.
func FixedGasFee(chain *cosmos.CosmosChain, gas uint64) (sdkmath.Int, error) {
	prices, err := sdk.ParseDecCoins(chain.Config().GasPrices)
	if err != nil {
		return sdkmath.Int{}, fmt.Errorf("parse gas prices %q: %w", chain.Config().GasPrices, err)
	}
	if len(prices) != 1 {
		return sdkmath.Int{}, fmt.Errorf("want one gas price, got %q", chain.Config().GasPrices)
	}
	return prices[0].Amount.MulInt64(int64(gas)).Ceil().TruncateInt(), nil //nolint:gosec // test gas limits are small
}
