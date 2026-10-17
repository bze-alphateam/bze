package keeper

import (
	"bytes"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// executeAsDao runs msgs in order as the DAO, on ctx itself: the caller decides
// whether ctx is cached. Every signer of every message must be the DAO account
// (port of x/group's ensureMsgAuthZ), so a DAO can never act for another
// address (D1, D24). The handlers' events are emitted on ctx.
func (k Keeper) executeAsDao(ctx sdk.Context, dao types.Dao, msgs []sdk.Msg) error {
	daoAddr, err := sdk.AccAddressFromBech32(dao.Address)
	if err != nil {
		return err
	}

	for i, msg := range msgs {
		if err := k.ensureDaoSigner(msg, daoAddr); err != nil {
			return errorsmod.Wrapf(err, "message %d", i)
		}

		handler := k.router.Handler(msg)
		if handler == nil {
			return errorsmod.Wrapf(types.ErrInvalidMsgs, "message %d: no handler for %s", i, sdk.MsgTypeURL(msg))
		}

		res, err := handler(ctx, msg)
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
