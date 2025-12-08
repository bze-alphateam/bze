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

// convertFees converts all non-native module account fees into the native denomination and returns the total as a coin.
func (k Keeper) convertFees(ctx sdk.Context, fromModule string) (toSend, skipped sdk.Coins, err error) {
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
			toSwap = toSwap.Add(c)
		} else {
			skipped = skipped.Add(c)
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
		return nil, skipped, nil
	}

	return toSend, skipped, nil
}
