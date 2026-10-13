package interchaintest_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

const (
	ibcBzeFunds  = 100_000_000_000 // 100,000 BZE: proposals and tx fees
	ibcSimdFunds = 10_000_000_000  // stake; simd txs are free
	// ibcOtherDenom is a second simd-origin denom, held by simdUser from genesis
	ibcOtherDenom = "utoken"
	ibcOtherFunds = 1_000_000_000

	ibcStakeIn     = 1_000_000 // 1. stake simd -> bze
	ibcBzeOut      = 1_000_000 // 1. and 5. ubze bze -> simd
	ibcBlockedIn   = 500_000   // 3. the refused stake transfer, 7. the one accepted again
	ibcOtherIn     = 250_000   // 4. utoken simd -> bze
	ibcStakeHome   = 400_000   // 5. stake vouchers bze -> simd
	ibcVoucherHome = 600_000   // 6. ubze vouchers simd -> bze
	ibcTimeoutOut  = 100_000   // 8. ubze bze -> simd, timed out
	ibcFeeIn       = 200_000   // 9. stake and utoken over the fee channel
	ibcScopeIn     = 150_000   // 10. stake over the second and the main channel
	ibcMultiHopOut = 300_000   // 10. stake vouchers bze -> simd over the second channel
	ibcMultiHopIn  = 100_000   // 10. those vouchers back to bze over the main channel, per transfer
	ibcErrAckOut   = 200_000   // 11. ubze bze -> an address simd cannot decode
	ibcUpgradeIn   = 120_000   // 12. and 13. stake and utoken over the upgraded channel

	// ibcTransferVersion is the plain ICS-20 channel version, ibcFeeVersion
	// the ICS-29 fee middleware wrapping it.
	ibcTransferVersion = "ics20-1"
	ibcFeeVersion      = `{"fee_version":"ics29-1","app_version":"ics20-1"}`

	ibcBlockedEvent = "bze.txfeecollector.BlockedIbcInboundEvent"
	// ibcCallbackErrorPrefix is what ibc-go puts on the type and attribute keys
	// of the events an OnRecvPacket callback emitted when it returns an error
	// ack: they are rewritten, not dropped (accepted as is in the v8.2.0 review).
	ibcCallbackErrorPrefix = "ibccallbackerror-"
	// ibcBlockedAckError is the error acknowledgement of ErrBlockedIbcInbound
	// (txfeecollector code 1101): ibc-go keeps only the ABCI code.
	ibcBlockedAckError = "ABCI code: 1101: error handling packet: see events for details"
	ibcTransferEvent   = "fungible_token_packet"
	// ibcInvalidReceiverAckError is the error acknowledgement of a receiver
	// the transfer app cannot decode: the bech32 error is not a registered
	// error, so it carries the internal ABCI code 1.
	ibcInvalidReceiverAckError = "ABCI code: 1: error handling packet: see events for details"
)

// TestBlockedInboundIbc connects bze to an ibc-go simd chain through the Hermes
// relayer and proves the txfeecollector inbound filter in the real transfer
// stack: a (channel, base denom) pair blocked by a governance proposal is
// answered with an error acknowledgement, refunded on simd and leaves no
// voucher on bze, while other denoms, outgoing transfers, returning BZE
// vouchers and timeouts behave as without the filter. Steps 9-13 add an
// ICS-29 fee channel, the channel scope and exact path match of the list, an
// error ack on an outgoing transfer and channel upgrades started from either
// chain, which reach the transfer app through the filter. Subtests run in order
// and share state; the step numbers match the scenario of the suite's design.
func TestBlockedInboundIbc(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()

	mnemonic, simdAddr := newSimdAccount(t)
	net := chain.StartIbc(t, ctx, []ibc.WalletAmount{{
		Address: simdAddr,
		Denom:   ibcOtherDenom,
		Amount:  sdkmath.NewInt(ibcOtherFunds),
	}})
	s := setupIbc(t, ctx, net, mnemonic)

	t.Run("1 baseline, nothing blocked", s.baseline)
	t.Run("2 block by governance", s.block)
	t.Run("3 blocked inbound is refunded", s.blockedInbound)
	t.Run("4 other denoms on the channel pass", s.otherDenom)
	t.Run("5 outgoing still works", s.outgoing)
	t.Run("6 returning vouchers pass", s.returningVouchers)
	t.Run("7 unblock by governance", s.unblock)
	t.Run("8 timeout refunds", s.timeout)
	t.Run("9 fee-enabled channel is filtered", s.feeChannel)
	t.Run("10 blocking is channel-scoped and path-exact", s.channelScope)
	t.Run("11 outgoing error ack refunds", s.outgoingErrorAck)
	t.Run("12 channel upgrade initiated on bze", s.upgradeFromBze)
	t.Run("13 channel upgrade initiated on simd", s.upgradeFromSimd)
}

type ibcSuite struct {
	ctx       context.Context
	net       chain.IbcNetwork
	bze, simd *cosmos.CosmosChain

	admin, bzeUser, simdUser ibc.Wallet
	gov                      string

	// stakeVoucher and otherVoucher are the simd denoms on bze, ubzeVoucher
	// is ubze on simd
	stakeVoucher, otherVoucher, ubzeVoucher string
	// bzeEscrow escrows ubze on bze, simdEscrow escrows stake on simd
	bzeEscrow, simdEscrow string

	// main is the channel StartIbc opened; fee (step 9) and second (step 10,
	// upgraded in steps 12 and 13) are opened by the suite
	main, fee, second ibcChannel
}

// ibcChannel is a transfer channel between bze and simd: its id and escrow
// account on each end, and whether the ICS-29 fee middleware is enabled on it.
type ibcChannel struct {
	bze, simd             string
	bzeEscrow, simdEscrow string
	fee                   bool
}

