package types

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	DefaultDenom                 = "ubze"
	DefaultAnonArticleCostAmount = 25000000000
)

var (
	KeyAnonArticleLimit            = []byte("AnonArticleLimit")
	DefaultAnonArticleLimit uint64 = 5
)

var (
	KeyAnonArticleCost     = []byte("AnonArticleCost")
	DefaultAnonArticleCost = sdk.NewCoin(DefaultDenom, sdk.NewInt(DefaultAnonArticleCostAmount))
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	anonArticleLimit uint64,
	anonArticleCost sdk.Coin,
) Params {
	return Params{
		AnonArticleLimit: anonArticleLimit,
		AnonArticleCost:  anonArticleCost,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultAnonArticleLimit,
		DefaultAnonArticleCost,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyAnonArticleLimit, &p.AnonArticleLimit, validateAnonArticleLimit),
		paramtypes.NewParamSetPair(KeyAnonArticleCost, &p.AnonArticleCost, validateAnonArticleCost),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateAnonArticleLimit(p.AnonArticleLimit); err != nil {
		return err
	}

	if err := validateAnonArticleCost(p.AnonArticleCost); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateAnonArticleLimit validates the AnonArticleLimit param
func validateAnonArticleLimit(v interface{}) error {
	anonArticleLimit, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if anonArticleLimit < 1 {
		return fmt.Errorf("invalid anonArticleLimit. Expected uint64 higher than 0 received %v", anonArticleLimit)
	}
	_ = anonArticleLimit

	return nil
}

// validateAnonArticleCost validates the AnonArticleCost param
func validateAnonArticleCost(v interface{}) error {
	anonArticleCost, ok := v.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter anonArticleLimit type: %T", v)
	}
	if !anonArticleCost.IsValid() {
		return fmt.Errorf("invalid anonArticleLimit coin: %s", anonArticleCost.String())
	}

	return nil
}
