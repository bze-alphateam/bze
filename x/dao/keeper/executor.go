package keeper

import (
	"bytes"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/feegrant"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// executionDepthKey is the context key of the number of DAO executions
// running on the stack.
type executionDepthKey struct{}

// executeAsDao runs msgs in order as the DAO, on ctx itself: the caller decides
// whether ctx is cached. Every signer of every message must be the DAO account
// (port of x/group's ensureMsgAuthZ), so a DAO can never act for another
// address (D1, D24). The handlers' events are emitted on ctx. A handler that
// panics fails with ErrExecutionPanic instead, except out of gas.
//
// Each call is one level of DAO execution, carried in ctx to the handlers: a
// DAO running MsgAdminExecute for its sub-DAO nests one more. More than
// MaxDaoExecutionDepth levels fail with ErrExecutionTooDeep. A settle never
// runs inside a DAO execution: settling is open to any account.
func (k Keeper) executeAsDao(ctx sdk.Context, daoAddr sdk.AccAddress, msgs []sdk.Msg) error {
	depth, _ := ctx.Value(executionDepthKey{}).(int)
	depth++
	if depth > types.MaxDaoExecutionDepth {
		return errorsmod.Wrapf(types.ErrExecutionTooDeep, "at most %d nested dao executions", types.MaxDaoExecutionDepth)
	}
	ctx = ctx.WithValue(executionDepthKey{}, depth)

	for i, msg := range msgs {
		if err := refuseSettle(msg); err != nil {
			return errorsmod.Wrapf(err, "message %d", i)
		}

		if err := k.ensureDaoSigner(msg, daoAddr); err != nil {
			return errorsmod.Wrapf(err, "message %d", i)
		}

		handler := k.router.Handler(msg)
		if handler == nil {
			return errorsmod.Wrapf(types.ErrInvalidMsgs, "message %d: no handler for %s", i, sdk.MsgTypeURL(msg))
		}

		res, err := safeHandle(ctx, handler, msg)
		if err != nil {
			return errorsmod.Wrapf(err, "message %d (%s)", i, sdk.MsgTypeURL(msg))
		}

		ctx.EventManager().EmitEvents(res.GetEvents())
	}

	return nil
}

// refuseSettle refuses MsgSettleProposal, also inside an authz exec.
func refuseSettle(msg sdk.Msg) error {
	switch m := msg.(type) {
	case *types.MsgSettleProposal:
		return errorsmod.Wrapf(types.ErrMsgNotAllowed, "%s", sdk.MsgTypeURL(msg))
	case *authz.MsgExec:
		inner, err := m.GetMessages()
		if err != nil {
			return errorsmod.Wrap(types.ErrInvalidMsgs, err.Error())
		}

		for _, innerMsg := range inner {
			if err := refuseSettle(innerMsg); err != nil {
				return err
			}
		}
	}

	return nil
}

// refuseAdminGrants refuses the messages that would leave the admin powers
// signed by the DAO once it is no longer the admin: authz grants and fee
// allowances. An authz exec is refused too, since it dispatches a message
// signed by its own grantee without any grant.
func refuseAdminGrants(msg sdk.Msg) error {
	switch msg.(type) {
	case *authz.MsgGrant, *feegrant.MsgGrantAllowance, *authz.MsgExec:
		return errorsmod.Wrapf(types.ErrMsgNotAllowed, "%s through the admin", sdk.MsgTypeURL(msg))
	}

	return nil
}

// ensureDaoSigner requires at least one signer and every signer to be the DAO.
func (k Keeper) ensureDaoSigner(msg sdk.Msg, daoAddr sdk.AccAddress) error {
	signers, _, err := k.cdc.GetMsgV1Signers(msg)
	if err != nil {
		return err
	}

	if len(signers) == 0 {
		return errorsmod.Wrapf(types.ErrUnauthorizedSigner, "%s has no signer", sdk.MsgTypeURL(msg))
	}

	for _, signer := range signers {
		if !bytes.Equal(signer, daoAddr) {
			return errorsmod.Wrapf(types.ErrUnauthorizedSigner, "expected %s, got %s", daoAddr, sdk.AccAddress(signer))
		}
	}

	return nil
}

// safeHandle runs the handler and turns a panic into ErrExecutionPanic, as
// x/gov's safeExecuteHandler does. Out of gas is panicked again: a settle is a
// transaction, so swallowing it would let anyone fail a passed proposal by
// settling it with a tiny gas limit; the whole transaction fails instead.
func safeHandle(ctx sdk.Context, handler baseapp.MsgServiceHandler, msg sdk.Msg) (res *sdk.Result, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		switch r.(type) {
		case storetypes.ErrorOutOfGas, storetypes.ErrorGasOverflow:
			panic(r)
		}

		err = errorsmod.Wrapf(types.ErrExecutionPanic, "%s: %v", sdk.MsgTypeURL(msg), r)
	}()

	return handler(ctx, msg)
}
