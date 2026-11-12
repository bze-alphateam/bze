package interchaintest_test

import (
	"context"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

const (
	gcGuardianFunds = 100_000_000_000 // ubze for the guardian's tx fees
	gcGas           = 500_000         // fixed gas limit of every tx whose refusal is asserted

	gcStakeIn    = 300_000 // 1. and 7. stake simd -> bze, accepted
	gcUbzeOut    = 200_000 // 1. and 7. ubze bze -> simd, accepted
	gcAckedOut   = 120_000 // 3. ubze in flight while the channel closes, acknowledged after
	gcTimeoutOut = 80_000  // 3. ubze in flight while the channel closes, timed out after
	gcRefusedOut = 50_000  // 4. ubze bze -> simd on the closed channel
	gcRefusedIn  = 70_000  // 5. stake simd -> bze on the closed channel
	gcVoucherIn  = 30_000  // 5. ubze vouchers simd -> bze on the closed channel

	gcUnknownChannel = "channel-99"

	gcUnauthorizedCode  = 1101 // guardian ErrUnauthorized
	gcNotFoundCode      = 1141 // guardian ErrChannelNotFound
	gcAlreadyClosedCode = 1142 // guardian ErrChannelAlreadyClosed
	gcClosedCode        = 1140 // txfeecollector ErrChannelClosed

	// gcClosedAckError is the error acknowledgement of a packet received on a
	// closed channel (txfeecollector ErrChannelClosed): ibc-go keeps only the
	// ABCI code.
	gcClosedAckError = "ABCI code: 1140: error handling packet: see events for details"
	gcClosedMsg      = "ibc transfers on this channel are stopped"
)

// TestGuardianChannelIbc proves the x/guardian channel switch in the real
// transfer stack: bze, an ibc-go simd and Hermes. Only the guardian (or
// governance) can close an existing transfer channel. Once closed, a transfer
// from bze fails as a tx with nothing escrowed, and a transfer from simd is
// answered with an error acknowledgement and refunded on simd, returning ubze
// vouchers included. Transfers already in flight when the channel closes are
// still acknowledged or refunded. The guardian cannot reopen; a governance
// proposal does, and transfers work both ways again. The state carries over
// between the subtests, which run in order.
func TestGuardianChannelIbc(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()

	simdMnemonic, simdAddr := newSimdAccount(t)
	guardianMnemonic, guardianAddr := newAccount(t, chain.Bech32Prefix)
	// the guardian is a test wallet set in genesis, so the suite does not
	// depend on the address the upgrade handler sets on mainnet
	net := chain.StartIbc(t, ctx, []ibc.WalletAmount{{
		Address: simdAddr,
		Denom:   ibcOtherDenom,
		Amount:  sdkmath.NewInt(ibcOtherFunds),
	}}, chain.WithGenesis(cosmos.NewGenesisKV("app_state.guardian.params.guardian", guardianAddr)))
	s := newGuardianChannelSuite(t, ctx, net, simdMnemonic, guardianMnemonic, guardianAddr)

	t.Run("1 baseline, no channel closed", s.baseline)
	t.Run("2 only the guardian closes an existing channel", s.closeRefused)
	t.Run("3 the guardian closes, in-flight transfers still settle", s.closeWithInFlight)
	t.Run("4 outbound on the closed channel is refused", s.outboundRefused)
	t.Run("5 inbound on the closed channel is refunded", s.inboundRefunded)
	t.Run("6 the guardian cannot reopen", s.guardianCannotReopen)
	t.Run("7 governance reopens", s.govReopens)
}

type guardianChannelSuite struct {
	*ibcSuite

	guardian ibc.Wallet
}

func newGuardianChannelSuite(t *testing.T, ctx context.Context, net chain.IbcNetwork, simdMnemonic, guardianMnemonic, guardianAddr string) *guardianChannelSuite {
	t.Helper()
	s := setupIbc(t, ctx, net, simdMnemonic)
	g := &guardianChannelSuite{ibcSuite: s}

	var err error
	g.guardian, err = interchaintest.GetAndFundTestUserWithMnemonic(ctx, "guardian", guardianMnemonic, sdkmath.NewInt(gcGuardianFunds), s.bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, g.guardian.FormattedAddress())
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, s.bze, s.simd))

	guardian, err := helpers.QueryGuardian(ctx, s.bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, guardian, "the genesis guardian")
	return g
}

