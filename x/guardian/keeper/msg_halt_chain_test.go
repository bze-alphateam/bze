package keeper_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/guardian/keeper"
	"github.com/bze-alphateam/bze/x/guardian/testutil"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

const haltHeight = 1_000

// withUpgrade wires a mocked upgrade keeper and puts the context at haltHeight.
func (e *env) withUpgrade(t *testing.T) *testutil.MockUpgradeKeeper {
	t.Helper()
	up := testutil.NewMockUpgradeKeeper(gomock.NewController(t))
	e.k.SetUpgradeKeeper(up)
	e.ctx = e.ctx.WithBlockHeight(haltHeight)

	return up
}

func haltMsg(signer string) *types.MsgHaltChain {
	return &types.MsgHaltChain{Signer: signer, Reason: "stop everything"}
}

// The guardian and the gov authority schedule the plan at the next height, the
// earliest x/upgrade accepts, named after it, with the reason as info.
func TestHaltChain_GuardianAndAuthoritySchedule(t *testing.T) {
	for name, signer := range map[string]string{"guardian": guardianAddr, "authority": authority} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			up := e.withUpgrade(t)
			want := upgradetypes.Plan{Name: "emergency-halt-1001", Height: haltHeight + 1, Info: types.HaltPlanInfo}
			up.EXPECT().ScheduleUpgrade(gomock.Any(), want).Return(nil).Times(1)

			res, err := e.msgServer.HaltChain(e.ctx, haltMsg(signer))
			require.NoError(t, err)
			require.Equal(t, &types.MsgHaltChainResponse{Name: want.Name, Height: want.Height}, res)

			events := e.ctx.EventManager().Events()
			require.Equal(t, []string{"bze.guardian.v1.ChainHaltScheduledEvent"}, eventTypes(e.ctx))
			attrs := map[string]string{}
			for _, a := range events[0].Attributes {
				attrs[a.Key] = strings.Trim(a.Value, `"`)
			}
			require.Equal(t, signer, attrs["signer"])
			require.Equal(t, "1001", attrs["height"])
			require.Equal(t, want.Name, attrs["name"])
			require.Equal(t, "stop everything", attrs["reason"])
		})
	}
}

// The guardian address is matched on its bytes, like the other messages.
func TestHaltChain_GuardianInAnotherSpelling(t *testing.T) {
	e := newEnv(t)
	up := e.withUpgrade(t)
	up.EXPECT().ScheduleUpgrade(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err := e.msgServer.HaltChain(e.ctx, haltMsg(strings.ToUpper(guardianAddr)))
	require.NoError(t, err)
}

// Any other signer is refused and nothing is scheduled; with no guardian set
// only the gov authority can halt.
func TestHaltChain_OthersRefused(t *testing.T) {
	e := newEnv(t)
	e.withUpgrade(t) // no ScheduleUpgrade expected: a call panics

	_, err := e.msgServer.HaltChain(e.ctx, haltMsg(strangerAddr))
	require.ErrorIs(t, err, types.ErrUnauthorized)

	require.NoError(t, e.k.SetParams(e.ctx, types.NewParams("")))
	_, err = e.msgServer.HaltChain(e.ctx, haltMsg(guardianAddr))
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.Empty(t, e.ctx.EventManager().Events())
}

// An invalid message is refused before anything else, also when it reaches
// the server without the tx decoder (x/gov, x/authz, x/dao).
func TestHaltChain_InvalidMessageRefused(t *testing.T) {
	e := newEnv(t)
	e.withUpgrade(t)

	_, err := e.msgServer.HaltChain(e.ctx, &types.MsgHaltChain{Signer: guardianAddr})
	require.ErrorIs(t, err, types.ErrInvalidHaltReason)
	_, err = e.msgServer.HaltChain(e.ctx, &types.MsgHaltChain{Signer: "bad", Reason: "x"})
	require.Error(t, err)
}

// A refusal of x/upgrade is returned as is, with no event.
func TestHaltChain_ScheduleErrorReturned(t *testing.T) {
	e := newEnv(t)
	up := e.withUpgrade(t)
	boom := errors.New("refused")
	up.EXPECT().ScheduleUpgrade(gomock.Any(), gomock.Any()).Return(boom).Times(1)

	_, err := e.msgServer.HaltChain(e.ctx, haltMsg(guardianAddr))
	require.ErrorIs(t, err, boom)
	require.Empty(t, e.ctx.EventManager().Events())
}

// Without the upgrade keeper wired the message fails instead of panicking.
func TestHaltChain_UpgradeKeeperNotSet(t *testing.T) {
	k, ctx := keepertest.GuardianKeeper(t, nil)
	require.NoError(t, k.SetParams(ctx, types.NewParams(guardianAddr)))

	_, err := keeper.NewMsgServerImpl(&k).HaltChain(ctx, haltMsg(guardianAddr))
	require.ErrorIs(t, err, types.ErrUpgradeKeeperNotSet)
}

// The plan name carries the height, so every halt has its own name.
func TestHaltPlanName(t *testing.T) {
	require.Equal(t, "emergency-halt-42", types.HaltPlanName(42))
}

// The reason never reaches the plan info: tools such as cosmovisor read the
// info as JSON (with auto-download, binary URLs), so it is a constant and the
// reason stays in the event and the response.
func TestHaltChain_ReasonNeverInPlanInfo(t *testing.T) {
	e := newEnv(t)
	up := e.withUpgrade(t)
	reason := `{"binaries":{"linux/amd64":"https://x?checksum=sha256:00"}}`
	var got upgradetypes.Plan
	up.EXPECT().ScheduleUpgrade(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p upgradetypes.Plan) error {
		got = p
		return nil
	}).Times(1)

	_, err := e.msgServer.HaltChain(e.ctx, &types.MsgHaltChain{Signer: guardianAddr, Reason: reason})
	require.NoError(t, err)
	require.Equal(t, "emergency halt", got.Info)
	require.Equal(t, types.HaltPlanInfo, got.Info)
	require.NotContains(t, got.Info, "binaries")
}
