package types_test

import (
	"math/big"
	"testing"

	"github.com/bze-alphateam/bze/bzeutils"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// Amounts must be canonical base-10 integers.
var nonCanonicalAmounts = []string{"0100", "0x10", "0o10", "0b10", "1_0", "+10", "00", "-0"}

func TestAmountFormat_CreateStakingRewardRejectsNonCanonical(t *testing.T) {
	for _, amount := range nonCanonicalAmounts {
		t.Run(amount, func(t *testing.T) {
			msg := types.NewMsgCreateStakingReward(sample.AccAddress(), amount, "ubze", "ubze", "5", "0", "0")
			_, err := msg.ToStakingReward()
			require.ErrorIs(t, err, types.ErrInvalidAmount, "prize amount %q", amount)
			require.ErrorIs(t, msg.ValidateBasic(), types.ErrInvalidAmount, "prize amount %q", amount)
		})
	}
}

func TestAmountFormat_CreateStakingRewardRejectsNonCanonicalMinStake(t *testing.T) {
	for _, minStake := range append([]string{"010"}, nonCanonicalAmounts...) {
		t.Run(minStake, func(t *testing.T) {
			msg := types.NewMsgCreateStakingReward(sample.AccAddress(), "100", "ubze", "ubze", "5", minStake, "0")
			require.ErrorIs(t, msg.ValidateBasic(), types.ErrInvalidMinStake, "min stake %q", minStake)
		})
	}

	msg := types.NewMsgCreateStakingReward(sample.AccAddress(), "100", "ubze", "ubze", "5", "0", "0")
	require.NoError(t, msg.ValidateBasic(), "a zero min stake stays valid")
}

func TestAmountFormat_DistributeStakingRewardsRejectsNonCanonical(t *testing.T) {
	for _, amount := range nonCanonicalAmounts {
		t.Run(amount, func(t *testing.T) {
			msg := types.NewMsgDistributeStakingRewards(sample.AccAddress(), "1", amount)
			require.ErrorIs(t, msg.ValidateBasic(), types.ErrInvalidAmount, "amount %q", amount)
		})
	}
}

func TestAmountFormat_JoinStakingRejectsNonCanonical(t *testing.T) {
	for _, amount := range nonCanonicalAmounts {
		t.Run(amount, func(t *testing.T) {
			msg := types.NewMsgJoinStaking(sample.AccAddress(), "1", amount)
			require.ErrorIs(t, msg.ValidateBasic(), types.ErrInvalidAmount, "amount %q", amount)
		})
	}
}

func TestAmountFormat_CreateTradingRewardRejectsNonCanonical(t *testing.T) {
	for _, amount := range nonCanonicalAmounts {
		t.Run(amount, func(t *testing.T) {
			msg := types.NewMsgCreateTradingReward(sample.AccAddress(), amount, "ubze", "5", "market", "1")
			require.ErrorIs(t, msg.ValidateBasic(), types.ErrInvalidAmount, "prize amount %q", amount)
		})
	}
}

func TestAmountFormat_CanonicalAmountsAccepted(t *testing.T) {
	for _, amount := range []string{"1", "100", "1000000000000000000000"} {
		require.NoError(t, types.NewMsgCreateStakingReward(sample.AccAddress(), amount, "ubze", "ubze", "5", "0", "0").ValidateBasic(), amount)
		require.NoError(t, types.NewMsgDistributeStakingRewards(sample.AccAddress(), "1", amount).ValidateBasic(), amount)
		require.NoError(t, types.NewMsgJoinStaking(sample.AccAddress(), "1", amount).ValidateBasic(), amount)
		require.NoError(t, types.NewMsgCreateTradingReward(sample.AccAddress(), amount, "ubze", "5", "market", "1").ValidateBasic(), amount)
	}
}

// The largest canonical 256-bit value is accepted; one beyond it is refused.
func TestAmountFormat_BitLengthBoundary(t *testing.T) {
	maxAmount := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).String()
	tooLarge := new(big.Int).Lsh(big.NewInt(1), 256).String()

	amt, err := bzeutils.ParseCanonicalAmount(maxAmount)
	require.NoError(t, err)
	require.Equal(t, maxAmount, amt.String())
	require.NoError(t, types.NewMsgDistributeStakingRewards(sample.AccAddress(), "1", maxAmount).ValidateBasic())

	_, err = bzeutils.ParseCanonicalAmount(tooLarge)
	require.Error(t, err)
	require.ErrorIs(t, types.NewMsgDistributeStakingRewards(sample.AccAddress(), "1", tooLarge).ValidateBasic(), types.ErrInvalidAmount)
}
