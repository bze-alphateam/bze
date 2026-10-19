package keeper

import (
	"bytes"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// executeAsDao runs msgs in order as the DAO, on ctx itself: the caller decides
// whether ctx is cached. Every signer of every message must be the DAO account
// (port of x/group's ensureMsgAuthZ), so a DAO can never act for another
// address (D1, D24). The handlers' events are emitted on ctx. A handler that
// panics fails with ErrExecutionPanic instead, except out of gas.
func (k Keeper) executeAsDao(ctx sdk.Context, daoAddr sdk.AccAddress, msgs []sdk.Msg) error {
	for i, msg := range msgs {
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
