package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	//periodic_burning_weeks
	KeyPeriodicBurningWeeks = []byte("PeriodicBurningWeeks")
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(periodicBurningWeeks int64) Params {
	return Params{
		PeriodicBurningWeeks: periodicBurningWeeks,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(1)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyPeriodicBurningWeeks, &p.PeriodicBurningWeeks, validatePeriodicBurningWeeks),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validatePeriodicBurningWeeks(p.PeriodicBurningWeeks); err != nil {
		return fmt.Errorf("invalid burning period weeks: %d", p.PeriodicBurningWeeks)
	}

	return nil
}

func validatePeriodicBurningWeeks(i interface{}) error {
	v, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v <= 0 {
		return fmt.Errorf("invalid burning period weeks: %d", v)
	}

	return nil
}