func setupIbc(t *testing.T, ctx context.Context, net chain.IbcNetwork, simdMnemonic string) *ibcSuite {
	s := &ibcSuite{ctx: ctx, net: net, bze: net.Bze, simd: net.Simd}

	users := interchaintest.GetAndFundTestUsers(t, ctx, "ibc", sdkmath.NewInt(ibcBzeFunds), s.bze, s.bze)
	s.admin, s.bzeUser = users[0], users[1]
	var err error
	s.simdUser, err = interchaintest.GetAndFundTestUserWithMnemonic(ctx, "ibc-simd", simdMnemonic, sdkmath.NewInt(ibcSimdFunds), s.simd)
	require.NoError(t, err)
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, s.bze, s.simd))
	require.Equal(t, sdkmath.NewInt(ibcOtherFunds), s.balance(t, s.simd, s.simdUser, ibcOtherDenom), "simd user holds the genesis utoken")

	s.gov, err = helpers.GovAuthority(ctx, s.bze)
	require.NoError(t, err)
	s.stakeVoucher = helpers.IbcDenom(net.BzeChannel, chain.SimdDenom)
	s.otherVoucher = helpers.IbcDenom(net.BzeChannel, ibcOtherDenom)
	s.ubzeVoucher = helpers.IbcDenom(net.SimdChannel, chain.Denom)
	s.bzeEscrow, err = helpers.EscrowAddress(chain.Bech32Prefix, net.BzeChannel)
	require.NoError(t, err)
	s.simdEscrow, err = helpers.EscrowAddress(chain.SimdBech32Prefix, net.SimdChannel)
	require.NoError(t, err)
	s.main = s.newChannel(t, net.BzeChannel, net.SimdChannel, false)

	require.Empty(t, s.blockedParam(t), "nothing is blocked by default")
	return s
}

// 1. With nothing blocked, stake arrives on bze as the voucher of
// transfer/<bze channel>/stake and ubze arrives on simd as a voucher.
func (s *ibcSuite) baseline(t *testing.T) {
	in := s.transferIn(t, chain.SimdDenom, ibcStakeIn)
	in.requireAccepted(t)
	require.Equal(t, sdkmath.NewInt(ibcStakeIn), s.balance(t, s.bze, s.bzeUser, s.stakeVoucher))
	require.Equal(t, sdkmath.NewInt(ibcStakeIn), s.supply(t, s.bze, s.stakeVoucher))

	s.transferOut(t, ibcBzeOut)
	require.Equal(t, sdkmath.NewInt(ibcBzeOut), s.balance(t, s.simd, s.simdUser, s.ubzeVoucher))
}

// 2. A txfeecollector MsgUpdateParams proposal carrying the full params
// blocks stake on the bze channel; the other params are unchanged.
func (s *ibcSuite) block(t *testing.T) {
	before := s.otherParams(t)
	s.setBlocked(t, "Block stake", blockedEntry{s.net.BzeChannel, chain.SimdDenom})
	require.Equal(t, []blockedEntry{{s.net.BzeChannel, chain.SimdDenom}}, s.blockedParam(t))
	require.Equal(t, before, s.otherParams(t), "only blocked_ibc_inbound changed")
}

// 3. The blocked transfer is answered with an error acknowledgement: bze
// emits BlockedIbcInboundEvent and mints nothing, simd refunds the sender.
func (s *ibcSuite) blockedInbound(t *testing.T) {
	s.transferIn(t, chain.SimdDenom, ibcBlockedIn).requireRefused(t)
}

// 4. A different simd denom on the same channel is not listed and arrives.
func (s *ibcSuite) otherDenom(t *testing.T) {
	in := s.transferIn(t, ibcOtherDenom, ibcOtherIn)
	in.requireAccepted(t)
	require.Equal(t, sdkmath.NewInt(ibcOtherIn), s.balance(t, s.bze, s.bzeUser, s.otherVoucher))
	require.Equal(t, sdkmath.NewInt(ibcOtherFunds-ibcOtherIn), s.balance(t, s.simd, s.simdUser, ibcOtherDenom))
}

// 5. Outgoing transfers are untouched: ubze leaves bze, and stake vouchers go
// home to simd, where they are released from escrow.
func (s *ibcSuite) outgoing(t *testing.T) {
	s.transferOut(t, ibcBzeOut)
	require.Equal(t, sdkmath.NewInt(2*ibcBzeOut), s.balance(t, s.simd, s.simdUser, s.ubzeVoucher))

	vouchers := s.balance(t, s.bze, s.bzeUser, s.stakeVoucher)
	voucherSupply := s.supply(t, s.bze, s.stakeVoucher)
	stake := s.balance(t, s.simd, s.simdUser, chain.SimdDenom)
	escrow := s.balanceOf(t, s.simd, s.simdEscrow, chain.SimdDenom)

	out, err := helpers.SendIbc(s.ctx, s.bze, s.bzeUser, s.net.BzeChannel, s.simdUser.FormattedAddress(), sdk.NewInt64Coin(s.stakeVoucher, ibcStakeHome))
	require.NoError(t, err)
	_, err = helpers.WaitForRecv(s.ctx, s.simd, s.net.SimdChannel, out.Sequence)
	require.NoError(t, err)
	_, err = helpers.WaitForAck(s.ctx, s.bze, out)
	require.NoError(t, err)

	require.Equal(t, vouchers.SubRaw(ibcStakeHome), s.balance(t, s.bze, s.bzeUser, s.stakeVoucher))
	require.Equal(t, voucherSupply.SubRaw(ibcStakeHome), s.supply(t, s.bze, s.stakeVoucher), "vouchers burned on bze")
	require.Equal(t, stake.AddRaw(ibcStakeHome), s.balance(t, s.simd, s.simdUser, chain.SimdDenom))
	require.Equal(t, escrow.SubRaw(ibcStakeHome), s.balanceOf(t, s.simd, s.simdEscrow, chain.SimdDenom), "released from simd escrow")
}

