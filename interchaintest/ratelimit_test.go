package interchaintest_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

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
	// rlUbzePercent is the ubze quota (send and receive) on the main channel,
	// in percent of the ubze supply when the rate limit is added. It is 1%, not
	// 10%: the supply is the whole genesis supply (about 3e13 ubze) and a 10%
	// threshold is more than the faucet can fund a sender with. The threshold is
	// never hard-coded: it is read from the stored channel value, because ubze is
	// minted every block, and computed with helpers.Threshold exactly as the
	// module does (value × percent / 100, integer division, refuse above).
	rlUbzePercent = 1
	// rlStakePercent is the quota (send and receive) on the stake voucher: the
	// voucher supply is small, so 10% is cheap to reach. The supply is
	// ibcStakeIn (baseline) + rlStakeSeed, so the threshold is exact:
	// (1,000,000 + 10,000,000) × 10 / 100 = 1,100,000.
	rlStakePercent = 10
	rlStakeSeed    = 10_000_000
	rlWindowHours  = 24

	rlGuardianFunds = 100_000_000_000 // ubze for the guardian's tx fees
	rlGas           = 500_000         // fixed gas limit of every tx whose refusal is asserted

	rlReturnStake  = 200_000 // 10. stake vouchers home before the block, makes room for inflow
	rlInFlight     = 50_000  // 10. stake vouchers in flight while the guardian blocks the denom
	rlBlockedIn    = 100_000 // 10. stake simd -> bze while blocked, and again after the unblock
	rlBackAmount   = 1_000_000
	rlTimeoutOut   = 100_000 // 6. ubze whose timeout is undone
	rlUpdateOut    = 1_000_000
	rlSecondExtra  = 1_000_000 // 7. ubze above the main channel's threshold, over the second channel
	rlQuotaErrCode = 4         // ratelimit ErrQuotaExceeded
	rlDenyErrCode  = 7         // ratelimit ErrDenomIsBlacklisted

	rlQuotaMsg     = "exceeds quota"
	rlBlacklistMsg = "is blacklisted"

	rlDeniedEvent       = "transfer_denied"
	rlReasonQuota       = "rate_limit_exceeded"
	rlReasonBlacklisted = "blacklisted_denom"

	rlGuardianUnauthorizedCode   = 1101
	rlGuardianQuotaRaiseCode     = 1103
	rlGuardianAlreadyBlockedCode = 1106
	rlGuardianDenomNotFoundCode  = 1109 // guardian ErrDenomNotFound
)

// TestRateLimitIbc proves IBC rate limiting in the real transfer stack: bze,
// an ibc-go simd and Hermes. A quota added by governance refuses an outbound
// transfer as a failed tx and an inbound one with an error acknowledgement
// that refunds the sender; the flow is net (returning vouchers make room), a
// timed-out transfer gives its outflow back, an unlimited channel is not
// affected, the guardian can tighten a quota (never raise it) and block a
// denom, and only governance loosens a quota and unblocks a denom. The state
// carries over between the subtests, which run in order. The suite never
// waits for a window to reset: windows are hour epochs of the module, and a
// start close to 00:00 UTC (when a 24 h window resets) waits that out first.
func TestRateLimitIbc(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	waitOutWindowReset(t)

	simdMnemonic, simdAddr := newSimdAccount(t)
	guardianMnemonic, guardianAddr := newAccount(t, chain.Bech32Prefix)
	// the guardian is a test wallet set in genesis, so the suite does not
	// depend on the address the upgrade handler sets on mainnet
	net := chain.StartIbc(t, ctx, []ibc.WalletAmount{{
		Address: simdAddr,
		Denom:   ibcOtherDenom,
		Amount:  sdkmath.NewInt(ibcOtherFunds),
	}}, chain.WithGenesis(cosmos.NewGenesisKV("app_state.guardian.params.guardian", guardianAddr)))
	s := newRateLimitSuite(t, ctx, net, simdMnemonic, guardianMnemonic, guardianAddr)

	t.Run("1 baseline, no rate limits", s.baseline)
	t.Run("2 governance adds a quota", s.addQuota)
	t.Run("3 outflow above the threshold is refused", s.outflowRefused)
	t.Run("4 flow is net", s.netFlow)
	t.Run("5 inflow above the threshold is refused and refunded", s.inflowRefused)
	t.Run("6 timeout undoes the outflow", s.timeoutUndo)
	t.Run("7 another channel is unaffected", s.otherChannel)
	t.Run("8 guardian tightens", s.guardianTightens)
	t.Run("9 governance loosens", s.govLoosens)
	t.Run("10 guardian blocks a denom", s.guardianBlocks)
	t.Run("11 audit", s.audit)
}

