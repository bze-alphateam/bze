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
	// DefaultCreateBoostFee is the default fee to create a boost — 25,000 BZE,
	// same as the staking/trading reward creation fee.
	DefaultCreateBoostFee sdk.Coin = sdk.NewInt64Coin("ubze", 25_000_000000)
)

const (
	DefaultExtraGasForExitStake uint64 = 1_000_000
	// DefaultMaxBoostsPerReward is the default cap on concurrently existing boost
	// records per reward.
	DefaultMaxBoostsPerReward uint32 = 10
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
	p := NewParams(
		DefaultCreateRewardFee,
		DefaultCreateRewardFee,
		DefaultExtraGasForExitStake,
	)
	p.CreateBoostFee = DefaultCreateBoostFee
	p.MaxBoostsPerReward = DefaultMaxBoostsPerReward

	return p
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

	if err := validateExtraGasForExitStake(p.ExtraGasForExitStake); err != nil {
		return err
	}

	if err := validateCreateBoostFee(p.CreateBoostFee); err != nil {
		return err
	}

	if err := validateMaxBoostsPerReward(p.MaxBoostsPerReward); err != nil {
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

// validateExtraGasForExitStake validates the ExtraGasForExitStake param
func validateExtraGasForExitStake(v interface{}) error {
	_, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}

// validateCreateBoostFee validates the CreateBoostFee param
func validateCreateBoostFee(v interface{}) error {
	createBoostFee, ok := v.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if !createBoostFee.IsValid() {
		return fmt.Errorf("invalid CreateBoostFee: %s", createBoostFee)
	}

	return nil
}

// validateMaxBoostsPerReward validates the MaxBoostsPerReward param
func validateMaxBoostsPerReward(v interface{}) error {
	maxBoostsPerReward, ok := v.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if maxBoostsPerReward == 0 {
		return fmt.Errorf("invalid MaxBoostsPerReward: must be greater than 0")
	}

	return nil
}
