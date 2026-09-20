package v2types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultParams_Validate(t *testing.T) {
	p := DefaultParams()
	require.Empty(t, p.HaltedDenoms, "the halt list ships empty")
	require.NoError(t, p.Validate())
}

func TestParams_Validate_HaltedDenoms(t *testing.T) {
	tooMany := make([]string, MaxHaltedDenoms+1)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("factory/bze1creator/denom%d", i)
	}
	atTheLimit := tooMany[:MaxHaltedDenoms]

	tests := []struct {
		name         string
		haltedDenoms []string
		nativeDenom  string
		wantErr      string
	}{
		{
			name:         "empty list is valid",
			haltedDenoms: nil,
			nativeDenom:  DefaultNativeDenom,
		},
		{
			name:         "explicit empty list is valid",
			haltedDenoms: []string{},
			nativeDenom:  DefaultNativeDenom,
		},
		{
			name: "valid list of ibc and factory denoms",
			haltedDenoms: []string{
				"ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4",
				"factory/bze1creator/token",
				"uatom",
			},
			nativeDenom: DefaultNativeDenom,
		},
		{
			name:         "exactly the maximum number of entries is valid",
			haltedDenoms: atTheLimit,
			nativeDenom:  DefaultNativeDenom,
		},
		{
			name:         "native denom is rejected",
			haltedDenoms: []string{"uatom", DefaultNativeDenom},
			nativeDenom:  DefaultNativeDenom,
			wantErr:      "the native denom ubze cannot be halted",
		},
		{
			name:         "native denom is rejected whatever it is",
			haltedDenoms: []string{"stake"},
			nativeDenom:  "stake",
			wantErr:      "the native denom stake cannot be halted",
		},
		{
			name:         "duplicate is rejected",
			haltedDenoms: []string{"uatom", "uosmo", "uatom"},
			nativeDenom:  DefaultNativeDenom,
			wantErr:      "duplicate halted denom uatom",
		},
		{
			name:         "malformed denom is rejected",
			haltedDenoms: []string{"not a denom"},
			nativeDenom:  DefaultNativeDenom,
			wantErr:      `invalid halted denom "not a denom"`,
		},
		{
			name:         "empty string is rejected",
			haltedDenoms: []string{""},
			nativeDenom:  DefaultNativeDenom,
			wantErr:      `invalid halted denom ""`,
		},
		{
			name:         "one entry over the maximum is rejected",
			haltedDenoms: tooMany,
			nativeDenom:  DefaultNativeDenom,
			wantErr:      fmt.Sprintf("at most %d halted denoms are allowed, got %d", MaxHaltedDenoms, MaxHaltedDenoms+1),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := DefaultParams()
			p.NativeDenom = tc.nativeDenom
			p.HaltedDenoms = tc.haltedDenoms

			err := p.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid HaltedDenoms")
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestParams_IsDenomHalted_ExactMatchOnly(t *testing.T) {
	p := DefaultParams()
	p.HaltedDenoms = []string{"uusdc", "ibc/ABCD"}

	require.True(t, p.IsDenomHalted("uusdc"))
	require.True(t, p.IsDenomHalted("ibc/ABCD"))

	require.False(t, p.IsDenomHalted("ubze"))
	require.False(t, p.IsDenomHalted(""))
	// market and pool ids that merely contain a halted denom never match
	require.False(t, p.IsDenomHalted("ubze/uusdc"))
	require.False(t, p.IsDenomHalted("ubze_uusdc"))
	require.False(t, p.IsDenomHalted("uusd"))
	require.False(t, p.IsDenomHalted("uusdcx"))

	require.False(t, DefaultParams().IsDenomHalted("uusdc"), "nothing is halted by default")
}
