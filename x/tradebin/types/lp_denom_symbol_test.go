package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// The symbol is "LP-" plus the first 6 uppercase hex characters of sha256(poolId), pinned with fixed vectors.
func TestLpDenomSymbol_PinnedVectors(t *testing.T) {
	require.Equal(t, "LP-FA16E9", types.LpDenomSymbol("abc_def"))
	require.Equal(t, "LP-A376DC", types.LpDenomSymbol("ubze_uusdc"))
	// legacy pools get a symbol from the same hash of their pool ID
	require.Equal(t, "LP-3E74C7", types.LpDenomSymbol("factory/bze1creator/token_ubze"))
}

func TestLpDenomSymbol_DistinctPerPool(t *testing.T) {
	require.NotEqual(t, types.LpDenomSymbol("ubze_uusdc"), types.LpDenomSymbol("abc_def"))
	require.NotEqual(t, types.LpDenomSymbol("factory/bze1creator/token_ubze"), types.LpDenomSymbol("uatom_ubze"))
}
