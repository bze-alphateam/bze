package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestCalculateOptimalSwapAmount() {
	tests := []struct {
		name           string
		pool           types.LiquidityPool
		inputCoin      sdk.Coin
		expectError    bool
		errorContains  string
		validateResult func(swapAmount math.Int, pool types.LiquidityPool, inputCoin sdk.Coin)
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
			validateResult: func(swapAmount math.Int, pool types.LiquidityPool, inputCoin sdk.Coin) {
				// The swap amount should be positive and less than input amount
				suite.Require().True(swapAmount.IsPositive(), "swap amount should be positive")
				suite.Require().True(swapAmount.LT(inputCoin.Amount), "swap amount should be less than input")

				// After swap, the remaining amounts should match the pool ratio
				// This is a simplified check - in reality, you'd simulate the swap and verify the ratio
				suite.T().Logf("Swap amount: %s out of %s", swapAmount.String(), inputCoin.Amount.String())
				suite.T().Logf("Remaining after swap: %s", inputCoin.Amount.Sub(swapAmount).String())
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
			validateResult: func(swapAmount math.Int, pool types.LiquidityPool, inputCoin sdk.Coin) {
				suite.Require().True(swapAmount.IsPositive(), "swap amount should be positive")
				suite.Require().True(swapAmount.LT(inputCoin.Amount), "swap amount should be less than input")
				suite.T().Logf("Swap amount: %s out of %s", swapAmount.String(), inputCoin.Amount.String())
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
			validateResult: func(swapAmount math.Int, pool types.LiquidityPool, inputCoin sdk.Coin) {
				// For very small amounts, swap might be 0 or very small
				suite.Require().False(swapAmount.IsNegative(), "swap amount should not be negative")
				suite.Require().True(swapAmount.LTE(inputCoin.Amount), "swap amount should not exceed input")
				suite.T().Logf("Small input swap amount: %s", swapAmount.String())
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			swapAmount, err := suite.k.CalculateOptimalSwapAmount(tt.pool, tt.inputCoin)

			if tt.expectError {
				suite.Require().Error(err)
				if tt.errorContains != "" {
					suite.Require().Contains(err.Error(), tt.errorContains)
				}
			} else {
				suite.Require().NoError(err)
				if tt.validateResult != nil {
					tt.validateResult(swapAmount, tt.pool, tt.inputCoin)
				}
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestCalculateOptimalSwapAmount_EdgeCases() {
	suite.Run("zero input amount", func() {
		pool := types.LiquidityPool{
			Id:           "tokenA_tokenB",
			Base:         "tokenA",
			Quote:        "tokenB",
			ReserveBase:  math.NewInt(1000000),
			ReserveQuote: math.NewInt(2000000),
			Fee:          math.LegacyNewDecWithPrec(3, 3),
		}
		inputCoin := sdk.NewCoin("tokenA", math.ZeroInt())

		swapAmount, err := suite.k.CalculateOptimalSwapAmount(pool, inputCoin)
		suite.Require().NoError(err)
		suite.Require().True(swapAmount.IsZero(), "swap amount should be zero for zero input")
	})

	suite.Run("very large reserves", func() {
		pool := types.LiquidityPool{
			Id:           "tokenA_tokenB",
			Base:         "tokenA",
			Quote:        "tokenB",
			ReserveBase:  math.NewInt(1_000_000_000_000), // 1 trillion
			ReserveQuote: math.NewInt(2_000_000_000_000), // 2 trillion
			Fee:          math.LegacyNewDecWithPrec(3, 3),
		}
		inputCoin := sdk.NewCoin("tokenA", math.NewInt(10_000_000_000)) // 10 billion

		swapAmount, err := suite.k.CalculateOptimalSwapAmount(pool, inputCoin)
		suite.Require().NoError(err)
		suite.Require().True(swapAmount.IsPositive())
		suite.Require().True(swapAmount.LT(inputCoin.Amount))
	})
}

func (suite *IntegrationTestSuite) TestCalculateOptimalInputForOutput() {
	tests := []struct {
		name           string
		pool           types.LiquidityPool
		outputCoin     sdk.Coin
		expectError    bool
		errorContains  string
		validateResult func(inputCoin sdk.Coin, pool types.LiquidityPool, outputCoin sdk.Coin)
	}{
		{
			name: "calculate input for base token output with 0.3% fee",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),            // 1M tokenA
				ReserveQuote: math.NewInt(2000000),            // 2M tokenB (ratio 1:2)
				Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3% fee
			},
			outputCoin:  sdk.NewCoin("tokenA", math.NewInt(100000)), // want 100k tokenA
			expectError: false,
			validateResult: func(inputCoin sdk.Coin, pool types.LiquidityPool, outputCoin sdk.Coin) {
				// The input should be in tokenB (the other token in the pool)
				suite.Require().Equal("tokenB", inputCoin.Denom, "input should be tokenB")
				suite.Require().True(inputCoin.IsPositive(), "input amount should be positive")

				// Manually calculate expected input:
				// For output = 100000 tokenA from 1M tokenA reserve
				// real_input = (100000 * 2000000) / (1000000 - 100000) = 200000000000 / 900000 ≈ 222222.22
				// input = 222222.22 / 0.997 ≈ 222889
				expectedInputApprox := math.NewInt(222889)

				// Allow some tolerance for rounding
				diff := inputCoin.Amount.Sub(expectedInputApprox).Abs()
				tolerance := math.NewInt(10)
				suite.Require().True(diff.LTE(tolerance),
					"input amount %s should be close to expected %s (diff: %s)",
					inputCoin.Amount.String(), expectedInputApprox.String(), diff.String())

				suite.T().Logf("To get %s, need to provide %s", outputCoin.String(), inputCoin.String())
			},
		},
		{
			name: "calculate input for quote token output with 0.3% fee",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
				Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3% fee
			},
			outputCoin:  sdk.NewCoin("tokenB", math.NewInt(200000)), // want 200k tokenB
			expectError: false,
			validateResult: func(inputCoin sdk.Coin, pool types.LiquidityPool, outputCoin sdk.Coin) {
				suite.Require().Equal("tokenA", inputCoin.Denom, "input should be tokenA")
				suite.Require().True(inputCoin.IsPositive(), "input amount should be positive")

				// real_input = (200000 * 1000000) / (2000000 - 200000) = 111111.11
				// input = 111111.11 / 0.997 ≈ 111445
				expectedInputApprox := math.NewInt(111445)
				diff := inputCoin.Amount.Sub(expectedInputApprox).Abs()
				tolerance := math.NewInt(10)
				suite.Require().True(diff.LTE(tolerance),
					"input amount %s should be close to expected %s",
					inputCoin.Amount.String(), expectedInputApprox.String())

				suite.T().Logf("To get %s, need to provide %s", outputCoin.String(), inputCoin.String())
			},
		},
		{
			name: "small output amount",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
				Fee:          math.LegacyNewDecWithPrec(3, 3),
			},
			outputCoin:  sdk.NewCoin("tokenA", math.NewInt(100)), // small output
			expectError: false,
			validateResult: func(inputCoin sdk.Coin, pool types.LiquidityPool, outputCoin sdk.Coin) {
				suite.Require().True(inputCoin.IsPositive(), "input should be positive")
				suite.T().Logf("For small output %s, need input %s", outputCoin.String(), inputCoin.String())
			},
		},
		{
			name: "output coin not in pool",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
				Fee:          math.LegacyNewDecWithPrec(3, 3),
			},
			outputCoin:    sdk.NewCoin("tokenC", math.NewInt(100000)),
			expectError:   true,
			errorContains: "does not exist in pool",
		},
		{
			name: "output exceeds reserve",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
				Fee:          math.LegacyNewDecWithPrec(3, 3),
			},
			outputCoin:    sdk.NewCoin("tokenA", math.NewInt(1000000)), // equals reserve
			expectError:   true,
			errorContains: "exceeds available reserve",
		},
		{
			name: "output greater than reserve",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
				Fee:          math.LegacyNewDecWithPrec(3, 3),
			},
			outputCoin:    sdk.NewCoin("tokenA", math.NewInt(1500000)), // exceeds reserve
			expectError:   true,
			errorContains: "exceeds available reserve",
		},
		{
			name: "zero output amount",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
				Fee:          math.LegacyNewDecWithPrec(3, 3),
			},
			outputCoin:    sdk.NewCoin("tokenA", math.ZeroInt()),
			expectError:   true,
			errorContains: "output amount must be positive",
		},
		{
			name: "high fee pool",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
				Fee:          math.LegacyNewDecWithPrec(5, 2), // 5% fee (maximum allowed)
			},
			outputCoin:  sdk.NewCoin("tokenA", math.NewInt(50000)),
			expectError: false,
			validateResult: func(inputCoin sdk.Coin, pool types.LiquidityPool, outputCoin sdk.Coin) {
				suite.Require().True(inputCoin.IsPositive(), "input should be positive even with high fee")
				suite.T().Logf("With 5%% fee, to get %s need %s", outputCoin.String(), inputCoin.String())
			},
		},
		{
			name: "large output relative to reserves",
			pool: types.LiquidityPool{
				Id:           "tokenA_tokenB",
				Base:         "tokenA",
				Quote:        "tokenB",
				ReserveBase:  math.NewInt(1000000),
				ReserveQuote: math.NewInt(2000000),
				Fee:          math.LegacyNewDecWithPrec(3, 3),
			},
			outputCoin:  sdk.NewCoin("tokenA", math.NewInt(900000)), // 90% of reserve
			expectError: false,
			validateResult: func(inputCoin sdk.Coin, pool types.LiquidityPool, outputCoin sdk.Coin) {
				suite.Require().True(inputCoin.IsPositive(), "input should be positive")
				// For large outputs, the required input grows exponentially due to AMM curve
				suite.T().Logf("For large output %s (90%% of reserve), need %s", outputCoin.String(), inputCoin.String())

				// The input should be very large relative to normal swaps
				// because we're draining most of the pool
				suite.Require().True(inputCoin.Amount.GT(outputCoin.Amount.MulRaw(2)),
					"for 90%% output, input should be significantly higher")
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			inputCoin, err := suite.k.CalculateOptimalInputForOutput(tt.pool, tt.outputCoin)

			if tt.expectError {
				suite.Require().Error(err)
				if tt.errorContains != "" {
					suite.Require().Contains(err.Error(), tt.errorContains)
				}
			} else {
				suite.Require().NoError(err)
				if tt.validateResult != nil {
					tt.validateResult(inputCoin, tt.pool, tt.outputCoin)
				}
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestCalculateOptimalInputForOutput_EdgeCases() {
	suite.Run("very large reserves", func() {
		pool := types.LiquidityPool{
			Id:           "tokenA_tokenB",
			Base:         "tokenA",
			Quote:        "tokenB",
			ReserveBase:  math.NewInt(1_000_000_000_000), // 1 trillion tokenA
			ReserveQuote: math.NewInt(2_000_000_000_000), // 2 trillion tokenB
			Fee:          math.LegacyNewDecWithPrec(3, 3),
		}
		outputCoin := sdk.NewCoin("tokenA", math.NewInt(10_000_000_000)) // 10 billion tokenA

		inputCoin, err := suite.k.CalculateOptimalInputForOutput(pool, outputCoin)
		suite.Require().NoError(err)
		suite.Require().True(inputCoin.IsPositive())
		suite.Require().Equal("tokenB", inputCoin.Denom)
		suite.T().Logf("Large reserves: to get %s need %s", outputCoin.String(), inputCoin.String())
	})

	suite.Run("unbalanced pool (1:1000 ratio)", func() {
		pool := types.LiquidityPool{
			Id:           "tokenA_tokenB",
			Base:         "tokenA",
			Quote:        "tokenB",
			ReserveBase:  math.NewInt(1_000_000),     // 1M tokenA
			ReserveQuote: math.NewInt(1_000_000_000), // 1B tokenB (1:1000 ratio)
			Fee:          math.LegacyNewDecWithPrec(3, 3),
		}
		outputCoin := sdk.NewCoin("tokenA", math.NewInt(10000)) // want 10k tokenA

		inputCoin, err := suite.k.CalculateOptimalInputForOutput(pool, outputCoin)
		suite.Require().NoError(err)
		suite.Require().True(inputCoin.IsPositive())
		suite.Require().Equal("tokenB", inputCoin.Denom)
		suite.T().Logf("Unbalanced pool: to get %s need %s", outputCoin.String(), inputCoin.String())
	})

	suite.Run("minimal output (1 unit)", func() {
		pool := types.LiquidityPool{
			Id:           "tokenA_tokenB",
			Base:         "tokenA",
			Quote:        "tokenB",
			ReserveBase:  math.NewInt(1000000),
			ReserveQuote: math.NewInt(2000000),
			Fee:          math.LegacyNewDecWithPrec(3, 3),
		}
		outputCoin := sdk.NewCoin("tokenA", math.NewInt(1)) // just 1 unit

		inputCoin, err := suite.k.CalculateOptimalInputForOutput(pool, outputCoin)
		suite.Require().NoError(err)
		suite.Require().True(inputCoin.IsPositive())
		suite.T().Logf("Minimal output: to get %s need %s", outputCoin.String(), inputCoin.String())
	})

	suite.Run("100% fee should fail", func() {
		pool := types.LiquidityPool{
			Id:           "tokenA_tokenB",
			Base:         "tokenA",
			Quote:        "tokenB",
			ReserveBase:  math.NewInt(1000000),
			ReserveQuote: math.NewInt(2000000),
			Fee:          math.LegacyOneDec(), // 100% fee
		}
		outputCoin := sdk.NewCoin("tokenA", math.NewInt(100))

		_, err := suite.k.CalculateOptimalInputForOutput(pool, outputCoin)
		suite.Require().Error(err)
		suite.Require().Contains(err.Error(), "fee is too high")
	})
}
