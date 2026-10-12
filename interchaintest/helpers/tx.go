package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
)

const (
	// DefaultFixedGas is the gas limit used with WithFees when WithGas is not set.
	DefaultFixedGas uint64 = 500_000

	txInclusionTimeout = 30 * time.Second
	pollInterval       = 500 * time.Millisecond
)

// TxResult is a broadcast transaction. Code is the CheckTx code when the tx was
// rejected before inclusion (Height is then 0), otherwise the DeliverTx code.
type TxResult struct {
	TxHash    string
	Height    int64
	Code      uint32
	Codespace string
	RawLog    string
	GasUsed   int64
	GasWanted int64
	Events    []Event
}

// SimulationError is returned when `--gas auto` simulation fails, i.e. the
// chain rejected the tx before it was broadcast. Suites asserting a rejection
// code should pass WithGas so the tx is broadcast and Code is set instead.
type SimulationError struct {
	Output string
}

func (e *SimulationError) Error() string {
	return "tx simulation failed: " + e.Output
}

type txOptions struct {
	fees string
	gas  uint64
}

// TxOption customises ExecTxWith.
type TxOption func(*txOptions)

// WithFees pays a fixed fee (e.g. "10000ufoo") instead of --gas-prices, with a
// fixed gas limit (WithGas, or DefaultFixedGas).
func WithFees(fees string) TxOption {
	return func(o *txOptions) { o.fees = fees }
}

// WithGas sets a fixed gas limit instead of --gas auto (skips simulation).
func WithGas(gas uint64) TxOption {
	return func(o *txOptions) { o.gas = gas }
}

type txResponse struct {
	Height    Int64      `json:"height"`
	TxHash    string     `json:"txhash"`
	Codespace string     `json:"codespace"`
	Code      uint32     `json:"code"`
	RawLog    string     `json:"raw_log"`
	GasWanted Int64      `json:"gas_wanted"`
	GasUsed   Int64      `json:"gas_used"`
	Events    []rawEvent `json:"events"`
}

func (r txResponse) result() TxResult {
	return TxResult{
		TxHash:    r.TxHash,
		Height:    int64(r.Height),
		Code:      r.Code,
		Codespace: r.Codespace,
		RawLog:    r.RawLog,
		GasUsed:   int64(r.GasUsed),
		GasWanted: int64(r.GasWanted),
		Events:    convertEvents(r.Events),
	}
}

// ExecTx signs `bzed tx <args>` with the user's key on the first validator,
// with --gas auto and the chain's gas prices, and waits for inclusion. A
// non-zero Code is not an error: suites assert rejections on it. Errors are
// infrastructure failures, or a *SimulationError.
func ExecTx(ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, args ...string) (TxResult, error) {
	return ExecTxWith(ctx, chain, user, nil, args...)
}

// ExecTxWith is ExecTx with options (WithFees, WithGas).
func ExecTxWith(ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, opts []TxOption, args ...string) (TxResult, error) {
	var o txOptions
	for _, opt := range opts {
		opt(&o)
	}

	cfg := chain.Config()
	cmd := append([]string{"tx"}, args...)
	cmd = append(cmd,
		"--from", user.KeyName(),
		"--keyring-backend", "test",
		"--chain-id", cfg.ChainID,
		"--output", "json",
		"-y",
	)
	switch {
	case o.fees != "":
		gas := o.gas
		if gas == 0 {
			gas = DefaultFixedGas
		}
		cmd = append(cmd, "--fees", o.fees, "--gas", strconv.FormatUint(gas, 10))
	case o.gas != 0:
		cmd = append(cmd, "--gas", strconv.FormatUint(o.gas, 10), "--gas-prices", cfg.GasPrices)
	default:
		cmd = append(cmd,
			"--gas", "auto",
			"--gas-adjustment", strconv.FormatFloat(cfg.GasAdjustment, 'f', -1, 64),
			"--gas-prices", cfg.GasPrices,
		)
	}

	node := chain.GetNode()
	stdout, stderr, err := node.Exec(ctx, node.NodeCommand(cmd...), cfg.Env)
	if err != nil {
		out := strings.TrimSpace(string(stderr) + "\n" + string(stdout))
		// With --gas auto the CLI simulates before signing; a chain rejection
		// then exits non-zero without printing a broadcast response.
		if o.gas == 0 && o.fees == "" && !json.Valid(stdout) {
			return TxResult{}, &SimulationError{Output: out}
		}
		return TxResult{}, fmt.Errorf("bzed tx %s: %w: %s", strings.Join(args, " "), err, out)
	}

	var broadcast txResponse
	if err := json.Unmarshal(stdout, &broadcast); err != nil {
		return TxResult{}, fmt.Errorf("decode broadcast response %q: %w", stdout, err)
	}
	if broadcast.Code != 0 {
		// Rejected by CheckTx: never included in a block.
		return broadcast.result(), nil
	}
	return WaitForTx(ctx, chain, broadcast.TxHash)
}

// WaitForTx polls `bzed q tx <hash>` until the tx is included or 30 s pass.
func WaitForTx(ctx context.Context, chain *cosmos.CosmosChain, txHash string) (TxResult, error) {
	ctx, cancel := context.WithTimeout(ctx, txInclusionTimeout)
	defer cancel()

	var lastErr error
	for {
		res, err := Query[txResponse](ctx, chain, "tx", txHash)
		if err == nil {
			return res.result(), nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return TxResult{}, fmt.Errorf("tx %s not included: %w (last error: %v)", txHash, ctx.Err(), lastErr)
		case <-time.After(pollInterval):
		}
	}
}
