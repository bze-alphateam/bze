package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

func (k Keeper) getNativeDenom(ctx sdk.Context) string {
	return k.GetParams(ctx).NativeDenom
}

func (k Keeper) IsNativeDenom(ctx sdk.Context, denom string) bool {
	return k.getNativeDenom(ctx) == denom
}

func (k Keeper) getDenomsLp(ctx sdk.Context, denom1, denom2 string) (lp types.LiquidityPool, exists bool) {
	_, _, poolId := k.CreatePoolId(denom1, denom2)

	return k.GetLiquidityPool(ctx, poolId)
}

// HasLiquidityWithNativeDenom checks if the provided denom has a liquidity pool with the native denom.
func (k Keeper) HasLiquidityWithNativeDenom(ctx sdk.Context, denom string) bool {
	nativeDenom := k.getNativeDenom(ctx)
	if nativeDenom == denom {
		return false
	}

	pool, exists := k.getDenomsLp(ctx, nativeDenom, denom)
	if !exists {
		return false
	}

	nativeLpCoins, _ := pool.GetReservesCoinsByDenom(nativeDenom)
	return nativeLpCoins.IsPositive()
}

// HasDeepLiquidityWithNativeDenom checks if the specified denom has sufficient liquidity when paired with the native denom.
// This function is the same as CanSwapForNativeDenom - except that it doesn't check the amount can be swapped.
// This function is useful when we DO NOT want to also check that the amount can be swapped.
func (k Keeper) HasDeepLiquidityWithNativeDenom(ctx sdk.Context, denom string) bool {
	nativeDenom := k.getNativeDenom(ctx)
	if nativeDenom == denom {
		return false
	}

	pool, exists := k.getDenomsLp(ctx, nativeDenom, denom)
	if !exists {
		return false
	}

	nativeLpCoins, _ := pool.GetReservesCoinsByDenom(nativeDenom)
	if !nativeLpCoins.IsPositive() {
		return false
	}

	params := k.GetParams(ctx)
	return !nativeLpCoins.Amount.LT(params.MinNativeLiquidityForModuleSwap)
}

func (k Keeper) GetDenomSpotPriceInNativeCoin(ctx sdk.Context, denom string) (sdk.DecCoin, error) {
	nativeDenom := k.getNativeDenom(ctx)

	// Return 1:1 for native denom or empty denom
	if denom == "" || nativeDenom == denom || nativeDenom == "" {
		return sdk.NewDecCoinFromDec(nativeDenom, math.LegacyOneDec()), nil
	}

	pool, exists := k.getDenomsLp(ctx, nativeDenom, denom)
	if !exists {
		return sdk.DecCoin{}, fmt.Errorf("no liquidity pool exists for %s/%s", denom, nativeDenom)
	}

	nativeCoin, otherCoin := pool.GetReservesCoinsByDenom(nativeDenom)
	if !nativeCoin.IsPositive() || !otherCoin.IsPositive() {
		return sdk.DecCoin{}, fmt.Errorf("pool has insufficient reserves")
	}

	// Spot price: how many native coins per 1 unit of denom
	// = native_reserve / other_reserve
	spotPrice := math.LegacyNewDecFromInt(nativeCoin.Amount).Quo(math.LegacyNewDecFromInt(otherCoin.Amount))

	return sdk.NewDecCoinFromDec(nativeDenom, spotPrice), nil
}

