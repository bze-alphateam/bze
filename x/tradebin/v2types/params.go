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
	DefaultCreateMarketFee                   = sdk.NewInt64Coin(DefaultNativeDenom, 25000000000)
	DefaultMarketMakerFee                    = sdk.NewInt64Coin(DefaultNativeDenom, 1000)
	DefaultMarketTakerFee                    = sdk.NewInt64Coin(DefaultNativeDenom, 100000)
	DefaultMakerFeeDestination               = FeeDestinationBurnerModule
	DefaultTakerFeeDestination               = FeeDestinationBurnerModule
	DefaultOrderBookExtraGasWindow           = uint64(100)
	DefaultOrderBookQueueExtraGas            = uint64(25000)
	DefaultFillOrdersExtraGas                = uint64(5000)
	DefaultOrderBookQueueMessageScanExtraGas = uint64(5000)
	DefaultMinNativeLiquidityForModuleSwap   = math.NewInt(100000000000)
	DefaultOrderBookPerBlockMessages         = uint64(500)
)

// DefaultParams returns a default set of v2 parameters
func DefaultParams() Params {
	return Params{
		CreateMarketFee:                   DefaultCreateMarketFee,
		MarketMakerFee:                    DefaultMarketMakerFee,
		MarketTakerFee:                    DefaultMarketTakerFee,
		MakerFeeDestination:               DefaultMakerFeeDestination,
		TakerFeeDestination:               DefaultTakerFeeDestination,
		NativeDenom:                       DefaultNativeDenom,
		OrderBookExtraGasWindow:           DefaultOrderBookExtraGasWindow,
		OrderBookQueueExtraGas:            DefaultOrderBookQueueExtraGas,
		FillOrdersExtraGas:                DefaultFillOrdersExtraGas,
		OrderBookQueueMessageScanExtraGas: DefaultOrderBookQueueMessageScanExtraGas,
		MinNativeLiquidityForModuleSwap:   DefaultMinNativeLiquidityForModuleSwap,
		OrderBookPerBlockMessages:         DefaultOrderBookPerBlockMessages,
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

	if p.OrderBookPerBlockMessages < 1 {
		return fmt.Errorf("order book per block messages must be at least 1")
	}

	return nil
}

func validateFeeDestination(dest string) error {
	if dest != FeeDestinationCommunityPool && dest != FeeDestinationBurnerModule {
		return fmt.Errorf("invalid fee destination: %s", dest)
	}

	return nil
}
