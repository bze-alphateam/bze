package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// DisableMessages adds message types to the x/circuit disable list. The app's
// message router then refuses every message of these types, whether it comes
// in a tx or is executed by another module (gov, authz, x/dao).
//
// x/guardian writes the list directly and holds no x/circuit permission, so
// the guardian cannot re-enable a message: only governance can, with
// x/circuit's MsgResetCircuitBreaker. Every url is checked before anything is
// written, so a refused message changes nothing.
func (k msgServer) DisableMessages(goCtx context.Context, msg *types.MsgDisableMessages) (*types.MsgDisableMessagesResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.authorize(ctx, msg.Signer); err != nil {
		return nil, err
	}

	if k.circuitKeeper == nil || k.msgRouter == nil {
		return nil, types.ErrCircuitKeeperNotSet
	}

	for _, url := range msg.MsgTypeUrls {
		if !k.msgRouter.IsRegistered(url) {
			return nil, errorsmod.Wrapf(types.ErrUnknownMsgType, "%s", url)
		}

		disabled, err := k.circuitKeeper.IsDisabled(ctx, url)
		if err != nil {
			return nil, err
		}
		if disabled {
			return nil, errorsmod.Wrapf(types.ErrAlreadyDisabled, "%s", url)
		}
	}

	for _, url := range msg.MsgTypeUrls {
		if err := k.circuitKeeper.Disable(ctx, url); err != nil {
			return nil, err
		}
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.MessagesDisabledEvent{
		Signer:      msg.Signer,
		MsgTypeUrls: msg.MsgTypeUrls,
	}); err != nil {
		return nil, err
	}

	return &types.MsgDisableMessagesResponse{}, nil
}
