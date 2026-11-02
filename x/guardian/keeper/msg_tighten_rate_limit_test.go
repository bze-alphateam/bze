package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/guardian/keeper"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

// expectStored makes the mocked keeper serve the limit and records what is
// written back.
func (e *env) expectStored(limit ratelimittypes.RateLimit) *[]ratelimittypes.RateLimit {
	written := &[]ratelimittypes.RateLimit{}
	e.rl.EXPECT().GetRateLimit(gomock.Any(), testDenom, testChannel).Return(limit, true).AnyTimes()
	e.rl.EXPECT().SetRateLimit(gomock.Any(), gomock.Any()).Do(func(_ sdk.Context, rl ratelimittypes.RateLimit) {
		*written = append(*written, rl)
	}).AnyTimes()

	return written
}

// The stored rate limit comes back with only the two percents changed: the
// window length, the path and the flow of the current window are untouched.
// Resetting the flow would hand out a fresh window (regression guard for the
// ratelimit keeper's UpdateRateLimit, which zeroes it).
func TestTightenRateLimit_GuardianLowersBothAndKeepsTheFlow(t *testing.T) {
	e := newEnv(t)
	before := storedLimit(10, 20)
	written := e.expectStored(before)

	_, err := e.msgServer.TightenRateLimit(e.ctx, tighten(guardianAddr, 5, 2))
	require.NoError(t, err)

	require.Len(t, *written, 1)
	got := (*written)[0]
	require.Equal(t, before.Path, got.Path)
	require.Equal(t, before.Flow, got.Flow)
	require.Equal(t, math.NewInt(123), got.Flow.Inflow)
	require.Equal(t, math.NewInt(456), got.Flow.Outflow)
	require.Equal(t, math.NewInt(1_000_000), got.Flow.ChannelValue)
	require.Equal(t, uint64(24), got.Quota.DurationHours)
	require.Equal(t, math.NewInt(5), got.Quota.MaxPercentSend)
	require.Equal(t, math.NewInt(2), got.Quota.MaxPercentRecv)
	// the object the keeper handed out is not edited in place
	require.Equal(t, math.NewInt(10), before.Quota.MaxPercentSend)
	require.Equal(t, math.NewInt(20), before.Quota.MaxPercentRecv)
}

func TestTightenRateLimit_LowersOneKeepsTheOtherEqual(t *testing.T) {
	e := newEnv(t)
	written := e.expectStored(storedLimit(10, 20))

	_, err := e.msgServer.TightenRateLimit(e.ctx, tighten(guardianAddr, 10, 7))
	require.NoError(t, err)

	require.Len(t, *written, 1)
	require.Equal(t, math.NewInt(10), (*written)[0].Quota.MaxPercentSend)
	require.Equal(t, math.NewInt(7), (*written)[0].Quota.MaxPercentRecv)
}

func TestTightenRateLimit_EqualValuesAreANoOpWrite(t *testing.T) {
	e := newEnv(t)
	before := storedLimit(10, 20)
	written := e.expectStored(before)

	_, err := e.msgServer.TightenRateLimit(e.ctx, tighten(guardianAddr, 10, 20))
	require.NoError(t, err)

	require.Equal(t, []ratelimittypes.RateLimit{before}, *written)
}

// 0/0 stops the path in both directions; only the ratelimit module's own
// messages forbid it.
func TestTightenRateLimit_ZeroZeroIsAllowed(t *testing.T) {
	e := newEnv(t)
	written := e.expectStored(storedLimit(10, 20))

	_, err := e.msgServer.TightenRateLimit(e.ctx, tighten(guardianAddr, 0, 0))
	require.NoError(t, err)

	require.Len(t, *written, 1)
	require.True(t, (*written)[0].Quota.MaxPercentSend.IsZero())
	require.True(t, (*written)[0].Quota.MaxPercentRecv.IsZero())
}

func TestTightenRateLimit_RaiseIsRefusedAndNothingIsWritten(t *testing.T) {
	for name, msg := range map[string]*types.MsgTightenRateLimit{
		"send":     tighten(guardianAddr, 11, 20),
		"recv":     tighten(guardianAddr, 10, 21),
		"both":     tighten(guardianAddr, 100, 100),
		"one up":   tighten(guardianAddr, 1, 21),
		"authorit": tighten(authority, 11, 20),
	} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			e.rl.EXPECT().GetRateLimit(gomock.Any(), testDenom, testChannel).Return(storedLimit(10, 20), true)
			// no SetRateLimit expectation: a write fails the test

			_, err := e.msgServer.TightenRateLimit(e.ctx, msg)
			require.ErrorIs(t, err, types.ErrQuotaRaise)
		})
	}
}

