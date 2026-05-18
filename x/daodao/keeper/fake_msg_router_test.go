package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// fakeMsgRouter is a test-only MsgRouter implementation. Tests register
// per-TypeURL handlers (closures) and can capture which messages were
// dispatched in what order, then assert post-hoc.
//
// Used by Epic 5's execute_basic / execute_self_modify tests. We don't
// use gomock here because the routing surface is "give me a handler"
// (callback-returning) which is awkward to express as a single mock
// EXPECT(); a small hand-rolled fake is clearer.
type fakeMsgRouter struct {
	// handlers maps a proto type URL (e.g. "/bze.daodao.MsgUpdateMembers")
	// to the handler closure that receives the unpacked sdk.Msg. Tests
	// register handlers via withHandler before triggering execution.
	handlers map[string]baseapp.MsgServiceHandler

	// invocations records the type URLs of dispatched messages in order
	// — useful for asserting atomic rollback (the failing index plus zero
	// further invocations).
	invocations []string
}

// Implements types.MsgRouter.
func (r *fakeMsgRouter) Handler(msg sdk.Msg) baseapp.MsgServiceHandler {
	typeURL := sdk.MsgTypeURL(msg)
	inner, ok := r.handlers[typeURL]
	if !ok {
		return nil // → dispatchProposalMsgs surfaces ErrUnknownMsgType
	}
	// Wrap so we can observe invocations even if `inner` panics on a
	// nil-Return route.
	return func(ctx sdk.Context, req sdk.Msg) (*sdk.Result, error) {
		r.invocations = append(r.invocations, sdk.MsgTypeURL(req))
		return inner(ctx, req)
	}
}

func newFakeMsgRouter() *fakeMsgRouter {
	return &fakeMsgRouter{handlers: map[string]baseapp.MsgServiceHandler{}}
}

// withHandler registers a handler for the supplied proto message's type.
func (r *fakeMsgRouter) withHandler(sample sdk.Msg, h baseapp.MsgServiceHandler) *fakeMsgRouter {
	r.handlers[sdk.MsgTypeURL(sample)] = h
	return r
}

// installRouter wires the fake router on the suite's keeper. Cleared at
// the end of each test by SetupTest replacing the keeper outright.
func (suite *IntegrationTestSuite) installRouter(r types.MsgRouter) {
	suite.k.SetMsgRouter(r)
}
