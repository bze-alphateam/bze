package v2types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	FeeDestinationCommunityPool = "community_pool"
	FeeDestinationBurnerModule  = "burner"

	DefaultNativeDenom = "ubze"
)

var (
	DefaultCreateMarketFee                 = sdk.NewInt64Coin(DefaultNativeDenom, 25000000000)
	DefaultMarketMakerFee                  = sdk.NewInt64Coin(DefaultNativeDenom, 1000)
	DefaultMarketTakerFee                  = sdk.NewInt64Coin(DefaultNativeDenom, 100000)
	DefaultMakerFeeDestination             = FeeDestinationBurnerModule
	DefaultTakerFeeDestination             = FeeDestinationBurnerModule
	DefaultPriceLevelOwnersWindow          = uint64(500)
	DefaultPriceLevelExtraGas              = uint64(25000)
	DefaultFillOrdersExtraGas              = uint64(5000)
	DefaultMinNativeLiquidityForModuleSwap = math.NewInt(100000000000)
	DefaultMaxFillsPerOrder                = uint64(500)
	DefaultPoolPriceHalfLifeBlocks         = uint64(20)
	DefaultFeeMaxPriceDrift                = math.LegacyNewDecWithPrec(10, 2)
	DefaultModuleMaxPriceDrift             = math.LegacyNewDecWithPrec(3, 2)
)

// MaxPoolPriceHalfLifeBlocks is the largest accepted half-life of the pool price average.
const MaxPoolPriceHalfLifeBlocks = uint64(100000)

// DefaultParams returns a default set of v2 parameters
func DefaultParams() Params {
	return Params{
		CreateMarketFee:                 DefaultCreateMarketFee,
		MarketMakerFee:                  DefaultMarketMakerFee,
		MarketTakerFee:                  DefaultMarketTakerFee,
		MakerFeeDestination:             DefaultMakerFeeDestination,
		TakerFeeDestination:             DefaultTakerFeeDestination,
		NativeDenom:                     DefaultNativeDenom,
		PriceLevelOwnersWindow:          DefaultPriceLevelOwnersWindow,
		PriceLevelExtraGas:              DefaultPriceLevelExtraGas,
		FillOrdersExtraGas:              DefaultFillOrdersExtraGas,
		MinNativeLiquidityForModuleSwap: DefaultMinNativeLiquidityForModuleSwap,
		MaxFillsPerOrder:                DefaultMaxFillsPerOrder,
		PoolPriceHalfLifeBlocks:         DefaultPoolPriceHalfLifeBlocks,
		FeeMaxPriceDrift:                DefaultFeeMaxPriceDrift,
		ModuleMaxPriceDrift:             DefaultModuleMaxPriceDrift,
	}
}

// Validate validates the Params fields.
func (p Params) Validate() error {
	if !p.CreateMarketFee.IsValid() {
		return fmt.Errorf("invalid CreateMarketFee: %s", p.CreateMarketFee)
	}

	if !p.MarketMakerFee.IsValid() {
		return fmt.Errorf("invalid MarketMakerFee: %s", p.MarketMakerFee)
	}

	if !p.MarketTakerFee.IsValid() {
		return fmt.Errorf("invalid MarketTakerFee: %s", p.MarketTakerFee)
	}

	if err := validateFeeDestination(p.MakerFeeDestination); err != nil {
		return fmt.Errorf("invalid MakerFeeDestination: %w", err)
	}

	if err := validateFeeDestination(p.TakerFeeDestination); err != nil {
		return fmt.Errorf("invalid TakerFeeDestination: %w", err)
	}

	if p.NativeDenom == "" {
		return fmt.Errorf("native denom cannot be an empty string")
	}

	if !p.MinNativeLiquidityForModuleSwap.IsPositive() {
		return fmt.Errorf("min native liquidity for module swap must be positive")
	}

	if p.PriceLevelOwnersWindow < 1 {
		return fmt.Errorf("price level owners window must be at least 1")
	}

	if p.PriceLevelExtraGas < 1 {
		return fmt.Errorf("price level extra gas must be at least 1")
	}

	if p.MaxFillsPerOrder < 1 {
		return fmt.Errorf("max fills per order must be at least 1")
	}

	if p.PoolPriceHalfLifeBlocks < 1 || p.PoolPriceHalfLifeBlocks > MaxPoolPriceHalfLifeBlocks {
		return fmt.Errorf("pool price half life must be between 1 and %d blocks", MaxPoolPriceHalfLifeBlocks)
	}

	if err := validateDrift(p.FeeMaxPriceDrift); err != nil {
		return fmt.Errorf("invalid FeeMaxPriceDrift: %w", err)
	}

	if err := validateDrift(p.ModuleMaxPriceDrift); err != nil {
		return fmt.Errorf("invalid ModuleMaxPriceDrift: %w", err)
	}

	return nil
}

// validateDrift accepts a ratio strictly between 0 and 1.
func validateDrift(d math.LegacyDec) error {
	if d.IsNil() {
		return fmt.Errorf("drift must be set")
	}

	if !d.IsPositive() || !d.LT(math.LegacyOneDec()) {
		return fmt.Errorf("drift must be greater than 0 and less than 1")
	}

	return nil
}

func validateFeeDestination(dest string) error {
	if dest != FeeDestinationCommunityPool && dest != FeeDestinationBurnerModule {
		return fmt.Errorf("invalid fee destination: %s", dest)
	}

	return nil
}
