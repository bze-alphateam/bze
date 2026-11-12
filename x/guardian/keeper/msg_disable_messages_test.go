package keeper_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/guardian/keeper"
	"github.com/bze-alphateam/bze/x/guardian/testutil"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

const (
	swapURL   = "/bze.tradebin.MsgMultiSwap"
	createURL = "/bze.tradebin.MsgCreateOrder"
)

func disable(signer string, urls ...string) *types.MsgDisableMessages {
	return &types.MsgDisableMessages{Signer: signer, MsgTypeUrls: urls}
}

// expectChecked answers the routing and disable-list checks of a url that
// passes both.
func (e *env) expectChecked(url string) {
	e.router.EXPECT().IsRegistered(url).Return(true)
	e.circuit.EXPECT().IsDisabled(gomock.Any(), url).Return(false, nil)
}

func TestDisableMessages_GuardianAndAuthorityDisable(t *testing.T) {
	for name, signer := range map[string]string{"guardian": guardianAddr, "authority": authority} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			e.expectChecked(swapURL)
			e.expectChecked(createURL)
			gomock.InOrder(
				e.circuit.EXPECT().Disable(gomock.Any(), swapURL).Return(nil),
				e.circuit.EXPECT().Disable(gomock.Any(), createURL).Return(nil),
			)

			_, err := e.msgServer.DisableMessages(e.ctx, disable(signer, swapURL, createURL))
			require.NoError(t, err)

			events := e.ctx.EventManager().Events()
			require.Len(t, events, 1)
			require.Equal(t, "bze.guardian.v1.MessagesDisabledEvent", events[0].Type)
			attrs := map[string]string{}
			for _, a := range events[0].Attributes {
				attrs[a.Key] = a.Value
			}
			require.Equal(t, `"`+signer+`"`, attrs["signer"])
			require.Equal(t, `["`+swapURL+`","`+createURL+`"]`, attrs["msg_type_urls"])
		})
	}
}

// Anyone but the guardian and the authority is refused before any lookup:
// no mock has an expectation, so a call would fail the test.
func TestDisableMessages_StrangerIsRefused(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.DisableMessages(e.ctx, disable(strangerAddr, swapURL))
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.Empty(t, e.ctx.EventManager().Events())
}

// With no guardian set, only the authority can sign.
func TestDisableMessages_NoGuardianOnlyAuthority(t *testing.T) {
	e := newEnv(t)
	require.NoError(t, e.k.SetParams(e.ctx, types.DefaultParams()))

	_, err := e.msgServer.DisableMessages(e.ctx, disable(guardianAddr, swapURL))
	require.ErrorIs(t, err, types.ErrUnauthorized)
}

func TestDisableMessages_ProtectedTypesAreRefused(t *testing.T) {
	for _, url := range []string{
		"/cosmos.gov.v1.MsgSubmitProposal",
		"/cosmos.gov.v1.MsgVote",
		"/cosmos.gov.v1beta1.MsgDeposit",
		"/cosmos.circuit.v1.MsgResetCircuitBreaker",
		"/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
		"/bze.guardian.v1.MsgDisableMessages",
		"/bze.guardian.v1.MsgBlockDenom",
		"/bze.dao.MsgSubmitProposal",
		"/bze.dao.MsgVote",
		"/bze.dao.MsgSettleProposal",
	} {
		t.Run(url, func(t *testing.T) {
			for name, signer := range map[string]string{"guardian": guardianAddr, "authority": authority} {
				e := newEnv(t)
				// a valid url first: nothing is written for it either
				_, err := e.msgServer.DisableMessages(e.ctx, disable(signer, swapURL, url))
				require.ErrorIs(t, err, types.ErrProtectedMsgType, name)
				require.Empty(t, e.ctx.EventManager().Events())
			}
		})
	}
}