type rateLimitSuite struct {
	*ibcSuite

	guardian ibc.Wallet
	// rich holds enough ubze to send the threshold of the ubze quota
	rich ibc.Wallet

	// ubzeThreshold is the ubze send/receive threshold of the main channel's
	// window
	ubzeThreshold sdkmath.Int
	// stakeThreshold is the threshold of the stake voucher's window
	stakeThreshold sdkmath.Int
	// returned is the ubze coming back to bze in step 4, room for the net flow
	returned sdkmath.Int
}

func newRateLimitSuite(t *testing.T, ctx context.Context, net chain.IbcNetwork, simdMnemonic, guardianMnemonic, guardianAddr string) *rateLimitSuite {
	t.Helper()
	s := setupIbc(t, ctx, net, simdMnemonic)
	r := &rateLimitSuite{ibcSuite: s}

	var err error
	r.guardian, err = interchaintest.GetAndFundTestUserWithMnemonic(ctx, "guardian", guardianMnemonic, sdkmath.NewInt(rlGuardianFunds), s.bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, r.guardian.FormattedAddress())

	// the rich sender holds 5% of the supply: five times the threshold of the
	// 1% ubze quota
	supply := s.supply(t, s.bze, chain.Denom)
	r.rich = interchaintest.GetAndFundTestUsers(t, ctx, "rl-rich", supply.QuoRaw(20), s.bze)[0]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, s.bze, s.simd))

	guardian, err := helpers.QueryGuardian(ctx, s.bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, guardian, "the genesis guardian")
	return r
}

// waitOutWindowReset waits until after 00:00 UTC when the test would start in
// the 15 minutes before it: the module resets a 24 h window when the hour
// epoch number is a multiple of 24, i.e. at 00:00 UTC, and the suite takes
// about 8 minutes.
func waitOutWindowReset(t *testing.T) {
	t.Helper()
	now := time.Now().UTC()
	midnight := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	if midnight.Sub(now) > 15*time.Minute {
		return
	}
	wait := midnight.Sub(now) + time.Minute
	t.Logf("a 24 h rate limit window resets at 00:00 UTC: waiting %s so the suite stays in one window", wait.Round(time.Second))
	time.Sleep(wait)
}

// ---------------------------------------------------------------------------
// queries and transactions

func (r *rateLimitSuite) limit(t *testing.T, denom, channel string) helpers.RateLimit {
	t.Helper()
	rl, err := helpers.QueryRateLimit(r.ctx, r.bze, channel, denom)
	require.NoError(t, err)
	return rl
}

func (r *rateLimitSuite) requireFlow(t *testing.T, denom, channel string, inflow, outflow sdkmath.Int) {
	t.Helper()
	rl := r.limit(t, denom, channel)
	require.True(t, rl.FlowIs(inflow, outflow), "flow of %s on %s: want in %s out %s, got in %s out %s", denom, channel, inflow, outflow, rl.Flow.Inflow, rl.Flow.Outflow)
}

// govAddLimit passes a proposal adding a quota of send/recv percent in a
// rlWindowHours window.
func (r *rateLimitSuite) govAddLimit(t *testing.T, title, denom, channel string, send, recv int64) {
	t.Helper()
	r.govRateLimitMsg(t, title, "/ratelimit.v1.MsgAddRateLimit", denom, channel, send, recv)
}

// govUpdateLimit passes a proposal replacing a quota. The module resets the
// flow of an updated rate limit and samples the channel value again.
func (r *rateLimitSuite) govUpdateLimit(t *testing.T, title, denom, channel string, send, recv int64) {
	t.Helper()
	r.govRateLimitMsg(t, title, "/ratelimit.v1.MsgUpdateRateLimit", denom, channel, send, recv)
}

func (r *rateLimitSuite) govRateLimitMsg(t *testing.T, title, msgType, denom, channel string, send, recv int64) {
	t.Helper()
	_, err := helpers.SubmitAndPassProposal(r.ctx, r.bze, r.admin, helpers.Proposal{
		Messages: []any{map[string]any{
			"@type":                msgType,
			"authority":            r.gov,
			"denom":                denom,
			"channel_or_client_id": channel,
			"max_percent_send":     strconv.FormatInt(send, 10),
			"max_percent_recv":     strconv.FormatInt(recv, 10),
			"duration_hours":       strconv.FormatInt(rlWindowHours, 10),
		}},
		Title:   title,
		Summary: "e2e: " + title,
	})
	require.NoError(t, err)
}

// tx sends a tx with a fixed gas limit, so a refusal is broadcast, included
// and charged, and comes back with its code and codespace.
func (r *rateLimitSuite) tx(t *testing.T, user ibc.Wallet, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTxWith(r.ctx, r.bze, user, []helpers.TxOption{helpers.WithGas(rlGas)}, args...)
	require.NoError(t, err)
	return res
}

func (r *rateLimitSuite) fixedFee(t *testing.T) sdkmath.Int {
	t.Helper()
	fee, err := helpers.FixedGasFee(r.bze, rlGas)
	require.NoError(t, err)
	return fee
}