// 6. ubze vouchers coming home are never blocked: even with entries for
// "ubze" and for the exact denom the packet carries
// ("transfer/<simd channel>/ubze"), they arrive as native ubze.
func (s *ibcSuite) returningVouchers(t *testing.T) {
	packetDenom := chain.TransferPort + "/" + s.net.SimdChannel + "/" + chain.Denom
	s.setBlocked(t, "Block ubze",
		blockedEntry{s.net.BzeChannel, chain.SimdDenom},
		blockedEntry{s.net.BzeChannel, chain.Denom},
		blockedEntry{s.net.BzeChannel, packetDenom},
	)
	require.Len(t, s.blockedParam(t), 3)

	ubze := s.balance(t, s.bze, s.bzeUser, chain.Denom)
	escrow := s.balanceOf(t, s.bze, s.bzeEscrow, chain.Denom)
	vouchers := s.balance(t, s.simd, s.simdUser, s.ubzeVoucher)
	voucherSupply := s.supply(t, s.simd, s.ubzeVoucher)

	out, err := helpers.SendIbc(s.ctx, s.simd, s.simdUser, s.net.SimdChannel, s.bzeUser.FormattedAddress(), sdk.NewInt64Coin(s.ubzeVoucher, ibcVoucherHome))
	require.NoError(t, err)
	recv, err := helpers.WaitForRecv(s.ctx, s.bze, s.net.BzeChannel, out.Sequence)
	require.NoError(t, err)
	_, err = helpers.WaitForAck(s.ctx, s.simd, out)
	require.NoError(t, err)

	require.Empty(t, blockedEvents(recv), "a returning voucher is never refused")
	accepted := helpers.EventsWith(helpers.TxEvents(recv, ibcTransferEvent), map[string]string{
		"denom": packetDenom, "amount": strconv.Itoa(ibcVoucherHome), "success": "true",
	})
	require.Len(t, accepted, 1, "the transfer app accepted the packet")

	require.Equal(t, ubze.AddRaw(ibcVoucherHome), s.balance(t, s.bze, s.bzeUser, chain.Denom), "arrived as native ubze")
	require.Equal(t, escrow.SubRaw(ibcVoucherHome), s.balanceOf(t, s.bze, s.bzeEscrow, chain.Denom), "released from bze escrow")
	require.Equal(t, vouchers.SubRaw(ibcVoucherHome), s.balance(t, s.simd, s.simdUser, s.ubzeVoucher))
	require.Equal(t, voucherSupply.SubRaw(ibcVoucherHome), s.supply(t, s.simd, s.ubzeVoucher), "vouchers burned on simd")
}

// 7. A proposal removes every entry and stake arrives again.
func (s *ibcSuite) unblock(t *testing.T) {
	s.setBlocked(t, "Unblock")
	require.Empty(t, s.blockedParam(t))

	vouchers := s.balance(t, s.bze, s.bzeUser, s.stakeVoucher)
	in := s.transferIn(t, chain.SimdDenom, ibcBlockedIn)
	in.requireAccepted(t)
	require.Equal(t, vouchers.AddRaw(ibcBlockedIn), s.balance(t, s.bze, s.bzeUser, s.stakeVoucher))
}

// 8. With the relayer stopped, a bze -> simd transfer outlives its timeout.
// Once the relayer is back it proves the timeout on bze and the sender is
// refunded: the filter forwards OnTimeoutPacket.
func (s *ibcSuite) timeout(t *testing.T) {
	r, rep := s.net.Relayer, s.net.Reporter
	require.NoError(t, r.StopRelayer(s.ctx, rep))

	ubze := s.balance(t, s.bze, s.bzeUser, chain.Denom)
	escrow := s.balanceOf(t, s.bze, s.bzeEscrow, chain.Denom)
	vouchers := s.balance(t, s.simd, s.simdUser, s.ubzeVoucher)

	// a timeout 1 s from now (the CLI's clock), passed on simd a few blocks
	// later while nothing relays the packet
	out, err := helpers.SendIbc(s.ctx, s.bze, s.bzeUser, s.net.BzeChannel, s.simdUser.FormattedAddress(), sdk.NewInt64Coin(chain.Denom, ibcTimeoutOut),
		"--packet-timeout-height", "0-0", "--packet-timeout-timestamp", "1000000000")
	require.NoError(t, err)
	fee, err := helpers.TxFee(out.Tx, chain.Denom)
	require.NoError(t, err)
	require.Equal(t, escrow.AddRaw(ibcTimeoutOut), s.balanceOf(t, s.bze, s.bzeEscrow, chain.Denom), "escrowed while in flight")
	require.NoError(t, testutil.WaitForBlocks(s.ctx, 3, s.simd))

	require.NoError(t, r.StartRelayer(s.ctx, rep, chain.IbcPath))
	res, err := helpers.WaitForTimeout(s.ctx, s.bze, out)
	require.NoError(t, err)
	refunds := helpers.EventsWith(helpers.TxEvents(res, "timeout"), map[string]string{
		"refund_receiver": s.bzeUser.FormattedAddress(),
		"refund_denom":    chain.Denom,
		"refund_amount":   strconv.Itoa(ibcTimeoutOut),
	})
	require.Len(t, refunds, 1, "the transfer app refunded the timed-out packet")

	require.Equal(t, ubze.Sub(fee), s.balance(t, s.bze, s.bzeUser, chain.Denom), "refunded, minus the tx fee only")
	require.Equal(t, escrow, s.balanceOf(t, s.bze, s.bzeEscrow, chain.Denom), "bze escrow released")
	require.Equal(t, vouchers, s.balance(t, s.simd, s.simdUser, s.ubzeVoucher), "nothing minted on simd")
}

// 9. A channel opened with the ICS-29 fee version stacks the fee middleware
// over the filter on both ends. A denom blocked on it is refused exactly as
// on a plain channel (the error ack wrapped in the fee acknowledgement) and
// an unlisted denom arrives.
func (s *ibcSuite) feeChannel(t *testing.T) {
	s.fee = s.openChannel(t, ibcFeeVersion)
	s.setBlocked(t, "Block stake on the fee channel", blockedEntry{s.fee.bze, chain.SimdDenom})

	s.transferInOn(t, s.fee, chain.SimdDenom, ibcFeeIn).requireRefused(t)
	s.transferInOn(t, s.fee, ibcOtherDenom, ibcFeeIn).requireAccepted(t)
}

