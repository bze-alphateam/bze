package v1types

import (
	"fmt"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyCreateTradingRewardFee = []byte("CreateTradingRewardFee")
	KeyCreateStakingRewardFee = []byte("CreateStakingRewardFee")
)

func (m *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyCreateStakingRewardFee, &m.CreateStakingRewardFee, validateCreateStakingRewardFee),
		paramtypes.NewParamSetPair(KeyCreateTradingRewardFee, &m.CreateTradingRewardFee, validateCreateTradingRewardFee),
	}
}

// Validate validates the set of params
func (m Params) Validate() error {
	if err := validateCreateStakingRewardFee(m.CreateStakingRewardFee); err != nil {
		return err
	}

	if err := validateCreateTradingRewardFee(m.CreateTradingRewardFee); err != nil {
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

	if createStakingRewardFee == "" {
		return fmt.Errorf("invalid CreateStakingRewardFee: %s", createStakingRewardFee)
	}

	return nil
}

// validateCreateTradingRewardFee validates the CreateTradingRewardFee param
func validateCreateTradingRewardFee(v interface{}) error {
	createTradingRewardFee, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if createTradingRewardFee == "" {
		return fmt.Errorf("invalid createTradingRewardFee: %s", createTradingRewardFee)
	}

	return nil
}

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}
