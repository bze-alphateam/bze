package keeper_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

const (
	pausedEvent  = "bze.guardian.v1.AutomationPausedEvent"
	resumedEvent = "bze.guardian.v1.AutomationResumedEvent"
)

func (e *env) pause(t *testing.T) {
	t.Helper()
	_, err := e.msgServer.PauseAutomation(e.ctx, &types.MsgPauseAutomation{Signer: guardianAddr})
	require.NoError(t, err)
	require.True(t, e.k.IsAutomationPaused(e.ctx))
}

func TestAutomation_NotPausedByDefault(t *testing.T) {
	e := newEnv(t)

	require.False(t, e.k.IsAutomationPaused(e.ctx))
}

func TestPauseAutomation_GuardianAndAuthorityPause(t *testing.T) {
	for name, signer := range map[string]string{"guardian": guardianAddr, "authority": authority} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)

			_, err := e.msgServer.PauseAutomation(e.ctx, &types.MsgPauseAutomation{Signer: signer})
			require.NoError(t, err)
			require.True(t, e.k.IsAutomationPaused(e.ctx))
			require.Equal(t, []string{pausedEvent}, eventTypes(e.ctx))
		})
	}
}

func TestPauseAutomation_StrangerIsRefused(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.PauseAutomation(e.ctx, &types.MsgPauseAutomation{Signer: strangerAddr})
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.False(t, e.k.IsAutomationPaused(e.ctx))
	require.Empty(t, eventTypes(e.ctx))
}

func TestPauseAutomation_EmptyGuardianLeavesOnlyTheAuthority(t *testing.T) {
	e := newEnv(t)
	require.NoError(t, e.k.SetParams(e.ctx, types.DefaultParams()))

	_, err := e.msgServer.PauseAutomation(e.ctx, &types.MsgPauseAutomation{Signer: guardianAddr})
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.False(t, e.k.IsAutomationPaused(e.ctx))

	_, err = e.msgServer.PauseAutomation(e.ctx, &types.MsgPauseAutomation{Signer: authority})
	require.NoError(t, err)
	require.True(t, e.k.IsAutomationPaused(e.ctx))
}

// The guardian address is compared by account bytes: an uppercase spelling is
// the same account.
func TestPauseAutomation_UppercaseGuardianSpelling(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.PauseAutomation(e.ctx, &types.MsgPauseAutomation{Signer: strings.ToUpper(guardianAddr)})
	require.NoError(t, err)
	require.True(t, e.k.IsAutomationPaused(e.ctx))
}

func TestPauseAutomation_DoublePauseIsRefused(t *testing.T) {
	e := newEnv(t)
	e.pause(t)
	before := len(e.ctx.EventManager().Events())

	_, err := e.msgServer.PauseAutomation(e.ctx, &types.MsgPauseAutomation{Signer: authority})
	require.ErrorIs(t, err, types.ErrAlreadyPaused)
	require.True(t, e.k.IsAutomationPaused(e.ctx))
	require.Len(t, e.ctx.EventManager().Events(), before)
}

func TestPauseAutomation_InvalidSigner(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.PauseAutomation(e.ctx, &types.MsgPauseAutomation{Signer: "nope"})
	require.Error(t, err)
	require.False(t, e.k.IsAutomationPaused(e.ctx))
}

func TestResumeAutomation_AuthorityResumes(t *testing.T) {
	e := newEnv(t)
	e.pause(t)

	_, err := e.msgServer.ResumeAutomation(e.ctx, &types.MsgResumeAutomation{Authority: authority})
	require.NoError(t, err)
	require.False(t, e.k.IsAutomationPaused(e.ctx))
	require.Equal(t, []string{pausedEvent, resumedEvent}, eventTypes(e.ctx))
}

// The guardian can stop the automation, never restart it.
func TestResumeAutomation_GuardianAndStrangerAreRefused(t *testing.T) {
	for name, signer := range map[string]string{"guardian": guardianAddr, "stranger": strangerAddr} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			e.pause(t)

			_, err := e.msgServer.ResumeAutomation(e.ctx, &types.MsgResumeAutomation{Authority: signer})
			require.ErrorIs(t, err, types.ErrUnauthorized)
			require.True(t, e.k.IsAutomationPaused(e.ctx))
			require.Equal(t, []string{pausedEvent}, eventTypes(e.ctx))
		})
	}
}

func TestResumeAutomation_NotPausedIsRefused(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.ResumeAutomation(e.ctx, &types.MsgResumeAutomation{Authority: authority})
	require.ErrorIs(t, err, types.ErrNotPaused)
	require.False(t, e.k.IsAutomationPaused(e.ctx))
	require.Empty(t, eventTypes(e.ctx))
}

func TestResumeAutomation_InvalidAuthority(t *testing.T) {
	e := newEnv(t)
	e.pause(t)

	_, err := e.msgServer.ResumeAutomation(e.ctx, &types.MsgResumeAutomation{Authority: "nope"})
	require.Error(t, err)
	require.True(t, e.k.IsAutomationPaused(e.ctx))
}

// A pause after a resume works again: the flag is a plain switch.
func TestAutomation_PauseResumePause(t *testing.T) {
	e := newEnv(t)
	e.pause(t)

	_, err := e.msgServer.ResumeAutomation(e.ctx, &types.MsgResumeAutomation{Authority: authority})
	require.NoError(t, err)

	e.pause(t)
}

func TestQueryAutomationPaused(t *testing.T) {
	e := newEnv(t)

	_, err := e.k.AutomationPaused(e.ctx, nil)
	require.Error(t, err)

	res, err := e.k.AutomationPaused(e.ctx, &types.QueryAutomationPausedRequest{})
	require.NoError(t, err)
	require.False(t, res.Paused)

	e.pause(t)

	res, err = e.k.AutomationPaused(e.ctx, &types.QueryAutomationPausedRequest{})
	require.NoError(t, err)
	require.True(t, res.Paused)
}

func TestSetAutomationPaused(t *testing.T) {
	e := newEnv(t)

	e.k.SetAutomationPaused(e.ctx, true)
	require.True(t, e.k.IsAutomationPaused(e.ctx))

	e.k.SetAutomationPaused(e.ctx, false)
	require.False(t, e.k.IsAutomationPaused(e.ctx))
}
