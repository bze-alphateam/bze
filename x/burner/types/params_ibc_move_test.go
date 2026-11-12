package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/burner/types"
)

func TestParams_DefaultIbcMoveLimits(t *testing.T) {
	p := types.DefaultParams()
	require.True(t, p.IbcMoveMaxReserveRatio.Equal(math.LegacyNewDecWithPrec(3, 2)))
	require.Equal(t, uint64(1000), p.IbcMoveCooldownBlocks)
	require.NoError(t, p.Validate())
}

func TestParams_ValidateIbcMoveLimits(t *testing.T) {
	tests := []struct {
		desc     string
		ratio    math.LegacyDec
		cooldown uint64
		valid    bool
	}{
		{desc: "nil ratio", ratio: math.LegacyDec{}, cooldown: 1},
		{desc: "zero ratio", ratio: math.LegacyZeroDec(), cooldown: 1},
		{desc: "negative ratio", ratio: math.LegacyNewDecWithPrec(-1, 2), cooldown: 1},
		{desc: "ratio above 1", ratio: math.LegacyNewDecWithPrec(1000001, 6), cooldown: 1},
		{desc: "zero cooldown", ratio: math.LegacyNewDecWithPrec(3, 2), cooldown: 0},
		{desc: "smallest ratio and cooldown", ratio: math.LegacySmallestDec(), cooldown: 1, valid: true},
		{desc: "ratio 1", ratio: math.LegacyOneDec(), cooldown: 1_000_000, valid: true},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			p := types.DefaultParams()
			p.IbcMoveMaxReserveRatio = tc.ratio
			p.IbcMoveCooldownBlocks = tc.cooldown
			err := p.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestGenesisState_ValidateIbcMoveLocks(t *testing.T) {
	tests := []struct {
		desc  string
		locks []types.IbcMoveLock
		valid bool
	}{
		{desc: "distinct denoms", locks: []types.IbcMoveLock{{Denom: "ibc/A", LastMoveHeight: 1}, {Denom: "ibc/B", LastMoveHeight: 2}}, valid: true},
		{desc: "duplicate denom", locks: []types.IbcMoveLock{{Denom: "ibc/A", LastMoveHeight: 1}, {Denom: "ibc/A", LastMoveHeight: 2}}},
		{desc: "empty denom", locks: []types.IbcMoveLock{{Denom: "", LastMoveHeight: 1}}},
		{desc: "negative height", locks: []types.IbcMoveLock{{Denom: "ibc/A", LastMoveHeight: -1}}},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			gs := types.DefaultGenesis()
			gs.IbcMoveLocks = tc.locks
			err := gs.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
