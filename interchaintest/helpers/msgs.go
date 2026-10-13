package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"sync/atomic"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
)

// txFileSeq keeps the tx files of one test process apart in the node volume.
var txFileSeq atomic.Uint64

// FeeForGas returns the fee `--gas-prices` would charge for a gas limit:
// ceil(price × gas) for each price, e.g. "0.01ubze" and 500,000 gas → 5000ubze.
func FeeForGas(gasPrices string, gas uint64) (sdk.Coins, error) {
	prices, err := sdk.ParseDecCoins(gasPrices)
	if err != nil {
		return nil, fmt.Errorf("parse gas prices %q: %w", gasPrices, err)
	}
	limit := sdkmath.LegacyNewDecFromInt(sdkmath.NewIntFromUint64(gas))
	fee := sdk.NewCoins()
	for _, p := range prices {
		fee = fee.Add(sdk.NewCoin(p.Denom, p.Amount.Mul(limit).Ceil().TruncateInt()))
	}
	return fee, nil
}

// UnsignedTx returns the JSON of an unsigned tx carrying msgs, in the shape
// `bzed tx … --generate-only` prints and `bzed tx sign` reads. Each msg is a
// JSON-encodable proto message with its "@type", e.g.
// map[string]any{"@type": "/bze.tradebin.MsgFillOrders", "creator": …}.
func UnsignedTx(fee sdk.Coins, gas uint64, msgs ...any) ([]byte, error) {
	if len(msgs) == 0 {
		return nil, fmt.Errorf("a tx needs at least one message")
	}
	amount := make([]map[string]string, 0, len(fee))
	for _, c := range fee {
		amount = append(amount, map[string]string{"denom": c.Denom, "amount": c.Amount.String()})
	}
	return json.Marshal(map[string]any{
		"body": map[string]any{
			"messages":                       msgs,
			"memo":                           "",
			"timeout_height":                 "0",
			"extension_options":              []any{},
			"non_critical_extension_options": []any{},
		},
		"auth_info": map[string]any{
			"signer_infos": []any{},
			"fee": map[string]any{
				"amount":    amount,
				"gas_limit": strconv.FormatUint(gas, 10),
				"payer":     "",
				"granter":   "",
			},
		},
		"signatures": []any{},
	})
}

// ExecMsgs signs msgs into one tx with the user's key, broadcasts it and waits
// for inclusion, like ExecTx. It is for messages the CLI can not express, such
// as a MsgFillOrders with several orders (autocli takes one repeated message
// argument). The gas limit is fixed (WithGas, or DefaultFixedGas) and the fee
// is WithFees, else the limit at the gas prices (WithGasPrices or the chain's).
func ExecMsgs(ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, opts []TxOption, msgs ...any) (TxResult, error) {
	var o txOptions
	for _, opt := range opts {
		opt(&o)
	}
	cfg := chain.Config()
	gas := o.gas
	if gas == 0 {
		gas = DefaultFixedGas
	}
	var fee sdk.Coins
	var err error
	if o.fees != "" {
		fee, err = sdk.ParseCoinsNormalized(o.fees)
	} else {
		fee, err = FeeForGas(o.prices(cfg), gas)
	}
	if err != nil {
		return TxResult{}, err
	}
	unsigned, err := UnsignedTx(fee, gas, msgs...)
	if err != nil {
		return TxResult{}, err
	}

	node := chain.GetNode()
	name := fmt.Sprintf("tx-%s-%d", user.KeyName(), txFileSeq.Add(1))
	if err := writeNodeFile(ctx, node, unsigned, name+".json"); err != nil {
		return TxResult{}, fmt.Errorf("write unsigned tx: %w", err)
	}
	unsignedPath := path.Join(node.HomeDir(), name+".json")
	signedPath := path.Join(node.HomeDir(), name+"-signed.json")

	sign := node.NodeCommand("tx", "sign", unsignedPath,
		"--from", user.KeyName(),
		"--keyring-backend", "test",
		"--chain-id", cfg.ChainID,
		"--output-document", signedPath,
	)
	if _, _, err := execNode(ctx, node, sign); err != nil {
		return TxResult{}, fmt.Errorf("sign tx: %w", err)
	}
	stdout, _, err := execNode(ctx, node, node.NodeCommand("tx", "broadcast", signedPath, "--output", "json"))
	if err != nil {
		return TxResult{}, fmt.Errorf("broadcast tx: %w", err)
	}
	return awaitBroadcast(ctx, chain, stdout)
}