func TestTightenRateLimit_UnknownPathIsRefused(t *testing.T) {
	e := newEnv(t)
	e.rl.EXPECT().GetRateLimit(gomock.Any(), testDenom, testChannel).Return(ratelimittypes.RateLimit{}, false)

	_, err := e.msgServer.TightenRateLimit(e.ctx, tighten(guardianAddr, 1, 1))
	require.ErrorIs(t, err, types.ErrRateLimitNotFound)
}

func TestTightenRateLimit_StrangerIsRefusedAndNothingIsTouched(t *testing.T) {
	e := newEnv(t)
	// no keeper expectation at all: the signer is checked first

	_, err := e.msgServer.TightenRateLimit(e.ctx, tighten(strangerAddr, 1, 1))
	require.ErrorIs(t, err, types.ErrUnauthorized)
}

func TestTightenRateLimit_AuthoritySignerIsAccepted(t *testing.T) {
	e := newEnv(t)
	written := e.expectStored(storedLimit(10, 20))

	_, err := e.msgServer.TightenRateLimit(e.ctx, tighten(authority, 1, 1))
	require.NoError(t, err)
	require.Len(t, *written, 1)
}

func TestTightenRateLimit_EmptyGuardianLeavesOnlyTheAuthority(t *testing.T) {
	e := newEnv(t)
	require.NoError(t, e.k.SetParams(e.ctx, types.DefaultParams()))
	written := e.expectStored(storedLimit(10, 20))

	_, err := e.msgServer.TightenRateLimit(e.ctx, tighten(guardianAddr, 1, 1))
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.Empty(t, *written)

	_, err = e.msgServer.TightenRateLimit(e.ctx, tighten(authority, 1, 1))
	require.NoError(t, err)
	require.Len(t, *written, 1)
}

// A message that skipped the transaction decoder (x/gov, x/authz) is still
// validated: a negative percent would pass the "not above the current" check.
func TestTightenRateLimit_InvalidMessageIsRefusedBeforeAnythingElse(t *testing.T) {
	e := newEnv(t)

	msg := tighten(guardianAddr, -1, 1)
	_, err := e.msgServer.TightenRateLimit(e.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidQuota)

	msg = tighten(guardianAddr, 1, 1)
	msg.MaxPercentRecv = math.Int{}
	_, err = e.msgServer.TightenRateLimit(e.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidQuota)
}

func TestTightenRateLimit_EmitsTheEventWithOldAndNewValues(t *testing.T) {
	e := newEnv(t)
	e.expectStored(storedLimit(10, 20))

	_, err := e.msgServer.TightenRateLimit(e.ctx, tighten(guardianAddr, 5, 0))
	require.NoError(t, err)

	var found []sdk.Event
	for _, ev := range e.ctx.EventManager().Events() {
		if ev.Type == "bze.guardian.v1.TightenRateLimitEvent" {
			found = append(found, ev)
		}
	}
	require.Len(t, found, 1)

	attrs := map[string]string{}
	for _, a := range found[0].Attributes {
		attrs[a.Key] = a.Value
	}
	require.Equal(t, `"`+guardianAddr+`"`, attrs["signer"])
	require.Equal(t, `"ubze"`, attrs["denom"])
	require.Equal(t, `"channel-0"`, attrs["channel_id"])
	require.Equal(t, `"10"`, attrs["old_send"])
	require.Equal(t, `"20"`, attrs["old_recv"])
	require.Equal(t, `"5"`, attrs["new_send"])
	require.Equal(t, `"0"`, attrs["new_recv"])
}

func TestTightenRateLimit_WithoutRateLimitKeeperIsRefused(t *testing.T) {
	k, ctx := newBareKeeper(t)

	_, err := keeper.NewMsgServerImpl(&k).TightenRateLimit(ctx, tighten(authority, 1, 1))
	require.ErrorIs(t, err, types.ErrRateLimitKeeperNotSet)
}