// CanSwapForNativeDenom determines if a given coin can be swapped for the native denomination in an existing liquidity pool.
func (k Keeper) CanSwapForNativeDenom(ctx sdk.Context, coin sdk.Coin) bool {
	nativeDenom := k.getNativeDenom(ctx)
	if nativeDenom == coin.Denom {
		return false
	}

	pool, exists := k.getDenomsLp(ctx, nativeDenom, coin.Denom)
	if !exists {
		return false
	}

	nativeLpCoins, _ := pool.GetReservesCoinsByDenom(nativeDenom)
	if !nativeLpCoins.IsPositive() {
		return false
	}

	params := k.GetParams(ctx)
	if nativeLpCoins.Amount.LT(params.MinNativeLiquidityForModuleSwap) {
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
// It fails if ANY swap of the provided coins fails.
func (k Keeper) ModuleSwapForNativeDenom(ctx sdk.Context, toModule string, coins sdk.Coins) (sdk.Coin, error) {
	cached, flush := ctx.CacheContext()
	nativeDenom := k.getNativeDenom(cached)
	if nativeDenom == "" {
		return sdk.Coin{}, fmt.Errorf("native denom not set")
	}

	//capture swapped coins from calling module
	err := k.bankKeeper.SendCoinsFromModuleToModule(cached, toModule, types.ModuleName, coins)
	if err != nil {
		return sdk.Coin{}, err
	}

	params := k.GetParams(cached)
	toModuleAcc := k.accountKeeper.GetModuleAccount(cached, toModule)
	swapResult := sdk.NewInt64Coin(nativeDenom, 0)
	var events []proto.Message
	for _, coin := range coins {
		if nativeDenom == coin.Denom {
			return sdk.Coin{}, fmt.Errorf("cannot swap native coin to native coin")
		}

		_, _, poolId := k.CreatePoolId(nativeDenom, coin.Denom)
		pool, exists := k.GetLiquidityPool(cached, poolId)
		if !exists {
			return sdk.Coin{}, fmt.Errorf("cannot find liquidity pool between native denom %s and provided denom %s", nativeDenom, coin.Denom)
		}

		nativeLpCoins, _ := pool.GetReservesCoinsByDenom(nativeDenom)
		if !nativeLpCoins.IsPositive() || !nativeLpCoins.Amount.GT(params.MinNativeLiquidityForModuleSwap) {
			return sdk.Coin{}, fmt.Errorf("not enough liquidity available to swap coin %s to native coin", coin.Denom)
		}

		sr, err := k.swapTokens(cached, coin, &pool)
		if err != nil {
			return sdk.Coin{}, err
		}

		swapResult = swapResult.Add(sr)
		events = append(events, &types.SwapEvent{
			Creator: toModuleAcc.GetAddress().String(),
			PoolId:  poolId,
			In:      coin,
			Out:     sr,
		})
	}

	//send swap resulting coins to the calling module
	err = k.bankKeeper.SendCoinsFromModuleToModule(cached, types.ModuleName, toModule, sdk.NewCoins(swapResult))
	if err != nil {
		return sdk.Coin{}, err
	}

	flush()

	err = ctx.EventManager().EmitTypedEvents(events...)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return swapResult, nil
}

// ModuleAddLiquidityWithNativeDenom swaps the optimal amount of provided coins to native denom and adds liquidity to the respective pools.
// The LP tokens are sent to the caller module.
// Any leftover coins after balancing are swapped to native denom and returned to the caller module as native.
// This ensures no value is lost and the caller receives a single denomination (native) for any unused amounts.
//
// The function processes coins individually and does not fail the entire operation if one coin encounters an issue.
// Coins that cannot be processed are refunded to the caller.
//
// Process per coin:
//  1. Validate pool exists and has liquidity
//     - If validation fails: add coin to refunds, continue to next coin
//  2. Calculate optimal swap amount to get native denom
//  3. Execute swap
//  4. Balance amounts to match pool ratio
//  5. Add liquidity and mint LP tokens
//  6. Swap any leftover input coin to native
//     - If swap succeeds: add to native leftover for return
//     - If swap fails (amount too small): donate leftover to pool reserves (benefits LP holders)
//
// Returns:
//   - addedCoins: coins successfully added as liquidity (non-native denominations)
//   - refundedCoins: coins that were refunded to caller (includes native leftovers)
//   - error: only for critical failures (native denom not set, initial transfer fails)
func (k Keeper) ModuleAddLiquidityWithNativeDenom(ctx sdk.Context, fromModule string, coins sdk.Coins) (addedCoins sdk.Coins, refundedCoins sdk.Coins, err error) {
	cached, flush := ctx.CacheContext()
	nativeDenom := k.getNativeDenom(cached)
	if nativeDenom == "" {
		return nil, nil, fmt.Errorf("cannot add liquidity with native denom: native denom not set")
	}

	//capture coins from calling module
	err = k.bankKeeper.SendCoinsFromModuleToModule(cached, fromModule, types.ModuleName, coins)
	if err != nil {
		return nil, nil, err
	}

	var events []proto.Message
	var totalNativeRefund math.Int = math.ZeroInt()

	//swap optimal amount of coins to native denom and add them as LP to the pool
	for _, coin := range coins {
		//validate coin is not native denom
		if coin.Denom == nativeDenom {
			k.Logger().Info("skipping native denom coin", "denom", coin.Denom)
			refundedCoins = refundedCoins.Add(coin)
			continue
		}

		_, _, poolId := k.CreatePoolId(nativeDenom, coin.Denom)
		pool, ok := k.GetLiquidityPool(cached, poolId)
		if !ok {
			k.Logger().Info("pool not found, refunding coin", "pool", poolId, "coin", coin)
			refundedCoins = refundedCoins.Add(coin)
			continue
		}

		poolBaseReserve := pool.ReserveBase
		poolQuoteReserve := pool.ReserveQuote
		if poolBaseReserve.IsZero() || poolQuoteReserve.IsZero() {
			k.Logger().Info("pool is empty, refunding coin", "pool", poolId, "coin", coin)
			refundedCoins = refundedCoins.Add(coin)
			continue
		}

		//compute the optimal amount to swap to native denom to add liquidity
		optimalSwapAmount, err := k.CalculateOptimalSwapAmount(pool, coin)
		if err != nil {
			k.Logger().Info("failed to calculate optimal swap, refunding coin", "pool", poolId, "coin", coin, "error", err)
			refundedCoins = refundedCoins.Add(coin)
			continue
		}

		//swap the optimal amount to get native denom
		//Note: the coins are already in the module from the previous SendCoinsFromModuleToModule
		resultedNative, swapErr := k.swapTokens(cached, sdk.NewCoin(coin.Denom, optimalSwapAmount), &pool)
		if swapErr != nil {
			k.Logger().Info("swap failed, refunding coin", "pool", poolId, "coin", coin, "error", swapErr)
			refundedCoins = refundedCoins.Add(coin)
			continue
		}

		//calculate remaining coin amount after the swap
		remainingCoinAmount := coin.Amount.Sub(optimalSwapAmount)

		//refresh pool state after swap. Defensive: the pool is passed by reference to swapTokens, but we must refresh
		//it here to avoid future bugs
		pool, ok = k.GetLiquidityPool(cached, poolId)
		if !ok {
			return nil, nil, fmt.Errorf("cannot add liquidity with native denom %s: pool disappeared after swap", coin.Denom)
		}

		//determine which coin is base and which is quote
		var baseAmount, quoteAmount math.Int
		if pool.GetBase() == nativeDenom {
			baseAmount = resultedNative.Amount
			quoteAmount = remainingCoinAmount
		} else {
			baseAmount = remainingCoinAmount
			quoteAmount = resultedNative.Amount
		}

		//balance the amounts to match pool ratio
		optimalBase, optimalQuote, err := k.BalanceProvidedAmounts(baseAmount, quoteAmount, pool.ReserveBase, pool.ReserveQuote)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot add liquidity with native denom: failed to balance amounts: %w", err)
		}

		//validate the optimal amounts are valid coins
		_, _, err = k.getProvidedReserves(pool.GetBase(), pool.GetQuote(), optimalBase, optimalQuote)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot add liquidity with native denom: failed to validate reserves: %w", err)
		}

		//store current reserves before minting
		currentBaseReserve := pool.ReserveBase
		currentQuoteReserve := pool.ReserveQuote

		//mint LP tokens
		minted, err := k.mintDepositLpTokens(cached, &optimalBase, &optimalQuote, &currentBaseReserve, &currentQuoteReserve, &pool)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot add liquidity with native denom: failed to mint LP tokens: %w", err)
		}
		refundedCoins = refundedCoins.Add(minted)

		//update pool reserves
		pool.ReserveBase = currentBaseReserve.Add(optimalBase)
		pool.ReserveQuote = currentQuoteReserve.Add(optimalQuote)

		k.SetLiquidityPool(cached, pool)

		//calculate leftover amounts
		var leftoverNative, leftoverCoin math.Int
		if pool.GetBase() == nativeDenom {
			leftoverNative = baseAmount.Sub(optimalBase)
			leftoverCoin = quoteAmount.Sub(optimalQuote)
		} else {
			leftoverNative = quoteAmount.Sub(optimalQuote)
			leftoverCoin = baseAmount.Sub(optimalBase)
		}

		//swap leftover coin to native and combine with leftover native
		totalNativeLeftover := leftoverNative
		if leftoverCoin.IsPositive() {
			//refresh pool state after liquidity add
			pool, ok = k.GetLiquidityPool(cached, poolId)
			if !ok {
				return nil, nil, fmt.Errorf("cannot add liquidity with native denom: pool disappeared after liquidity add")
			}

			//swap leftover coin to native
			leftoverCoinToSwap := sdk.NewCoin(coin.Denom, leftoverCoin)
			swappedNative, err := k.swapTokens(cached, leftoverCoinToSwap, &pool)
			if err != nil {
				//if swap fails (amount too small, etc), donate the leftover coin to the pool
				//this benefits LP holders and ensures no funds are lost
				k.Logger().Info("donating leftover coin to pool (swap failed)", "error", err, "amount", leftoverCoin, "pool", poolId)

				//add leftover coin directly to pool reserves (no LP minting)
				if pool.GetQuote() == coin.Denom {
					pool.ReserveQuote = pool.ReserveQuote.Add(leftoverCoin)
				} else {
					pool.ReserveBase = pool.ReserveBase.Add(leftoverCoin)
				}
				k.SetLiquidityPool(cached, pool)
			} else {
				totalNativeLeftover = totalNativeLeftover.Add(swappedNative.Amount)
			}
		}

		//accumulate native leftover for final refund
		totalNativeRefund = totalNativeRefund.Add(totalNativeLeftover)

		//track successfully added coin (non-native only)
		addedCoins = addedCoins.Add(sdk.NewCoin(coin.Denom, optimalQuote))

		//prepare liquidity added event
		events = append(events, &types.LiquidityAddedEvent{
			Creator:      fromModule,
			BaseAmount:   optimalBase,
			QuoteAmount:  optimalQuote,
			MintedAmount: minted.Amount,
			PoolId:       pool.GetId(),
		})
	}

	//send all refunds back to caller in a single call
	//combine native refund with other refunded coins
	if totalNativeRefund.IsPositive() {
		refundedCoins = refundedCoins.Add(sdk.NewCoin(nativeDenom, totalNativeRefund))
	}

	if len(refundedCoins) > 0 {
		err = k.bankKeeper.SendCoinsFromModuleToModule(cached, types.ModuleName, fromModule, refundedCoins)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to send refunds to caller: %w", err)
		}
	}

	//flush the cached context to persist all changes
	flush()

	//emit all events
	err = ctx.EventManager().EmitTypedEvents(events...)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return addedCoins, refundedCoins, nil
}
