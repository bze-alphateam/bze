package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// cheapTokenPool stores a 200,000 BZE / 2e17 ucheap pool with a 1% fee: one ucheap is worth 0.000001 ubze.
func (suite *IntegrationTestSuite) cheapTokenPool() {
	suite.Require().NoError(suite.k.SetParams(suite.ctx, denomTestParams("ubze")))
	suite.k.SetLiquidityPool(suite.ctx, types.LiquidityPool{
		Id:           "ubze_ucheap",
		Base:         "ubze",
		Quote:        "ucheap",
		ReserveBase:  math.NewInt(200_000_000_000),
		ReserveQuote: math.NewInt(200_000_000_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(1, 2),
	})
}

// 10,000 ucheap pays a 100 ucheap fee but swaps to 200e9 x 9,900 / (2e17 + 9,900) ~ 0.0099 ubze, truncated to 0:
// the coin is not swappable.
func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_PositiveFeeZeroOutput() {
	suite.cheapTokenPool()

	suite.Require().True(suite.k.HasDeepLiquidityWithNativeDenom(suite.ctx, "ucheap"), "the pool is deep enough")
	suite.Require().False(suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin("ucheap", 10_000)))
}

// On the same pool, the smallest coin that swaps to 1 ubze is swappable and the one just below it is not.
// After the 1% fee, 1,010,102 ucheap leaves a real input of 1,000,001: 200e9 x 1,000,001 / (2e17 + 1,000,001) = 1.
// 1,010,101 leaves 1,000,000 and swaps to 0.999999...
func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_OutputBoundary() {
	suite.cheapTokenPool()

	suite.Require().False(suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin("ucheap", 1_010_101)))
	suite.Require().True(suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin("ucheap", 1_010_102)))
	suite.Require().True(suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin("ucheap", 1_000_000_000)))
}

// The output check works on a pool whose native denom is the quote (the other denom sorts first).
func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_ZeroOutputNativeQuote() {
	suite.Require().NoError(suite.k.SetParams(suite.ctx, denomTestParams("ubze")))
	suite.k.SetLiquidityPool(suite.ctx, types.LiquidityPool{
		Id:           "ibc/CHEAP_ubze",
		Base:         "ibc/CHEAP",
		Quote:        "ubze",
		ReserveBase:  math.NewInt(200_000_000_000_000_000),
		ReserveQuote: math.NewInt(200_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(1, 2),
	})

	suite.Require().False(suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin("ibc/CHEAP", 10_000)))
	suite.Require().True(suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewInt64Coin("ibc/CHEAP", 1_010_102)))
}