// 10. The filter compares the packet's destination channel and denom exactly.
// stake blocked on the main channel arrives through a second channel between
// the same chains. A voucher path is matched whole: simd sends back stake
// vouchers it holds under "transfer/<simd second>/transfer/<bze main>/stake";
// an entry for the base denom stake does not catch that path, an entry for
// the path does and no longer catches plain stake.
func (s *ibcSuite) channelScope(t *testing.T) {
	s.second = s.openChannel(t, ibcTransferVersion)
	s.setBlocked(t, "Block stake on the main channel", blockedEntry{s.main.bze, chain.SimdDenom})

	s.transferInOn(t, s.second, chain.SimdDenom, ibcScopeIn).requireAccepted(t)
	s.transferIn(t, chain.SimdDenom, ibcScopeIn).requireRefused(t)

	// stake vouchers of the main channel leave bze over the second channel
	path := strings.Join([]string{chain.TransferPort, s.second.simd, chain.TransferPort, s.main.bze, chain.SimdDenom}, "/")
	out, err := helpers.SendIbc(s.ctx, s.bze, s.bzeUser, s.second.bze, s.simdUser.FormattedAddress(), sdk.NewInt64Coin(s.stakeVoucher, ibcMultiHopOut))
	require.NoError(t, err)
	_, err = helpers.WaitForRecv(s.ctx, s.simd, s.second.simd, out.Sequence)
	require.NoError(t, err)
	_, err = helpers.WaitForAck(s.ctx, s.bze, out)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(ibcMultiHopOut), s.balance(t, s.simd, s.simdUser, simdDenomOf(path)), "simd holds the two-hop vouchers")

	// back over the main channel the packet carries the path, not "stake"
	s.transferIn(t, path, ibcMultiHopIn).requireAccepted(t)

	s.setBlocked(t, "Block the stake voucher path on the main channel", blockedEntry{s.main.bze, path})
	s.transferIn(t, path, ibcMultiHopIn).requireRefused(t)
	s.transferIn(t, chain.SimdDenom, ibcScopeIn).requireAccepted(t)
}

// 11. An outgoing transfer to a receiver simd cannot decode (a bze address)
// is answered by simd's transfer app with an error acknowledgement. The
// filter forwards it to the transfer app, which refunds the sender exactly:
// the escrow is back where it was and nothing was minted on simd.
func (s *ibcSuite) outgoingErrorAck(t *testing.T) {
	ubze := s.balance(t, s.bze, s.bzeUser, chain.Denom)
	escrow := s.balanceOf(t, s.bze, s.main.bzeEscrow, chain.Denom)
	voucherSupply := s.supply(t, s.simd, s.ubzeVoucher)
	receiver := s.bzeUser.FormattedAddress()

	out, err := helpers.SendIbc(s.ctx, s.bze, s.bzeUser, s.main.bze, receiver, sdk.NewInt64Coin(chain.Denom, ibcErrAckOut))
	require.NoError(t, err)
	fee, err := helpers.TxFee(out.Tx, chain.Denom)
	require.NoError(t, err)
	require.Equal(t, escrow.AddRaw(ibcErrAckOut), s.balanceOf(t, s.bze, s.main.bzeEscrow, chain.Denom), "escrowed while in flight")

	// simd: the transfer app refused the receiver
	recv, err := helpers.WaitForRecv(s.ctx, s.simd, s.main.simd, out.Sequence)
	require.NoError(t, err)
	refused := helpers.EventsWith(callbackEvents(recv, ibcTransferEvent), map[string]string{
		"receiver": receiver,
		"denom":    chain.Denom,
		"amount":   strconv.Itoa(ibcErrAckOut),
		"success":  "false",
	})
	require.Len(t, refused, 1, "simd's transfer app refused the packet: %+v", callbackEvents(recv, ibcTransferEvent))
	require.Equal(t, "failed to decode receiver address: "+receiver+": invalid Bech32 prefix; expected "+chain.SimdBech32Prefix+", got "+chain.Bech32Prefix,
		refused[0].Attributes["error"])
	require.Equal(t, `{"error":"`+ibcInvalidReceiverAckError+`"}`, packetAck(t, recv, s.main.simd, out.Sequence))

	// bze: the error ack reached the transfer app and refunded the sender
	ack, err := helpers.WaitForAck(s.ctx, s.bze, out)
	require.NoError(t, err)
	require.Len(t, helpers.EventsWith(helpers.TxEvents(ack, ibcTransferEvent), map[string]string{"error": ibcInvalidReceiverAckError}), 1, "bze processed the error ack")

	require.Equal(t, ubze.Sub(fee), s.balance(t, s.bze, s.bzeUser, chain.Denom), "refunded, minus the tx fee only")
	require.Equal(t, escrow, s.balanceOf(t, s.bze, s.main.bzeEscrow, chain.Denom), "bze escrow released")
	require.Equal(t, voucherSupply, s.supply(t, s.simd, s.ubzeVoucher), "nothing minted on simd")
}

// 12. A gov proposal on bze starts upgrading the second channel to the fee
// version (MsgChannelUpgradeInit is authority-gated) and Hermes completes
// the handshake: TRY on simd, ACK on bze, CONFIRM on simd, OPEN on bze. On
// bze the INIT, ACK and OPEN callbacks reach the transfer app through the
// filter. Both ends end open on the fee version and the channel still
// carries packets, filtered.
func (s *ibcSuite) upgradeFromBze(t *testing.T) {
	bzeEnd, simdEnd := s.upgradeEnds(t, s.second)
	s.relayerStopped(t, func() {
		s.initUpgrade(t, bzeEnd, s.admin, "", ibcFeeVersion)
		ch := s.channelEnd(t, s.bze, s.second.bze)
		require.Equal(t, "STATE_OPEN", ch.State, "INIT keeps the channel open")
		require.Equal(t, helpers.Int64(1), ch.UpgradeSequence, "INIT stored the upgrade")
		s.completeUpgrade(t, bzeEnd, simdEnd)
	})
	s.second.fee = true
	s.requireChannelOpen(t, s.second, ibcFeeVersion, 1)

	s.setBlocked(t, "Block stake on the upgraded channel", blockedEntry{s.second.bze, chain.SimdDenom})
	s.transferInOn(t, s.second, chain.SimdDenom, ibcUpgradeIn).requireRefused(t)
	s.transferInOn(t, s.second, ibcOtherDenom, ibcUpgradeIn).requireAccepted(t)
}