// A refusal on the last url writes nothing, also for the urls before it.
func TestDisableMessages_UnknownTypeWritesNothing(t *testing.T) {
	e := newEnv(t)
	e.expectChecked(swapURL)
	e.router.EXPECT().IsRegistered("/bze.tradebin.MsgNope").Return(false)
	e.circuit.EXPECT().Disable(gomock.Any(), gomock.Any()).Times(0)

	_, err := e.msgServer.DisableMessages(e.ctx, disable(guardianAddr, swapURL, "/bze.tradebin.MsgNope"))
	require.ErrorIs(t, err, types.ErrUnknownMsgType)
	require.Empty(t, e.ctx.EventManager().Events())
}

func TestDisableMessages_AlreadyDisabledWritesNothing(t *testing.T) {
	e := newEnv(t)
	e.expectChecked(swapURL)
	e.router.EXPECT().IsRegistered(createURL).Return(true)
	e.circuit.EXPECT().IsDisabled(gomock.Any(), createURL).Return(true, nil)
	e.circuit.EXPECT().Disable(gomock.Any(), gomock.Any()).Times(0)

	_, err := e.msgServer.DisableMessages(e.ctx, disable(guardianAddr, swapURL, createURL))
	require.ErrorIs(t, err, types.ErrAlreadyDisabled)
	require.Empty(t, e.ctx.EventManager().Events())
}

func TestDisableMessages_DuplicatesAreRefused(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.DisableMessages(e.ctx, disable(guardianAddr, swapURL, swapURL))
	require.ErrorIs(t, err, types.ErrInvalidMsgTypeURLs)
}

func TestDisableMessages_EmptyListIsRefused(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.DisableMessages(e.ctx, disable(guardianAddr))
	require.ErrorIs(t, err, types.ErrInvalidMsgTypeURLs)
}

func TestDisableMessages_LookupErrorsAreReturned(t *testing.T) {
	boom := errors.New("store error")

	e := newEnv(t)
	e.router.EXPECT().IsRegistered(swapURL).Return(true)
	e.circuit.EXPECT().IsDisabled(gomock.Any(), swapURL).Return(false, boom)
	_, err := e.msgServer.DisableMessages(e.ctx, disable(guardianAddr, swapURL))
	require.ErrorIs(t, err, boom)

	e = newEnv(t)
	e.expectChecked(swapURL)
	e.circuit.EXPECT().Disable(gomock.Any(), swapURL).Return(boom)
	_, err = e.msgServer.DisableMessages(e.ctx, disable(guardianAddr, swapURL))
	require.ErrorIs(t, err, boom)
	require.Empty(t, e.ctx.EventManager().Events())
}

// A keeper without the circuit keeper or the router refuses, after the signer
// check, and touches nothing.
func TestDisableMessages_NotWired(t *testing.T) {
	ctrl := gomock.NewController(t)

	k, ctx := keepertest.GuardianKeeper(t, nil)
	require.NoError(t, k.SetParams(ctx, types.NewParams(guardianAddr)))
	srv := keeper.NewMsgServerImpl(&k)

	_, err := srv.DisableMessages(ctx, disable(strangerAddr, swapURL))
	require.ErrorIs(t, err, types.ErrUnauthorized)

	_, err = srv.DisableMessages(ctx, disable(guardianAddr, swapURL))
	require.ErrorIs(t, err, types.ErrCircuitKeeperNotSet)

	// the circuit keeper alone is not enough: the router is needed too
	k.SetCircuitKeeper(testutil.NewMockCircuitKeeper(ctrl))
	_, err = srv.DisableMessages(ctx, disable(guardianAddr, swapURL))
	require.ErrorIs(t, err, types.ErrCircuitKeeperNotSet)

	k.SetCircuitKeeper(nil)
	k.SetMsgRouter(testutil.NewMockMsgRouter(ctrl))
	_, err = srv.DisableMessages(ctx, disable(guardianAddr, swapURL))
	require.ErrorIs(t, err, types.ErrCircuitKeeperNotSet)
	require.Empty(t, ctx.EventManager().Events())
}
