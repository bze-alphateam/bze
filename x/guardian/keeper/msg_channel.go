package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// SetChannelKeeper wires the IBC channel keeper. The IBC keepers are not built
// by depinject: the app creates them after the depinject container.
func (k *Keeper) SetChannelKeeper(channelKeeper types.ChannelKeeper) {
	k.channelKeeper = channelKeeper
}

// CloseChannel stops every ICS-20 transfer on an existing transfer channel, in
// both directions: sends fail and received packets get an error acknowledgement,
// so the sending chain refunds them. Acknowledgements and timeouts of packets
// already in flight still go through. The channel stays open in IBC.
func (k msgServer) CloseChannel(goCtx context.Context, msg *types.MsgCloseChannel) (*types.MsgCloseChannelResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.authorize(ctx, msg.Signer); err != nil {
		return nil, err
	}

	if k.channelKeeper == nil {
		return nil, types.ErrChannelKeeperNotSet
	}

	if _, found := k.channelKeeper.GetChannel(ctx, ibctransfertypes.PortID, msg.ChannelId); !found {
		return nil, errorsmod.Wrapf(types.ErrChannelNotFound, "%s/%s", ibctransfertypes.PortID, msg.ChannelId)
	}

	if k.IsChannelClosed(ctx, msg.ChannelId) {
		return nil, errorsmod.Wrapf(types.ErrChannelAlreadyClosed, "channel %s", msg.ChannelId)
	}

	k.SetClosedChannel(ctx, msg.ChannelId)

	if err := ctx.EventManager().EmitTypedEvent(&types.ChannelClosedEvent{
		Signer:    msg.Signer,
		ChannelId: msg.ChannelId,
	}); err != nil {
		return nil, err
	}

	return &types.MsgCloseChannelResponse{}, nil
}

// ReopenChannel lets transfers through a closed channel again. Only the
// governance authority can sign it: the guardian can close, never reopen.
func (k msgServer) ReopenChannel(goCtx context.Context, msg *types.MsgReopenChannel) (*types.MsgReopenChannelResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if !k.isAuthority(msg.Authority) {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "only the governance authority can reopen, got %s", msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if !k.IsChannelClosed(ctx, msg.ChannelId) {
		return nil, errorsmod.Wrapf(types.ErrChannelNotClosed, "channel %s", msg.ChannelId)
	}

	k.RemoveClosedChannel(ctx, msg.ChannelId)

	if err := ctx.EventManager().EmitTypedEvent(&types.ChannelReopenedEvent{
		Authority: msg.Authority,
		ChannelId: msg.ChannelId,
	}); err != nil {
		return nil, err
	}

	return &types.MsgReopenChannelResponse{}, nil
}
