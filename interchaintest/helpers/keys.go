package helpers

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
)

// NewWallet adds a key to the first validator's test keyring, inside the
// running container, and returns its wallet. It is the fast path for suites
// that need many unfunded accounts (fund them with one MsgMultiSend);
// interchaintest's BuildWallet starts a container per key.
func NewWallet(ctx context.Context, chain *cosmos.CosmosChain, keyName string) (ibc.Wallet, error) {
	node := chain.GetNode()
	cfg := chain.Config()
	keys := func(args ...string) []string {
		return append(append([]string{cfg.Bin, "keys"}, args...), "--keyring-backend", "test", "--home", node.HomeDir())
	}
	if _, _, err := execNode(ctx, node, keys("add", keyName)); err != nil {
		return nil, fmt.Errorf("add key %s: %w", keyName, err)
	}
	stdout, _, err := execNode(ctx, node, keys("show", keyName, "-a"))
	if err != nil {
		return nil, fmt.Errorf("show key %s: %w", keyName, err)
	}
	bech32 := strings.TrimSpace(string(stdout))
	addr, err := sdk.GetFromBech32(bech32, cfg.Bech32Prefix)
	if err != nil {
		return nil, fmt.Errorf("parse address %q of key %s: %w", bech32, keyName, err)
	}
	return cosmos.NewWallet(keyName, addr, "", cfg), nil
}

// CLI runs `bzed <args>` against the first validator, inside its running
// container, and returns stdout; e.g. a `tx … --generate-only` that prints an
// unsigned tx instead of broadcasting it. A non-zero exit is an error holding
// the CLI's output.
func CLI(ctx context.Context, chain *cosmos.CosmosChain, args ...string) ([]byte, error) {
	node := chain.GetNode()
	stdout, _, err := execNode(ctx, node, node.NodeCommand(args...))
	return stdout, err
}
