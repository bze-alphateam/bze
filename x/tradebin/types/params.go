package types

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	FeeDestinationCommunityPool = "community_pool"
	FeeDestinationBurnerModule  = "burner"
)

var (
	KeyCreateMarketFee     = []byte("CreateMarketFee")
	DefaultCreateMarketFee = "25000000000ubze"
)

var (
	KeyMarketMakerFee     = []byte("MarketMakerFee")
	DefaultMarketMakerFee = "1000ubze"
)

var (
	KeyMarketTakerFee     = []byte("MarketTakerFee")
	DefaultMarketTakerFee = "100000ubze"
)

var (
	KeyMakerFeeDestination     = []byte("MakerFeeDestination")
	DefaultMakerFeeDestination = FeeDestinationBurnerModule
)

var (
	KeyTakerFeeDestination     = []byte("TakerFeeDestination")
	DefaultTakerFeeDestination = FeeDestinationBurnerModule
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
) Params {
	return Params{
		CreateMarketFee:     createMarketFee,
		MarketMakerFee:      marketMakerFee,
		MarketTakerFee:      marketTakerFee,
		MakerFeeDestination: makerFeeDestination,
		TakerFeeDestination: takerFeeDestination,
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

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
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
