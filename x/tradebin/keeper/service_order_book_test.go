package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/stretchr/testify/require"
)

func TestCalculateMinAmountFromPriceDec(t *testing.T) {
	tests := []struct {
		name     string
		price    math.LegacyDec
		expected math.Int
	}{
		{
			name:     "price = 1 -> ceil(1/1)*2 = 2",
			price:    math.LegacyNewDec(1),
			expected: math.NewInt(2),
		},
		{
			name:     "price = 2 -> ceil(1/2)*2 = 2",
			price:    math.LegacyNewDec(2),
			expected: math.NewInt(2),
		},
		{
			name:     "price = 10 -> ceil(1/10)*2 = 2",
			price:    math.LegacyNewDec(10),
			expected: math.NewInt(2),
		},
		{
			name:     "price = 100 -> ceil(1/100)*2 = 2",
			price:    math.LegacyNewDec(100),
			expected: math.NewInt(2),
		},
		{
			name:     "price = 0.1 -> ceil(1/0.1)*2 = 20",
			price:    math.LegacyMustNewDecFromStr("0.1"),
			expected: math.NewInt(20),
		},
		{
			name:     "price = 0.01 -> ceil(1/0.01)*2 = 200",
			price:    math.LegacyMustNewDecFromStr("0.01"),
			expected: math.NewInt(200),
		},
		{
			name:     "price = 0.001 -> ceil(1/0.001)*2 = 2000",
			price:    math.LegacyMustNewDecFromStr("0.001"),
			expected: math.NewInt(2000),
		},
		{
			name:     "price = 0.5 -> ceil(1/0.5)*2 = 4",
			price:    math.LegacyMustNewDecFromStr("0.5"),
			expected: math.NewInt(4),
		},
		{
			name:     "price = 0.3 -> ceil(1/0.3)*2 = 8",
			price:    math.LegacyMustNewDecFromStr("0.3"),
			expected: math.NewInt(8),
		},
		{
			name:     "price = 0.7 -> ceil(1/0.7)*2 = 4",
			price:    math.LegacyMustNewDecFromStr("0.7"),
			expected: math.NewInt(4),
		},
		{
			name:     "price = 3 -> ceil(1/3)*2 = 2",
			price:    math.LegacyNewDec(3),
			expected: math.NewInt(2),
		},
		{
			name:     "price = 1.5 -> ceil(1/1.5)*2 = 2",
			price:    math.LegacyMustNewDecFromStr("1.5"),
			expected: math.NewInt(2),
		},
		{
			name:     "price = 0.0001 -> ceil(1/0.0001)*2 = 20000",
			price:    math.LegacyMustNewDecFromStr("0.0001"),
			expected: math.NewInt(20000),
		},
		{
			name:     "very small price = 0.000000001 -> ceil(1/0.000000001)*2 = 2000000000",
			price:    math.LegacyMustNewDecFromStr("0.000000001"),
			expected: math.NewInt(2000000000),
		},
		{
			name:     "very large price = 1000000 -> ceil(1/1000000)*2 = 2",
			price:    math.LegacyNewDec(1000000),
			expected: math.NewInt(2),
		},
		{
			name:     "fractional result: price = 0.6 -> ceil(1/0.6)*2 = 4",
			price:    math.LegacyMustNewDecFromStr("0.6"),
			expected: math.NewInt(4),
		},
		{
			name:     "fractional result: price = 0.9 -> ceil(1/0.9)*2 = 4",
			price:    math.LegacyMustNewDecFromStr("0.9"),
			expected: math.NewInt(4),
		},
		{
			name:     "fractional result: price = 7 -> ceil(1/7)*2 = 2",
			price:    math.LegacyNewDec(7),
			expected: math.NewInt(2),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := keeper.CalculateMinAmountFromPriceDec(tc.price)
			require.NoError(t, err)
			require.True(t, tc.expected.Equal(result), "expected %s, got %s", tc.expected.String(), result.String())
		})
	}
}

func TestCalculateMinAmountFromPriceDec_MatchesStringVersion(t *testing.T) {
	// Verify that CalculateMinAmountFromPriceDec produces the same results
	// as the deprecated CalculateMinAmount for equivalent inputs
	prices := []string{"1", "2", "0.1", "0.01", "0.5", "0.3", "10", "100", "0.001", "1.5", "7", "0.0001"}

	for _, priceStr := range prices {
		t.Run("price="+priceStr, func(t *testing.T) {
			fromString, err := keeper.CalculateMinAmount(priceStr)
			require.NoError(t, err)

			priceDec := math.LegacyMustNewDecFromStr(priceStr)
			fromDec, err := keeper.CalculateMinAmountFromPriceDec(priceDec)
			require.NoError(t, err)

			require.True(t, fromString.Equal(fromDec), "mismatch for price %s: string=%s, dec=%s", priceStr, fromString.String(), fromDec.String())
		})
	}
}

func TestCalculateMinAmountFromPriceDec_ZeroPrice(t *testing.T) {
	// Zero price should panic due to division by zero in Quo
	require.Panics(t, func() {
		_, _ = keeper.CalculateMinAmountFromPriceDec(math.LegacyZeroDec())
	})
}

func TestCalculateMinAmountFromPriceDec_NegativePrice(t *testing.T) {
	// Negative price produces a negative result (caller should validate before calling)
	result, err := keeper.CalculateMinAmountFromPriceDec(math.LegacyNewDec(-1))
	require.NoError(t, err)
	require.True(t, result.IsNegative(), "expected negative result for negative price, got %s", result.String())
}

func TestCalculateMinAmountFromPriceDec_AlwaysAtLeastTwo(t *testing.T) {
	// For any positive price, the minimum amount should always be >= 2
	// because ceil(1/price) >= 1 for any positive price, and we multiply by 2
	prices := []math.LegacyDec{
		math.LegacyMustNewDecFromStr("0.000001"),
		math.LegacyMustNewDecFromStr("0.01"),
		math.LegacyMustNewDecFromStr("0.5"),
		math.LegacyNewDec(1),
		math.LegacyNewDec(100),
		math.LegacyNewDec(999999),
	}

	for _, price := range prices {
		result, err := keeper.CalculateMinAmountFromPriceDec(price)
		require.NoError(t, err)
		require.True(t, result.GTE(math.NewInt(2)), "expected >= 2 for price %s, got %s", price.String(), result.String())
	}
}
