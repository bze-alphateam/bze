package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	FeeDestinationCommunityPool = "community_pool"
	FeeDestinationBurnerModule  = "burner"
)

var (
	KeyCreateMarketFee     = []byte("CreateMarketFee")
	DefaultCreateMarketFee = "25000000000ubze"

	KeyMarketMakerFee     = []byte("MarketMakerFee")
	DefaultMarketMakerFee = "1000ubze"

	KeyMarketTakerFee     = []byte("MarketTakerFee")
	DefaultMarketTakerFee = "100000ubze"

	KeyMakerFeeDestination     = []byte("MakerFeeDestination")
	DefaultMakerFeeDestination = FeeDestinationBurnerModule

	KeyTakerFeeDestination     = []byte("TakerFeeDestination")
	DefaultTakerFeeDestination = FeeDestinationBurnerModule

	KeyOrderBookExtraGasWindow     = []byte("OrderBookExtraGasWindow")
	DefaultOrderBookExtraGasWindow = uint64(100)

	KeyOrderBookQueueExtraGas     = []byte("OrderBookQueueExtraGas")
	DefaultOrderBookQueueExtraGas = uint64(25000)

	KeyFillOrdersExtraGas     = []byte("FillOrdersExtraGas")
	DefaultFillOrdersExtraGas = uint64(5000)

	KeyOrderBookQueueMessageScanExtraGas     = []byte("OrderBookQueueMessageScanExtraGas")
	DefaultOrderBookQueueMessageScanExtraGas = uint64(5000)

	KeyMinNativeLiquidityForModuleSwap     = []byte("MinNativeLiquidityForModuleSwap")
	DefaultMinNativeLiquidityForModuleSwap = math.NewInt(100000000000)

	KeyOrderBookPerBlockMessages     = []byte("OrderBookPerBlockMessages")
	DefaultOrderBookPerBlockMessages = uint64(500)

	DefaultNativeDenom = "ubze"
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	createMarketFee string,
	marketMakerFee string,
	marketTakerFee string,
	makerFeeDestination string,
	takerFeeDestination string,
	nativeDenom string,
	orderBookExtraGasWindow uint64,
	orderBookQueueExtraGas uint64,
	fillOrdersExtraGas uint64,
	orderBookQueueMessageScanExtraGas uint64,
	minNativeLiquidityForModuleSwap math.Int,
	orderBookPerBlockMessages uint64,
) Params {
	return Params{
		CreateMarketFee:                   createMarketFee,
		MarketMakerFee:                    marketMakerFee,
		MarketTakerFee:                    marketTakerFee,
		MakerFeeDestination:               makerFeeDestination,
		TakerFeeDestination:               takerFeeDestination,
		NativeDenom:                       nativeDenom,
		OrderBookExtraGasWindow:           orderBookExtraGasWindow,
		OrderBookQueueExtraGas:            orderBookQueueExtraGas,
		FillOrdersExtraGas:                fillOrdersExtraGas,
		OrderBookQueueMessageScanExtraGas: orderBookQueueMessageScanExtraGas,
		MinNativeLiquidityForModuleSwap:   minNativeLiquidityForModuleSwap,
		OrderBookPerBlockMessages:         orderBookPerBlockMessages,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultCreateMarketFee,
		DefaultMarketMakerFee,
		DefaultMarketTakerFee,
		DefaultMakerFeeDestination,
		DefaultTakerFeeDestination,
		DefaultNativeDenom,
		DefaultOrderBookExtraGasWindow,
		DefaultOrderBookQueueExtraGas,
		DefaultFillOrdersExtraGas,
		DefaultOrderBookQueueMessageScanExtraGas,
		DefaultMinNativeLiquidityForModuleSwap,
		DefaultOrderBookPerBlockMessages,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyCreateMarketFee, &p.CreateMarketFee, validateCreateMarketFee),
		paramtypes.NewParamSetPair(KeyMarketMakerFee, &p.MarketMakerFee, validateMarketMakerFee),
		paramtypes.NewParamSetPair(KeyMarketTakerFee, &p.MarketTakerFee, validateMarketTakerFee),
		paramtypes.NewParamSetPair(KeyMakerFeeDestination, &p.MakerFeeDestination, validateMakerFeeDestination),
		paramtypes.NewParamSetPair(KeyTakerFeeDestination, &p.TakerFeeDestination, validateTakerFeeDestination),
		paramtypes.NewParamSetPair(KeyOrderBookExtraGasWindow, &p.OrderBookExtraGasWindow, validateOrderBookExtraGasWindow),
		paramtypes.NewParamSetPair(KeyOrderBookQueueExtraGas, &p.OrderBookQueueExtraGas, validateOrderBookQueueExtraGas),
		paramtypes.NewParamSetPair(KeyFillOrdersExtraGas, &p.FillOrdersExtraGas, validateFillOrdersExtraGas),
		paramtypes.NewParamSetPair(KeyOrderBookQueueMessageScanExtraGas, &p.OrderBookQueueMessageScanExtraGas, validateOrderBookQueueMessageScanExtraGas),
		paramtypes.NewParamSetPair(KeyMinNativeLiquidityForModuleSwap, &p.MinNativeLiquidityForModuleSwap, validateMinNativeLiquidityForModuleSwap),
		paramtypes.NewParamSetPair(KeyOrderBookPerBlockMessages, &p.OrderBookPerBlockMessages, validateOrderBookPerBlockMessages),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateCreateMarketFee(p.CreateMarketFee); err != nil {
		return err
	}

	if err := validateMarketMakerFee(p.MarketMakerFee); err != nil {
		return err
	}

	if err := validateMarketTakerFee(p.MarketTakerFee); err != nil {
		return err
	}

	if err := validateMakerFeeDestination(p.MakerFeeDestination); err != nil {
		return err
	}

	if err := validateTakerFeeDestination(p.TakerFeeDestination); err != nil {
		return err
	}

	if err := validateNativeDenom(p.NativeDenom); err != nil {
		return err
	}

	if err := validateOrderBookExtraGasWindow(p.OrderBookExtraGasWindow); err != nil {
		return err
	}

	if err := validateOrderBookQueueExtraGas(p.OrderBookQueueExtraGas); err != nil {
		return err
	}

	if err := validateFillOrdersExtraGas(p.FillOrdersExtraGas); err != nil {
		return err
	}

	if err := validateOrderBookQueueMessageScanExtraGas(p.OrderBookQueueMessageScanExtraGas); err != nil {
		return err
	}

	if err := validateMinNativeLiquidityForModuleSwap(p.MinNativeLiquidityForModuleSwap); err != nil {
		return err
	}

	if err := validateOrderBookPerBlockMessages(p.OrderBookPerBlockMessages); err != nil {
		return err
	}

	return nil
}

