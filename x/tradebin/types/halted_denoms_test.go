package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

func TestValidateHaltedDenoms(t *testing.T) {
	many := make([]string, 1500)
	for i := range many {
		many[i] = fmt.Sprintf("udenom%d", i)
	}

	tests := []struct {
		name   string
		denoms []string
		max    int
		native string
		errMsg string
	}{
		{"empty list is valid", nil, 0, "ubze", ""},
		{"valid denoms", []string{"uusdc", "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4", "factory/bze1m33n82r5x3eyjmjtwjkl82zzdlrnv8pevd8u9r/spam"}, 0, "ubze", ""},
		{"no cap when max is 0", many, 0, "ubze", ""},
		{"within the cap", many[:1000], 1000, "ubze", ""},
		{"over the cap", many[:1001], 1000, "ubze", "at most 1000 denoms"},
		{"malformed denom", []string{"1bad"}, 0, "ubze", "invalid denom"},
		{"empty denom", []string{""}, 0, "ubze", "invalid denom"},
		{"native denom refused", []string{"uusdc", "ubze"}, 0, "ubze", "cannot be halted"},
		{"native check off when no native denom is given", []string{"ubze"}, 0, "", ""},
		{"duplicate", []string{"uusdc", "uusdc"}, 0, "ubze", "duplicate denom"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := types.ValidateHaltedDenoms(tc.denoms, tc.max, tc.native)
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}
