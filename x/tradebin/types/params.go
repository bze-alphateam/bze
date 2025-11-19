package types

import (
	"fmt"
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
) Params {
	return Params{
		CreateMarketFee:     createMarketFee,
		MarketMakerFee:      marketMakerFee,
		MarketTakerFee:      marketTakerFee,
		MakerFeeDestination: makerFeeDestination,
		TakerFeeDestination: takerFeeDestination,
		NativeDenom:         nativeDenom,
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
