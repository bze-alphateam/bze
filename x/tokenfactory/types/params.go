package types

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyCreateDenomFee           = []byte("CreateDenomFee")
	DefaultCreateDenomFee int64 = 25000000000
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	createDenomFee sdk.Coin,
) Params {
	return Params{
		CreateDenomFee: createDenomFee,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		sdk.NewInt64Coin("ubze", DefaultCreateDenomFee),
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

// validateCreateDenomFee validates the CreateDenomFee param
func validateCreateDenomFee(v interface{}) error {
	createDenomFee, ok := v.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if !createDenomFee.IsValid() {
		return fmt.Errorf("invalid CreateDenomFee: %s", createDenomFee)
	}

	return nil
}
