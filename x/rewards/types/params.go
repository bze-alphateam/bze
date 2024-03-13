package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyCreateStakingRewardFee     = []byte("CreateStakingRewardFee")
	DefaultCreateStakingRewardFee = "10000000000ubze"
)

var (
	KeyCreateTradingRewardFee     = []byte("CreateTradingRewardFee")
	DefaultCreateTradingRewardFee = "10000000000ubze"
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	createStakingRewardFee string,
	createTradingRewardFee string,
) Params {
	return Params{
		CreateStakingRewardFee: createStakingRewardFee,
		CreateTradingRewardFee: createTradingRewardFee,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultCreateStakingRewardFee,
		DefaultCreateTradingRewardFee,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyCreateStakingRewardFee, &p.CreateStakingRewardFee, validateCreateStakingRewardFee),
		paramtypes.NewParamSetPair(KeyCreateTradingRewardFee, &p.CreateTradingRewardFee, validateCreateTradingRewardFee),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateCreateStakingRewardFee(p.CreateStakingRewardFee); err != nil {
		return err
	}

	if err := validateCreateTradingRewardFee(p.CreateTradingRewardFee); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateCreateStakingRewardFee validates the CreateStakingRewardFee param
func validateCreateStakingRewardFee(v interface{}) error {
	createStakingRewardFee, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = createStakingRewardFee

	return nil
}

// validateCreateTradingRewardFee validates the CreateTradingRewardFee param
func validateCreateTradingRewardFee(v interface{}) error {
	createTradingRewardFee, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = createTradingRewardFee

	return nil
}
