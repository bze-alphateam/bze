package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	DefaultDenom                 = "ubze"
	DefaultAnonArticleCostAmount = 25000000000
)

var (
	KeyAnonArticleLimit            = []byte("AnonArticleLimit")
	DefaultAnonArticleLimit uint64 = 5

	KeyAnonArticleCost     = []byte("AnonArticleCost")
	DefaultAnonArticleCost = sdk.NewCoin(DefaultDenom, math.NewInt(DefaultAnonArticleCostAmount))

	KeyPublisherRespectParams     = []byte("PublisherRespectParams")
	DefaultPublisherRespectParams = PublisherRespectParams{
		Denom: DefaultDenom,
		Tax:   math.LegacyNewDecWithPrec(20, 2), //20%
	}
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	anonArticleLimit uint64,
	anonArticleCost sdk.Coin,
	PublisherRespectParams PublisherRespectParams,
) Params {
	return Params{
		AnonArticleLimit:       anonArticleLimit,
		AnonArticleCost:        anonArticleCost,
		PublisherRespectParams: PublisherRespectParams,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultAnonArticleLimit,
		DefaultAnonArticleCost,
		DefaultPublisherRespectParams,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyAnonArticleLimit, &p.AnonArticleLimit, validateAnonArticleLimit),
		paramtypes.NewParamSetPair(KeyAnonArticleCost, &p.AnonArticleCost, validateAnonArticleCost),
		paramtypes.NewParamSetPair(KeyPublisherRespectParams, &p.PublisherRespectParams, validatePublisherRespectParams),
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

	if err := validatePublisherRespectParams(p.PublisherRespectParams); err != nil {
		return err
	}

	return nil
}

// validateAnonArticleLimit validates the AnonArticleLimit param
func validateAnonArticleLimit(v interface{}) error {
	anonArticleLimit, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if anonArticleLimit < 0 {
		return fmt.Errorf("invalid anonArticleLimit. Expected uint64 higher than or equal with 0 received %v", anonArticleLimit)
	}

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

// validateAnonArticleCost validates the AnonArticleCost param
func validatePublisherRespectParams(v interface{}) error {
	publisherRespectParams, ok := v.(PublisherRespectParams)
	if !ok {
		return fmt.Errorf("invalid parameter publisherRespectParams type: %T", v)
	}

	if !publisherRespectParams.Tax.IsPositive() {
		return fmt.Errorf("publisherRespectParams tax should be positive: %s", publisherRespectParams.Tax.String())
	}

	err := sdk.ValidateDenom(publisherRespectParams.Denom)
	if err != nil {
		return err
	}

	return nil
}
