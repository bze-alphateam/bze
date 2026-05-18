package keeper

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestComputeOutcome_NearMaxUint64_DoesNotOverflow exercises the 128-bit
// cross-multiplication helpers (`mulGreaterOrEqual` / `mulLess`) with
// voting powers near MaxUint64. Under naive uint64 cross-multiplication
// `yes * 10_000` overflows once yes > MaxUint64/10_000 ≈ 1.8e15, which
// can flip pass/reject for adversarially-chosen STATIC weights.
//
// These tests run against the pure-function tally code (no keeper / store
// needed), so they live in the `keeper` package directly rather than
// `keeper_test`.
func TestComputeOutcome_NearMaxUint64_DoesNotOverflow(t *testing.T) {
	// total at MaxUint64; yes = total → should PASS under any threshold < 100%.
	total := uint64(math.MaxUint64)
	gov := types.GovernanceConfig{
		ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
		ThresholdBps: 5_000,
	}
	require.Equal(t, types.OutcomePass, types.ComputeOutcome(gov, types.Tally{
		YesPower: total, TotalPower: total,
	}))

	// yes = exactly threshold * total / 10000 → boundary PASS.
	// For threshold=5000, yes = total/2. With total=MaxUint64 (odd), use a
	// multiple of 10000 to avoid integer-division rounding ambiguity.
	bigEven := uint64(1 << 60) // 2^60, divisible by every reasonable threshold
	require.Equal(t, types.OutcomePass, types.ComputeOutcome(gov, types.Tally{
		YesPower: bigEven / 2, TotalPower: bigEven,
	}))
	require.Equal(t, types.OutcomeReject, types.ComputeOutcome(gov, types.Tally{
		YesPower: bigEven/2 - 1, TotalPower: bigEven,
	}))
}

func TestOutcomeWithQuorum_NearMaxUint64_DoesNotOverflow(t *testing.T) {
	// voted = total, yes = no = total/2 → quorum 100%, threshold 50%.
	// Threshold check: yes * 10000 >= 5000 * (yes+no). With yes=no=total/2,
	// yes+no = total. yes * 10000 = (total/2) * 10000 — overflows in uint64
	// for total ≥ ~3.7e15.
	bigEven := uint64(1 << 60)
	gov := types.GovernanceConfig{
		ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITH_QUORUM,
		ThresholdBps: 5_000,
		QuorumBps:    5_000,
	}
	require.Equal(t, types.OutcomePass, types.ComputeOutcome(gov, types.Tally{
		YesPower: bigEven / 2, NoPower: bigEven / 2, TotalPower: bigEven,
	}), "yes = no with 50% threshold should PASS at the boundary")

	// yes strictly less than no → should REJECT.
	require.Equal(t, types.OutcomeReject, types.ComputeOutcome(gov, types.Tally{
		YesPower: bigEven/2 - 1, NoPower: bigEven/2 + 1, TotalPower: bigEven,
	}))
}

func TestCheckEarlyClose_NearMaxUint64_DoesNotOverflow(t *testing.T) {
	// WITHOUT_QUORUM, threshold 50%. yes alone meets threshold of total.
	bigEven := uint64(1 << 60)
	gov := types.GovernanceConfig{
		ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
		ThresholdBps: 5_000,
	}
	require.Equal(t, earlyClosePass, checkEarlyClose(gov, types.Tally{
		YesPower: bigEven / 2, TotalPower: bigEven,
	}))

	// yes + remaining still below threshold → reject-locked.
	require.Equal(t, earlyCloseReject, checkEarlyClose(gov, types.Tally{
		NoPower: bigEven/2 + 1, TotalPower: bigEven,
	}))
}
