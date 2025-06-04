package keeper

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var minNativeAmountForSwap = math.NewInt(50_000_000_000)

func (k Keeper) getNativeDenom(ctx sdk.Context) string {
	return k.GetParams(ctx).NativeDenom
}

func (k Keeper) IsNativeDenom(ctx sdk.Context, denom string) bool {
	return k.getNativeDenom(ctx) == denom
}

func (k Keeper) CanSwapForNativeDenom(ctx sdk.Context, denom string) bool {
	nativeDenom := k.getNativeDenom(ctx)
	if nativeDenom == denom {
		return false
	}

	_, _, poolId := k.CreatePoolId(nativeDenom, denom)
	pool, exists := k.GetLiquidityPool(ctx, poolId)
	if !exists {
		return false
	}

	nativeLpCoins, _ := pool.GetReservesCoinsByDenom(nativeDenom)
	if !nativeLpCoins.IsPositive() {
		return false
	}

	return nativeLpCoins.Amount.GT(minNativeAmountForSwap)
}

func (k Keeper) ModuleSwapForNativeDenom(ctx sdk.Context, toModule string, coins sdk.Coins) (sdk.Coin, error) {
	nativeDenom := k.getNativeDenom(ctx)
	if nativeDenom == "" {
		return sdk.Coin{}, fmt.Errorf("native denom not set")
	}

	toModuleAcc := k.accountKeeper.GetModuleAccount(ctx, toModule)
	swapResult := sdk.NewInt64Coin(nativeDenom, 0)
	for _, coin := range coins {
		if nativeDenom == coin.Denom {
			return swapResult, fmt.Errorf("cannot swap native coin to native coin")
		}

		_, _, poolId := k.CreatePoolId(nativeDenom, coin.Denom)
		pool, exists := k.GetLiquidityPool(ctx, poolId)
		if !exists {
			return swapResult, fmt.Errorf("cannot find liquidity pool between native denom %s and provided denom %s", nativeDenom, coin.Denom)
		}

		nativeLpCoins, _ := pool.GetReservesCoinsByDenom(nativeDenom)
		if !nativeLpCoins.IsPositive() || !nativeLpCoins.Amount.GT(minNativeAmountForSwap) {
			return swapResult, fmt.Errorf("not enough liquidity available to swap coin %s to native coin", coin.Denom)
		}

		sr, err := k.swapTokens(ctx, coin, &pool)
		if err != nil {
			return swapResult, err
		}

		swapResult = swapResult.Add(sr)

		err = ctx.EventManager().EmitTypedEvent(
			&types.SwapEvent{
				Creator: toModuleAcc.GetAddress().String(),
				PoolId:  poolId,
				In:      coin,
				Out:     sr,
			},
		)

		if err != nil {
			k.Logger().Error(err.Error())
		}
	}

	//capture swapped coins from calling module
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, toModule, types.ModuleName, coins)
	if err != nil {
		return swapResult, err
	}

	//send swap resulting coins to the calling module
	err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, toModule, sdk.NewCoins(swapResult))
	if err != nil {
		return swapResult, err
	}

	return swapResult, nil
}