// requireSendRefused sends coin from user over ch and asserts the transfer
// app's send was refused by the rate limit module: the tx is included and
// failed with the codespace ratelimit and code, no packet was committed, and
// the user lost the fee and nothing else.
func (r *rateLimitSuite) requireSendRefused(t *testing.T, user ibc.Wallet, ch ibcChannel, coin sdk.Coin, code uint32, msg string) {
	t.Helper()
	ubze := r.balance(t, r.bze, user, chain.Denom)
	held := r.balance(t, r.bze, user, coin.Denom)
	escrow := r.balanceOf(t, r.bze, ch.bzeEscrow, coin.Denom)

	res, err := helpers.TrySendIbc(r.ctx, r.bze, user, ch.bze, r.simdUser.FormattedAddress(), coin, rlGas)
	require.NoError(t, err)
	require.NotZero(t, res.Code, "the transfer of %s is refused: %s", coin, res.RawLog)
	require.Equal(t, "ratelimit", res.Codespace, res.RawLog)
	require.EqualValues(t, code, res.Code, res.RawLog)
	require.Contains(t, res.RawLog, msg)
	require.Empty(t, helpers.TxEvents(res, "send_packet"), "no packet was committed")

	fee := r.fixedFee(t)
	wantUbze := ubze.Sub(fee)
	wantHeld := held
	if coin.Denom == chain.Denom {
		wantHeld = wantUbze
	}
	require.Equal(t, wantUbze, r.balance(t, r.bze, user, chain.Denom), "only the fee was lost")
	require.Equal(t, wantHeld, r.balance(t, r.bze, user, coin.Denom), "the coin stayed with the sender")
	require.Equal(t, escrow, r.balanceOf(t, r.bze, ch.bzeEscrow, coin.Denom), "nothing was escrowed")
}

// sendUbze sends amount ubze from user to simdUser over ch and waits for the
// packet to be received and acknowledged on both chains.
func (r *rateLimitSuite) sendUbze(t *testing.T, user ibc.Wallet, ch ibcChannel, amount sdkmath.Int) {
	t.Helper()
	escrow := r.balanceOf(t, r.bze, ch.bzeEscrow, chain.Denom)
	voucher := helpers.IbcDenom(ch.simd, chain.Denom)
	vouchers := r.balance(t, r.simd, r.simdUser, voucher)

	out, err := helpers.SendIbc(r.ctx, r.bze, user, ch.bze, r.simdUser.FormattedAddress(), sdk.NewCoin(chain.Denom, amount))
	require.NoError(t, err)
	_, err = helpers.WaitForRecv(r.ctx, r.simd, ch.simd, out.Sequence)
	require.NoError(t, err)
	_, err = helpers.WaitForAck(r.ctx, r.bze, out)
	require.NoError(t, err)

	require.Equal(t, escrow.Add(amount), r.balanceOf(t, r.bze, ch.bzeEscrow, chain.Denom), "escrowed")
	require.Equal(t, vouchers.Add(amount), r.balance(t, r.simd, r.simdUser, voucher), "minted on simd")
}

// returnUbze sends amount ubze vouchers from simdUser home to bzeUser over
// the main channel and returns the recv tx on bze. The packet is received
// (not denied), so it counts as inflow of (ubze, main channel).
func (r *rateLimitSuite) returnUbze(t *testing.T, amount int64) helpers.TxResult {
	t.Helper()
	ubze := r.balance(t, r.bze, r.bzeUser, chain.Denom)
	escrow := r.balanceOf(t, r.bze, r.bzeEscrow, chain.Denom)

	out, err := helpers.SendIbc(r.ctx, r.simd, r.simdUser, r.net.SimdChannel, r.bzeUser.FormattedAddress(), sdk.NewInt64Coin(r.ubzeVoucher, amount))
	require.NoError(t, err)
	recv, err := helpers.WaitForRecv(r.ctx, r.bze, r.net.BzeChannel, out.Sequence)
	require.NoError(t, err)
	_, err = helpers.WaitForAck(r.ctx, r.simd, out)
	require.NoError(t, err)

	require.Empty(t, callbackEvents(recv, rlDeniedEvent), "not denied")
	require.Equal(t, ubze.AddRaw(amount), r.balance(t, r.bze, r.bzeUser, chain.Denom), "arrived as native ubze")
	require.Equal(t, escrow.SubRaw(amount), r.balanceOf(t, r.bze, r.bzeEscrow, chain.Denom), "released from escrow")
	return recv
}

