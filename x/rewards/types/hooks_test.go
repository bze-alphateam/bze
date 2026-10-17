package types

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// recordingHooks is a StakingRewardHooks implementation that records the order
// in which its methods are called and can be forced to return an error.
type recordingHooks struct {
	name  string
	calls *[]string
	err   error
}

func (r recordingHooks) AfterStakingRewardJoin(_ sdk.Context, _, _ string, _ math.Int, _ string) error {
	*r.calls = append(*r.calls, r.name+":join")
	return r.err
}

func (r recordingHooks) AfterStakingRewardIncrease(_ sdk.Context, _, _ string, _, _ math.Int, _ string) error {
	*r.calls = append(*r.calls, r.name+":increase")
	return r.err
}

func (r recordingHooks) AfterStakingRewardExit(_ sdk.Context, _, _ string, _ math.Int, _ string) error {
	*r.calls = append(*r.calls, r.name+":exit")
	return r.err
}

func (r recordingHooks) BeforeStakingRewardRemoval(_ sdk.Context, _ string) error {
	*r.calls = append(*r.calls, r.name+":removal")
	return r.err
}

func TestMultiStakingRewardHooks_CallsAllInOrder(t *testing.T) {
	var calls []string
	multi := NewMultiStakingRewardHooks(
		recordingHooks{name: "first", calls: &calls},
		recordingHooks{name: "second", calls: &calls},
	)

	ctx := sdk.Context{}
	amount := math.NewInt(100)

	require.NoError(t, multi.AfterStakingRewardJoin(ctx, "01", "addr", amount, "ubze"))
	require.NoError(t, multi.AfterStakingRewardIncrease(ctx, "01", "addr", amount, amount, "ubze"))
	require.NoError(t, multi.AfterStakingRewardExit(ctx, "01", "addr", amount, "ubze"))
	require.NoError(t, multi.BeforeStakingRewardRemoval(ctx, "01"))

	require.Equal(t, []string{
		"first:join", "second:join",
		"first:increase", "second:increase",
		"first:exit", "second:exit",
		"first:removal", "second:removal",
	}, calls)
}

func TestMultiStakingRewardHooks_FirstErrorShortCircuits(t *testing.T) {
	var calls []string
	expectedErr := fmt.Errorf("hook failure")
	multi := NewMultiStakingRewardHooks(
		recordingHooks{name: "first", calls: &calls, err: expectedErr},
		recordingHooks{name: "second", calls: &calls},
	)

	ctx := sdk.Context{}
	amount := math.NewInt(100)

	err := multi.AfterStakingRewardJoin(ctx, "01", "addr", amount, "ubze")
	require.ErrorIs(t, err, expectedErr)
	//the second hook must not have been called
	require.Equal(t, []string{"first:join"}, calls)

	calls = nil
	err = multi.AfterStakingRewardIncrease(ctx, "01", "addr", amount, amount, "ubze")
	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, []string{"first:increase"}, calls)

	calls = nil
	err = multi.AfterStakingRewardExit(ctx, "01", "addr", amount, "ubze")
	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, []string{"first:exit"}, calls)

	calls = nil
	err = multi.BeforeStakingRewardRemoval(ctx, "01")
	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, []string{"first:removal"}, calls)
}

func TestMultiStakingRewardHooks_EmptyReturnsNil(t *testing.T) {
	multi := NewMultiStakingRewardHooks()

	ctx := sdk.Context{}
	amount := math.NewInt(100)

	require.NoError(t, multi.AfterStakingRewardJoin(ctx, "01", "addr", amount, "ubze"))
	require.NoError(t, multi.AfterStakingRewardIncrease(ctx, "01", "addr", amount, amount, "ubze"))
	require.NoError(t, multi.AfterStakingRewardExit(ctx, "01", "addr", amount, "ubze"))
	require.NoError(t, multi.BeforeStakingRewardRemoval(ctx, "01"))
}

// recordingDenomHooks is the DenomRewardHooks counterpart of recordingHooks.
type recordingDenomHooks struct {
	name  string
	calls *[]string
	err   error
}

func (r recordingDenomHooks) AfterDenomRewardJoin(_ sdk.Context, _, _ string, _ math.Int) error {
	*r.calls = append(*r.calls, r.name+":join")
	return r.err
}

func (r recordingDenomHooks) AfterDenomRewardIncrease(_ sdk.Context, _, _ string, _, _ math.Int) error {
	*r.calls = append(*r.calls, r.name+":increase")
	return r.err
}

func (r recordingDenomHooks) AfterDenomRewardExit(_ sdk.Context, _, _ string, _ math.Int) error {
	*r.calls = append(*r.calls, r.name+":exit")
	return r.err
}

func TestMultiDenomRewardHooks_CallsAllInOrder(t *testing.T) {
	var calls []string
	multi := NewMultiDenomRewardHooks(
		recordingDenomHooks{name: "first", calls: &calls},
		recordingDenomHooks{name: "second", calls: &calls},
	)

	ctx := sdk.Context{}
	amount := math.NewInt(100)

	require.NoError(t, multi.AfterDenomRewardJoin(ctx, "ubze", "addr", amount))
	require.NoError(t, multi.AfterDenomRewardIncrease(ctx, "ubze", "addr", amount, amount))
	require.NoError(t, multi.AfterDenomRewardExit(ctx, "ubze", "addr", amount))

	require.Equal(t, []string{
		"first:join", "second:join",
		"first:increase", "second:increase",
		"first:exit", "second:exit",
	}, calls)
}

func TestMultiDenomRewardHooks_FirstErrorShortCircuits(t *testing.T) {
	var calls []string
	expectedErr := fmt.Errorf("hook failure")
	multi := NewMultiDenomRewardHooks(
		recordingDenomHooks{name: "first", calls: &calls, err: expectedErr},
		recordingDenomHooks{name: "second", calls: &calls},
	)

	ctx := sdk.Context{}
	amount := math.NewInt(100)

	require.ErrorIs(t, multi.AfterDenomRewardJoin(ctx, "ubze", "addr", amount), expectedErr)
	require.ErrorIs(t, multi.AfterDenomRewardIncrease(ctx, "ubze", "addr", amount, amount), expectedErr)
	require.ErrorIs(t, multi.AfterDenomRewardExit(ctx, "ubze", "addr", amount), expectedErr)
	//the second hook must never have been called
	require.Equal(t, []string{"first:join", "first:increase", "first:exit"}, calls)
}

func TestMultiDenomRewardHooks_EmptyReturnsNil(t *testing.T) {
	multi := NewMultiDenomRewardHooks()

	ctx := sdk.Context{}
	amount := math.NewInt(100)

	require.NoError(t, multi.AfterDenomRewardJoin(ctx, "ubze", "addr", amount))
	require.NoError(t, multi.AfterDenomRewardIncrease(ctx, "ubze", "addr", amount, amount))
	require.NoError(t, multi.AfterDenomRewardExit(ctx, "ubze", "addr", amount))
}
