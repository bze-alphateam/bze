package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/stretchr/testify/require"
)

// Pins the duration bounds of ValidateBasic: 1 to HundredYearsInDays days, parsed as a 32-bit integer
// like the message server.
func TestCreateDenomRewardSchedule_DurationBounds(t *testing.T) {
	msg := func(duration string) *types.MsgCreateDenomRewardSchedule {
		return types.NewMsgCreateDenomRewardSchedule(sample.AccAddress(), "ubze", "ubze", math.NewInt(100), duration)
	}

	for _, d := range []string{"2147483648", "3000000000", "9223372036854775807", "99999999999999999999", "36501", "0", "-1"} {
		t.Run(d, func(t *testing.T) {
			require.ErrorIs(t, msg(d).ValidateBasic(), types.ErrInvalidDuration, "duration %q must be refused", d)
		})
	}

	for _, d := range []string{"1", "365", "36500"} {
		t.Run(d, func(t *testing.T) {
			require.NoError(t, msg(d).ValidateBasic())
		})
	}
}
