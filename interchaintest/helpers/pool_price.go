package helpers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/testutil"
)

// ParseDec reads a LegacyDec as the CLI prints it. A field with the
// cosmos.Dec scalar comes out in decimal form ("0.030000000000000000"); a
// plain LegacyDec custom type comes out as its raw 18-decimal integer
// ("30000000000000000"). The decimal form always has a point.
func ParseDec(s string) (sdkmath.LegacyDec, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return sdkmath.LegacyDec{}, fmt.Errorf("empty decimal")
	}
	if strings.Contains(s, ".") {
		return sdkmath.LegacyNewDecFromStr(s)
	}
	i, ok := sdkmath.NewIntFromString(s)
	if !ok {
		return sdkmath.LegacyDec{}, fmt.Errorf("parse decimal %q", s)
	}
	return sdkmath.LegacyNewDecFromIntWithPrec(i, sdkmath.LegacyPrecision), nil
}

// PoolPriceDrifted is the chain's price guard rule: the price is further from
// its moving average than limit × average. Exactly at the limit is not
// drifted; a non-positive average counts as drifted.
func PoolPriceDrifted(price, avg, limit sdkmath.LegacyDec) bool {
	if avg.IsNil() || !avg.IsPositive() {
		return true
	}
	return price.Sub(avg).Abs().GT(limit.Mul(avg))
}

// PoolPriceAverage is `bzed q tradebin pool-price-average <pool-id>`: the
// moving average brought up to the query height, and the height its record
// was last written.
type PoolPriceAverage struct {
	AvgPrice   sdkmath.LegacyDec
	LastHeight int64
}

// QueryPoolPriceAverage reads the moving average of a pool's price.
func QueryPoolPriceAverage(ctx context.Context, chain *cosmos.CosmosChain, poolID string) (PoolPriceAverage, error) {
	return QueryPoolPriceAverageAt(ctx, chain, poolID, 0)
}

// atHeight appends --height to query args; 0 is the latest height.
func atHeight(height int64, args ...string) []string {
	if height > 0 {
		args = append(args, "--height", strconv.FormatInt(height, 10))
	}
	return args
}

// QueryPoolPriceAverageAt reads the moving average of a pool's price at a
// height (0: the latest), brought up to that height.
func QueryPoolPriceAverageAt(ctx context.Context, chain *cosmos.CosmosChain, poolID string, height int64) (PoolPriceAverage, error) {
	res, err := Query[struct {
		AvgPrice   string `json:"avg_price"`
		LastHeight Int64  `json:"last_height"`
	}](ctx, chain, atHeight(height, "tradebin", "pool-price-average", poolID)...)
	if err != nil {
		return PoolPriceAverage{}, err
	}
	avg, err := ParseDec(res.AvgPrice)
	if err != nil {
		return PoolPriceAverage{}, fmt.Errorf("pool %s average: %w", poolID, err)
	}
	return PoolPriceAverage{AvgPrice: avg, LastHeight: int64(res.LastHeight)}, nil
}

// QueryPoolPrice returns the price of a pool as the chain defines it: quote
// reserve / base reserve.
func QueryPoolPrice(ctx context.Context, chain *cosmos.CosmosChain, poolID string) (sdkmath.LegacyDec, error) {
	return QueryPoolPriceAt(ctx, chain, poolID, 0)
}

// QueryPoolPriceAt returns the price of a pool at a height (0: the latest).
func QueryPoolPriceAt(ctx context.Context, chain *cosmos.CosmosChain, poolID string, height int64) (sdkmath.LegacyDec, error) {
	res, err := Query[struct {
		Pool struct {
			ReserveBase  string `json:"reserve_base"`
			ReserveQuote string `json:"reserve_quote"`
		} `json:"pool"`
	}](ctx, chain, atHeight(height, "tradebin", "liquidity-pool", poolID)...)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}
	base, ok := sdkmath.NewIntFromString(res.Pool.ReserveBase)
	if !ok || !base.IsPositive() {
		return sdkmath.LegacyDec{}, fmt.Errorf("pool %s base reserve %q", poolID, res.Pool.ReserveBase)
	}
	quote, ok := sdkmath.NewIntFromString(res.Pool.ReserveQuote)
	if !ok || !quote.IsPositive() {
		return sdkmath.LegacyDec{}, fmt.Errorf("pool %s quote reserve %q", poolID, res.Pool.ReserveQuote)
	}
	return sdkmath.LegacyNewDecFromInt(quote).Quo(sdkmath.LegacyNewDecFromInt(base)), nil
}

// PriceDriftParams are the tradebin params of the pool price guard.
type PriceDriftParams struct {
	HalfLifeBlocks uint64
	FeeMaxDrift    sdkmath.LegacyDec
	ModuleMaxDrift sdkmath.LegacyDec
}