// 13. A gov proposal on simd upgrades the channel back to plain ics20-1 and
// Hermes completes the handshake the other way round: TRY on bze reaches the
// transfer app through the filter, and CONFIRM on bze opens the channel (the
// OPEN callback). The blocked list still applies on the plain channel.
func (s *ibcSuite) upgradeFromSimd(t *testing.T) {
	bzeEnd, simdEnd := s.upgradeEnds(t, s.second)
	s.relayerStopped(t, func() {
		s.initUpgrade(t, simdEnd, s.simdUser, chain.MinDeposit+chain.SimdDenom, ibcTransferVersion)
		s.completeUpgrade(t, simdEnd, bzeEnd)
	})
	s.second.fee = false
	s.requireChannelOpen(t, s.second, ibcTransferVersion, 2)

	s.transferInOn(t, s.second, chain.SimdDenom, ibcUpgradeIn).requireRefused(t)
	s.transferInOn(t, s.second, ibcOtherDenom, ibcUpgradeIn).requireAccepted(t)
}

// inbound is a simd -> bze transfer from simdUser to bzeUser, relayed both
// ways, with the balances taken before it was sent.
type inbound struct {
	ch ibcChannel
	// denom is the denom the packet carries (a base denom or a path),
	// simdDenom the coin simdUser sent
	denom, simdDenom string
	amount           int64
	send             helpers.IbcTransfer
	recv, ack        helpers.TxResult
	fee              sdkmath.Int // simd tx fee in stake

	senderBefore, escrowBefore, receiverBefore, supplyBefore sdkmath.Int
	s                                                        *ibcSuite
}

// transferIn sends amount of a simd denom from simdUser to bzeUser over the
// main channel and waits until the packet is received on bze and
// acknowledged on simd.
func (s *ibcSuite) transferIn(t *testing.T, denom string, amount int64) inbound {
	t.Helper()
	return s.transferInOn(t, s.main, denom, amount)
}

// transferInOn is transferIn over ch. denom is the denom the packet carries:
// a simd base denom, or the path of a voucher simdUser holds.
func (s *ibcSuite) transferInOn(t *testing.T, ch ibcChannel, denom string, amount int64) inbound {
	t.Helper()
	voucher := helpers.IbcDenom(ch.bze, denom)
	in := inbound{
		ch:             ch,
		denom:          denom,
		simdDenom:      simdDenomOf(denom),
		amount:         amount,
		receiverBefore: s.balance(t, s.bze, s.bzeUser, voucher),
		supplyBefore:   s.supply(t, s.bze, voucher),
		s:              s,
	}
	in.senderBefore = s.balance(t, s.simd, s.simdUser, in.simdDenom)
	in.escrowBefore = s.balanceOf(t, s.simd, ch.simdEscrow, in.simdDenom)
	var err error
	in.send, err = helpers.SendIbc(s.ctx, s.simd, s.simdUser, ch.simd, s.bzeUser.FormattedAddress(), sdk.NewInt64Coin(in.simdDenom, amount))
	require.NoError(t, err)
	in.fee, err = helpers.TxFee(in.send.Tx, chain.SimdDenom)
	require.NoError(t, err)
	in.recv, err = helpers.WaitForRecv(s.ctx, s.bze, ch.bze, in.send.Sequence)
	require.NoError(t, err)
	in.ack, err = helpers.WaitForAck(s.ctx, s.simd, in.send)
	require.NoError(t, err)
	return in
}

// simdDenomOf returns the coin simd holds for a packet denom: a base denom is
// itself, a path "transfer/<simd channel>/<rest>" is the voucher of rest
// received on that channel.
func simdDenomOf(packetDenom string) string {
	parts := strings.SplitN(packetDenom, "/", 3)
	if len(parts) < 3 {
		return packetDenom
	}
	return helpers.IbcDenom(parts[1], parts[2])
}

// writtenAck returns the acknowledgement bze wrote for the packet, as written
// (wrapped in the ICS-29 acknowledgement on a fee channel).
func (in inbound) writtenAck(t *testing.T) string {
	t.Helper()
	return packetAck(t, in.recv, in.ch.bze, in.send.Sequence)
}

// appAck returns the transfer acknowledgement bze wrote for the packet. On a
// fee channel it is unwrapped from the ICS-29 acknowledgement, whose
// underlying_app_success must be appSuccess.
func (in inbound) appAck(t *testing.T, appSuccess bool) string {
	t.Helper()
	written := in.writtenAck(t)
	if !in.ch.fee {
		return written
	}
	var wrapped struct {
		AppAcknowledgement   []byte `json:"app_acknowledgement"`
		UnderlyingAppSuccess bool   `json:"underlying_app_success"`
	}
	require.NoError(t, json.Unmarshal([]byte(written), &wrapped), "fee channel ack %s", written)
	require.NotEmpty(t, wrapped.AppAcknowledgement, "the fee middleware wrapped the ack: %s", written)
	require.Equal(t, appSuccess, wrapped.UnderlyingAppSuccess, "underlying_app_success of %s", written)
	return string(wrapped.AppAcknowledgement)
}

// recvTransferEvents returns the transfer app's events on bze for this
// packet, under the callback error prefix too.
func (in inbound) recvTransferEvents() []helpers.Event {
	return helpers.EventsWith(callbackEvents(in.recv, ibcTransferEvent), map[string]string{
		"sender":   in.s.simdUser.FormattedAddress(),
		"receiver": in.s.bzeUser.FormattedAddress(),
		"denom":    in.denom,
		"amount":   strconv.FormatInt(in.amount, 10),
	})
}

// requireAccepted asserts the transfer app minted the voucher and simd
// processed a success acknowledgement.
func (in inbound) requireAccepted(t *testing.T) {
	t.Helper()
	require.Empty(t, blockedEvents(in.recv), "not refused")
	require.Equal(t, `{"result":"AQ=="}`, in.appAck(t, true), "success acknowledgement")
	recvd := in.recvTransferEvents()
	require.Len(t, recvd, 1)
	require.Equal(t, "true", recvd[0].Attributes["success"])
	require.Empty(t, helpers.EventsWith(helpers.TxEvents(in.ack, ibcTransferEvent), map[string]string{"error": ibcBlockedAckError}))
	require.Equal(t, in.receiverBefore.AddRaw(in.amount), in.s.balance(t, in.s.bze, in.s.bzeUser, helpers.IbcDenom(in.ch.bze, in.denom)))
}

