package v1types

import (
	"fmt"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyCreateDenomFee = []byte("CreateDenomFee")
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
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

// validateCreateDenomFee validates the CreateDenomFee param
func validateCreateDenomFee(v interface{}) error {
	createDenomFee, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if createDenomFee == "" {
		return fmt.Errorf("invalid CreateDenomFee: %s", createDenomFee)
	}

	return nil
}
