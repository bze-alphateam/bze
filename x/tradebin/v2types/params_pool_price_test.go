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
		err    string
	}{
		"half life 1":            {func(p *v2types.Params) { p.PoolPriceHalfLifeBlocks = 1 }, ""},
		"half life max":          {func(p *v2types.Params) { p.PoolPriceHalfLifeBlocks = 100000 }, ""},
		"half life zero":         {func(p *v2types.Params) { p.PoolPriceHalfLifeBlocks = 0 }, "pool price half life must be between 1 and 100000 blocks"},
		"half life above max":    {func(p *v2types.Params) { p.PoolPriceHalfLifeBlocks = 100001 }, "pool price half life must be between 1 and 100000 blocks"},
		"fee drift smallest":     {func(p *v2types.Params) { p.FeeMaxPriceDrift = dec("0.000000000000000001") }, ""},
		"fee drift just below 1": {func(p *v2types.Params) { p.FeeMaxPriceDrift = dec("0.999999999999999999") }, ""},
		"fee drift zero":         {func(p *v2types.Params) { p.FeeMaxPriceDrift = math.LegacyZeroDec() }, "invalid FeeMaxPriceDrift: drift must be greater than 0 and less than 1"},
		"fee drift one":          {func(p *v2types.Params) { p.FeeMaxPriceDrift = math.LegacyOneDec() }, "invalid FeeMaxPriceDrift: drift must be greater than 0 and less than 1"},
		"fee drift negative":     {func(p *v2types.Params) { p.FeeMaxPriceDrift = dec("-0.1") }, "invalid FeeMaxPriceDrift: drift must be greater than 0 and less than 1"},
		"fee drift unset":        {func(p *v2types.Params) { p.FeeMaxPriceDrift = math.LegacyDec{} }, "invalid FeeMaxPriceDrift: drift must be set"},
		"module drift 0.5":       {func(p *v2types.Params) { p.ModuleMaxPriceDrift = dec("0.5") }, ""},
		"module drift zero":      {func(p *v2types.Params) { p.ModuleMaxPriceDrift = math.LegacyZeroDec() }, "invalid ModuleMaxPriceDrift: drift must be greater than 0 and less than 1"},
		"module drift one":       {func(p *v2types.Params) { p.ModuleMaxPriceDrift = math.LegacyOneDec() }, "invalid ModuleMaxPriceDrift: drift must be greater than 0 and less than 1"},
		"module drift above one": {func(p *v2types.Params) { p.ModuleMaxPriceDrift = dec("1.5") }, "invalid ModuleMaxPriceDrift: drift must be greater than 0 and less than 1"},
		"module drift unset":     {func(p *v2types.Params) { p.ModuleMaxPriceDrift = math.LegacyDec{} }, "invalid ModuleMaxPriceDrift: drift must be set"},
	} {
		t.Run(name, func(t *testing.T) {
			p := v2types.DefaultParams()
			tc.mutate(&p)
			if tc.err == "" {
				require.NoError(t, p.Validate())
			} else {
				require.EqualError(t, p.Validate(), tc.err)
			}
		})
	}
}
