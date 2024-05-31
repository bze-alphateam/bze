package keeper

import (
	"fmt"
	"github.com/bze-alphateam/bze/bzeutils"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AfterEpochEnd gets called at the end of the epoch, end of epoch is the timestamp of first block produced after epoch duration.
func (k Keeper) AfterEpochEnd(ctx sdk.Context, identifier string, epochNumber int64) {
	for _, h := range k.hooks {
		k.safeHookCall(ctx, h.AfterEpochEnd, identifier, epochNumber, h.GetName())
	}
}

// BeforeEpochStart new epoch is next block of epoch end block
func (k Keeper) BeforeEpochStart(ctx sdk.Context, identifier string, epochNumber int64) {
	for _, h := range k.hooks {
		k.safeHookCall(ctx, h.BeforeEpochStart, identifier, epochNumber, h.GetName())
	}
}

func (k Keeper) safeHookCall(
	ctx sdk.Context,
	hookFn func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error,
	epochIdentifier string,
	epochNumber int64,
	hookName string,
) {
	// wrap function
	wrappedHookFn := func(ctx sdk.Context) error {
		return hookFn(ctx, epochIdentifier, epochNumber)
	}

	// safely call the wrapped hook function
	err := bzeutils.ApplyFuncIfNoError(ctx, wrappedHookFn)
	if err != nil {
		ctx.Logger().Error(fmt.Sprintf("error in %s epoch hook %v", hookName, err))
	}
}
