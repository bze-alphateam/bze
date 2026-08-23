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

	// Denom Rewards defaults
	DefaultAddDenomRewardScheduleFee sdk.Coin = sdk.NewInt64Coin("ubze", 25_000_000000)
)

const (
	DefaultExtraGasForExitStake uint64 = 1_000_000

	// Denom Rewards defaults
	DefaultMaxPrizeDenomsPerDr  uint32 = 50
	DefaultExtraGasForDenomExit uint64 = 1_000_000
	DefaultDenomRewardLock      uint32 = 7
	DefaultDenomRewardMinStake  uint64 = 0
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
	// NewParams keeps the original v3->v4 signature (still used by the frozen v3
	// migrator). The additive Denom Rewards params are set directly here.
	p := NewParams(
		DefaultCreateRewardFee,
		DefaultCreateRewardFee,
		DefaultExtraGasForExitStake,
	)
	p.CreateDenomRewardFee = DefaultCreateRewardFee
	p.CreateDenomRewardPrizeFee = DefaultCreateRewardFee
	p.AddDenomRewardScheduleFee = DefaultAddDenomRewardScheduleFee
	p.MaxPrizeDenomsPerDr = DefaultMaxPrizeDenomsPerDr
	p.ExtraGasForDenomExit = DefaultExtraGasForDenomExit
	p.DenomRewardLock = DefaultDenomRewardLock
	p.DenomRewardMinStake = DefaultDenomRewardMinStake

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

	if err := validateFeeCoin(p.CreateDenomRewardFee); err != nil {
		return err
	}

	if err := validateFeeCoin(p.CreateDenomRewardPrizeFee); err != nil {
		return err
	}

	if err := validateFeeCoin(p.AddDenomRewardScheduleFee); err != nil {
		return err
	}

	if err := validateUint32(p.MaxPrizeDenomsPerDr); err != nil {
		return err
	}

	if err := validateUint64(p.ExtraGasForDenomExit); err != nil {
		return err
	}

	if err := validateUint32(p.DenomRewardLock); err != nil {
		return err
	}

	if err := validateUint64(p.DenomRewardMinStake); err != nil {
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

// validateFeeCoin validates a Coin-typed fee param (shared by the Denom Rewards fees).
func validateFeeCoin(v interface{}) error {
	fee, ok := v.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if !fee.IsValid() {
		return fmt.Errorf("invalid fee coin: %s", fee)
	}

	return nil
}

// validateUint32 validates a uint32-typed param.
func validateUint32(v interface{}) error {
	_, ok := v.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}

// validateUint64 validates a uint64-typed param.
func validateUint64(v interface{}) error {
	_, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}
