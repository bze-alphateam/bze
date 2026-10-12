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

	ibcBlockedEvent = "bze.txfeecollector.BlockedIbcInboundEvent"
	// ibcCallbackErrorPrefix is what ibc-go puts on the type and attribute keys
	// of the events an OnRecvPacket callback emitted when it returns an error
	// ack: they are rewritten, not dropped (accepted as is in the v8.2.0 review).
	ibcCallbackErrorPrefix = "ibccallbackerror-"
	// ibcBlockedAckError is the error acknowledgement of ErrBlockedIbcInbound
	// (txfeecollector code 1101): ibc-go keeps only the ABCI code.
	ibcBlockedAckError = "ABCI code: 1101: error handling packet: see events for details"
	ibcTransferEvent   = "fungible_token_packet"
)

// TestBlockedInboundIbc connects bze to an ibc-go simd chain through the Hermes
// relayer and proves the txfeecollector inbound filter in the real transfer
// stack: a (channel, base denom) pair blocked by a governance proposal is
// answered with an error acknowledgement, refunded on simd and leaves no
// voucher on bze, while other denoms, outgoing transfers, returning BZE
// vouchers and timeouts behave as without the filter. Subtests run in order
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
	voucherSupply := s.supply(t, s.bze, s.stakeVoucher)
	in := s.transferIn(t, chain.SimdDenom, ibcBlockedIn)

	// bze: the filter answered, the transfer app never saw the packet
	events := helpers.EventsWith(blockedEvents(in.recv), map[string]string{
		"channel":  s.net.BzeChannel,
		"denom":    chain.SimdDenom,
		"amount":   strconv.Itoa(ibcBlockedIn),
		"sender":   s.simdUser.FormattedAddress(),
		"receiver": s.bzeUser.FormattedAddress(),
	})
	require.Len(t, events, 1, "one BlockedIbcInboundEvent with the packet's fields: %+v", blockedEvents(in.recv))
	require.Empty(t, helpers.TxEvents(in.recv, ibcBlockedEvent), "emitted only under the callback error prefix")
	require.Equal(t, `{"error":"`+ibcBlockedAckError+`"}`, in.writtenAck(t))
	require.Empty(t, in.recvTransferEvents(), "the transfer app did not handle the packet")

	// simd: the error ack was relayed and the transfer refunded
	require.Len(t, helpers.EventsWith(helpers.TxEvents(in.ack, ibcTransferEvent), map[string]string{"error": ibcBlockedAckError}), 1, "simd processed an error ack")

	// exact balances on both chains
	require.Equal(t, in.senderBefore.Sub(in.fee), s.balance(t, s.simd, s.simdUser, chain.SimdDenom), "simd user refunded, minus the tx fee only")
	require.Equal(t, in.escrowBefore, s.balanceOf(t, s.simd, s.simdEscrow, chain.SimdDenom), "simd escrow released")
	require.Equal(t, in.receiverBefore, s.balance(t, s.bze, s.bzeUser, s.stakeVoucher), "no voucher minted")
	require.Equal(t, voucherSupply, s.supply(t, s.bze, s.stakeVoucher), "voucher supply unchanged")
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

// inbound is a simd -> bze transfer from simdUser to bzeUser, relayed both
// ways, with the balances taken before it was sent.
type inbound struct {
	denom     string
	amount    int64
	send      helpers.IbcTransfer
	recv, ack helpers.TxResult
	fee       sdkmath.Int // simd tx fee in stake

	senderBefore, escrowBefore, receiverBefore sdkmath.Int
	s                                          *ibcSuite
}

// transferIn sends amount of a simd denom from simdUser to bzeUser and waits
// until the packet is received on bze and acknowledged on simd.
func (s *ibcSuite) transferIn(t *testing.T, denom string, amount int64) inbound {
	t.Helper()
	in := inbound{
		denom:          denom,
		amount:         amount,
		senderBefore:   s.balance(t, s.simd, s.simdUser, chain.SimdDenom),
		escrowBefore:   s.balanceOf(t, s.simd, s.simdEscrow, chain.SimdDenom),
		receiverBefore: s.balance(t, s.bze, s.bzeUser, helpers.IbcDenom(s.net.BzeChannel, denom)),
		s:              s,
	}
	var err error
	in.send, err = helpers.SendIbc(s.ctx, s.simd, s.simdUser, s.net.SimdChannel, s.bzeUser.FormattedAddress(), sdk.NewInt64Coin(denom, amount))
	require.NoError(t, err)
	in.fee, err = helpers.TxFee(in.send.Tx, chain.SimdDenom)
	require.NoError(t, err)
	in.recv, err = helpers.WaitForRecv(s.ctx, s.bze, s.net.BzeChannel, in.send.Sequence)
	require.NoError(t, err)
	in.ack, err = helpers.WaitForAck(s.ctx, s.simd, in.send)
	require.NoError(t, err)
	return in
}

// writtenAck returns the acknowledgement bze wrote for the packet.
func (in inbound) writtenAck(t *testing.T) string {
	t.Helper()
	acks := helpers.EventsWith(helpers.TxEvents(in.recv, "write_acknowledgement"), map[string]string{
		"packet_dst_channel": in.s.net.BzeChannel,
		"packet_sequence":    strconv.FormatUint(in.send.Sequence, 10),
	})
	require.Len(t, acks, 1, "one write_acknowledgement for the packet")
	return acks[0].Attributes["packet_ack"]
}

// recvTransferEvents returns the transfer app's events on bze for this packet.
func (in inbound) recvTransferEvents() []helpers.Event {
	return helpers.EventsWith(helpers.TxEvents(in.recv, ibcTransferEvent), map[string]string{
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
	require.Equal(t, `{"result":"AQ=="}`, in.writtenAck(t), "success acknowledgement")
	recvd := in.recvTransferEvents()
	require.Len(t, recvd, 1)
	require.Equal(t, "true", recvd[0].Attributes["success"])
	require.Empty(t, helpers.EventsWith(helpers.TxEvents(in.ack, ibcTransferEvent), map[string]string{"error": ibcBlockedAckError}))
	require.Equal(t, in.receiverBefore.AddRaw(in.amount), in.s.balance(t, in.s.bze, in.s.bzeUser, helpers.IbcDenom(in.s.net.BzeChannel, in.denom)))
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
	out := helpers.TxEvents(res, ibcBlockedEvent)
	for _, e := range helpers.TxEvents(res, ibcCallbackErrorPrefix+ibcBlockedEvent) {
		ev := helpers.Event{Type: ibcBlockedEvent, Attributes: make(map[string]string, len(e.Attributes))}
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
