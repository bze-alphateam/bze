package types

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyCreateTradingRewardFee          = []byte("CreateTradingRewardFee")
	KeyCreateStakingRewardFee          = []byte("CreateStakingRewardFee")
	DefaultCreateRewardFee    sdk.Coin = sdk.NewInt64Coin("ubze", 25_000_000000)
)

const (
	DefaultExtraGasForExitStake uint64 = 1_000_000
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	createStakingRewardFee sdk.Coin,
	createTradingRewardFee sdk.Coin,
	extraGasForExitStake uint64,
) Params {
	return Params{
		CreateStakingRewardFee: createStakingRewardFee,
		CreateTradingRewardFee: createTradingRewardFee,
		ExtraGasForExitStake:   extraGasForExitStake,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultCreateRewardFee,
		DefaultCreateRewardFee,
		DefaultExtraGasForExitStake,
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

// validateCreateStakingRewardFee validates the CreateStakingRewardFee param
func validateCreateStakingRewardFee(v interface{}) error {
	createStakingRewardFee, ok := v.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if !createStakingRewardFee.IsValid() {
		return fmt.Errorf("invalid CreateStakingRewardFee: %s", createStakingRewardFee)
	}

	return nil
}

// validateCreateTradingRewardFee validates the CreateTradingRewardFee param
func validateCreateTradingRewardFee(v interface{}) error {
	createTradingRewardFee, ok := v.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if !createTradingRewardFee.IsValid() {
		return fmt.Errorf("invalid createTradingRewardFee: %s", createTradingRewardFee)
	}

	return nil
}
