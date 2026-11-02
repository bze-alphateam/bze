package helpers_test

import (
	"encoding/json"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

func TestThreshold(t *testing.T) {
	// exact percents
	require.Equal(t, sdkmath.NewInt(100), helpers.Threshold(sdkmath.NewInt(1000), 10))
	require.True(t, helpers.Threshold(sdkmath.NewInt(1000), 0).IsZero())
	require.Equal(t, sdkmath.NewInt(1000), helpers.Threshold(sdkmath.NewInt(1000), 100))
	// integer division rounds down, like the module: 2% of 150 is 3
	require.Equal(t, sdkmath.NewInt(3), helpers.Threshold(sdkmath.NewInt(150), 2))
	require.True(t, helpers.Threshold(sdkmath.NewInt(99), 1).IsZero())
	// values beyond int64 do not overflow
	big, ok := sdkmath.NewIntFromString("100000000000000000000000")
	require.True(t, ok)
	want, ok := sdkmath.NewIntFromString("1000000000000000000000")
	require.True(t, ok)
	require.Equal(t, want, helpers.Threshold(big, 1))
}

func TestRateLimitDecodes(t *testing.T) {
	const out = `{"path":{"denom":"ubze","channel_or_client_id":"channel-0"},` +
		`"quota":{"max_percent_send":"1","max_percent_recv":"0","duration_hours":"24"},` +
		`"flow":{"inflow":"30","outflow":"100","channel_value":"1000000"}}`
	var rl helpers.RateLimit
	require.NoError(t, json.Unmarshal([]byte(out), &rl))
	require.Equal(t, "ubze", rl.Path.Denom)
	require.Equal(t, "channel-0", rl.Path.ChannelOrClientID)
	require.Equal(t, sdkmath.NewInt(1), rl.Quota.MaxPercentSend)
	require.True(t, rl.Quota.MaxPercentRecv.IsZero())
	require.EqualValues(t, 24, rl.Quota.DurationHours)
	require.Equal(t, sdkmath.NewInt(70), rl.NetOutflow())
	require.True(t, rl.FlowIs(sdkmath.NewInt(30), sdkmath.NewInt(100)))
	require.False(t, rl.FlowIs(sdkmath.NewInt(0), sdkmath.NewInt(100)))
}

func TestRateLimitOmittedFlowReadsAsZero(t *testing.T) {
	var rl helpers.RateLimit
	require.NoError(t, json.Unmarshal([]byte(`{"flow":{"channel_value":"5"}}`), &rl))
	require.True(t, rl.NetOutflow().IsZero())
	require.True(t, rl.FlowIs(sdkmath.ZeroInt(), sdkmath.ZeroInt()))
}
