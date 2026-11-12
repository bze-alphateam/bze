package app_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	guardiantypes "github.com/bze-alphateam/bze/x/guardian/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// newChannelSwitchApp is the real app with an open transfer channel, a funded sender
// and the guardian set.
func newChannelSwitchApp(t *testing.T) (*app.App, sdk.Context, sdk.AccAddress) {
	t.Helper()

	bApp, ctx := newRateLimitApp(t)
	sender := fundSender(t, bApp, ctx)
	require.NoError(t, bApp.GuardianKeeper.SetParams(ctx, guardiantypes.NewParams(guardianSigner)))

	return bApp, ctx, sender
}

func closeChannel(t *testing.T, bApp *app.App, ctx sdk.Context) {
	t.Helper()
	require.NoError(t, handle(bApp, ctx, &guardiantypes.MsgCloseChannel{Signer: guardianSigner, ChannelId: rlChannel}))
}

// inboundPacket is an ICS-20 packet from the counterparty arriving on rlChannel.
func inboundPacket(receiver sdk.AccAddress) channeltypes.Packet {
	data := ibctransfertypes.NewFungibleTokenPacketData("uatom", "100", "cosmos1sender", receiver.String(), "")

	return channeltypes.NewPacket(
		data.GetBytes(), 1,
		ibctransfertypes.PortID, "channel-9",
		ibctransfertypes.PortID, rlChannel,
		clienttypes.NewHeight(1, 1000), 0,
	)
}

// A transfer message on a closed channel fails through the real message router: the
// transfer keeper sends through the closed-channel gate. Run in a cache context like a
// failed tx, the sender keeps its funds.
func TestGuardianChannel_SendOnClosedChannelFails(t *testing.T) {
	bApp, ctx, sender := newChannelSwitchApp(t)
	closeChannel(t, bApp, ctx)

	before := bApp.BankKeeper.GetBalance(ctx, sender, rlDenom)
	cacheCtx, _ := ctx.CacheContext()
	err := handle(bApp, cacheCtx, newTransfer(sender))

	require.ErrorIs(t, err, txfeecollectortypes.ErrChannelClosed)
	require.Equal(t, before, bApp.BankKeeper.GetBalance(ctx, sender, rlDenom))
	require.Empty(t, bApp.IBCKeeper.ChannelKeeper.GetPacketCommitment(ctx, ibctransfertypes.PortID, rlChannel, 1))
}

// A packet received on a closed channel is answered with an error acknowledgement by
// the routed transfer stack, so nothing is minted for the receiver.
func TestGuardianChannel_RecvOnClosedChannelIsRefused(t *testing.T) {
	bApp, ctx, _ := newChannelSwitchApp(t)
	closeChannel(t, bApp, ctx)
	bApp.TransferKeeper.SetParams(ctx, ibctransfertypes.DefaultParams())

	route, found := bApp.IBCKeeper.PortKeeper.Router.Route(ibctransfertypes.ModuleName)
	require.True(t, found)

	receiver := sdk.AccAddress("channel_receiver____")
	ack := route.OnRecvPacket(ctx, ibctransfertypes.V1, inboundPacket(receiver), sdk.AccAddress{})

	require.False(t, ack.Success())
	require.Equal(t, channeltypes.NewErrorAcknowledgement(txfeecollectortypes.ErrChannelClosed), ack)
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, receiver).IsZero())
}

// A transfer sent before the close still gets its timeout refund after the close.
func TestGuardianChannel_InFlightTransferIsRefundedAfterClose(t *testing.T) {
	bApp, ctx, sender := newChannelSwitchApp(t)
	before := bApp.BankKeeper.GetBalance(ctx, sender, rlDenom)

	msg := newTransfer(sender)
	res, err := bApp.TransferKeeper.Transfer(ctx, msg)
	require.NoError(t, err)
	require.Equal(t, before.Sub(msg.Token), bApp.BankKeeper.GetBalance(ctx, sender, rlDenom))

	closeChannel(t, bApp, ctx)

	data := ibctransfertypes.NewFungibleTokenPacketData(rlDenom, msg.Token.Amount.String(), sender.String(), msg.Receiver, "")
	packet := channeltypes.NewPacket(
		data.GetBytes(), res.Sequence,
		ibctransfertypes.PortID, rlChannel,
		ibctransfertypes.PortID, "channel-9",
		clienttypes.ZeroHeight(), msg.TimeoutTimestamp,
	)

	route, _ := bApp.IBCKeeper.PortKeeper.Router.Route(ibctransfertypes.ModuleName)
	require.NoError(t, route.OnTimeoutPacket(ctx, ibctransfertypes.V1, packet, sdk.AccAddress{}))
	require.Equal(t, before, bApp.BankKeeper.GetBalance(ctx, sender, rlDenom))
}

// The guardian cannot reopen; a governance reopen lets transfers through again.
func TestGuardianChannel_OnlyGovReopens(t *testing.T) {
	bApp, ctx, sender := newChannelSwitchApp(t)
	closeChannel(t, bApp, ctx)

	err := handle(bApp, ctx, &guardiantypes.MsgReopenChannel{Authority: guardianSigner, ChannelId: rlChannel})
	require.ErrorIs(t, err, guardiantypes.ErrUnauthorized)

	require.NoError(t, handle(bApp, ctx, &guardiantypes.MsgReopenChannel{Authority: govAuthority(), ChannelId: rlChannel}))

	res, err := bApp.TransferKeeper.Transfer(ctx, newTransfer(sender))
	require.NoError(t, err)
	require.Equal(t, uint64(1), res.Sequence)
}

// The real channel keeper is wired: an unknown channel cannot be closed.
func TestGuardianChannel_UnknownChannelIsRefused(t *testing.T) {
	bApp, ctx, _ := newChannelSwitchApp(t)

	err := handle(bApp, ctx, &guardiantypes.MsgCloseChannel{Signer: guardianSigner, ChannelId: "channel-77"})
	require.ErrorIs(t, err, guardiantypes.ErrChannelNotFound)
}