// requireRefused asserts the filter answered the packet with an error
// acknowledgement: bze emitted BlockedIbcInboundEvent and minted nothing, simd
// processed the error ack and refunded the sender.
func (in inbound) requireRefused(t *testing.T) {
	t.Helper()
	s := in.s

	// bze: the filter answered, the transfer app never saw the packet
	events := helpers.EventsWith(blockedEvents(in.recv), map[string]string{
		"channel":  in.ch.bze,
		"denom":    in.denom,
		"amount":   strconv.FormatInt(in.amount, 10),
		"sender":   s.simdUser.FormattedAddress(),
		"receiver": s.bzeUser.FormattedAddress(),
	})
	require.Len(t, events, 1, "one BlockedIbcInboundEvent with the packet's fields: %+v", blockedEvents(in.recv))
	require.Empty(t, helpers.TxEvents(in.recv, ibcBlockedEvent), "emitted only under the callback error prefix")
	require.Equal(t, `{"error":"`+ibcBlockedAckError+`"}`, in.appAck(t, false))
	require.Empty(t, in.recvTransferEvents(), "the transfer app did not handle the packet")

	// simd: the error ack was relayed and the transfer refunded
	require.Len(t, helpers.EventsWith(helpers.TxEvents(in.ack, ibcTransferEvent), map[string]string{"error": ibcBlockedAckError}), 1, "simd processed an error ack")

	// exact balances on both chains
	sender := in.senderBefore
	if in.simdDenom == chain.SimdDenom {
		sender = sender.Sub(in.fee)
	}
	voucher := helpers.IbcDenom(in.ch.bze, in.denom)
	require.Equal(t, sender, s.balance(t, s.simd, s.simdUser, in.simdDenom), "simd user refunded, minus the tx fee only")
	require.Equal(t, in.escrowBefore, s.balanceOf(t, s.simd, in.ch.simdEscrow, in.simdDenom), "simd escrow released")
	require.Equal(t, in.receiverBefore, s.balance(t, s.bze, s.bzeUser, voucher), "no voucher minted")
	require.Equal(t, in.supplyBefore, s.supply(t, s.bze, voucher), "voucher supply unchanged")
}

// packetAck returns the acknowledgement a recv tx wrote for the packet with
// destination channel and sequence.
func packetAck(t *testing.T, recv helpers.TxResult, channel string, sequence uint64) string {
	t.Helper()
	acks := helpers.EventsWith(helpers.TxEvents(recv, "write_acknowledgement"), map[string]string{
		"packet_dst_channel": channel,
		"packet_sequence":    strconv.FormatUint(sequence, 10),
	})
	require.Len(t, acks, 1, "one write_acknowledgement for the packet")
	return acks[0].Attributes["packet_ack"]
}

// transferOut sends amount ubze from bzeUser to simdUser, waits for the ack
// and checks the sender paid amount + fee and the escrow took amount.
func (s *ibcSuite) transferOut(t *testing.T, amount int64) {
	t.Helper()
	ubze := s.balance(t, s.bze, s.bzeUser, chain.Denom)
	escrow := s.balanceOf(t, s.bze, s.bzeEscrow, chain.Denom)

	out, err := helpers.SendIbc(s.ctx, s.bze, s.bzeUser, s.net.BzeChannel, s.simdUser.FormattedAddress(), sdk.NewInt64Coin(chain.Denom, amount))
	require.NoError(t, err)
	_, err = helpers.WaitForRecv(s.ctx, s.simd, s.net.SimdChannel, out.Sequence)
	require.NoError(t, err)
	_, err = helpers.WaitForAck(s.ctx, s.bze, out)
	require.NoError(t, err)

	fee, err := helpers.TxFee(out.Tx, chain.Denom)
	require.NoError(t, err)
	require.Equal(t, ubze.SubRaw(amount).Sub(fee), s.balance(t, s.bze, s.bzeUser, chain.Denom))
	require.Equal(t, escrow.AddRaw(amount), s.balanceOf(t, s.bze, s.bzeEscrow, chain.Denom))
}

// blockedEvents returns the BlockedIbcInboundEvents of a recv tx with the
// callback error prefix removed from type and keys and the values unquoted,
// plus any emitted without the prefix.
func blockedEvents(res helpers.TxResult) []helpers.Event {
	return callbackEvents(res, ibcBlockedEvent)
}

// callbackEvents returns the events of eventType in a recv tx: those emitted
// as is, plus those ibc-go rewrote with the callback error prefix (because
// the callback returned an error ack), with the prefix removed from type and
// keys and the values unquoted.
func callbackEvents(res helpers.TxResult, eventType string) []helpers.Event {
	out := helpers.TxEvents(res, eventType)
	for _, e := range helpers.TxEvents(res, ibcCallbackErrorPrefix+eventType) {
		ev := helpers.Event{Type: eventType, Attributes: make(map[string]string, len(e.Attributes))}
		for k, v := range e.Attributes {
			ev.Attributes[strings.TrimPrefix(k, ibcCallbackErrorPrefix)] = helpers.Unquote(v)
		}
		out = append(out, ev)
	}
	return out
}

type blockedEntry struct {
	ChannelID, BaseDenom string
}

// txfeecollector params, by proto field name. The CLI prints them camelCase
// or snake_case depending on the encoder, and the proposal JSON wants the
// proto names, so keys are matched without case and underscores.
var txFeeParamNames = map[string]string{
	"validatormingasfee":   "ValidatorMinGasFee",
	"maxbalanceiterations": "MaxBalanceIterations",
	"blockedibcinbound":    "BlockedIbcInbound",
}

const blockedParamName = "BlockedIbcInbound"

func normalizeKey(k string) string {
	return strings.ToLower(strings.ReplaceAll(k, "_", ""))
}