// ---------------------------------------------------------------------------
// queries and transactions

func (g *guardianChannelSuite) closedChannels(t *testing.T) []string {
	t.Helper()
	res, err := helpers.Query[struct {
		ChannelIds []string `json:"channel_ids"`
	}](g.ctx, g.bze, "guardian", "closed-channels")
	require.NoError(t, err)
	return res.ChannelIds
}

// tx sends a tx with a fixed gas limit, so a refusal is broadcast, included
// and charged, and comes back with its code and codespace.
func (g *guardianChannelSuite) tx(t *testing.T, user ibc.Wallet, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTxWith(g.ctx, g.bze, user, []helpers.TxOption{helpers.WithGas(gcGas)}, args...)
	require.NoError(t, err)
	return res
}

func (g *guardianChannelSuite) requireCode(t *testing.T, res helpers.TxResult, codespace string, code uint32) {
	t.Helper()
	require.Equal(t, codespace, res.Codespace, res.RawLog)
	require.EqualValues(t, code, res.Code, res.RawLog)
}

// reopenProposal is a governance proposal reopening the main channel with
// authority as the message signer.
func (g *guardianChannelSuite) reopenProposal(authority, title string) helpers.Proposal {
	return helpers.Proposal{
		Messages: []any{map[string]any{"@type": "/bze.guardian.v1.MsgReopenChannel", "authority": authority, "channel_id": g.main.bze}},
		Title:    title,
		Summary:  "e2e: " + title,
	}
}

// requireInboundRefunded asserts bze answered the inbound transfer with the
// closed-channel error acknowledgement without handling it, and simd refunded
// the sender.
func (g *guardianChannelSuite) requireInboundRefunded(t *testing.T, in inbound) {
	t.Helper()
	require.Equal(t, `{"error":"`+gcClosedAckError+`"}`, in.writtenAck(t))
	require.Empty(t, in.recvTransferEvents(), "the transfer app did not handle the packet")
	require.Len(t, helpers.EventsWith(helpers.TxEvents(in.ack, ibcTransferEvent), map[string]string{"error": gcClosedAckError}), 1, "simd processed the error ack")

	sender := in.senderBefore
	if in.simdDenom == chain.SimdDenom {
		sender = sender.Sub(in.fee)
	}
	voucher := helpers.IbcDenom(in.ch.bze, in.denom)
	require.Equal(t, sender, g.balance(t, g.simd, g.simdUser, in.simdDenom), "simd user refunded, minus the tx fee only")
	require.Equal(t, in.escrowBefore, g.balanceOf(t, g.simd, in.ch.simdEscrow, in.simdDenom), "simd escrow restored")
	require.Equal(t, in.receiverBefore, g.balance(t, g.bze, g.bzeUser, voucher), "nothing credited on bze")
	require.Equal(t, in.supplyBefore, g.supply(t, g.bze, voucher), "bze supply unchanged")
}

// ---------------------------------------------------------------------------
// the subtests

// 1. Nothing is closed after genesis and transfers work both ways.
func (g *guardianChannelSuite) baseline(t *testing.T) {
	require.Empty(t, g.closedChannels(t), "no channel closed after genesis")

	g.transferIn(t, chain.SimdDenom, gcStakeIn).requireAccepted(t)
	g.transferOut(t, gcUbzeOut)
}

// 2. A user who is neither the guardian nor governance cannot close the
// channel, and the guardian cannot close a channel that does not exist.
func (g *guardianChannelSuite) closeRefused(t *testing.T) {
	res := g.tx(t, g.bzeUser, "guardian", "close-channel", g.main.bze)
	g.requireCode(t, res, "guardian", gcUnauthorizedCode)

	res = g.tx(t, g.guardian, "guardian", "close-channel", gcUnknownChannel)
	g.requireCode(t, res, "guardian", gcNotFoundCode)

	require.Empty(t, g.closedChannels(t), "still open")
	g.transferOut(t, gcUbzeOut)
}

