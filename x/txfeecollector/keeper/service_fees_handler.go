package keeper

import (
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (k Keeper) ConvertCollectedFeesToNativeDenom(ctx sdk.Context) error {
	return k.convertFeesAndSend(ctx, types.ModuleName, authtypes.FeeCollectorName, false)
}

func (k Keeper) ConvertBurnerFeesToNativeDenom(ctx sdk.Context) error {
	return k.convertFeesAndSend(ctx, types.BurnerFeeCollector, burnermoduletypes.ModuleName, true)
}

func (k Keeper) ConvertCommunityPoolFeesToNativeDenom(ctx sdk.Context) error {
	toSend, skipped, err := k.convertFees(ctx, types.CpFeeCollector)
	if err != nil {
		return err
	}

	//TODO: non native coins should be sent to the token's community pool (NOT BZE Community Pool) when they are available.
	// For now we just send them to burner module if they can't be swapped to native denom.
	if skipped.Len() > 0 {
		//if some coins were skipped because we couldn't convert them, send them to the buner module
		err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.CpFeeCollector, burnermoduletypes.ModuleName, skipped)
		if err != nil {
			return err
		}
	}

	if toSend == nil || toSend.IsZero() {
		return nil
	}

	moduleAddr := k.accountKeeper.GetModuleAddress(types.CpFeeCollector)
	err = k.distrKeeper.FundCommunityPool(ctx, toSend, moduleAddr)

	return err
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
func (k Keeper) convertFees(ctx sdk.Context, fromModule string) (toSend, nonSwapable sdk.Coins, err error) {
	moduleAddr := k.accountKeeper.GetModuleAddress(fromModule)
	allCoins := k.bankKeeper.GetAllBalances(ctx, moduleAddr)
	if allCoins.IsZero() {
		//nothing to burn at this moment
		return nil, nil, nil
	}

	//group swappable coins to one collection
	toSwap := sdk.NewCoins()
	for _, c := range allCoins {
		if k.tradeKeeper.IsNativeDenom(ctx, c.Denom) {
			toSend = toSend.Add(c)
			continue
		}

		if !c.IsPositive() {
			continue
		}

		if k.tradeKeeper.CanSwapForNativeDenom(ctx, c) {
			//swap if you can
			toSwap = toSwap.Add(c)
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

	if !toSwap.IsZero() {
		swapped, err := k.tradeKeeper.ModuleSwapForNativeDenom(ctx, fromModule, toSwap)
		if err != nil {
			return nil, nil, err
		}

		toSend = toSend.Add(swapped)
	}

	if !toSend.IsAllPositive() {
		return nil, nonSwapable, nil
	}

	return toSend, nonSwapable, nil
}
