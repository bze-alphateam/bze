package helpers

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
)

// AppVersion returns the version the running bzed reports in ABCI Info, the
// one `bzed version` prints.
func AppVersion(ctx context.Context, chain *cosmos.CosmosChain) (string, error) {
	node := chain.GetNode()
	if node.Client == nil {
		return "", fmt.Errorf("node %s has no RPC client", node.Name())
	}
	res, err := node.Client.ABCIInfo(ctx)
	if err != nil {
		return "", fmt.Errorf("abci_info: %w", err)
	}
	return res.Response.Version, nil
}

// WaitForHeight blocks until the chain's latest height is at least h and
// returns it, or fails when ctx ends. Bound it with a context deadline.
func WaitForHeight(ctx context.Context, chain *cosmos.CosmosChain, h int64) (int64, error) {
	var cur int64
	var err error
	for {
		cur, err = chain.Height(ctx)
		if err == nil && cur >= h {
			return cur, nil
		}
		select {
		case <-ctx.Done():
			return cur, fmt.Errorf("height %d not reached (latest %d, last error %v): %w", h, cur, err, ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// WaitForHalt waits until the chain reaches haltHeight, then watches it for
// settle and fails if it moves on. A chain halted by an upgrade plan stops at
// the plan height: its block store holds the block the old binary refused to
// execute, its state is the one of the block before.
func WaitForHalt(ctx context.Context, chain *cosmos.CosmosChain, haltHeight int64, settle time.Duration) error {
	cur, err := WaitForHeight(ctx, chain, haltHeight)
	if err != nil {
		return err
	}
	if cur != haltHeight {
		return fmt.Errorf("chain went past the halt height %d to %d", haltHeight, cur)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(settle):
	}
	cur, err = chain.Height(ctx)
	if err != nil {
		return err
	}
	if cur != haltHeight {
		return fmt.Errorf("chain did not halt at %d: at %d after %s", haltHeight, cur, settle)
	}
	return nil
}

// SupplyChange sums the coins minted (bank "coinbase" events, by minter) and
// burned (bank "burn" events, by burner) in events, e.g. the finalize-block
// events of one height.
func SupplyChange(events []Event) (minted, burned map[string]sdk.Coins, err error) {
	minted, burned = map[string]sdk.Coins{}, map[string]sdk.Coins{}
	for _, e := range events {
		var into map[string]sdk.Coins
		var who string
		switch e.Type {
		case "coinbase":
			into, who = minted, e.Attributes["minter"]
		case "burn":
			into, who = burned, e.Attributes["burner"]
		default:
			continue
		}
		coins, err := sdk.ParseCoinsNormalized(e.Attributes["amount"])
		if err != nil {
			return nil, nil, fmt.Errorf("parse %s amount %q: %w", e.Type, e.Attributes["amount"], err)
		}
		into[who] = into[who].Add(coins...)
	}
	return minted, burned, nil
}
