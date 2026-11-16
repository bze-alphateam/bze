package bzeutils_test

import (
	"testing"

	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/stretchr/testify/require"
)

func TestIsLpTokenDenom(t *testing.T) {
	tests := []struct {
		denom string
		want  bool
	}{
		{"ulp/0A1B2C3D4E5F", true},
		{"ulp_ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4_ubze", true},
		{"ulp_utoken_ubze", true},
		{"ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4", false},
		{"factory/bze1abc/token", false},
		{"ubze", false},
		{"ulp", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.denom, func(t *testing.T) {
			require.Equal(t, tt.want, bzeutils.IsLpTokenDenom(tt.denom))
		})
	}
}