// 3. With the relayer stopped, bzeUser sends two ubze transfers: one with the
// default timeout and one that times out on simd. The guardian closes the
// channel, then the relayer resumes: the first is received on simd and its
// acknowledgement still reaches bze, the second is refunded on bze by its
// timeout. Neither is stranded by the close.
func (g *guardianChannelSuite) closeWithInFlight(t *testing.T) {
	r, rep := g.net.Relayer, g.net.Reporter
	ubze := g.balance(t, g.bze, g.bzeUser, chain.Denom)
	escrow := g.balanceOf(t, g.bze, g.bzeEscrow, chain.Denom)
	vouchers := g.balance(t, g.simd, g.simdUser, g.ubzeVoucher)

	require.NoError(t, r.StopRelayer(g.ctx, rep))
	acked, err := helpers.SendIbc(g.ctx, g.bze, g.bzeUser, g.main.bze, g.simdUser.FormattedAddress(), sdk.NewInt64Coin(chain.Denom, gcAckedOut))
	require.NoError(t, err)
	// a timeout 1 s from now (the CLI's clock), passed on simd a few blocks
	// later while nothing relays the packet
	timedOut, err := helpers.SendIbc(g.ctx, g.bze, g.bzeUser, g.main.bze, g.simdUser.FormattedAddress(), sdk.NewInt64Coin(chain.Denom, gcTimeoutOut),
		"--packet-timeout-height", "0-0", "--packet-timeout-timestamp", "1000000000")
	require.NoError(t, err)
	require.Equal(t, escrow.AddRaw(gcAckedOut+gcTimeoutOut), g.balanceOf(t, g.bze, g.bzeEscrow, chain.Denom), "both escrowed while in flight")
	require.NoError(t, testutil.WaitForBlocks(g.ctx, 3, g.simd))

	res := g.tx(t, g.guardian, "guardian", "close-channel", g.main.bze)
	require.Zero(t, res.Code, res.RawLog)
	require.Equal(t, []string{g.main.bze}, g.closedChannels(t))

	require.NoError(t, r.StartRelayer(g.ctx, rep, chain.IbcPath))
	_, err = helpers.WaitForRecv(g.ctx, g.simd, g.main.simd, acked.Sequence)
	require.NoError(t, err)
	_, err = helpers.WaitForAck(g.ctx, g.bze, acked)
	require.NoError(t, err)
	_, err = helpers.WaitForTimeout(g.ctx, g.bze, timedOut)
	require.NoError(t, err)

	ackedFee, err := helpers.TxFee(acked.Tx, chain.Denom)
	require.NoError(t, err)
	timeoutFee, err := helpers.TxFee(timedOut.Tx, chain.Denom)
	require.NoError(t, err)
	require.Equal(t, ubze.SubRaw(gcAckedOut).Sub(ackedFee).Sub(timeoutFee), g.balance(t, g.bze, g.bzeUser, chain.Denom), "the timed-out transfer was refunded")
	require.Equal(t, escrow.AddRaw(gcAckedOut), g.balanceOf(t, g.bze, g.bzeEscrow, chain.Denom), "only the acknowledged transfer stays escrowed")
	require.Equal(t, vouchers.AddRaw(gcAckedOut), g.balance(t, g.simd, g.simdUser, g.ubzeVoucher), "the acknowledged transfer arrived on simd")
}

