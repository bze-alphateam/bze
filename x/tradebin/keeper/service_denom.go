package keeper

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

var minNativeAmountForSwap = math.NewInt(50_000_000_000)

func (k Keeper) getNativeDenom(ctx sdk.Context) string {
	return k.GetParams(ctx).NativeDenom
}

func (k Keeper) IsNativeDenom(ctx sdk.Context, denom string) bool {
	return k.getNativeDenom(ctx) == denom
}

func (k Keeper) HasLiquidityWithNativeDenom(ctx sdk.Context, denom string) bool {
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

	return true
}

func (k Keeper) CanSwapForNativeDenom(ctx sdk.Context, coin sdk.Coin) bool {
	nativeDenom := k.getNativeDenom(ctx)
	if nativeDenom == coin.Denom {
		return false
	}

	_, _, poolId := k.CreatePoolId(nativeDenom, coin.Denom)
	pool, exists := k.GetLiquidityPool(ctx, poolId)
	if !exists {
		return false
	}

	nativeLpCoins, _ := pool.GetReservesCoinsByDenom(nativeDenom)
	if !nativeLpCoins.IsPositive() {
		return false
	}

	if nativeLpCoins.Amount.LT(minNativeAmountForSwap) {
		return false
	}

	//check if the amount is too low
	_, fee := k.calculateSwapInputAndFee(coin, &pool)
	if pool.Fee.IsPositive() && !fee.IsPositive() {
		return false
	}

	return true
}

// ModuleSwapForNativeDenom swaps a specified set of coins from a module account to the native denomination if liquidity exists.
// It ensures the proper liquidity pool is available and updates account balances accordingly.
// Returns the resulting native denomination coin and an error if the process is unsuccessful.
func (k Keeper) ModuleSwapForNativeDenom(ctx sdk.Context, toModule string, coins sdk.Coins) (sdk.Coin, error) {
	cached, flush := ctx.CacheContext()
	nativeDenom := k.getNativeDenom(cached)
	if nativeDenom == "" {
		return sdk.Coin{}, fmt.Errorf("native denom not set")
	}

	toModuleAcc := k.accountKeeper.GetModuleAccount(cached, toModule)
	swapResult := sdk.NewInt64Coin(nativeDenom, 0)
	var events []proto.Message
	for _, coin := range coins {
		if nativeDenom == coin.Denom {
			return swapResult, fmt.Errorf("cannot swap native coin to native coin")
		}

		_, _, poolId := k.CreatePoolId(nativeDenom, coin.Denom)
		pool, exists := k.GetLiquidityPool(cached, poolId)
		if !exists {
			return swapResult, fmt.Errorf("cannot find liquidity pool between native denom %s and provided denom %s", nativeDenom, coin.Denom)
		}

		nativeLpCoins, _ := pool.GetReservesCoinsByDenom(nativeDenom)
		if !nativeLpCoins.IsPositive() || !nativeLpCoins.Amount.GT(minNativeAmountForSwap) {
			return swapResult, fmt.Errorf("not enough liquidity available to swap coin %s to native coin", coin.Denom)
		}

		sr, err := k.swapTokens(cached, coin, &pool)
		if err != nil {
			return swapResult, err
		}

		swapResult = swapResult.Add(sr)
		events = append(events, &types.SwapEvent{
			Creator: toModuleAcc.GetAddress().String(),
			PoolId:  poolId,
			In:      coin,
			Out:     sr,
		})
	}

	//capture swapped coins from calling module
	err := k.bankKeeper.SendCoinsFromModuleToModule(cached, toModule, types.ModuleName, coins)
	if err != nil {
		return swapResult, err
	}

	//send swap resulting coins to the calling module
	err = k.bankKeeper.SendCoinsFromModuleToModule(cached, types.ModuleName, toModule, sdk.NewCoins(swapResult))
	if err != nil {
		return swapResult, err
	}

	flush()

	err = ctx.EventManager().EmitTypedEvents(events...)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return swapResult, nil
}
