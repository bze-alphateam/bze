package types

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// The exact denom values are consensus-critical: every validator must derive the same
// LP denom for the same pool ID. Pin them with fixed vectors so any accidental change
// to the hashing or formatting breaks a test instead of forking the chain.
func TestGetLpDenom_PinnedVectors(t *testing.T) {
	tests := []struct {
		name       string
		poolId     string
		wantDenom  string
		wantScaled string
	}{
		{
			name:       "simple pool id",
			poolId:     "abc_def",
			wantDenom:  "ulp/FA16E98D11CBA9D68283859B3F135969243FE74288468E29022B303559324FA0",
			wantScaled: "lp/FA16E98D11CBA9D68283859B3F135969243FE74288468E29022B303559324FA0",
		},
		{
			name:       "two ibc denoms pool id (the >128 chars regression case)",
			poolId:     "ibc/19488A79F091167225A4BFA34BD3D04F11621E682EB14A58B4CA5D6234BA9487_ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4",
			wantDenom:  "ulp/8DD6964DBD1D912E1AA9FBF53BFDBE46219E48639AE366B9110B3A2C8123B8F5",
			wantScaled: "lp/8DD6964DBD1D912E1AA9FBF53BFDBE46219E48639AE366B9110B3A2C8123B8F5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantDenom, GetLpDenom(tt.poolId))
			require.Equal(t, tt.wantScaled, GetLpScaledDenom(tt.poolId))
		})
	}
}

func TestGetLpDenom_Properties(t *testing.T) {
	// worst case today: two max-size denoms (IBC-style, 68 chars each) joined by "_"
	longPoolId := strings.Repeat("a", 68) + "_" + strings.Repeat("b", 68)

	for _, poolId := range []string{"abc_def", "ubze_uusdc", longPoolId} {
		base := GetLpDenom(poolId)
		scaled := GetLpScaledDenom(poolId)

		require.NoError(t, sdk.ValidateDenom(base), "base LP denom must be a valid SDK denom")
		require.NoError(t, sdk.ValidateDenom(scaled), "scaled LP denom must be a valid SDK denom")
		// fixed length regardless of the pool id: "ulp/" + 64 hex chars
		require.Len(t, base, 68)
		require.Len(t, scaled, 67)
		require.Equal(t, "u"+scaled, base)
		// deterministic
		require.Equal(t, base, GetLpDenom(poolId))
	}

	// different pool ids must not collide
	require.NotEqual(t, GetLpDenom("abc_def"), GetLpDenom("abc_deg"))
}
