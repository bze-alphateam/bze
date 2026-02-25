package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const DefaultMaxBalanceIterations = uint64(100)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(validatorMinGasFee sdk.DecCoin, maxBalanceIterations uint64) Params {
	return Params{
		ValidatorMinGasFee:   validatorMinGasFee,
		MaxBalanceIterations: maxBalanceIterations,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		sdk.NewDecCoinFromDec("ubze", sdkmath.LegacyNewDecWithPrec(1, 2)), // 0.01ubze
		DefaultMaxBalanceIterations,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateValidatorMinGasFee(p.ValidatorMinGasFee); err != nil {
		return err
	}

	if err := validateMaxBalanceIterations(p.MaxBalanceIterations); err != nil {
		return err
	}

	return nil
}

func validateValidatorMinGasFee(i interface{}) error {
	v, ok := i.(sdk.DecCoin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.Denom != "ubze" {
		return fmt.Errorf("validator min gas fee denom must be ubze")
	}

	if v.Amount.IsNegative() {
		return fmt.Errorf("validator min gas fee amount cannot be negative: %s", v.Amount)
	}

	if !v.IsValid() {
		return fmt.Errorf("invalid validator min gas fee: %s", v)
	}

	return nil
}

func validateMaxBalanceIterations(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("max balance iterations must be greater than 0")
	}

	return nil
}
