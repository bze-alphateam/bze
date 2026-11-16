package keeper_test

import (
	"context"

	"cosmossdk.io/x/feegrant"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// fakeAuthz is an in-memory x/authz Msg service: it records the grants and
// execs it receives, so the executor's routing can be checked without the
// authz keeper.
type fakeAuthz struct {
	authz.UnimplementedMsgServer

	grants []*authz.MsgGrant
	execs  []*authz.MsgExec
}

func (a *fakeAuthz) Grant(_ context.Context, msg *authz.MsgGrant) (*authz.MsgGrantResponse, error) {
	a.grants = append(a.grants, msg)

	return &authz.MsgGrantResponse{}, nil
}

func (a *fakeAuthz) Exec(_ context.Context, msg *authz.MsgExec) (*authz.MsgExecResponse, error) {
	a.execs = append(a.execs, msg)

	return &authz.MsgExecResponse{}, nil
}

// fakeFeegrant is an in-memory x/feegrant Msg service that records the
// allowances it receives.
type fakeFeegrant struct {
	feegrant.UnimplementedMsgServer

	allowances []*feegrant.MsgGrantAllowance
}

func (f *fakeFeegrant) GrantAllowance(_ context.Context, msg *feegrant.MsgGrantAllowance) (*feegrant.MsgGrantAllowanceResponse, error) {
	f.allowances = append(f.allowances, msg)

	return &feegrant.MsgGrantAllowanceResponse{}, nil
}
