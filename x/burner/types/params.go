package types

import (
	"fmt"

	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	//periodic_burning_weeks
	KeyPeriodicBurningWeeks = []byte("PeriodicBurningWeeks")

	// DefaultIbcMoveMaxReserveRatio is the default share of the pool's IBC reserve one MsgMoveIbcLockedCoins moves at most.
	DefaultIbcMoveMaxReserveRatio = math.LegacyNewDecWithPrec(3, 2)
)

const (
	// DefaultIbcMoveCooldownBlocks is the default number of blocks between two moves of the same denom.
	DefaultIbcMoveCooldownBlocks uint64 = 1000
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance with the given burning period and the default limits of
// MsgMoveIbcLockedCoins.
func NewParams(periodicBurningWeeks int64) Params {
	return Params{
		PeriodicBurningWeeks:   periodicBurningWeeks,
		IbcMoveMaxReserveRatio: DefaultIbcMoveMaxReserveRatio,
		IbcMoveCooldownBlocks:  DefaultIbcMoveCooldownBlocks,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(1)
}

// ParamSetPairs get the params.ParamSet. It is used only to read the legacy x/params subspace, which never held the
// MsgMoveIbcLockedCoins limits.
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

	if err := validateIbcMoveMaxReserveRatio(p.IbcMoveMaxReserveRatio); err != nil {
		return err
	}

	return validateIbcMoveCooldownBlocks(p.IbcMoveCooldownBlocks)
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

func validateIbcMoveMaxReserveRatio(v math.LegacyDec) error {
	if v.IsNil() {
		return fmt.Errorf("ibc move max reserve ratio is required")
	}

	if !v.IsPositive() || v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("ibc move max reserve ratio must be in (0, 1]: %s", v)
	}

	return nil
}

func validateIbcMoveCooldownBlocks(v uint64) error {
	if v == 0 {
		return fmt.Errorf("ibc move cooldown blocks must be at least 1")
	}

	return nil
}