// queryParams returns the txfeecollector params by proto field name. An
// unknown param fails the test: MsgUpdateParams must carry the full params,
// so the suite must learn about new ones.
func (s *ibcSuite) queryParams(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	res, err := helpers.Query[struct {
		Params map[string]json.RawMessage `json:"params"`
	}](s.ctx, s.bze, "txfeecollector", "params")
	require.NoError(t, err)
	out := make(map[string]json.RawMessage, len(res.Params))
	for k, v := range res.Params {
		name, ok := txFeeParamNames[normalizeKey(k)]
		require.True(t, ok, "unknown txfeecollector param %q: teach the IBC suite about it", k)
		out[name] = v
	}
	return out
}

// otherParams returns every param but the blocked list.
func (s *ibcSuite) otherParams(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	p := s.queryParams(t)
	delete(p, blockedParamName)
	require.NotEmpty(t, p)
	return p
}

func (s *ibcSuite) blockedParam(t *testing.T) []blockedEntry {
	t.Helper()
	raw, ok := s.queryParams(t)[blockedParamName]
	if !ok || string(raw) == "null" {
		return nil
	}
	var entries []map[string]string
	require.NoError(t, json.Unmarshal(raw, &entries))
	out := make([]blockedEntry, 0, len(entries))
	for _, e := range entries {
		var b blockedEntry
		for k, v := range e {
			switch normalizeKey(k) {
			case "channelid":
				b.ChannelID = v
			case "basedenom":
				b.BaseDenom = v
			default:
				t.Fatalf("unknown blocked ibc inbound field %q", k)
			}
		}
		out = append(out, b)
	}
	return out
}

// setBlocked passes a proposal with one txfeecollector MsgUpdateParams that
// carries the current params with the blocked list replaced by entries.
func (s *ibcSuite) setBlocked(t *testing.T, title string, entries ...blockedEntry) {
	t.Helper()
	params := map[string]any{}
	for k, v := range s.otherParams(t) {
		params[k] = v
	}
	// the query prints the DecCoin as a coin string; the proposal needs the message
	raw, ok := params["ValidatorMinGasFee"].(json.RawMessage)
	require.True(t, ok, "ValidatorMinGasFee missing from the params")
	var minGasFee string
	if json.Unmarshal(raw, &minGasFee) == nil {
		coin, err := sdk.ParseDecCoin(minGasFee)
		require.NoError(t, err)
		params["ValidatorMinGasFee"] = map[string]string{"denom": coin.Denom, "amount": coin.Amount.String()}
	}
	blocked := make([]map[string]string, 0, len(entries))
	for _, e := range entries {
		blocked = append(blocked, map[string]string{"channel_id": e.ChannelID, "base_denom": e.BaseDenom})
	}
	params[blockedParamName] = blocked

	_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: []any{map[string]any{
			"@type":     "/bze.txfeecollector.MsgUpdateParams",
			"authority": s.gov,
			"params":    params,
		}},
		Title:   title,
		Summary: "e2e: " + title,
	})
	require.NoError(t, err)
}

// channelEnd is a transfer channel end as `q ibc channel` prints it.
type channelEnd struct {
	State        string `json:"state"`
	Ordering     string `json:"ordering"`
	Counterparty struct {
		ChannelID string `json:"channel_id"`
	} `json:"counterparty"`
	ConnectionHops  []string      `json:"connection_hops"`
	Version         string        `json:"version"`
	ChannelID       string        `json:"channel_id"`
	UpgradeSequence helpers.Int64 `json:"upgrade_sequence"`
}

func (s *ibcSuite) newChannel(t *testing.T, bzeID, simdID string, fee bool) ibcChannel {
	t.Helper()
	ch := ibcChannel{bze: bzeID, simd: simdID, fee: fee}
	var err error
	ch.bzeEscrow, err = helpers.EscrowAddress(chain.Bech32Prefix, bzeID)
	require.NoError(t, err)
	ch.simdEscrow, err = helpers.EscrowAddress(chain.SimdBech32Prefix, simdID)
	require.NoError(t, err)
	return ch
}

func (s *ibcSuite) channels(t *testing.T, c *cosmos.CosmosChain) []channelEnd {
	t.Helper()
	res, err := helpers.Query[struct {
		Channels []channelEnd `json:"channels"`
	}](s.ctx, c, "ibc", "channel", "channels")
	require.NoError(t, err)
	return res.Channels
}

func (s *ibcSuite) channelEnd(t *testing.T, c *cosmos.CosmosChain, channel string) channelEnd {
	t.Helper()
	res, err := helpers.Query[struct {
		Channel channelEnd `json:"channel"`
	}](s.ctx, c, "ibc", "channel", "end", chain.TransferPort, channel)
	require.NoError(t, err)
	return res.Channel
}

// openChannel opens another unordered transfer channel between bze and simd
// with version, on the existing connection, and returns it.
func (s *ibcSuite) openChannel(t *testing.T, version string) ibcChannel {
	t.Helper()
	known := map[string]bool{}
	for _, c := range s.channels(t, s.bze) {
		known[c.ChannelID] = true
	}
	s.relayerStopped(t, func() {
		require.NoError(t, s.net.Relayer.CreateChannel(s.ctx, s.net.Reporter, chain.IbcPath, ibc.CreateChannelOptions{
			SourcePortName: chain.TransferPort,
			DestPortName:   chain.TransferPort,
			Order:          ibc.Unordered,
			Version:        version,
		}))
	})
	var opened []channelEnd
	for _, c := range s.channels(t, s.bze) {
		if !known[c.ChannelID] {
			opened = append(opened, c)
		}
	}
	require.Len(t, opened, 1, "one new channel on bze")
	ch := s.newChannel(t, opened[0].ChannelID, opened[0].Counterparty.ChannelID, version != ibcTransferVersion)
	s.requireChannelOpen(t, ch, version, 0)
	return ch
}

