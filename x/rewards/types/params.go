package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)


var (
	KeyCreateStakingRewardFee = []byte("CreateStakingRewardFee")
	// TODO: Determine the default value
	DefaultCreateStakingRewardFee string = "create_staking_reward_fee"
)

var (
	KeyCreateTradingRewardFee = []byte("CreateTradingRewardFee")
	// TODO: Determine the default value
	DefaultCreateTradingRewardFee string = "create_trading_reward_fee"
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
