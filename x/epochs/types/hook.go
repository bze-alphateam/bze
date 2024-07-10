package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type EpochHook interface {
	// AfterEpochEnd executed at first block with a timestamp after epoch duration (epoch end)
	AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) error

	// BeforeEpochStart new epoch is called after AfterEpochEnd in the same BeginBlocker
	BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64) error

	// GetName Returns the name of the hook
	GetName() string
}
