package types_test

import (
	"strings"
	"testing"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/stretchr/testify/require"
)

// The pool price average prefix must not collide with another module key: none is a prefix of it and it is
// a prefix of none.
func TestPoolPriceAveragePrefixDoesNotCollide(t *testing.T) {
	others := []string{
		types.LpKeyPrefix, types.HaltedDenomKeyPrefix, types.UserDustKeyPrefix,
		types.MarketKeyPrefix, types.MarketAliasKeyPrefix,
		types.OrderKeyPrefix, types.UserOrderKeyPrefix, types.PriceOrderKeyPrefix, types.AggOrderKeyPrefix,
		types.HistoryOrderKeyPrefix, types.OrderCounterPrefix, types.OwnerPriceKeyPrefix,
		string(types.ParamsKey),
		"Qm/value/", "Qm/counter/", "Qm/pc/", // queue keys removed by the v5 migration
	}
	mine := types.PoolPriceAverageKeyPrefix
	for _, o := range others {
		require.False(t, strings.HasPrefix(mine, o), "%q is a prefix of %q", o, mine)
		require.False(t, strings.HasPrefix(o, mine), "%q is a prefix of %q", mine, o)
	}
}