// validateCreateMarketFee validates the CreateMarketFee param
func validateCreateMarketFee(v interface{}) error {
	createMarketFee, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	normCoin, err := sdk.ParseCoinNormalized(createMarketFee)
	if err != nil {
		return err
	}

	if normCoin.IsNegative() {
		return fmt.Errorf("negative amount provided")
	}

	return nil
}

// validateMarketMakerFee validates the MarketMakerFee param
func validateMarketMakerFee(v interface{}) error {
	marketMakerFee, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	normCoin, err := sdk.ParseCoinNormalized(marketMakerFee)
	if err != nil {
		return err
	}

	if normCoin.IsNegative() {
		return fmt.Errorf("negative amount provided")
	}

	return nil
}

// validateMarketTakerFee validates the MarketTakerFee param
func validateMarketTakerFee(v interface{}) error {
	marketTakerFee, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	normCoin, err := sdk.ParseCoinNormalized(marketTakerFee)
	if err != nil {
		return err
	}

	if normCoin.IsNegative() {
		return fmt.Errorf("negative amount provided")
	}

	return nil
}

// validateMakerFeeDestination validates the MakerFeeDestination param
func validateMakerFeeDestination(v interface{}) error {
	makerFeeDestination, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return validateFeeDestination(makerFeeDestination)
}

// validateTakerFeeDestination validates the TakerFeeDestination param
func validateTakerFeeDestination(v interface{}) error {
	takerFeeDestination, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return validateFeeDestination(takerFeeDestination)
}

func validateFeeDestination(dest string) error {
	if dest != FeeDestinationCommunityPool && dest != FeeDestinationBurnerModule {
		return fmt.Errorf("invalid fee destination: %s", dest)
	}

	return nil
}

func validateNativeDenom(v interface{}) error {
	nativeDenom, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid native denom parameter type: %T", v)
	}

	if nativeDenom == "" {
		return fmt.Errorf("native denom cannot be an empty string")
	}

	return nil
}

func validateOrderBookExtraGasWindow(v interface{}) error {
	_, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}

func validateOrderBookQueueExtraGas(v interface{}) error {
	_, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}

func validateFillOrdersExtraGas(v interface{}) error {
	_, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}

func validateOrderBookQueueMessageScanExtraGas(v interface{}) error {
	_, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}

func validateMinNativeLiquidityForModuleSwap(v interface{}) error {
	minLiquidity, ok := v.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if !minLiquidity.IsPositive() {
		return fmt.Errorf("min native liquidity for module swap must be positive")
	}

	return nil
}

func validateOrderBookPerBlockMessages(v interface{}) error {
	orderBookPerBlockMessages, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if orderBookPerBlockMessages < 1 {
		return fmt.Errorf("order book per block messages must be at least 1")
	}

	return nil
}
