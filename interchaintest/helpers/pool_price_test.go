package helpers_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

func TestParseDec(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"0.030000000000000000", "0.03"},
		{"30000000000000000", "0.03"},
		{"100000000000000000", "0.1"},
		{"1000000000000000000", "1"},
		{"2.5", "2.5"},
		{" 0.5 ", "0.5"},
	} {
		got, err := helpers.ParseDec(c.in)
		require.NoError(t, err, c.in)
		require.True(t, sdkmath.LegacyMustNewDecFromStr(c.want).Equal(got), "%s: got %s", c.in, got)
	}
	for _, in := range []string{"", "abc", "1.2.3"} {
		_, err := helpers.ParseDec(in)
		require.Error(t, err, in)
	}
}

func TestPoolPriceDrifted(t *testing.T) {
	d := sdkmath.LegacyMustNewDecFromStr
	limit := d("0.1")
	for _, c := range []struct {
		price, avg string
		want       bool
	}{
		{"1", "1", false},
		{"1.1", "1", false}, // exactly at the limit
		{"0.9", "1", false}, // exactly at the limit, below
		{"1.11", "1", true}, // just over
		{"0.89", "1", true}, // just under
		{"5", "0", true},    // no average
		{"10.5", "10", false},
	} {
		require.Equal(t, c.want, helpers.PoolPriceDrifted(d(c.price), d(c.avg), limit), "price %s avg %s", c.price, c.avg)
	}
}
