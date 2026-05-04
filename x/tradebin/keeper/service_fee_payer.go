package keeper

import (
	"fmt"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/bze-alphateam/bze/x/txfeecollector/ante"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CtxFeeDenomKey mirrors the txfeecollector ante.FeeDenomKey constant.
// Kept as a local constant to avoid a cross-module import dependency.
const CtxFeeDenomKey = ante.FeeDenomKey

// CaptureAndSwapUserFee captures a fee from a user, optionally swapping it to the native denomination if needed.
// It utilizes liquidity pools to swap the fees to the preferred denomination if sufficient liquidity exists.
// If the swap cannot be performed due to insufficient liquidity or payer balance, the native denomination is used directly.
func (k Keeper) CaptureAndSwapUserFee(ctx sdk.Context, payer sdk.AccAddress, fee sdk.Coins, toModule string) (coins sdk.Coins, err error) {
	if !fee.IsAllPositive() {
		return nil, fmt.Errorf("can not capture user fees that are not all positive")
	}

	nativeDenom := k.getNativeDenom(ctx)
	ctxDenom := k.getCtxDenom(ctx)
	k.Logger().Debug("capturing user fee", "payer", payer.String(), "fee", fee.String(), "preferred_denom", ctxDenom)

	if ctxDenom == "" || nativeDenom == ctxDenom {
		k.Logger().Debug("no swap needed, using native denom", "denom", nativeDenom)
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	//if the user prefers another fee denom instead of the native fee, we try to find out what amount he has to pay in
	//his preferred fee denom. We try to capture this amount in his preferred fee denom and then swap it to the native
	//denom. if he doesn't have enough balance we try to capture the native denom
	ok, nativeFee := fee.Find(nativeDenom)
	if !ok || nativeFee.IsZero() {
		//if the fee he has to pay has no native denom we can just capture the fee that it was provided to this function
		k.Logger().Debug("no native fee found in fee, capturing as-is")
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	_, _, poolId := k.CreatePoolId(nativeDenom, ctxDenom)
	pool, ok := k.GetLiquidityPool(ctx, poolId)
	if !ok {
		//the provided ctx denom does not have a pool with native denom. (should never happen)
		k.Logger().Debug("liquidity pool not found, falling back to native denom", "pool_id", poolId)
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	nativeLpCoins, _ := pool.GetReservesCoinsByDenom(nativeDenom)
	if !nativeLpCoins.IsPositive() {
		k.Logger().Debug("no liquidity available, falling back to native denom", "pool", poolId)
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	params := k.GetParams(ctx)
	// check the liquidity available still meets the criteria of minimum liquidity.
	// At this point the check was already done by tx ante handlers, so we allow the liquidity to be at least 75% (appx)
	// of the liquidity required. This decision assumes that params.MinNativeLiquidityForModuleSwap will always be a
	// big amount (changed via gov proposals, and it should be {BIGGEST_BZE_TAX} * 2 at least).
	// we decide to allow going below the threshold to ensure smooth experience for in-flight transactions that already
	// passed the ante-handler.
	if nativeLpCoins.Amount.LT(params.MinNativeLiquidityForModuleSwap.MulRaw(3).QuoRaw(4)) {
		k.Logger().Debug("insufficient liquidity available, falling back to native denom", "pool", poolId)
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	//for the required native coin, we calculate the amount the user needs to pay in his preferred fee denom
	requiredFeeCoins, err := k.CalculateOptimalInputForOutput(pool, nativeFee)
	if err != nil {
		k.Logger().Debug("failed to calculate swap amount, falling back to native denom", "native_fee", nativeFee.String())
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	k.Logger().Debug("calculated required fee in preferred denom", "required", requiredFeeCoins.String(), "native_fee", nativeFee.String())

	payerBalances := k.bankKeeper.SpendableCoins(ctx, payer)
	toCapture := fee.Sub(nativeFee).Add(requiredFeeCoins)
	if payerBalances.IsAllGTE(toCapture) {
		//if the user has enough balance to pay the required fee in his preferred fee denom, we just remove the native
		//fee and add the required fee.
		//capture the amount and swap the fee to the native denom

		//capture the entire amount in tradebin so we can perform the swap
		_, err = k.payerCoinsToModule(ctx, payer, toCapture, types.ModuleName)
		if err != nil {
			return nil, err
		}

		swapOutput, err := k.swapTokens(ctx, requiredFeeCoins, &pool)
		if err != nil {
			return nil, err
		}

		k.Logger().Info("swapped user fee to native denom", "input", requiredFeeCoins.String(), "output", swapOutput.String(), "payer", payer.String())

		//we subtract the required fee that we swapped from the entire capture amount and add the swap output
		capturedAndSwapped := toCapture.Sub(requiredFeeCoins).Add(swapOutput)

		//send swapped coins back to the caller module if the caller is not tradebin module
		if toModule != types.ModuleName {
			err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, toModule, capturedAndSwapped)
			if err != nil {
				return nil, err
			}
		}

		//we return to the caller the fee that was provided to this function plus the fee that was swapped to native
		//denom from preferred denom
		return capturedAndSwapped, nil
	}

	//the user can not pay the required fee in his preferred fee denom, so we just try to capture the fee in native denom
	k.Logger().Debug("insufficient balance for preferred denom, falling back to native", "required", toCapture.String())

	return k.payerCoinsToModule(ctx, payer, fee, toModule)
}

func (k Keeper) payerCoinsToModule(ctx sdk.Context, payer sdk.AccAddress, coins sdk.Coins, toModule string) (sdk.Coins, error) {
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, payer, toModule, coins)
	if err != nil {
		return nil, err
	}

	return coins, nil
}

func (k Keeper) getCtxDenom(ctx sdk.Context) string {
	ctxDenom := ctx.Value(CtxFeeDenomKey)
	if ctxDenom == nil {
		return ""
	}

	ctxDenomStr, ok := ctxDenom.(string)
	if !ok {
		return ""
	}

	return ctxDenomStr
}

// CaptureAndTryToSwapUserFeesOrSendItAsIs attempts to capture user fees and swap them to the native denom if a preferred denom is specified.
// Fallbacks include using provided fees as-is or defaulting to the native denom based on user balance and liquidity conditions.
// ctx: The current transaction context.
// payer: The address of the user paying fees.
// fee: The coins representing the user fee to be processed.
// toModule: The target module where fees should be transferred.
// Returns captured coins and an error if any part of the process fails.
// Note: this is an exact copy of CaptureAndSwapUserFee with a small change:
// - CaptureAndSwapUserFee - fails if the fee is paid in other denom than the native denom when swap fails.
// - CaptureAndTryToSwapUserFeesOrSendItAsIs - it does NOT fail when the user pays in a different denom than native if
// the failure reason is amount too small to be traded.
// Example: we have small trading fees of 0.1 BZE for market taker and 0.001BZE for market maker. If the user pays fees
// USD, the amount in USD we're supposed to capture for 0.001 BZE is like 0.000050 USD. Which is too low to be traded
// due to swap fees. The resulted swap fee would be lower than 1uusd (smallest unit), but the fee is valid. So the
// blockchain decides to accept the 0.00005 USD as-is. The caller is responsible for routing the captured coins to the
// appropriate fee collector (e.g. BurnerFeeCollector or CpFeeCollector) where txfeecollector's EndBlock accumulates
// small amounts and swaps them once they reach a swappable amount.
func (k Keeper) CaptureAndTryToSwapUserFeesOrSendItAsIs(ctx sdk.Context, payer sdk.AccAddress, fee sdk.Coins, toModule string) (coins sdk.Coins, err error) {
	if !fee.IsAllPositive() {
		return nil, fmt.Errorf("can not capture user fees that are not all positive")
	}

	nativeDenom := k.getNativeDenom(ctx)
	ctxDenom := k.getCtxDenom(ctx)
	k.Logger().Debug("capturing user fee", "payer", payer.String(), "fee", fee.String(), "preferred_denom", ctxDenom)

	if ctxDenom == "" || nativeDenom == ctxDenom {
		k.Logger().Debug("no swap needed, using native denom", "denom", nativeDenom)
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	//if the user prefers another fee denom instead of the native fee, we try to find out what amount he has to pay in
	//his preferred fee denom. We try to capture this amount in his preferred fee denom and then swap it to the native
	//denom. if he doesn't have enough balance we try to capture the native denom
	ok, nativeFee := fee.Find(nativeDenom)
	if !ok || nativeFee.IsZero() {
		//if the fee he has to pay has no native denom we can just capture the fee that it was provided to this function
		k.Logger().Debug("no native fee found in fee, capturing as-is")
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	_, _, poolId := k.CreatePoolId(nativeDenom, ctxDenom)
	pool, ok := k.GetLiquidityPool(ctx, poolId)
	if !ok {
		//the provided ctx denom does not have a pool with native denom. (should never happen)
		k.Logger().Debug("liquidity pool not found, falling back to native denom", "pool_id", poolId)
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	nativeLpCoins, _ := pool.GetReservesCoinsByDenom(nativeDenom)
	if !nativeLpCoins.IsPositive() {
		k.Logger().Debug("no liquidity available, falling back to native denom", "pool", poolId)
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	params := k.GetParams(ctx)
	// check the liquidity available still meets the criteria of minimum liquidity.
	// At this point the check was already done by tx ante handlers, so we allow the liquidity to be at least 75% (appx)
	// of the liquidity required. This decision assumes that params.MinNativeLiquidityForModuleSwap will always be a
	// big amount (changed via gov proposals, and it should be {BIGGEST_BZE_TAX} * 2 at least).
	// we decide to allow going below the threshold to ensure smooth experience for in-flight transactions that already
	// passed the ante-handler.
	if nativeLpCoins.Amount.LT(params.MinNativeLiquidityForModuleSwap.MulRaw(3).QuoRaw(4)) {
		k.Logger().Debug("insufficient liquidity available, falling back to native denom", "pool", poolId)
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	//for the required native coin, we calculate the amount the user needs to pay in his preferred fee denom
	requiredFeeCoins, err := k.CalculateOptimalInputForOutput(pool, nativeFee)
	if err != nil {
		k.Logger().Debug("failed to calculate swap amount, falling back to native denom", "native_fee", nativeFee.String())
		return k.payerCoinsToModule(ctx, payer, fee, toModule)
	}

	k.Logger().Debug("calculated required fee in preferred denom", "required", requiredFeeCoins.String(), "native_fee", nativeFee.String())

	payerBalances := k.bankKeeper.SpendableCoins(ctx, payer)
	toCapture := fee.Sub(nativeFee).Add(requiredFeeCoins)
	if payerBalances.IsAllGTE(toCapture) {
		//first we have to check if the swap would fail due to amount too low.
		//we do this by checking if the swap process can capture the fee.
		_, swapFee := k.calculateSwapInputAndFee(requiredFeeCoins, &pool)
		if pool.Fee.IsPositive() && !swapFee.IsPositive() {
			//the swap fee is too low so we capture the fees and instead of native we capture the amount in
			//his preferred fee denom.
			return k.payerCoinsToModule(ctx, payer, toCapture, toModule)
		}

		//if the user has enough balance to pay the required fee in his preferred fee denom, we just remove the native
		//fee and add the required fee.
		//capture the amount and swap the fee to the native denom

		//capture the entire amount in tradebin so we can perform the swap
		_, err = k.payerCoinsToModule(ctx, payer, toCapture, types.ModuleName)
		if err != nil {
			return nil, err
		}

		swapOutput, err := k.swapTokens(ctx, requiredFeeCoins, &pool)
		if err != nil {
			return nil, err
		}

		k.Logger().Info("swapped user fee to native denom", "input", requiredFeeCoins.String(), "output", swapOutput.String(), "payer", payer.String())

		//we subtract the required fee that we swapped from the entire capture amount and add the swap output
		capturedAndSwapped := toCapture.Sub(requiredFeeCoins).Add(swapOutput)

		//send swapped coins back to the caller module if the caller is not tradebin module
		if toModule != types.ModuleName {
			err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, toModule, capturedAndSwapped)
			if err != nil {
				return nil, err
			}
		}

		//we return to the caller the fee that was provided to this function plus the fee that was swapped to native
		//denom from preferred denom
		return capturedAndSwapped, nil
	}

	//the user can not pay the required fee in his preferred fee denom, so we just try to capture the fee in native denom
	k.Logger().Debug("insufficient balance for preferred denom, falling back to native", "required", toCapture.String())

	return k.payerCoinsToModule(ctx, payer, fee, toModule)
}
