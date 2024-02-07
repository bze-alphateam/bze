package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyCreateDenomFee            = []byte("CreateDenomFee")
	DefaultCreateDenomFee string = "10000000000ubze"
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	createDenomFee string,
) Params {
	return Params{
		CreateDenomFee: createDenomFee,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultCreateDenomFee,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyCreateDenomFee, &p.CreateDenomFee, validateCreateDenomFee),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateCreateDenomFee(p.CreateDenomFee); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateCreateDenomFee validates the CreateDenomFee param
func validateCreateDenomFee(v interface{}) error {
	createDenomFee, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = createDenomFee

	return nil
}
