package keeper

import (
	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) burnModuleCoins(ctx sdk.Context) error {
	moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
	allCoins := k.bankKeeper.GetAllBalances(ctx, moduleAcc.GetAddress())
	if allCoins.IsZero() {
		//nothing to burn at this moment
		return nil
	}

	err := k.BurnAnyCoins(ctx, types.ModuleName, allCoins)
	if err != nil {
		return err
	}

	err = k.SaveBurnedCoins(ctx, allCoins)
	if err != nil {
		ctx.Logger().Error("error saving burned coins", "error", err)
	}

	err = ctx.EventManager().EmitTypedEvent(&types.CoinsBurnedEvent{Burned: allCoins.String()})
	if err != nil {
		return err
	}

	k.Logger().With("coins", allCoins.String()).Info("coins successfully burned")

	return nil
}

func (k Keeper) BurnAnyCoins(ctx sdk.Context, fromModule string, coins sdk.Coins) error {
	//holds coins that can be burned from bank module
	var burnable sdk.Coins
	//holds coins that can not be burned (IBC coins) but can be exchanged to native coin and burned
	var exchangeable sdk.Coins
	//holds coins that should not be burned because the total supply in bank module should not be modified
	var lockable sdk.Coins
	for _, c := range coins {
		if !c.IsPositive() {
			continue
		}

		// native coins can be burned directly
		// factory tokens can be burned directly
		if k.tradeKeeper.IsNativeDenom(ctx, c.Denom) || bzeutils.IsTokenFactoryDenom(c.Denom) {
			burnable = burnable.Add(c)
			continue
		}

		//LP shares cannot be burned, but they can be locked
		if bzeutils.IsLpTokenDenom(c.Denom) {
			lockable = lockable.Add(c)
			continue
		}

		//not native, not LP share, not token factory -> it should be an IBC denom
		if k.tradeKeeper.CanSwapForNativeDenom(ctx, c.Denom) {
			exchangeable = exchangeable.Add(c)
			continue
		}

		//this should never be reached, but in case it does, let's keep these coins locked
		lockable = lockable.Add(c)
	}

	if len(exchangeable) > 0 {
		//swap coins to native and add them to burn
		swapped, err := k.tradeKeeper.ModuleSwapForNativeDenom(ctx, fromModule, coins)
		if err != nil {
			k.Logger().Error("error on swapping coins to burn", "error", err)
		} else {
			burnable = burnable.Add(swapped)
		}
	}

	//lock coins in black hole address
	if len(lockable) > 0 {
		err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromModule, types.BlackHoleModuleName, lockable)
		if err != nil {
			return err
		}
	}

	if burnable.IsAllPositive() {
		//burn coins eligible to burn
		err := k.bankKeeper.BurnCoins(ctx, fromModule, burnable)
		if err != nil {
			return err
		}
	}

	return nil
}
