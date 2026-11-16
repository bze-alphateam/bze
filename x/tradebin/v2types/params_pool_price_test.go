package v2types_test

import (
	"testing"

	"cosmossdk.io/math"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	"github.com/stretchr/testify/require"
)

func TestDefaultPoolPriceParams(t *testing.T) {
	p := v2types.DefaultParams()
	require.Equal(t, uint64(20), p.PoolPriceHalfLifeBlocks)
	require.Equal(t, math.LegacyMustNewDecFromStr("0.1"), p.FeeMaxPriceDrift)
	require.Equal(t, math.LegacyMustNewDecFromStr("0.03"), p.ModuleMaxPriceDrift)
	require.NoError(t, p.Validate())
}

func TestPoolPriceParamsValidation(t *testing.T) {
	dec := math.LegacyMustNewDecFromStr
	for name, tc := range map[string]struct {
		mutate func(p *v2types.Params)
		valid  bool
	}{
		"half life 1":            {func(p *v2types.Params) { p.PoolPriceHalfLifeBlocks = 1 }, true},
		"half life max":          {func(p *v2types.Params) { p.PoolPriceHalfLifeBlocks = 100000 }, true},
		"half life zero":         {func(p *v2types.Params) { p.PoolPriceHalfLifeBlocks = 0 }, false},
		"half life above max":    {func(p *v2types.Params) { p.PoolPriceHalfLifeBlocks = 100001 }, false},
		"fee drift smallest":     {func(p *v2types.Params) { p.FeeMaxPriceDrift = dec("0.000000000000000001") }, true},
		"fee drift just below 1": {func(p *v2types.Params) { p.FeeMaxPriceDrift = dec("0.999999999999999999") }, true},
		"fee drift zero":         {func(p *v2types.Params) { p.FeeMaxPriceDrift = math.LegacyZeroDec() }, false},
		"fee drift one":          {func(p *v2types.Params) { p.FeeMaxPriceDrift = math.LegacyOneDec() }, false},
		"fee drift negative":     {func(p *v2types.Params) { p.FeeMaxPriceDrift = dec("-0.1") }, false},
		"fee drift unset":        {func(p *v2types.Params) { p.FeeMaxPriceDrift = math.LegacyDec{} }, false},
		"module drift 0.5":       {func(p *v2types.Params) { p.ModuleMaxPriceDrift = dec("0.5") }, true},
		"module drift zero":      {func(p *v2types.Params) { p.ModuleMaxPriceDrift = math.LegacyZeroDec() }, false},
		"module drift one":       {func(p *v2types.Params) { p.ModuleMaxPriceDrift = math.LegacyOneDec() }, false},
		"module drift above one": {func(p *v2types.Params) { p.ModuleMaxPriceDrift = dec("1.5") }, false},
		"module drift unset":     {func(p *v2types.Params) { p.ModuleMaxPriceDrift = math.LegacyDec{} }, false},
	} {
		t.Run(name, func(t *testing.T) {
			p := v2types.DefaultParams()
			tc.mutate(&p)
			if tc.valid {
				require.NoError(t, p.Validate())
			} else {
				require.Error(t, p.Validate())
			}
		})
	}
}