// 4. A transfer from bze on the closed channel fails as a tx: no packet is
// committed, nothing is escrowed and the sender loses only the fee.
func (g *guardianChannelSuite) outboundRefused(t *testing.T) {
	ubze := g.balance(t, g.bze, g.bzeUser, chain.Denom)
	escrow := g.balanceOf(t, g.bze, g.bzeEscrow, chain.Denom)

	res, err := helpers.TrySendIbc(g.ctx, g.bze, g.bzeUser, g.main.bze, g.simdUser.FormattedAddress(), sdk.NewInt64Coin(chain.Denom, gcRefusedOut), gcGas)
	require.NoError(t, err)
	g.requireCode(t, res, "txfeecollector", gcClosedCode)
	require.Contains(t, res.RawLog, gcClosedMsg)
	require.Empty(t, helpers.TxEvents(res, "send_packet"), "no packet was committed")

	fee, err := helpers.FixedGasFee(g.bze, gcGas)
	require.NoError(t, err)
	require.Equal(t, ubze.Sub(fee), g.balance(t, g.bze, g.bzeUser, chain.Denom), "only the fee was lost")
	require.Equal(t, escrow, g.balanceOf(t, g.bze, g.bzeEscrow, chain.Denom), "nothing was escrowed")
}

// 5. A transfer from simd on the closed channel is answered with an error
// acknowledgement and refunded on simd. Returning ubze vouchers, which an
// inbound denom filter would let through, are refused too: the whole channel
// is stopped.
func (g *guardianChannelSuite) inboundRefunded(t *testing.T) {
	g.requireInboundRefunded(t, g.transferIn(t, chain.SimdDenom, gcRefusedIn))

	ubze := g.balance(t, g.bze, g.bzeUser, chain.Denom)
	escrow := g.balanceOf(t, g.bze, g.bzeEscrow, chain.Denom)
	vouchers := g.balance(t, g.simd, g.simdUser, g.ubzeVoucher)
	voucherSupply := g.supply(t, g.simd, g.ubzeVoucher)

	out, err := helpers.SendIbc(g.ctx, g.simd, g.simdUser, g.main.simd, g.bzeUser.FormattedAddress(), sdk.NewInt64Coin(g.ubzeVoucher, gcVoucherIn))
	require.NoError(t, err)
	recv, err := helpers.WaitForRecv(g.ctx, g.bze, g.main.bze, out.Sequence)
	require.NoError(t, err)
	ack, err := helpers.WaitForAck(g.ctx, g.simd, out)
	require.NoError(t, err)

	require.Equal(t, `{"error":"`+gcClosedAckError+`"}`, packetAck(t, recv, g.main.bze, out.Sequence))
	require.Len(t, helpers.EventsWith(helpers.TxEvents(ack, ibcTransferEvent), map[string]string{"error": gcClosedAckError}), 1, "simd processed the error ack")
	require.Equal(t, ubze, g.balance(t, g.bze, g.bzeUser, chain.Denom), "nothing released on bze")
	require.Equal(t, escrow, g.balanceOf(t, g.bze, g.bzeEscrow, chain.Denom), "bze escrow unchanged")
	require.Equal(t, vouchers, g.balance(t, g.simd, g.simdUser, g.ubzeVoucher), "vouchers refunded on simd")
	require.Equal(t, voucherSupply, g.supply(t, g.simd, g.ubzeVoucher), "voucher supply restored on simd")
}

// 6. The guardian cannot reopen: the reopen message has the gov authority as
// signer, and closing a closed channel again is refused.
func (g *guardianChannelSuite) guardianCannotReopen(t *testing.T) {
	_, err := helpers.SubmitAndPassProposal(g.ctx, g.bze, g.guardian, g.reopenProposal(g.guardian.FormattedAddress(), "Reopen as the guardian"))
	require.ErrorContains(t, err, "expected gov account as only signer")
	require.Equal(t, []string{g.main.bze}, g.closedChannels(t), "still closed")

	res := g.tx(t, g.guardian, "guardian", "close-channel", g.main.bze)
	g.requireCode(t, res, "guardian", gcAlreadyClosedCode)
}

// 7. A governance proposal reopens the channel and transfers work both ways
// again.
func (g *guardianChannelSuite) govReopens(t *testing.T) {
	_, err := helpers.SubmitAndPassProposal(g.ctx, g.bze, g.admin, g.reopenProposal(g.gov, "Reopen the main channel"))
	require.NoError(t, err)
	require.Empty(t, g.closedChannels(t))

	g.transferIn(t, chain.SimdDenom, gcStakeIn).requireAccepted(t)
	g.transferOut(t, gcUbzeOut)
}
