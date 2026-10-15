package v800

import (
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// The legacy x/params key table of x/mint, as shipped by cosmos-sdk v0.50.
// cosmos-sdk v0.53 removed minttypes.ParamKeyTable; this upgrade ran on
// cosmos-sdk v0.50, so the table is kept here unchanged to keep the
// historical handler compiling.

var (
	keyMintDenom           = []byte("MintDenom")
	keyInflationRateChange = []byte("InflationRateChange")
	keyInflationMax        = []byte("InflationMax")
	keyInflationMin        = []byte("InflationMin")
	keyGoalBonded          = []byte("GoalBonded")
	keyBlocksPerYear       = []byte("BlocksPerYear")
)

type legacyMintParams struct {
	minttypes.Params
}

func mintParamKeyTable() paramstypes.KeyTable {
	return paramstypes.NewKeyTable().RegisterParamSet(&legacyMintParams{})
}

func (p *legacyMintParams) ParamSetPairs() paramstypes.ParamSetPairs {
	return paramstypes.ParamSetPairs{
		paramstypes.NewParamSetPair(keyMintDenom, &p.MintDenom, validateMintDenom),
		paramstypes.NewParamSetPair(keyInflationRateChange, &p.InflationRateChange, validateUnitDec("inflation rate change", false)),
		paramstypes.NewParamSetPair(keyInflationMax, &p.InflationMax, validateUnitDec("max inflation", false)),
		paramstypes.NewParamSetPair(keyInflationMin, &p.InflationMin, validateUnitDec("min inflation", false)),
		paramstypes.NewParamSetPair(keyGoalBonded, &p.GoalBonded, validateUnitDec("goal bonded", true)),
		paramstypes.NewParamSetPair(keyBlocksPerYear, &p.BlocksPerYear, validateBlocksPerYear),
	}
}

func validateMintDenom(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if strings.TrimSpace(v) == "" {
		return errors.New("mint denom cannot be blank")
	}

	return sdk.ValidateDenom(v)
}

// validateUnitDec checks a decimal lies in [0, 1], or in (0, 1] when positive is set.
func validateUnitDec(name string, positive bool) func(interface{}) error {
	return func(i interface{}) error {
		v, ok := i.(math.LegacyDec)
		if !ok {
			return fmt.Errorf("invalid parameter type: %T", i)
		}
		if v.IsNil() {
			return fmt.Errorf("%s cannot be nil: %s", name, v)
		}
		if v.IsNegative() || (positive && v.IsZero()) {
			return fmt.Errorf("%s out of range: %s", name, v)
		}
		if v.GT(math.LegacyOneDec()) {
			return fmt.Errorf("%s too large: %s", name, v)
		}

		return nil
	}
}

func validateBlocksPerYear(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("blocks per year must be positive: %d", v)
	}

	return nil
}
