package keeper

import (
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	txFeeConverterHookName = "tx_fee_convert"

	txFeeEpoch = "hour"
)

func (k Keeper) GetTxFeeConverterHook() types.EpochHook {
	return types.NewAfterEpochHook(txFeeConverterHookName, func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
		if epochIdentifier != txFeeEpoch {
			return nil
		}

		k.Logger().
			With("epoch", epochIdentifier, "epoch_number", epochNumber, "hook_name", txFeeConverterHookName).
			Debug("preparing to execute hook")

		return k.convertCollectedFeesToNativeDenom(ctx)
	})
}

func (k Keeper) convertCollectedFeesToNativeDenom(ctx sdk.Context) error {
	moduleAddr := k.accountKeeper.GetModuleAddress(types.ModuleName)
	allCoins := k.bankKeeper.GetAllBalances(ctx, moduleAddr)
	if allCoins.IsZero() {
		//nothing to burn at this moment
		return nil
	}

	//group swappable coins to one collection
	toSwap := sdk.NewCoins()
	for _, c := range allCoins {
		if k.tradeKeeper.CanSwapForNativeDenom(ctx, c) {
			toSwap = toSwap.Add(c)
		}
	}

	//if they are not all positive we'll try next round
	if !toSwap.IsAllPositive() {
		return nil
	}

	swapped, err := k.tradeKeeper.ModuleSwapForNativeDenom(ctx, types.ModuleName, toSwap)
	if err != nil {
		return err
	}

	//if no swap happened we can try again next time
	//this should not happen as the swap would return an error if the swap result is 0
	if !swapped.IsPositive() {
		return nil
	}

	//send the swapped coins to SDK fee collector to distribute it to the delegators/validators
	err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, authtypes.FeeCollectorName, sdk.NewCoins(swapped))

	return err
}
