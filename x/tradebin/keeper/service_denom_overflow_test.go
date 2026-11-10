package keeper_test

import (
	"math/big"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// pow2 returns 2^n as a math.Int (n <= 255).
func pow2(n uint) math.Int {
	return math.NewIntFromBigInt(new(big.Int).Lsh(big.NewInt(1), n))
}

// deepTokenPool stores a 200,000 BZE / 200,000 ibc/BIG pool with a 1% fee, deep enough for module swaps.
func (suite *IntegrationTestSuite) deepTokenPool() {
	suite.Require().NoError(suite.k.SetParams(suite.ctx, denomTestParams("ubze")))
	suite.k.SetLiquidityPool(suite.ctx, types.LiquidityPool{
		Id:           "ibc/BIG_ubze",
		Base:         "ibc/BIG",
		Quote:        "ubze",
		ReserveBase:  math.NewInt(200_000_000_000),
		ReserveQuote: math.NewInt(200_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(1, 2),
	})
}

// A coin so large that the swap output product doesn't fit in 256 bits can't be swapped: the check reports it as not
// swappable instead of panicking.
func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_OversizedAmountNotSwappable() {
	suite.deepTokenPool()

	oversized := sdk.NewCoin("ibc/BIG", pow2(240))
	suite.Require().NotPanics(func() {
		suite.Require().False(suite.k.CanSwapForNativeDenom(suite.ctx, oversized))
	})
}

// A large coin whose output product still fits stays swappable.
func (suite *IntegrationTestSuite) TestServiceDenom_CanSwapForNativeDenom_LargeAmountInRangeSwappable() {
	suite.deepTokenPool()

	// 2e11 x 2^200 fits easily in 256 bits
	suite.Require().True(suite.k.CanSwapForNativeDenom(suite.ctx, sdk.NewCoin("ibc/BIG", pow2(200))))
}

// Operands too large for the optimal swap formula return an error instead of panicking.
func (suite *IntegrationTestSuite) TestCalculateOptimalSwapAmount_OversizedOperandsReturnError() {
	pool := types.LiquidityPool{
		Id:           "ibc/BIG_ubze",
		Base:         "ibc/BIG",
		Quote:        "ubze",
		ReserveBase:  pow2(200),
		ReserveQuote: math.NewInt(200_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(1, 2),
	}

	var (
		amount math.Int
		err    error
	)
	suite.Require().NotPanics(func() {
		amount, err = suite.k.CalculateOptimalSwapAmount(pool, sdk.NewCoin("ibc/BIG", pow2(200)))
	})
	suite.Require().Error(err)
	suite.Require().True(amount.IsZero())
}

// Large operands that still fit the formula keep returning a result.
func (suite *IntegrationTestSuite) TestCalculateOptimalSwapAmount_LargeOperandsInRange() {
	pool := types.LiquidityPool{
		Id:           "ibc/BIG_ubze",
		Base:         "ibc/BIG",
		Quote:        "ubze",
		ReserveBase:  pow2(100),
		ReserveQuote: math.NewInt(200_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(1, 2),
	}

	amount, err := suite.k.CalculateOptimalSwapAmount(pool, sdk.NewCoin("ibc/BIG", pow2(100)))
	suite.Require().NoError(err)
	suite.Require().True(amount.IsPositive())
}
