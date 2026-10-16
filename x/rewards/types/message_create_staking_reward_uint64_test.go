package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// min_stake is a string in the tx but a uint64 in the store and the event: a
// value past uint64 is refused instead of panicking in the conversion.
func TestMsgCreateStakingReward_MinStakeUint64Bound(t *testing.T) {
	msg := func(minStake string) *types.MsgCreateStakingReward {
		return types.NewMsgCreateStakingReward(sample.AccAddress(), "1000", "utoken", "ustake", "30", minStake, "7")
	}

	t.Run("max uint64 is accepted", func(t *testing.T) {
		m := msg("18446744073709551615")
		require.NoError(t, m.ValidateBasic())
		sr, err := m.ToStakingReward()
		require.NoError(t, err)
		require.Equal(t, uint64(18446744073709551615), sr.MinStake)
	})

	for _, minStake := range []string{"18446744073709551616", "1000000000000000000000000000000"} {
		t.Run(minStake+" is refused", func(t *testing.T) {
			m := msg(minStake)
			require.NotPanics(t, func() {
				require.ErrorIs(t, m.ValidateBasic(), types.ErrInvalidMinStake)
			})
			_, err := m.ToStakingReward()
			require.ErrorIs(t, err, types.ErrInvalidMinStake)
		})
	}
}
