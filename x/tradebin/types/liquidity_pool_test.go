package types

import (
	"cosmossdk.io/math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestLiquidityPool_HasDenom(t *testing.T) {
	tests := []struct {
		name     string
		pool     LiquidityPool
		denom    string
		expected bool
	}{
		{
			name: "base denom exists",
			pool: LiquidityPool{
				Base:  "ubze",
				Quote: "uusdc",
			},
			denom:    "ubze",
			expected: true,
		},
		{
			name: "quote denom exists",
			pool: LiquidityPool{
				Base:  "ubze",
				Quote: "uusdc",
			},
			denom:    "uusdc",
			expected: true,
		},
		{
			name: "denom does not exist",
			pool: LiquidityPool{
				Base:  "ubze",
				Quote: "uusdc",
			},
			denom:    "uatom",
			expected: false,
		},
		{
			name: "empty denom",
			pool: LiquidityPool{
				Base:  "ubze",
				Quote: "uusdc",
			},
			denom:    "",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.pool.HasDenom(tc.denom)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestLiquidityPool_GetReservesCoinsByDenom(t *testing.T) {
	tests := []struct {
		name              string
		pool              LiquidityPool
		denom             string
		expectedDenomCoin sdk.Coin
		expectedCounter   sdk.Coin
		hasDenom          bool
	}{
		{
			name: "get reserves by base denom",
			pool: LiquidityPool{
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			denom:             "ubze",
			expectedDenomCoin: sdk.NewCoin("ubze", math.NewInt(1000)),
			expectedCounter:   sdk.NewCoin("uusdc", math.NewInt(2000)),
			hasDenom:          true,
		},
		{
			name: "get reserves by quote denom",
			pool: LiquidityPool{
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			denom:             "uusdc",
			expectedDenomCoin: sdk.NewCoin("uusdc", math.NewInt(2000)),
			expectedCounter:   sdk.NewCoin("ubze", math.NewInt(1000)),
			hasDenom:          true,
		},
		{
			name: "denom not in pool",
			pool: LiquidityPool{
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			denom:             "uatom",
			expectedDenomCoin: sdk.Coin{},
			expectedCounter:   sdk.Coin{},
			hasDenom:          false,
		},
		{
			name: "zero reserves",
			pool: LiquidityPool{
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(0),
				ReserveQuote: math.NewInt(0),
			},
			denom:             "ubze",
			expectedDenomCoin: sdk.NewCoin("ubze", math.NewInt(0)),
			expectedCounter:   sdk.NewCoin("uusdc", math.NewInt(0)),
			hasDenom:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			denomCoin, counterCoin := tc.pool.GetReservesCoinsByDenom(tc.denom)

			if !tc.hasDenom {
				require.Equal(t, sdk.Coin{}, denomCoin)
				require.Equal(t, sdk.Coin{}, counterCoin)
				return
			}

			require.Equal(t, tc.expectedDenomCoin.Denom, denomCoin.Denom)
			require.True(t, tc.expectedDenomCoin.Amount.Equal(denomCoin.Amount))

			require.Equal(t, tc.expectedCounter.Denom, counterCoin.Denom)
			require.True(t, tc.expectedCounter.Amount.Equal(counterCoin.Amount))
		})
	}
}

func TestLiquidityPool_ChangeReserves(t *testing.T) {
	tests := []struct {
		name          string
		pool          LiquidityPool
		add           sdk.Coin
		subtract      sdk.Coin
		expectError   bool
		errorContains string
		expectedBase  math.Int
		expectedQuote math.Int
	}{
		{
			name: "add to base, subtract from quote",
			pool: LiquidityPool{
				Id:           "pool1",
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			add:           sdk.NewCoin("ubze", math.NewInt(500)),
			subtract:      sdk.NewCoin("uusdc", math.NewInt(300)),
			expectError:   false,
			expectedBase:  math.NewInt(1500),
			expectedQuote: math.NewInt(1700),
		},
		{
			name: "add to quote, subtract from base",
			pool: LiquidityPool{
				Id:           "pool1",
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			add:           sdk.NewCoin("uusdc", math.NewInt(500)),
			subtract:      sdk.NewCoin("ubze", math.NewInt(300)),
			expectError:   false,
			expectedBase:  math.NewInt(700),
			expectedQuote: math.NewInt(2500),
		},
		{
			name: "same denom error",
			pool: LiquidityPool{
				Id:           "pool1",
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			add:           sdk.NewCoin("ubze", math.NewInt(500)),
			subtract:      sdk.NewCoin("ubze", math.NewInt(300)),
			expectError:   true,
			errorContains: "can not change reserves with amounts of the same denom",
		},
		{
			name: "denom not in pool - add",
			pool: LiquidityPool{
				Id:           "pool1",
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			add:           sdk.NewCoin("uatom", math.NewInt(500)),
			subtract:      sdk.NewCoin("uusdc", math.NewInt(300)),
			expectError:   true,
			errorContains: "can not change reserves of pool",
		},
		{
			name: "denom not in pool - subtract",
			pool: LiquidityPool{
				Id:           "pool1",
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			add:           sdk.NewCoin("ubze", math.NewInt(500)),
			subtract:      sdk.NewCoin("uatom", math.NewInt(300)),
			expectError:   true,
			errorContains: "can not change reserves of pool",
		},
		{
			name: "insufficient quote reserve",
			pool: LiquidityPool{
				Id:           "pool1",
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			add:           sdk.NewCoin("ubze", math.NewInt(500)),
			subtract:      sdk.NewCoin("uusdc", math.NewInt(3000)),
			expectError:   true,
			errorContains: "insufficient quote reserve",
		},
		{
			name: "insufficient base reserve",
			pool: LiquidityPool{
				Id:           "pool1",
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			add:           sdk.NewCoin("uusdc", math.NewInt(500)),
			subtract:      sdk.NewCoin("ubze", math.NewInt(1500)),
			expectError:   true,
			errorContains: "insufficient base reserve",
		},
		{
			name: "zero add amount",
			pool: LiquidityPool{
				Id:           "pool1",
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			add:           sdk.NewCoin("ubze", math.ZeroInt()),
			subtract:      sdk.NewCoin("uusdc", math.NewInt(300)),
			expectError:   false,
			expectedBase:  math.NewInt(1000),
			expectedQuote: math.NewInt(1700),
		},
		{
			name: "zero subtract amount",
			pool: LiquidityPool{
				Id:           "pool1",
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
			},
			add:           sdk.NewCoin("ubze", math.NewInt(500)),
			subtract:      sdk.NewCoin("uusdc", math.ZeroInt()),
			expectError:   false,
			expectedBase:  math.NewInt(1500),
			expectedQuote: math.NewInt(2000),
		},
		{
			name: "large amounts",
			pool: LiquidityPool{
				Id:           "pool1",
				Base:         "ubze",
				Quote:        "uusdc",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
			},
			add:           sdk.NewCoin("ubze", math.NewInt(500000)),
			subtract:      sdk.NewCoin("uusdc", math.NewInt(300000)),
			expectError:   false,
			expectedBase:  math.NewInt(1500000),
			expectedQuote: math.NewInt(1700000),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a copy of the pool for testing
			pool := tc.pool

			err := pool.ChangeReserves(tc.add, tc.subtract)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.True(t, tc.expectedBase.Equal(pool.ReserveBase))
			require.True(t, tc.expectedQuote.Equal(pool.ReserveQuote))
		})
	}
}

func TestLiquidityPool_ChangeReserves_EdgeCases(t *testing.T) {
	// Test for very large numbers
	t.Run("very large numbers", func(t *testing.T) {
		// Create a large number but one that can be safely represented
		largeNumber := math.NewInt(9223372036854775807) // Max int64

		pool := LiquidityPool{
			Id:           "pool1",
			Base:         "ubze",
			Quote:        "uusdc",
			ReserveBase:  math.NewInt(1000),
			ReserveQuote: largeNumber,
		}

		// Add a more modest amount that won't cause issues
		add := sdk.NewCoin("uusdc", math.NewInt(2000))
		subtract := sdk.NewCoin("ubze", math.NewInt(500))

		err := pool.ChangeReserves(add, subtract)
		require.NoError(t, err)

		// Calculate expected values more explicitly
		expectedBase := math.NewInt(500) // 1000 - 500
		expectedQuote := largeNumber.Add(math.NewInt(2000))

		require.True(t, expectedBase.Equal(pool.ReserveBase))
		require.True(t, expectedQuote.Equal(pool.ReserveQuote))
	})

	// Test for exact reserve depletion
	t.Run("exact reserve depletion", func(t *testing.T) {
		pool := LiquidityPool{
			Id:           "pool1",
			Base:         "ubze",
			Quote:        "uusdc",
			ReserveBase:  math.NewInt(1000),
			ReserveQuote: math.NewInt(2000),
		}

		// Subtract exactly the available amount
		add := sdk.NewCoin("ubze", math.NewInt(500))
		subtract := sdk.NewCoin("uusdc", math.NewInt(2000))

		err := pool.ChangeReserves(add, subtract)
		require.NoError(t, err)
		require.True(t, math.NewInt(1500).Equal(pool.ReserveBase))
		require.True(t, math.ZeroInt().Equal(pool.ReserveQuote))
	})
}