// requireChannelOpen asserts both ends of ch are open, unordered, point at
// each other and carry version at upgrade sequence upgradeSequence.
func (s *ibcSuite) requireChannelOpen(t *testing.T, ch ibcChannel, version string, upgradeSequence int64) {
	t.Helper()
	for _, end := range []struct {
		c                     *cosmos.CosmosChain
		channel, counterparty string
	}{{s.bze, ch.bze, ch.simd}, {s.simd, ch.simd, ch.bze}} {
		got := s.channelEnd(t, end.c, end.channel)
		require.Equal(t, "STATE_OPEN", got.State, "%s %s", end.c.Config().ChainID, end.channel)
		require.Equal(t, "ORDER_UNORDERED", got.Ordering, "%s %s", end.c.Config().ChainID, end.channel)
		require.Equal(t, end.counterparty, got.Counterparty.ChannelID, "%s %s", end.c.Config().ChainID, end.channel)
		require.Equal(t, version, got.Version, "%s %s", end.c.Config().ChainID, end.channel)
		require.Equal(t, helpers.Int64(upgradeSequence), got.UpgradeSequence, "%s %s", end.c.Config().ChainID, end.channel)
	}
}

// relayerStopped runs f with the relayer stopped, so Hermes CLI commands do
// not race the running relayer on the same channels and keys, and restarts
// it afterwards (also when f fails).
func (s *ibcSuite) relayerStopped(t *testing.T, f func()) {
	t.Helper()
	r, rep := s.net.Relayer, s.net.Reporter
	require.NoError(t, r.StopRelayer(s.ctx, rep))
	defer func() {
		if err := r.StartRelayer(s.ctx, rep, chain.IbcPath); err != nil {
			t.Errorf("restart relayer: %v", err)
		}
	}()
	f()
}

// upgradeEnd is one end of a channel to upgrade.
type upgradeEnd struct {
	c                   *cosmos.CosmosChain
	channel, connection string
}

func (s *ibcSuite) upgradeEnds(t *testing.T, ch ibcChannel) (bzeEnd, simdEnd upgradeEnd) {
	t.Helper()
	b, sd := s.channelEnd(t, s.bze, ch.bze), s.channelEnd(t, s.simd, ch.simd)
	require.Len(t, b.ConnectionHops, 1)
	require.Len(t, sd.ConnectionHops, 1)
	return upgradeEnd{s.bze, ch.bze, b.ConnectionHops[0]}, upgradeEnd{s.simd, ch.simd, sd.ConnectionHops[0]}
}

// initUpgrade passes a proposal on end's chain with a MsgChannelUpgradeInit
// to version (same ordering and connection); deposit "" is bze's default.
func (s *ibcSuite) initUpgrade(t *testing.T, end upgradeEnd, proposer ibc.Wallet, deposit, version string) {
	t.Helper()
	authority, err := helpers.GovAuthority(s.ctx, end.c)
	require.NoError(t, err)
	_, err = helpers.SubmitAndPassProposal(s.ctx, end.c, proposer, helpers.Proposal{
		Messages: []any{map[string]any{
			"@type":      "/ibc.core.channel.v1.MsgChannelUpgradeInit",
			"port_id":    chain.TransferPort,
			"channel_id": end.channel,
			"fields": map[string]any{
				"ordering":        "ORDER_UNORDERED",
				"connection_hops": []string{end.connection},
				"version":         version,
			},
			"signer": authority,
		}},
		Deposit: deposit,
		Title:   "Upgrade " + end.channel,
		Summary: "e2e: upgrade " + end.channel + " to " + version,
	})
	require.NoError(t, err)
}

// completeUpgrade relays the handshake an INIT on init started, with the
// relayer stopped: TRY on the counterparty, ACK on init, CONFIRM on the
// counterparty (which opens it there, nothing being in flight), OPEN on init.
func (s *ibcSuite) completeUpgrade(t *testing.T, init, counterparty upgradeEnd) {
	t.Helper()
	s.hermesUpgradeStep(t, "try", counterparty, init)
	s.hermesUpgradeStep(t, "ack", init, counterparty)
	s.hermesUpgradeStep(t, "confirm", counterparty, init)
	s.hermesUpgradeStep(t, "open", init, counterparty)
}

// hermesUpgradeStep runs `hermes tx chan-upgrade-<step>`, sending the step's
// message to dst with proofs from src.
func (s *ibcSuite) hermesUpgradeStep(t *testing.T, step string, dst, src upgradeEnd) {
	t.Helper()
	res := s.net.Relayer.Exec(s.ctx, s.net.Reporter, []string{
		"hermes", "--json", "tx", "chan-upgrade-" + step,
		"--dst-chain", dst.c.Config().ChainID,
		"--src-chain", src.c.Config().ChainID,
		"--dst-connection", dst.connection,
		"--dst-port", chain.TransferPort,
		"--src-port", chain.TransferPort,
		"--dst-channel", dst.channel,
		"--src-channel", src.channel,
	}, nil)
	require.NoError(t, res.Err, "hermes chan-upgrade-%s: %s %s", step, res.Stdout, res.Stderr)
}

func (s *ibcSuite) balance(t *testing.T, c *cosmos.CosmosChain, user ibc.Wallet, denom string) sdkmath.Int {
	t.Helper()
	return s.balanceOf(t, c, user.FormattedAddress(), denom)
}

func (s *ibcSuite) balanceOf(t *testing.T, c *cosmos.CosmosChain, addr, denom string) sdkmath.Int {
	t.Helper()
	b, err := helpers.Balance(s.ctx, c, addr, denom)
	require.NoError(t, err)
	return b
}

func (s *ibcSuite) supply(t *testing.T, c *cosmos.CosmosChain, denom string) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Amount struct {
			Amount string `json:"amount"`
		} `json:"amount"`
	}](s.ctx, c, "bank", "total-supply-of", denom)
	require.NoError(t, err)
	return parseInt(res.Amount.Amount)
}

// newSimdAccount returns a fresh mnemonic and its simd address, so the
// account can be funded in simd's genesis before its key exists.
func newSimdAccount(t *testing.T) (string, string) {
	t.Helper()
	entropy, err := bip39.NewEntropy(256)
	require.NoError(t, err)
	mnemonic, err := bip39.NewMnemonic(entropy)
	require.NoError(t, err)
	derived, err := hd.Secp256k1.Derive()(mnemonic, "", hd.CreateHDPath(sdk.CoinType, 0, 0).String())
	require.NoError(t, err)
	addr, err := sdk.Bech32ifyAddressBytes(chain.SimdBech32Prefix, hd.Secp256k1.Generate()(derived).PubKey().Address())
	require.NoError(t, err)
	return mnemonic, addr
}