// requireDenied asserts an inbound transfer was answered with the rate limit
// module's error acknowledgement (ABCI code), with a transfer_denied event of
// reason, and refunded on simd without minting anything on bze.
func (r *rateLimitSuite) requireDenied(t *testing.T, in inbound, reason string, code int) {
	t.Helper()
	voucher := helpers.IbcDenom(in.ch.bze, in.denom)
	ackErr := "ABCI code: " + strconv.Itoa(code) + ": error handling packet: see events for details"

	denied := helpers.EventsWith(callbackEvents(in.recv, rlDeniedEvent), map[string]string{
		"reason":            reason,
		"denom":             voucher,
		"channel_or_client": in.ch.bze,
		"action":            "packet_recv",
		"amount":            strconv.FormatInt(in.amount, 10),
	})
	require.Len(t, denied, 1, "one transfer_denied event with reason %s: %+v", reason, callbackEvents(in.recv, rlDeniedEvent))
	require.Equal(t, `{"error":"`+ackErr+`"}`, in.writtenAck(t))
	require.Empty(t, in.recvTransferEvents(), "the transfer app did not handle the packet")
	require.Len(t, helpers.EventsWith(helpers.TxEvents(in.ack, ibcTransferEvent), map[string]string{"error": ackErr}), 1, "simd processed the error ack")

	sender := in.senderBefore
	if in.simdDenom == chain.SimdDenom {
		sender = sender.Sub(in.fee)
	}
	require.Equal(t, sender, r.balance(t, r.simd, r.simdUser, in.simdDenom), "simd user refunded, minus the tx fee only")
	require.Equal(t, in.escrowBefore, r.balanceOf(t, r.simd, in.ch.simdEscrow, in.simdDenom), "simd escrow released")
	require.Equal(t, in.receiverBefore, r.balance(t, r.bze, r.bzeUser, voucher), "no voucher minted")
	require.Equal(t, in.supplyBefore, r.supply(t, r.bze, voucher), "voucher supply unchanged")
}

// sendTimingOut stops the relayer and sends coin with a timeout 1 s from now,
// which simd has passed a few blocks later, while nothing relays the packet.
func (r *rateLimitSuite) sendTimingOut(t *testing.T, user ibc.Wallet, ch ibcChannel, coin sdk.Coin) helpers.IbcTransfer {
	t.Helper()
	require.NoError(t, r.net.Relayer.StopRelayer(r.ctx, r.net.Reporter))
	out, err := helpers.SendIbc(r.ctx, r.bze, user, ch.bze, r.simdUser.FormattedAddress(), coin,
		"--packet-timeout-height", "0-0", "--packet-timeout-timestamp", "1000000000")
	require.NoError(t, err)
	require.NoError(t, testutil.WaitForBlocks(r.ctx, 3, r.simd))
	return out
}

