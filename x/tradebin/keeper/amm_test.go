package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func (suite *IntegrationTestSuite) TestCreatePoolId() {
	base := "abc"
	quote := "xyz"
	newBase, newQuote, poolId := suite.k.CreatePoolId(base, quote)
	suite.Require().Equal(newBase, base)
	suite.Require().Equal(newQuote, quote)
	suite.Require().Contains(poolId, quote)
	suite.Require().Contains(poolId, base)

	newBase, newQuote, poolId = suite.k.CreatePoolId(quote, base)
	suite.Require().Equal(newBase, base)
	suite.Require().Equal(newQuote, quote)
	suite.Require().Contains(poolId, quote)
	suite.Require().Contains(poolId, base)
}

func (suite *IntegrationTestSuite) TestBalanceProvidedAmounts() {
	testCases := []struct {
		name           string
		base           uint64
		quote          uint64
		reserveBase    uint64
		reserveQuote   uint64
		expectedBase   sdk.Int
		expectedQuote  sdk.Int
		expectError    bool
		errorSubstring string
	}{
		{
			name:           "Empty pool - error",
			base:           100,
			quote:          200,
			reserveBase:    0,
			reserveQuote:   1000,
			expectedBase:   sdk.ZeroInt(),
			expectedQuote:  sdk.ZeroInt(),
			expectError:    true,
			errorSubstring: "pool is empty",
		},
		{
			name:           "Empty pool (quote) - error",
			base:           100,
			quote:          200,
			reserveBase:    1000,
			reserveQuote:   0,
			expectedBase:   sdk.ZeroInt(),
			expectedQuote:  sdk.ZeroInt(),
			expectError:    true,
			errorSubstring: "pool is empty",
		},
		{
			name:          "Base is limiting factor",
			base:          100,
			quote:         300,
			reserveBase:   1000,
			reserveQuote:  2000,
			expectedBase:  sdk.NewInt(100),
			expectedQuote: sdk.NewInt(200), // 100 * 2000 / 1000 = 200
			expectError:   false,
		},
		{
			name:          "Quote is limiting factor",
			base:          300,
			quote:         200,
			reserveBase:   1000,
			reserveQuote:  2000,
			expectedBase:  sdk.NewInt(100), // 200 * 1000 / 2000 = 100
			expectedQuote: sdk.NewInt(200),
			expectError:   false,
		},
		{
			name:          "Exact ratio provided",
			base:          500,
			quote:         1000,
			reserveBase:   1000,
			reserveQuote:  2000,
			expectedBase:  sdk.NewInt(500),
			expectedQuote: sdk.NewInt(1000),
			expectError:   false,
		},
		{
			name:          "Large numbers",
			base:          1000000,
			quote:         3000000,
			reserveBase:   5000000,
			reserveQuote:  10000000,
			expectedBase:  sdk.NewInt(1000000),
			expectedQuote: sdk.NewInt(2000000), // 1000000 * 10000000 / 5000000 = 2000000
			expectError:   false,
		},
		{
			name:          "Small amounts",
			base:          10,
			quote:         15,
			reserveBase:   1000,
			reserveQuote:  2000,
			expectedBase:  sdk.NewInt(7),
			expectedQuote: sdk.NewInt(15), // 10 * 2000 / 1000 = 20
			expectError:   false,
		},
		{
			name:          "Uneven ratio in pool",
			base:          100,
			quote:         200,
			reserveBase:   1000,
			reserveQuote:  1500,
			expectedBase:  sdk.NewInt(100),
			expectedQuote: sdk.NewInt(150), // 100 * 1500 / 1000 = 150
			expectError:   false,
		},
	}

	t := suite.T()
	k := suite.k
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			optimalBase, optimalQuote, err := k.BalanceProvidedAmounts(tc.base, tc.quote, tc.reserveBase, tc.reserveQuote)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorSubstring != "" {
					require.Contains(t, err.Error(), tc.errorSubstring)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedBase.String(), optimalBase.String())
				require.Equal(t, tc.expectedQuote.String(), optimalQuote.String())
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestBalanceProvidedAmounts_EdgeCases() {
	k := suite.k
	t := suite.T()
	// Very small ratio differences
	t.Run("Very small ratio difference - base limiting", func(t *testing.T) {
		base := uint64(100)
		quote := uint64(201) // Just slightly more than needed
		reserveBase := uint64(1000)
		reserveQuote := uint64(2000)

		optimalBase, optimalQuote, err := k.BalanceProvidedAmounts(base, quote, reserveBase, reserveQuote)

		require.NoError(t, err)
		require.Equal(t, sdk.NewInt(100).String(), optimalBase.String())
		require.Equal(t, sdk.NewInt(200).String(), optimalQuote.String()) // 100 * 2000 / 1000 = 200
	})

	t.Run("Very small ratio difference - quote limiting", func(t *testing.T) {
		base := uint64(101) // Just slightly more than needed
		quote := uint64(200)
		reserveBase := uint64(1000)
		reserveQuote := uint64(2000)

		optimalBase, optimalQuote, err := k.BalanceProvidedAmounts(base, quote, reserveBase, reserveQuote)

		require.NoError(t, err)
		require.Equal(t, sdk.NewInt(100).String(), optimalBase.String()) // 200 * 1000 / 2000 = 100
		require.Equal(t, sdk.NewInt(200).String(), optimalQuote.String())
	})

	// Handle division with remainder
	t.Run("Division with remainder", func(t *testing.T) {
		base := uint64(100)
		quote := uint64(300)
		reserveBase := uint64(1000)
		reserveQuote := uint64(3001) // Not evenly divisible

		optimalBase, optimalQuote, err := k.BalanceProvidedAmounts(base, quote, reserveBase, reserveQuote)

		require.NoError(t, err)
		// 100 * 3001 / 1000 = 300.1, which should truncate to 300
		require.Equal(t, sdk.NewInt(100).String(), optimalBase.String())
		require.Equal(t, sdk.NewInt(300).String(), optimalQuote.String())
	})
}
