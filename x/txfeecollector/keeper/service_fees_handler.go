package keeper

import (
	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (k Keeper) ConvertCollectedFeesToNativeDenom(ctx sdk.Context) error {
	// change code to use sendSkipped = true here because:
	// 1. the skipped coins are only the ones that cannot be swapped for native denom
	// 2. the coins that cannot be swapped are those that don't have a LP with BZE or the LP is not deep enough
	// 3. the coins that can not be swapped should never reach this module's address because the ante handler that sends
	// them here is usually checking the LP
	// 4. In case we have such coins (there are edge cases that can allow this) we should send them to the SDK's fee
	// collector for performance on the end block of this module.
	//
	// Bottom line is: we allow the coins to be sent to validator/stakers instead of letting them grow here to be forever checked
	// in each block.
	return k.convertFeesAndSend(ctx, types.ModuleName, authtypes.FeeCollectorName, true)
}

// convertFeesAndSend transfers converted fees from one module to another if fees are available and conversion is successful.
func (k Keeper) convertFeesAndSend(ctx sdk.Context, fromModule, toModule string, sendSkipped bool) error {
	toSend, skipped, err := k.convertFees(ctx, fromModule)
	if err != nil {
		return err
	}
	if sendSkipped && skipped.Len() > 0 {
		toSend = toSend.Add(skipped...)
	}

	if toSend == nil || toSend.IsZero() {
		return nil
	}

	err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromModule, toModule, toSend)

	return err
}

// convertFees converts module-held fees into native denominations and categorizes them as swappable or non-swappable.
// Returns the swappable fees, non-swappable fees, and any error encountered during the process.
//
// The coins that have deep enough liquidity but can't be swapped due to the amount being too low remain untouched
// until they can be swapped.
// It looks at one page of the module's balances, the ones after the module's cursor (see readCollectorPage).
// Each coin is checked and swapped on its own (see swapToNative): a coin that fails stays in the collector without
// discarding the other coins' swaps or the cursor move.
func (k Keeper) convertFees(ctx sdk.Context, fromModule string) (toSend, nonSwapable sdk.Coins, err error) {
	moduleAddr := k.accountKeeper.GetModuleAddress(fromModule)
	allCoins := k.readCollectorPage(ctx, fromModule, moduleAddr)
	if allCoins.IsZero() {
		//nothing to burn at this moment
		return nil, nil, nil
	}

	params := k.GetParams(ctx)
	maxIterations := int(params.MaxBalanceIterations)

	for i, c := range allCoins {
		if i >= maxIterations {
			if toSend.IsZero() && nonSwapable.IsZero() {
				k.Logger().Warn("max iterations reached without finding any coins to swap or non-swapable coins.")
			}
			break
		}

		if k.tradeKeeper.IsNativeDenom(ctx, c.Denom) {
			toSend = toSend.Add(c)
			continue
		}

		if !c.IsPositive() {
			continue
		}

		swapped, swappable, swapErr := k.swapToNative(ctx, fromModule, c)
		if swapErr != nil {
			//the coin stays in the collector, the rest of the page is still handled and the cursor still moves on
			k.Logger().Error("failed to swap collected fee coin, keeping it", "coin", c.String(), "error", swapErr)
			continue
		}

		if swappable {
			toSend = toSend.Add(swapped)
		} else if k.tradeKeeper.HasDeepLiquidityWithNativeDenom(ctx, c.Denom) {
			//if you can't swap, but there is deep liquidity, it means the amount is too low to be swapped at the moment,
			//so we ignore it.
			//we let the coins live here until the amount is big enough to be swapped.
			//we do NOT return them as nonSwapable, because they are swapable, but we need a bigger amount to swap them.
			continue
		} else {
			//coins that don't have deep liquidity should not reach this point
			//if they do, they should be sent to burner module, to avoid iterating through them every time this function
			//is called (usually at EndBlock)
			nonSwapable = nonSwapable.Add(c)
		}
	}

	if !toSend.IsAllPositive() {
		return nil, nonSwapable, nil
	}

	return toSend, nonSwapable, nil
}

// swapToNative checks and swaps one collected coin to native denom in its own cached context, so a failure (an error
// or a panic, e.g. an arithmetic overflow on an oversized amount) discards only this coin's writes and leaves the coin
// in the collector. swappable is false when the coin can't be swapped right now; nothing is written then.
func (k Keeper) swapToNative(ctx sdk.Context, fromModule string, c sdk.Coin) (swapped sdk.Coin, swappable bool, err error) {
	var out sdk.Coin
	var ok bool
	err = bzeutils.ApplyFuncIfNoError(ctx, func(cached sdk.Context) error {
		if !k.tradeKeeper.CanSwapForNativeDenom(cached, c) {
			return nil
		}

		res, swapErr := k.tradeKeeper.ModuleSwapForNativeDenom(cached, fromModule, sdk.NewCoins(c))
		if swapErr != nil {
			return swapErr
		}

		out, ok = res, true

		return nil
	})
	if err != nil {
		return sdk.Coin{}, false, err
	}

	return out, ok, nil
}
