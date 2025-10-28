package keeper

import (
	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) burnModuleCoins(ctx sdk.Context) error {
	logger := k.Logger().With("method", "burnModuleCoins")
	logger.Debug("running burn module coins")

	moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
	allCoins := k.bankKeeper.GetAllBalances(ctx, moduleAcc.GetAddress())
	if allCoins.IsZero() {
		//nothing to burn at this moment
		return nil
	}

	burned, err := k.BurnAnyCoins(ctx, types.ModuleName, allCoins)
	if err != nil {
		return err
	}

	if burned.IsZero() {
		logger.Info("no module coins to burn")
		return nil
	}

	err = k.SaveBurnedCoins(ctx, burned)
	if err != nil {
		logger.Error("error saving burned coins", "error", err)
	}

	err = ctx.EventManager().EmitTypedEvent(&types.CoinsBurnedEvent{Burned: burned.String()})
	if err != nil {
		return err
	}

	logger.With("coins", burned.String()).Info("coins successfully burned")

	return nil
}

// BurnAnyCoins attempts to burn, lock, or exchange specified coins from a module account.
// It directly burns native and token factory denominations, locks LP tokens, and exchanges IBC tokens for native ones.
// It returns the coins that were successfully burned.
func (k Keeper) BurnAnyCoins(ctx sdk.Context, fromModule string, coins sdk.Coins) (sdk.Coins, error) {
	logger := k.Logger().With("method", "BurnAnyCoins", "fromModule", fromModule)
	logger.Debug("iterating through provided coins")

	//holds coins that can be burned from bank module
	var burnable sdk.Coins
	//holds coins that can not be burned (IBC coins) but can be exchanged to native coin and burned
	var exchangeable sdk.Coins
	//holds coins that should not be burned because the total supply in bank module should not be modified
	var lockable sdk.Coins
	for _, c := range coins {
		logger.Debug("checking coin", "coin", c.String())
		if !c.IsPositive() {
			continue
		}

		// native coins can be burned directly
		// factory tokens can be burned directly
		if k.tradeKeeper.IsNativeDenom(ctx, c.Denom) || bzeutils.IsTokenFactoryDenom(c.Denom) {
			logger.Debug("coin is native or factory token, can be burned directly")
			burnable = burnable.Add(c)
			continue
		}

		//LP shares cannot be burned, but they can be locked
		if bzeutils.IsLpTokenDenom(c.Denom) {
			logger.Debug("coin is LP token, can be locked")
			lockable = lockable.Add(c)
			continue
		}
		//it must be an IBC token

		//not native, not LP share, not token factory -> it should be an IBC denom
		if k.tradeKeeper.CanSwapForNativeDenom(ctx, c) {
			logger.Debug("coin is IBC token, can be exchanged to native coin for burning")
			exchangeable = exchangeable.Add(c)
			continue
		}

		if k.tradeKeeper.HasLiquidityWithNativeDenom(ctx, c.Denom) {
			logger.Debug("coin is IBC token, it has liquidity with native denom, CAN NOT be exchanged to native coin for burning. Will be burned in next run.")
			//if the coin has liquidity with native denom, but it cannot be swapped yet (previous if statement checks this)
			//we let the coins be burned in the next run, hoping that the liquidity will be added by then soon
			continue
		}

		logger.Debug("coin is IBC token, cannot be burned or exchanged to native coin. Will be locked forever")
		lockable = lockable.Add(c)
	}

	if len(exchangeable) > 0 {
		logger.Debug("there are coins that can be exchanged to native coin for burning")
		//swap exchangeable coins to native and add them to burn
		swapped, err := k.tradeKeeper.ModuleSwapForNativeDenom(ctx, fromModule, exchangeable)
		if err != nil {
			logger.Error("error on swapping coins to burn", "error", err)
		} else {
			logger.Debug("swapped coins to burn", "swapped", swapped.String())
			burnable = burnable.Add(swapped)
		}
	}

	//lock coins in black hole address
	if len(lockable) > 0 {
		err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromModule, types.BlackHoleModuleName, lockable)
		if err != nil {
			return nil, err
		}
		logger.Debug("coins locked", "locked", lockable.String())
	}

	if burnable.IsAllPositive() {
		//burn coins eligible to burn
		err := k.bankKeeper.BurnCoins(ctx, fromModule, burnable)
		if err != nil {
			return nil, err
		}
		logger.Debug("coins burned", "burned", burnable.String())
	}

	return burnable, nil
}
