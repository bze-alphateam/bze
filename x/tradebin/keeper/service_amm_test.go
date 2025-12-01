package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestCalculateOptimalSwapAmount(t *testing.T) {
	keeper := Keeper{}

	tests := []struct {
		name           string
		pool           types.LiquidityPool
		inputCoin      sdk.Coin
		expectError    bool
		errorContains  string
		validateResult func(t *testing.T, swapAmount math.Int, pool types.LiquidityPool, inputCoin sdk.Coin)
	}{
		{
			name: "optimal swap with 0.3% fee",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),            // 1M tokenA
				ReserveQuote: math.NewInt(2000000),            // 2M tokenB (ratio 1:2)
				Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3% fee
			},
			inputCoin:   sdk.NewCoin("tokenA", math.NewInt(500000)), // 500k tokenA
			expectError: false,
			validateResult: func(t *testing.T, swapAmount math.Int, pool types.LiquidityPool, inputCoin sdk.Coin) {
				// The swap amount should be positive and less than input amount
				require.True(t, swapAmount.IsPositive(), "swap amount should be positive")
				require.True(t, swapAmount.LT(inputCoin.Amount), "swap amount should be less than input")

				// After swap, the remaining amounts should match the pool ratio
				// This is a simplified check - in reality, you'd simulate the swap and verify the ratio
				t.Logf("Swap amount: %s out of %s", swapAmount.String(), inputCoin.Amount.String())
				t.Logf("Remaining after swap: %s", inputCoin.Amount.Sub(swapAmount).String())
			},
		},
		{
			name: "optimal swap with different denomination (quote token)",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
				Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3% fee
			},
			inputCoin:   sdk.NewCoin("tokenB", math.NewInt(1000000)), // 1M tokenB
			expectError: false,
			validateResult: func(t *testing.T, swapAmount math.Int, pool types.LiquidityPool, inputCoin sdk.Coin) {
				require.True(t, swapAmount.IsPositive(), "swap amount should be positive")
				require.True(t, swapAmount.LT(inputCoin.Amount), "swap amount should be less than input")
				t.Logf("Swap amount: %s out of %s", swapAmount.String(), inputCoin.Amount.String())
			},
		},
		{
			name: "input coin not in pool",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
				Fee:          math.LegacyNewDecWithPrec(3, 3),
			},
			inputCoin:     sdk.NewCoin("tokenC", math.NewInt(500000)),
			expectError:   true,
			errorContains: "does not exist in pool",
		},
		{
			name: "small input amount",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
				Fee:          math.LegacyNewDecWithPrec(3, 3),
			},
			inputCoin:   sdk.NewCoin("tokenA", math.NewInt(100)),
			expectError: false,
			validateResult: func(t *testing.T, swapAmount math.Int, pool types.LiquidityPool, inputCoin sdk.Coin) {
				// For very small amounts, swap might be 0 or very small
				require.False(t, swapAmount.IsNegative(), "swap amount should not be negative")
				require.True(t, swapAmount.LTE(inputCoin.Amount), "swap amount should not exceed input")
				t.Logf("Small input swap amount: %s", swapAmount.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			swapAmount, err := keeper.CalculateOptimalSwapAmount(tt.pool, tt.inputCoin)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				if tt.validateResult != nil {
					tt.validateResult(t, swapAmount, tt.pool, tt.inputCoin)
				}
			}
		})
	}
}

func TestCalculateOptimalSwapAmount_EdgeCases(t *testing.T) {
	keeper := Keeper{}

	t.Run("zero input amount", func(t *testing.T) {
		pool := types.LiquidityPool{
			Id:           "tokenA_tokenB",
			Base:         "tokenA",
			Quote:        "tokenB",
			ReserveBase:  math.NewInt(1000000),
			ReserveQuote: math.NewInt(2000000),
			Fee:          math.LegacyNewDecWithPrec(3, 3),
		}
		inputCoin := sdk.NewCoin("tokenA", math.ZeroInt())

		swapAmount, err := keeper.CalculateOptimalSwapAmount(pool, inputCoin)
		require.NoError(t, err)
		require.True(t, swapAmount.IsZero(), "swap amount should be zero for zero input")
	})

	t.Run("very large reserves", func(t *testing.T) {
		pool := types.LiquidityPool{
			Id:           "tokenA_tokenB",
			Base:         "tokenA",
			Quote:        "tokenB",
			ReserveBase:  math.NewInt(1_000_000_000_000), // 1 trillion
			ReserveQuote: math.NewInt(2_000_000_000_000), // 2 trillion
			Fee:          math.LegacyNewDecWithPrec(3, 3),
		}
		inputCoin := sdk.NewCoin("tokenA", math.NewInt(10_000_000_000)) // 10 billion

		swapAmount, err := keeper.CalculateOptimalSwapAmount(pool, inputCoin)
		require.NoError(t, err)
		require.True(t, swapAmount.IsPositive())
		require.True(t, swapAmount.LT(inputCoin.Amount))
	})
}