// relayTimeout restarts the relayer, which proves the timeout on bze.
func (r *rateLimitSuite) relayTimeout(t *testing.T, out helpers.IbcTransfer) {
	t.Helper()
	require.NoError(t, r.net.Relayer.StartRelayer(r.ctx, r.net.Reporter, chain.IbcPath))
	_, err := helpers.WaitForTimeout(r.ctx, r.bze, out)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// the subtests

// 1. Nothing is limited after genesis: no rate limits, no blocked denoms. A
// ubze transfer out and a stake transfer in work as in the IBC suite.
func (r *rateLimitSuite) baseline(t *testing.T) {
	limits, err := helpers.QueryRateLimits(r.ctx, r.bze)
	require.NoError(t, err)
	require.Empty(t, limits, "no rate limits after genesis")
	blocked, err := helpers.QueryBlockedDenoms(r.ctx, r.bze)
	require.NoError(t, err)
	require.Empty(t, blocked, "no blocked denoms after genesis")

	r.ibcSuite.baseline(t)
}

// 2. A governance MsgAddRateLimit sets the quota of ubze on the main channel.
// The stored channel value is the ubze supply when the proposal executed, and
// the flow starts at zero.
func (r *rateLimitSuite) addQuota(t *testing.T) {
	before := r.supply(t, r.bze, chain.Denom)
	r.govAddLimit(t, "Limit ubze on the main channel", chain.Denom, r.net.BzeChannel, rlUbzePercent, rlUbzePercent)

	rl := r.limit(t, chain.Denom, r.net.BzeChannel)
	require.Equal(t, sdkmath.NewInt(rlUbzePercent), rl.Quota.MaxPercentSend)
	require.Equal(t, sdkmath.NewInt(rlUbzePercent), rl.Quota.MaxPercentRecv)
	require.EqualValues(t, rlWindowHours, rl.Quota.DurationHours)
	require.True(t, rl.Flow.ChannelValue.GTE(before), "channel value %s is the supply when the quota was added, at least %s", rl.Flow.ChannelValue, before)
	require.True(t, rl.Flow.ChannelValue.LTE(r.supply(t, r.bze, chain.Denom)), "channel value is not above the current supply")
	r.requireFlow(t, chain.Denom, r.net.BzeChannel, sdkmath.ZeroInt(), sdkmath.ZeroInt())
	r.ubzeThreshold = helpers.Threshold(rl.Flow.ChannelValue, rlUbzePercent)
	require.True(t, r.ubzeThreshold.IsPositive())
	require.True(t, r.balance(t, r.bze, r.rich, chain.Denom).GT(r.ubzeThreshold.MulRaw(2)), "the rich sender can cover twice the threshold")

	limits, err := helpers.QueryRateLimits(r.ctx, r.bze)
	require.NoError(t, err)
	require.Len(t, limits, 1)
}

// 3. threshold + 1 is refused as a failed tx that leaves only the fee behind;
// exactly the threshold is accepted and received on simd.
func (r *rateLimitSuite) outflowRefused(t *testing.T) {
	r.requireSendRefused(t, r.rich, r.main, sdk.NewCoin(chain.Denom, r.ubzeThreshold.AddRaw(1)), rlQuotaErrCode, rlQuotaMsg)
	r.requireFlow(t, chain.Denom, r.net.BzeChannel, sdkmath.ZeroInt(), sdkmath.ZeroInt())

	r.sendUbze(t, r.rich, r.main, r.ubzeThreshold)
	r.requireFlow(t, chain.Denom, r.net.BzeChannel, sdkmath.ZeroInt(), r.ubzeThreshold)
}

// 4. The quota is on the net flow: a voucher coming home counts as inflow and
// makes room. With the outflow at the threshold, X vouchers back leave room for
// exactly X ubze: X + 1 is refused.
func (r *rateLimitSuite) netFlow(t *testing.T) {
	// the outflow is at the threshold: one more ubze is refused
	r.requireSendRefused(t, r.rich, r.main, sdk.NewInt64Coin(chain.Denom, 1), rlQuotaErrCode, rlQuotaMsg)

	r.returned = r.ubzeThreshold.QuoRaw(2)
	recv := r.returnUbze(t, r.returned.Int64())
	require.Empty(t, callbackEvents(recv, rlDeniedEvent))
	r.requireFlow(t, chain.Denom, r.net.BzeChannel, r.returned, r.ubzeThreshold)
	require.Equal(t, r.ubzeThreshold.Sub(r.returned), r.limit(t, chain.Denom, r.net.BzeChannel).NetOutflow(), "net outflow is lower by what came back")

	r.requireSendRefused(t, r.rich, r.main, sdk.NewCoin(chain.Denom, r.returned.AddRaw(1)), rlQuotaErrCode, rlQuotaMsg)
}

// 5. A quota on the stake voucher: threshold + 1 coming in is received on bze
// with an error acknowledgement (quota exceeded), nothing is minted and simd
// refunds the sender when the ack is relayed; exactly the threshold is
// accepted.
func (r *rateLimitSuite) inflowRefused(t *testing.T) {
	// seed first: the channel value is the voucher supply when the quota is added
	r.transferIn(t, chain.SimdDenom, rlStakeSeed).requireAccepted(t)
	r.govAddLimit(t, "Limit the stake voucher", r.stakeVoucher, r.net.BzeChannel, rlStakePercent, rlStakePercent)
	rl := r.limit(t, r.stakeVoucher, r.net.BzeChannel)
	require.Equal(t, r.supply(t, r.bze, r.stakeVoucher), rl.Flow.ChannelValue, "channel value is the voucher supply")
	r.stakeThreshold = helpers.Threshold(rl.Flow.ChannelValue, rlStakePercent)
	require.Equal(t, sdkmath.NewInt((ibcStakeIn+rlStakeSeed)*rlStakePercent/100), r.stakeThreshold, "exact arithmetic")

	over := r.transferIn(t, chain.SimdDenom, r.stakeThreshold.AddRaw(1).Int64())
	r.requireDenied(t, over, rlReasonQuota, rlQuotaErrCode)
	r.requireFlow(t, r.stakeVoucher, r.net.BzeChannel, sdkmath.ZeroInt(), sdkmath.ZeroInt())

	exact := r.transferIn(t, chain.SimdDenom, r.stakeThreshold.Int64())
	exact.requireAccepted(t)
	r.requireFlow(t, r.stakeVoucher, r.net.BzeChannel, r.stakeThreshold, sdkmath.ZeroInt())
}

// 6. A transfer that times out gives its outflow back: with the relayer
// stopped a ubze transfer outlives its timeout; once relayed the sender is
// refunded and the window's flow is what it was. The room it held is usable
// again: X ubze, which only fit because the timed-out amount was undone.
func (r *rateLimitSuite) timeoutUndo(t *testing.T) {
	before := r.limit(t, chain.Denom, r.net.BzeChannel)
	ubze := r.balance(t, r.bze, r.rich, chain.Denom)
	escrow := r.balanceOf(t, r.bze, r.bzeEscrow, chain.Denom)
	vouchers := r.balance(t, r.simd, r.simdUser, r.ubzeVoucher)

	out := r.sendTimingOut(t, r.rich, r.main, sdk.NewInt64Coin(chain.Denom, rlTimeoutOut))
	fee, err := helpers.TxFee(out.Tx, chain.Denom)
	require.NoError(t, err)
	r.requireFlow(t, chain.Denom, r.net.BzeChannel, before.Flow.Inflow, before.Flow.Outflow.AddRaw(rlTimeoutOut))
	require.Equal(t, escrow.AddRaw(rlTimeoutOut), r.balanceOf(t, r.bze, r.bzeEscrow, chain.Denom), "escrowed while in flight")

	r.relayTimeout(t, out)
	require.Equal(t, ubze.Sub(fee), r.balance(t, r.bze, r.rich, chain.Denom), "refunded, minus the tx fee only")
	require.Equal(t, escrow, r.balanceOf(t, r.bze, r.bzeEscrow, chain.Denom), "escrow released")
	require.Equal(t, vouchers, r.balance(t, r.simd, r.simdUser, r.ubzeVoucher), "nothing minted on simd")
	r.requireFlow(t, chain.Denom, r.net.BzeChannel, before.Flow.Inflow, before.Flow.Outflow)

	// the room is back: the net flow of step 4 plus X is exactly the threshold
	r.sendUbze(t, r.rich, r.main, r.returned)
	r.requireFlow(t, chain.Denom, r.net.BzeChannel, r.returned, r.ubzeThreshold.Add(r.returned))
	r.requireSendRefused(t, r.rich, r.main, sdk.NewInt64Coin(chain.Denom, 1), rlQuotaErrCode, rlQuotaMsg)
}

// 7. The quota is per channel: over a second channel with no rate limit, more
// than the first channel's threshold goes through.
func (r *rateLimitSuite) otherChannel(t *testing.T) {
	r.second = r.openChannel(t)
	_, err := helpers.QueryRateLimit(r.ctx, r.bze, r.second.bze, chain.Denom)
	require.Error(t, err, "no rate limit on the second channel")

	r.sendUbze(t, r.rich, r.second, r.ubzeThreshold.AddRaw(rlSecondExtra))
	r.requireFlow(t, chain.Denom, r.net.BzeChannel, r.returned, r.ubzeThreshold.Add(r.returned))
}

// 8. The guardian lowers the quota: the window and the flow stay, the next
// outflow is refused, a voucher coming home is still received. Raising a quota
// is refused as is a tighten signed by anyone else.
func (r *rateLimitSuite) guardianTightens(t *testing.T) {
	before := r.limit(t, chain.Denom, r.net.BzeChannel)
	require.True(t, before.NetOutflow().GTE(sdkmath.ZeroInt()), "net outflow is not negative: a send quota of 0 refuses 1 ubze")

	res := r.tx(t, r.guardian, "guardian", "tighten-rate-limit", chain.Denom, r.net.BzeChannel, "0", strconv.Itoa(rlUbzePercent))
	require.Zero(t, res.Code, res.RawLog)
	tightened := r.limit(t, chain.Denom, r.net.BzeChannel)
	require.True(t, tightened.Quota.MaxPercentSend.IsZero(), "send quota is 0")
	require.Equal(t, sdkmath.NewInt(rlUbzePercent), tightened.Quota.MaxPercentRecv)
	require.Equal(t, before.Quota.DurationHours, tightened.Quota.DurationHours, "the window is unchanged")
	require.Equal(t, before.Flow.Inflow, tightened.Flow.Inflow, "the flow is kept")
	require.Equal(t, before.Flow.Outflow, tightened.Flow.Outflow, "the flow is kept")
	require.Equal(t, before.Flow.ChannelValue, tightened.Flow.ChannelValue, "the channel value is kept")

	r.requireSendRefused(t, r.rich, r.main, sdk.NewInt64Coin(chain.Denom, 1), rlQuotaErrCode, rlQuotaMsg)
	r.returnUbze(t, rlBackAmount)

	// a raise is refused, the quota stays
	res = r.tx(t, r.guardian, "guardian", "tighten-rate-limit", chain.Denom, r.net.BzeChannel, "0", strconv.Itoa(rlUbzePercent+1))
	require.EqualValues(t, rlGuardianQuotaRaiseCode, res.Code, res.RawLog)
	require.Equal(t, "guardian", res.Codespace)
	require.Equal(t, tightened.Quota, r.limit(t, chain.Denom, r.net.BzeChannel).Quota)

	// anyone else is not authorized
	res = r.tx(t, r.bzeUser, "guardian", "tighten-rate-limit", chain.Denom, r.net.BzeChannel, "0", "0")
	require.EqualValues(t, rlGuardianUnauthorizedCode, res.Code, res.RawLog)
	require.Equal(t, "guardian", res.Codespace)
	require.Equal(t, tightened.Quota, r.limit(t, chain.Denom, r.net.BzeChannel).Quota)
}

// 9. Governance loosens the quota back. UpdateRateLimit resets the flow (and
// samples the channel value again): the window starts empty.
func (r *rateLimitSuite) govLoosens(t *testing.T) {
	r.govUpdateLimit(t, "Restore the ubze quota", chain.Denom, r.net.BzeChannel, rlUbzePercent, rlUbzePercent)
	rl := r.limit(t, chain.Denom, r.net.BzeChannel)
	require.Equal(t, sdkmath.NewInt(rlUbzePercent), rl.Quota.MaxPercentSend)
	require.Equal(t, sdkmath.NewInt(rlUbzePercent), rl.Quota.MaxPercentRecv)
	r.requireFlow(t, chain.Denom, r.net.BzeChannel, sdkmath.ZeroInt(), sdkmath.ZeroInt())
	r.ubzeThreshold = helpers.Threshold(rl.Flow.ChannelValue, rlUbzePercent)

	r.sendUbze(t, r.rich, r.main, sdkmath.NewInt(rlUpdateOut))
	r.requireFlow(t, chain.Denom, r.net.BzeChannel, sdkmath.ZeroInt(), sdkmath.NewInt(rlUpdateOut))
}

// 10. The guardian blocks the stake voucher on every channel. A packet already
// in flight when it happens times out and is refunded. While blocked an inbound
// stake transfer is answered with an error acknowledgement and refunded, an
// outbound voucher is refused as a tx, and the list shows the denom. The
// guardian cannot lift it: a proposal needs the gov authority as signer. A
// governance proposal lifts it and inbound works again.
func (r *rateLimitSuite) guardianBlocks(t *testing.T) {
	voucher := r.stakeVoucher

	// vouchers home first: the stake window has only room for what leaves
	r.sendVouchers(t, rlReturnStake)

	// in flight while the guardian blocks
	held := r.balance(t, r.bze, r.bzeUser, voucher)
	ubze := r.balance(t, r.bze, r.bzeUser, chain.Denom)
	supply := r.supply(t, r.bze, voucher)
	flow := r.limit(t, voucher, r.net.BzeChannel).Flow
	out := r.sendTimingOut(t, r.bzeUser, r.main, sdk.NewInt64Coin(voucher, rlInFlight))
	fee, err := helpers.TxFee(out.Tx, chain.Denom)
	require.NoError(t, err)
	require.Equal(t, held.SubRaw(rlInFlight), r.balance(t, r.bze, r.bzeUser, voucher), "burned while in flight")

	res := r.tx(t, r.guardian, "guardian", "block-denom", voucher)
	require.Zero(t, res.Code, res.RawLog)
	blocked, err := helpers.QueryBlockedDenoms(r.ctx, r.bze)
	require.NoError(t, err)
	require.Equal(t, []string{voucher}, blocked)

	r.relayTimeout(t, out)
	require.Equal(t, held, r.balance(t, r.bze, r.bzeUser, voucher), "the in-flight packet was refunded although the denom is blocked")
	require.Equal(t, ubze.Sub(fee), r.balance(t, r.bze, r.bzeUser, chain.Denom), "minus the tx fee only")
	require.Equal(t, supply, r.supply(t, r.bze, voucher), "voucher supply restored")
	rl := r.limit(t, voucher, r.net.BzeChannel)
	require.Equal(t, flow.Inflow, rl.Flow.Inflow, "the flow is what it was")
	require.Equal(t, flow.Outflow, rl.Flow.Outflow, "the flow is what it was")

	// inbound is refused and refunded
	r.requireDenied(t, r.transferIn(t, chain.SimdDenom, rlBlockedIn), rlReasonBlacklisted, rlDenyErrCode)
	// outbound is refused as a tx
	r.requireSendRefused(t, r.bzeUser, r.main, sdk.NewInt64Coin(voucher, rlInFlight), rlDenyErrCode, rlBlacklistMsg)
	// ubze is not the blocked denom and still moves
	r.returnUbze(t, rlBackAmount)

	// the guardian cannot lift it: the unblock message has the gov authority as signer
	guardianAddr := r.guardian.FormattedAddress()
	_, err = helpers.SubmitAndPassProposal(r.ctx, r.bze, r.guardian, helpers.Proposal{
		Messages: []any{map[string]any{"@type": "/bze.guardian.v1.MsgUnblockDenom", "authority": guardianAddr, "denom": voucher}},
		Title:    "Unblock as the guardian",
		Summary:  "e2e: only governance unblocks",
	})
	require.ErrorContains(t, err, "expected gov account as only signer")
	blocked, err = helpers.QueryBlockedDenoms(r.ctx, r.bze)
	require.NoError(t, err)
	require.Equal(t, []string{voucher}, blocked, "still blocked")

	// blocking twice is refused
	res = r.tx(t, r.guardian, "guardian", "block-denom", voucher)
	require.EqualValues(t, rlGuardianAlreadyBlockedCode, res.Code, res.RawLog)
	require.Equal(t, "guardian", res.Codespace, res.RawLog)

	// only a denom with a supply on bze can be blocked, in its exact
	// spelling: other spellings of the voucher, its base name and trace path,
	// and an unknown denom are refused and block nothing
	for _, denom := range []string{
		"ibc/" + strings.ToLower(strings.TrimPrefix(voucher, "ibc/")),
		chain.SimdDenom,
		chain.TransferPort + "/" + r.net.BzeChannel + "/" + chain.SimdDenom,
		"unothere",
	} {
		res = r.tx(t, r.guardian, "guardian", "block-denom", denom)
		require.EqualValues(t, rlGuardianDenomNotFoundCode, res.Code, "%s: %s", denom, res.RawLog)
		require.Equal(t, "guardian", res.Codespace, res.RawLog)
		require.Contains(t, res.RawLog, "denom does not exist on chain")
		blocked, err = helpers.QueryBlockedDenoms(r.ctx, r.bze)
		require.NoError(t, err)
		require.Equal(t, []string{voucher}, blocked, "%s blocked nothing", denom)
	}

	// governance unblocks
	_, err = helpers.SubmitAndPassProposal(r.ctx, r.bze, r.admin, helpers.Proposal{
		Messages: []any{map[string]any{"@type": "/bze.guardian.v1.MsgUnblockDenom", "authority": r.gov, "denom": voucher}},
		Title:    "Unblock the stake voucher",
		Summary:  "e2e: governance unblocks",
	})
	require.NoError(t, err)
	blocked, err = helpers.QueryBlockedDenoms(r.ctx, r.bze)
	require.NoError(t, err)
	require.Empty(t, blocked)

	r.transferIn(t, chain.SimdDenom, rlBlockedIn).requireAccepted(t)
}

// sendVouchers sends amount stake vouchers from bzeUser home to simdUser over
// the main channel.
func (r *rateLimitSuite) sendVouchers(t *testing.T, amount int64) {
	t.Helper()
	held := r.balance(t, r.bze, r.bzeUser, r.stakeVoucher)
	supply := r.supply(t, r.bze, r.stakeVoucher)
	stake := r.balance(t, r.simd, r.simdUser, chain.SimdDenom)

	out, err := helpers.SendIbc(r.ctx, r.bze, r.bzeUser, r.net.BzeChannel, r.simdUser.FormattedAddress(), sdk.NewInt64Coin(r.stakeVoucher, amount))
	require.NoError(t, err)
	_, err = helpers.WaitForRecv(r.ctx, r.simd, r.net.SimdChannel, out.Sequence)
	require.NoError(t, err)
	_, err = helpers.WaitForAck(r.ctx, r.bze, out)
	require.NoError(t, err)
	require.Equal(t, held.SubRaw(amount), r.balance(t, r.bze, r.bzeUser, r.stakeVoucher))
	require.Equal(t, supply.SubRaw(amount), r.supply(t, r.bze, r.stakeVoucher), "vouchers burned on bze")
	require.Equal(t, stake.AddRaw(amount), r.balance(t, r.simd, r.simdUser, chain.SimdDenom), "released from simd escrow")
}

// 11. Conservation of coins. The refusals created and lost nothing: for every
// channel the ubze escrowed on bze is the ubze voucher supply on simd, and the
// stake escrowed on simd is the stake voucher supply on bze. (The total ubze
// supply is not asserted: it grows with every block's mint.)
func (r *rateLimitSuite) audit(t *testing.T) {
	ubzeEscrowed, ubzeVouchers := sdkmath.ZeroInt(), sdkmath.ZeroInt()
	for _, ch := range []ibcChannel{r.main, r.second} {
		ubzeEscrowed = ubzeEscrowed.Add(r.balanceOf(t, r.bze, ch.bzeEscrow, chain.Denom))
		ubzeVouchers = ubzeVouchers.Add(r.supply(t, r.simd, helpers.IbcDenom(ch.simd, chain.Denom)))
		require.Equal(t, r.balanceOf(t, r.bze, ch.bzeEscrow, chain.Denom), r.supply(t, r.simd, helpers.IbcDenom(ch.simd, chain.Denom)), "ubze escrowed on %s is the voucher supply on simd", ch.bze)
	}
	require.Equal(t, ubzeEscrowed, ubzeVouchers)
	require.True(t, ubzeEscrowed.IsPositive())

	require.Equal(t, r.balanceOf(t, r.simd, r.simdEscrow, chain.SimdDenom), r.supply(t, r.bze, r.stakeVoucher), "stake escrowed on simd is the voucher supply on bze")
	require.True(t, r.balanceOf(t, r.simd, r.second.simdEscrow, chain.SimdDenom).IsZero(), "nothing crossed the second channel as stake")
}
