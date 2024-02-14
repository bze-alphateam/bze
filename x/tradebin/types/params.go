package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyCreateMarketFee = []byte("CreateMarketFee")
	// TODO: Determine the default value
	DefaultCreateMarketFee string = "create_market_fee"
)

var (
	KeyMarketMakerFee = []byte("MarketMakerFee")
	// TODO: Determine the default value
	DefaultMarketMakerFee string = "market_maker_fee"
)

var (
	KeyMarketTakerFee = []byte("MarketTakerFee")
	// TODO: Determine the default value
	DefaultMarketTakerFee string = "market_taker_fee"
)

var (
	KeyMakerFeeDestination = []byte("MakerFeeDestination")
	// TODO: Determine the default value
	DefaultMakerFeeDestination string = "maker_fee_destination"
)

var (
	KeyTakerFeeDestination = []byte("TakerFeeDestination")
	// TODO: Determine the default value
	DefaultTakerFeeDestination string = "taker_fee_destination"
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

	// TODO implement validation
	_ = createMarketFee

	return nil
}

// validateMarketMakerFee validates the MarketMakerFee param
func validateMarketMakerFee(v interface{}) error {
	marketMakerFee, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = marketMakerFee

	return nil
}

// validateMarketTakerFee validates the MarketTakerFee param
func validateMarketTakerFee(v interface{}) error {
	marketTakerFee, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = marketTakerFee

	return nil
}

// validateMakerFeeDestination validates the MakerFeeDestination param
func validateMakerFeeDestination(v interface{}) error {
	makerFeeDestination, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = makerFeeDestination

	return nil
}

// validateTakerFeeDestination validates the TakerFeeDestination param
func validateTakerFeeDestination(v interface{}) error {
	takerFeeDestination, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = takerFeeDestination

	return nil
}
