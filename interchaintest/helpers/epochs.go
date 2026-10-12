package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"
)

// EpochInfo is one entry of `bzed q epochs all`.
type EpochInfo struct {
	Identifier              string `json:"identifier"`
	Duration                string `json:"duration"`
	CurrentEpoch            Int64  `json:"current_epoch"`
	CurrentEpochStartHeight Int64  `json:"current_epoch_start_height"`
	EpochCountingStarted    bool   `json:"epoch_counting_started"`
}

// Epochs returns every epoch (`bzed q epochs all`).
func Epochs(ctx context.Context, chain *cosmos.CosmosChain) ([]EpochInfo, error) {
	res, err := Query[struct {
		Epochs []EpochInfo `json:"epochs"`
	}](ctx, chain, "epochs", "all")
	return res.Epochs, err
}

// Epoch returns one epoch by identifier ("day", "hour", "week").
func Epoch(ctx context.Context, chain *cosmos.CosmosChain, identifier string) (EpochInfo, error) {
	epochs, err := Epochs(ctx, chain)
	if err != nil {
		return EpochInfo{}, err
	}
	for _, e := range epochs {
		if e.Identifier == identifier {
			return e, nil
		}
	}
	return EpochInfo{}, fmt.Errorf("epoch %q not found", identifier)
}

// CurrentEpoch returns the current number of an epoch (`bzed q epochs identifier <id>`).
func CurrentEpoch(ctx context.Context, chain *cosmos.CosmosChain, identifier string) (int64, error) {
	res, err := Query[struct {
		CurrentEpoch Int64 `json:"current_epoch"`
	}](ctx, chain, "epochs", "identifier", identifier)
	return int64(res.CurrentEpoch), err
}

// WaitForEpoch blocks until the epoch's current number is at least n, or ctx
// ends. Bound it with a context deadline.
func WaitForEpoch(ctx context.Context, chain *cosmos.CosmosChain, identifier string, n int64) error {
	for {
		cur, err := CurrentEpoch(ctx, chain, identifier)
		if err == nil && cur >= n {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("epoch %s did not reach %d (current %d, last error %v): %w", identifier, n, cur, err, ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}
