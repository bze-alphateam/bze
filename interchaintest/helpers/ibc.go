package helpers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
)

const (
	transferPort = "transfer"
	// packetTimeout bounds the wait for a relayed packet, ack or timeout.
	packetTimeout = 2 * time.Minute
)

// IbcDenom returns the voucher denom of a token that arrived through
// port/channel on the receiving chain: `ibc/<sha256 of "transfer/<channel>/<denom>">`.
// denom is the denom as carried by the packet (a base denom, or a path for
// multi-hop tokens).
func IbcDenom(channel, denom string) string {
	return transfertypes.ExtractDenomFromPath(transferPort + "/" + channel + "/" + denom).IBCDenom()
}

// EscrowAddress returns the bech32 address that escrows the chain's own
// tokens sent out through the transfer channel.
func EscrowAddress(prefix, channel string) (string, error) {
	return sdk.Bech32ifyAddressBytes(prefix, transfertypes.GetEscrowAddress(transferPort, channel))
}

// IbcTransfer is a sent ICS-20 transfer: the tx and the packet's source
// channel and sequence.
type IbcTransfer struct {
	Tx       TxResult
	Channel  string
	Sequence uint64
}

// SendIbc sends coin from user to receiver over the transfer channel with
// `tx ibc-transfer transfer` and the default timeouts, or the extra flags
// (e.g. "--packet-timeout-timestamp", "1"). It fails unless the tx is
// accepted and emits a send_packet event.
func SendIbc(ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, channel, receiver string, coin sdk.Coin, flags ...string) (IbcTransfer, error) {
	args := append([]string{"ibc-transfer", "transfer", transferPort, channel, receiver, coin.String()}, flags...)
	res, err := ExecTx(ctx, chain, user, args...)
	if err != nil {
		return IbcTransfer{}, err
	}
	if res.Code != 0 {
		return IbcTransfer{}, fmt.Errorf("ibc transfer: code %d: %s", res.Code, res.RawLog)
	}
	sends := EventsWith(TxEvents(res, "send_packet"), map[string]string{"packet_src_channel": channel})
	if len(sends) != 1 {
		return IbcTransfer{}, fmt.Errorf("ibc transfer %s: want one send_packet event on %s, got %d", res.TxHash, channel, len(sends))
	}
	seq, err := strconv.ParseUint(sends[0].Attributes["packet_sequence"], 10, 64)
	if err != nil {
		return IbcTransfer{}, fmt.Errorf("parse packet_sequence: %w", err)
	}
	return IbcTransfer{Tx: res, Channel: channel, Sequence: seq}, nil
}

// WaitForRecv waits until the relayer delivered the packet on the receiving
// chain and returns that tx; channel is the packet's destination channel.
func WaitForRecv(ctx context.Context, dst *cosmos.CosmosChain, channel string, sequence uint64) (TxResult, error) {
	return waitForPacketTx(ctx, dst, "recv_packet", "packet_dst_channel", channel, sequence)
}

// WaitForAck waits until the relayer delivered the packet's acknowledgement
// back on the sending chain and returns that tx.
func WaitForAck(ctx context.Context, src *cosmos.CosmosChain, t IbcTransfer) (TxResult, error) {
	return waitForPacketTx(ctx, src, "acknowledge_packet", "packet_src_channel", t.Channel, t.Sequence)
}

// WaitForTimeout waits until the relayer proved the packet timed out on the
// sending chain and returns that tx.
func WaitForTimeout(ctx context.Context, src *cosmos.CosmosChain, t IbcTransfer) (TxResult, error) {
	return waitForPacketTx(ctx, src, "timeout_packet", "packet_src_channel", t.Channel, t.Sequence)
}

type txSearch struct {
	Txs []txResponse `json:"txs"`
}

// waitForPacketTx waits up to two minutes for the successful tx emitting
// eventType for the packet, following the chain's blocks, and reads it with
// `q tx`. A relayer may batch several packets into one tx, so callers select
// the packet's events by their attributes.
func waitForPacketTx(ctx context.Context, chain *cosmos.CosmosChain, eventType, channelAttr, channel string, sequence uint64) (TxResult, error) {
	ctx, cancel := context.WithTimeout(ctx, packetTimeout)
	defer cancel()

	w, err := watchBlocks(ctx, chain)
	if err != nil {
		return TxResult{}, err
	}
	defer w.Close()

	// Watching first, then checking the tx index for an earlier delivery.
	query := fmt.Sprintf("%s.%s='%s' AND %s.packet_sequence='%d'", eventType, channelAttr, channel, eventType, sequence)
	if res, err := Query[txSearch](ctx, chain, "txs", "--query", query); err == nil {
		for _, tx := range res.Txs {
			if tx.Code == 0 {
				return tx.result(), nil
			}
		}
	}
	attrs := map[string]string{channelAttr: channel, "packet_sequence": strconv.FormatUint(sequence, 10)}
	var hash string
	_, err = w.until(ctx, func(b streamBlock) (bool, error) {
		for _, tx := range b.Txs {
			if tx.Code == 0 && len(EventsWith(filterEvents(tx.Events, eventType), attrs)) > 0 {
				hash = tx.Hash
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return TxResult{}, fmt.Errorf("no %s tx for %s sequence %d: %w", eventType, channel, sequence, err)
	}
	return queryIncludedTx(ctx, chain, hash)
}
