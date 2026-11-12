package helpers_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

func TestDecParamsForTx(t *testing.T) {
	params := map[string]json.RawMessage{
		"feeMaxPriceDrift":    json.RawMessage(`"100000000000000000"`),
		"moduleMaxPriceDrift": json.RawMessage(`"0.030000000000000000"`),
		"maxFillsPerOrder":    json.RawMessage(`"100000000000000000"`),
	}
	require.NoError(t, helpers.DecParamsForTx(params, append(helpers.TradebinDecParams, "notThere")...))
	require.JSONEq(t, `"0.100000000000000000"`, string(params["feeMaxPriceDrift"]))
	require.JSONEq(t, `"0.030000000000000000"`, string(params["moduleMaxPriceDrift"]))
	require.JSONEq(t, `"100000000000000000"`, string(params["maxFillsPerOrder"]), "not a Dec field: unchanged")
	require.NotContains(t, params, "notThere")

	for _, bad := range []string{`0.1`, `"abc"`, `""`} {
		err := helpers.DecParamsForTx(map[string]json.RawMessage{"x": json.RawMessage(bad)}, "x")
		require.Error(t, err, bad)
	}
}