// QueryPriceDriftParams reads the pool price guard params.
func QueryPriceDriftParams(ctx context.Context, chain *cosmos.CosmosChain) (PriceDriftParams, error) {
	res, err := Query[struct {
		Params struct {
			HalfLife    Int64  `json:"poolPriceHalfLifeBlocks"`
			FeeDrift    string `json:"feeMaxPriceDrift"`
			ModuleDrift string `json:"moduleMaxPriceDrift"`
		} `json:"params"`
	}](ctx, chain, "tradebin", "params")
	if err != nil {
		return PriceDriftParams{}, err
	}
	fee, err := ParseDec(res.Params.FeeDrift)
	if err != nil {
		return PriceDriftParams{}, fmt.Errorf("feeMaxPriceDrift: %w", err)
	}
	module, err := ParseDec(res.Params.ModuleDrift)
	if err != nil {
		return PriceDriftParams{}, fmt.Errorf("moduleMaxPriceDrift: %w", err)
	}
	return PriceDriftParams{HalfLifeBlocks: uint64(res.Params.HalfLife), FeeMaxDrift: fee, ModuleMaxDrift: module}, nil
}

// PoolPriceState is a pool's price next to its moving average.
type PoolPriceState struct {
	Price, Avg sdkmath.LegacyDec
	Height     int64
}

// QueryPoolPriceState reads the price and the average of a pool.
func QueryPoolPriceState(ctx context.Context, chain *cosmos.CosmosChain, poolID string) (PoolPriceState, error) {
	price, err := QueryPoolPrice(ctx, chain, poolID)
	if err != nil {
		return PoolPriceState{}, err
	}
	avg, err := QueryPoolPriceAverage(ctx, chain, poolID)
	if err != nil {
		return PoolPriceState{}, err
	}
	return PoolPriceState{Price: price, Avg: avg.AvgPrice, Height: avg.LastHeight}, nil
}

// WaitForPoolPriceCalm waits, one block at a time and for at most maxBlocks
// blocks, until the price of each pool is within both drift params of its
// moving average, so the next protocol fee swap and module swap through
// those pools go ahead. The average only moves toward the price while the
// pool is left alone, so a calm pool stays calm until it is traded again.
func WaitForPoolPriceCalm(ctx context.Context, chain *cosmos.CosmosChain, maxBlocks int, poolIDs ...string) error {
	params, err := QueryPriceDriftParams(ctx, chain)
	if err != nil {
		return err
	}
	return WaitForPoolPriceWithin(ctx, chain, maxBlocks, sdkmath.LegacyMinDec(params.FeeMaxDrift, params.ModuleMaxDrift), poolIDs...)
}

// WaitForPoolPriceWithin waits, one block at a time and for at most
// maxBlocks blocks, until the price of each pool is within limit of its
// moving average.
func WaitForPoolPriceWithin(ctx context.Context, chain *cosmos.CosmosChain, maxBlocks int, limit sdkmath.LegacyDec, poolIDs ...string) error {
	for i := 0; ; i++ {
		calm := true
		var last string
		for _, id := range poolIDs {
			st, err := QueryPoolPriceState(ctx, chain, id)
			if err != nil {
				return err
			}
			if PoolPriceDrifted(st.Price, st.Avg, limit) {
				calm = false
				last = fmt.Sprintf("pool %s price %s average %s", id, st.Price, st.Avg)
				break
			}
		}
		if calm {
			return nil
		}
		if i >= maxBlocks {
			return fmt.Errorf("pool price still more than %s from its average after %d blocks: %s", limit, maxBlocks, last)
		}
		if err := testutil.WaitForBlocks(ctx, 1, chain); err != nil {
			return err
		}
	}
}

// WaitForPoolPriceDrifted waits until the price of a pool is further than
// limit from its moving average, at most maxBlocks blocks. A suite uses it
// after a large swap to make sure the guard is on before the next step.
func WaitForPoolPriceDrifted(ctx context.Context, chain *cosmos.CosmosChain, maxBlocks int, poolID string, limit sdkmath.LegacyDec) (PoolPriceState, error) {
	for i := 0; ; i++ {
		st, err := QueryPoolPriceState(ctx, chain, poolID)
		if err != nil {
			return st, err
		}
		if PoolPriceDrifted(st.Price, st.Avg, limit) {
			return st, nil
		}
		if i >= maxBlocks {
			return st, fmt.Errorf("pool %s price %s stayed within %s of its average %s", poolID, st.Price, limit, st.Avg)
		}
		if err := testutil.WaitForBlocks(ctx, 1, chain); err != nil {
			return st, err
		}
	}
}
