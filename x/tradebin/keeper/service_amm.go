package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	lpDenomPrefix = "lp"

	sharesScaleExponent = 6
)

// CreatePoolId - orders assets alphabetically and returns them in current order and their respective pool id
func (k Keeper) CreatePoolId(base, quote string) (newBase, newQuote, id string) {
	//sort assets alphabetically
	if base > quote {
		//reverse the order of assets
		base, quote = quote, base
	}

	return base, quote, k.getPoolId(base, quote)
}

// getPoolId - creates Pool ID from given assets
func (k Keeper) getPoolId(base, quote string) string {
	return fmt.Sprintf("%s_%s", base, quote)
}

func (k Keeper) getPoolDenom(poolId string) string {
	return fmt.Sprintf("u%s", k.getPoolScaledDenom(poolId))
}

func (k Keeper) getPoolScaledDenom(poolId string) string {
	return fmt.Sprintf("%s_%s", lpDenomPrefix, poolId)
}

// BalanceProvidedAmounts calculates optimal base and quote amounts maintaining pool reserve ratios.
// Returns the optimal base and quote amounts along with an error if the input values are invalid or the pool is empty.
func (k Keeper) BalanceProvidedAmounts(base, quote, reserveBase, reserveQuote math.Int) (math.Int, math.Int, error) {
	if base.IsNil() || quote.IsNil() {
		return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("can not balance with non positive base or quote")
	}

	if reserveBase.IsZero() || reserveQuote.IsZero() {
		//pools should not be empty, they are created with a desired price
		return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("pool is empty")
	}

	// Calculate how much would be needed for the provided amounts
	possibleQuote := base.Mul(reserveQuote).Quo(reserveBase)
	possibleBase := quote.Mul(reserveBase).Quo(reserveQuote)

	var optimalBase, optimalQuote math.Int
	// Use the lesser amounts to maintain the ratio
	if possibleQuote.LTE(quote) {
		optimalBase = base
		optimalQuote = possibleQuote
	} else {
		optimalBase = possibleBase
		optimalQuote = quote
	}

	return optimalBase, optimalQuote, nil
}

// swapTokens swaps `input` tokens for `output` tokens within the specified `pool` based on the pool's reserves and fees.
// Returns the `output` tokens and a potential error if the operation cannot be completed.
func (k Keeper) swapTokens(ctx sdk.Context, input sdk.Coin, pool *types.LiquidityPool) (output sdk.Coin, err error) {
	// MAKE SURE YOU CAPTURED THE FUNDS BEFORE CALLING THIS FUNCTION.
	// THE MODULE WILL SEND FEES TO THEIR DESTINATION AND IT NEEDS TO BE CAPTURED BEFORE THIS FUNCTION IS CALLED.
	if !pool.HasDenom(input.Denom) {
		return output, fmt.Errorf("denom %s does not exist in pool %s", input.Denom, pool.GetId())
	}

	realInput, fee := k.calculateSwapInputAndFee(input, pool)
	if pool.Fee.IsPositive() && !fee.IsPositive() {
		return output, fmt.Errorf("amount is too low to be traded")
	}

	feeToPool, err := k.collectSwapFee(ctx, fee, pool)
	if err != nil {
		return output, err
	}

	inputReserve, outputReserve := pool.GetReservesCoinsByDenom(input.Denom)

	//output_reserve x real_input (the input - fee)
	prod := math.LegacyNewDecFromInt(outputReserve.Amount.Mul(realInput.Amount))

	//input_reserve + real_input (the input - fee)
	quo := math.LegacyNewDecFromInt(inputReserve.Amount.Add(realInput.Amount))
	if !quo.IsPositive() || !prod.IsPositive() {
		return output, fmt.Errorf("non positive product or quotient on swap tokens")
	}

	outputAmount := prod.Quo(quo).TruncateInt()
	output = sdk.NewCoin(outputReserve.Denom, outputAmount)
	if !output.IsPositive() {
		return output, fmt.Errorf("non positive output on swap tokens")
	}

	//add the part of the fee that should remain in the LP (as LP Reward to LP providers)
	err = pool.ChangeReserves(realInput.Add(feeToPool), output)
	if err != nil {
		return output, err
	}

	k.SetLiquidityPool(ctx, *pool)

	return output, nil
}

// collectSwapFee - calculates the distribution of the fee, and it returns the part of the fee that should be added to
// LP (for LP Providers rewards)
func (k Keeper) collectSwapFee(ctx sdk.Context, fee sdk.Coin, pool *types.LiquidityPool) (sdk.Coin, error) {
	if !fee.IsPositive() {
		//return 0 coin
		return sdk.NewCoin(fee.Denom, math.ZeroInt()), nil
	}

	feeDec := math.LegacyNewDecFromInt(fee.Amount)
	treasury := sdk.NewCoin(fee.Denom, feeDec.Mul(pool.FeeDest.Treasury).TruncateInt())
	if treasury.IsPositive() {
		fee = fee.Sub(treasury)
		moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
		err := k.distrKeeper.FundCommunityPool(ctx, sdk.NewCoins(treasury), moduleAcc.GetAddress())
		if err != nil {
			return sdk.Coin{}, err
		}

		if !fee.IsPositive() {

			return sdk.NewCoin(fee.Denom, math.ZeroInt()), nil
		}
	}

	burner := sdk.NewCoin(fee.Denom, feeDec.Mul(pool.FeeDest.Burner).TruncateInt())
	if burner.IsPositive() {
		fee = fee.Sub(burner)
		err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, burnermoduletypes.ModuleName, sdk.NewCoins(burner))
		if err != nil {
			return sdk.Coin{}, err
		}
	}

	//just to make sure it's never negative (thinking that truncating the Dec might result in 0 or even negative value)
	if !fee.IsPositive() {
		//return 0 coin
		return sdk.NewCoin(fee.Denom, math.ZeroInt()), nil
	}

	return fee, nil
}

func (k Keeper) calculateSwapInputAndFee(input sdk.Coin, pool *types.LiquidityPool) (remainingInput, fee sdk.Coin) {
	feeAmount := math.LegacyNewDecFromInt(input.Amount).Mul(pool.Fee).TruncateInt()
	rAmount := input.Amount.Sub(feeAmount)

	return sdk.NewCoin(input.GetDenom(), rAmount), sdk.NewCoin(input.GetDenom(), feeAmount)
}

// CalculateOptimalSwapAmount calculates the optimal amount to swap from the input coin
// to obtain the correct ratio of both tokens for adding liquidity to the pool.
//
// The function uses the constant product AMM formula (x * y = k) to determine
// the swap amount that will result in token amounts matching the pool's reserve ratio.
//
// Mathematical derivation:
// Given:
//   - X: total amount of input token we have
//   - Ra, Rb: current reserves of token A (input) and token B (output)
//   - s: amount to swap
//   - f = 1 - fee: fee multiplier
//
// After swapping s amount of A, we get: b = Rb * s_net / (Ra + s_net), where s_net = s * f
// Remaining A: a = X - s
// For optimal ratio: a / b = Ra_new / Rb_new
//
// This leads to the quadratic equation:
// s^2 * f^2 + s * Ra * (1 + f) - X * Ra = 0
//
// Solution:
// s = (sqrt(Ra * (Ra * (1 + f)^2 + 4 * f^2 * X)) - Ra * (1 + f)) / (2 * f^2)
//
// Parameters:
//   - pool: the liquidity pool to add liquidity to
//   - inputCoin: the single token the user has
//
// Returns:
//   - swapAmount: the optimal amount of inputCoin to swap
//   - error: if the input coin denom is not in the pool or calculation fails
func (k Keeper) CalculateOptimalSwapAmount(pool types.LiquidityPool, inputCoin sdk.Coin) (swapAmount math.Int, err error) {
	if !pool.HasDenom(inputCoin.Denom) {
		return math.ZeroInt(), fmt.Errorf("denom %s does not exist in pool %s", inputCoin.Denom, pool.GetId())
	}

	// Get the reserves for the input token
	inputReserve, _ := pool.GetReservesCoinsByDenom(inputCoin.Denom)

	// X = total amount of input token
	X := math.LegacyNewDecFromInt(inputCoin.Amount)

	// Ra = reserve of input token
	Ra := math.LegacyNewDecFromInt(inputReserve.Amount)

	// f = 1 - fee (the multiplier for the net input after fee)
	f := math.LegacyOneDec().Sub(pool.Fee)

	// If fee is 1 (100%), no swap is possible
	if !f.IsPositive() {
		return math.ZeroInt(), fmt.Errorf("fee is too high, cannot calculate optimal swap")
	}

	// Calculate (1 + f)
	onePlusF := math.LegacyOneDec().Add(f)

	// Calculate f^2
	fSquared := f.Mul(f)

	// Calculate Ra * (1 + f)^2
	raTimeOnePlusFSquared := Ra.Mul(onePlusF).Mul(onePlusF)

	// Calculate 4 * f^2 * X
	fourFSquaredX := fSquared.Mul(X).Mul(math.LegacyNewDec(4))

	// Calculate Ra * (Ra * (1 + f)^2 + 4 * f^2 * X)
	underSqrt := Ra.Mul(raTimeOnePlusFSquared.Add(fourFSquaredX))

	// Calculate sqrt(Ra * (Ra * (1 + f)^2 + 4 * f^2 * X))
	sqrtValue, err := underSqrt.ApproxSqrt()
	if err != nil {
		return math.ZeroInt(), fmt.Errorf("failed to calculate square root: %w", err)
	}

	// Calculate numerator: sqrt(...) - Ra * (1 + f)
	numerator := sqrtValue.Sub(Ra.Mul(onePlusF))

	// Calculate denominator: 2 * f^2
	denominator := fSquared.Mul(math.LegacyNewDec(2))

	if !denominator.IsPositive() {
		return math.ZeroInt(), fmt.Errorf("denominator is not positive")
	}

	// Calculate s = numerator / denominator
	swapAmountDec := numerator.Quo(denominator)

	// Ensure the swap amount is non-negative
	if swapAmountDec.IsNegative() {
		return math.ZeroInt(), nil
	}

	// Ensure the swap amount doesn't exceed the input amount
	if swapAmountDec.GT(X) {
		return math.ZeroInt(), fmt.Errorf("calculated swap amount exceeds input amount")
	}

	swapAmount = swapAmountDec.TruncateInt()

	return swapAmount, nil
}

// CalculateOptimalInputForOutput calculates the required input amount to obtain a desired output amount
// from a liquidity pool, accounting for fees.
//
// The function uses the constant product AMM formula (x * y = k) in reverse to determine
// the input amount needed to receive a specific output amount.
//
// Mathematical derivation:
// Given the swap formula:
//
//	output = (output_reserve * real_input) / (input_reserve + real_input)
//
// Where real_input = input * (1 - fee), we need to solve for input given a desired output.
//
// Rearranging:
//
//	output * (input_reserve + real_input) = output_reserve * real_input
//	output * input_reserve = real_input * (output_reserve - output)
//	real_input = (output * input_reserve) / (output_reserve - output)
//
// Since real_input = input * f, where f = (1 - fee):
//
//	input = real_input / f
//
// Parameters:
//   - pool: the liquidity pool to swap from
//   - outputCoin: the desired output token and amount
//
// Returns:
//   - inputCoin: the required input token and amount (including fees)
//   - error: if the output coin denom is not in the pool, output exceeds reserves, or calculation fails
func (k Keeper) CalculateOptimalInputForOutput(pool types.LiquidityPool, outputCoin sdk.Coin) (requiredInput sdk.Coin, err error) {
	if !pool.HasDenom(outputCoin.Denom) {
		return requiredInput, fmt.Errorf("denom %s does not exist in pool %s", outputCoin.Denom, pool.GetId())
	}

	if !outputCoin.IsPositive() {
		return requiredInput, fmt.Errorf("output amount must be positive")
	}

	// Get reserves - outputReserve is for the token we want to receive
	// inputReserve is for the token we need to provide
	outputReserve, inputReserve := pool.GetReservesCoinsByDenom(outputCoin.Denom)

	// explicit reserve sanity
	if !outputReserve.Amount.IsPositive() || !inputReserve.Amount.IsPositive() {
		return requiredInput, fmt.Errorf("pool %s has insufficient liquidity", pool.GetId())
	}

	// Check if the desired output exceeds the available reserve
	if outputCoin.Amount.GTE(outputReserve.Amount) {
		return requiredInput, fmt.Errorf("desired output %s exceeds available reserve %s", outputCoin.Amount.String(), outputReserve.Amount.String())
	}

	// Convert to Dec for precise calculation
	outputAmount := math.LegacyNewDecFromInt(outputCoin.Amount)
	outputReserveAmount := math.LegacyNewDecFromInt(outputReserve.Amount)
	inputReserveAmount := math.LegacyNewDecFromInt(inputReserve.Amount)

	// Calculate the denominator: (output_reserve - output)
	denominator := outputReserveAmount.Sub(outputAmount)
	if !denominator.IsPositive() {
		return requiredInput, fmt.Errorf("invalid denominator in calculation")
	}

	// Calculate real_input needed (after fee deduction):
	// real_input = (output * input_reserve) / (output_reserve - output)
	numerator := outputAmount.Mul(inputReserveAmount)
	realInput := numerator.Quo(denominator)

	// f = 1 - fee (the multiplier for the net input after fee)
	f := math.LegacyOneDec().Sub(pool.Fee)

	// If fee is 1 (100%), no swap is possible
	if !f.IsPositive() {
		return requiredInput, fmt.Errorf("fee is too high, cannot calculate required input")
	}

	// Calculate the actual input needed (before fee deduction):
	// input = real_input / f
	inputAmount := realInput.Quo(f)

	// Round up to ensure we get at least the desired output
	// (using Ceil to avoid truncation that might result in slightly less output)
	inputAmountInt := inputAmount.Ceil().TruncateInt()

	requiredInput = sdk.NewCoin(inputReserve.Denom, inputAmountInt)

	if !requiredInput.IsPositive() {
		return requiredInput, fmt.Errorf("calculated input amount is not positive")
	}

	return requiredInput, nil
}

func (k Keeper) getProvidedReserves(baseDenom, quoteDenom string, baseAmt, quoteAmt math.Int) (baseCoin, quoteCoin sdk.Coin, err error) {
	baseCoin = sdk.NewCoin(baseDenom, baseAmt)
	quoteCoin = sdk.NewCoin(quoteDenom, quoteAmt)
	if !baseCoin.IsValid() || !quoteCoin.IsValid() {
		err = errors.Wrap(sdkerrors.ErrInvalidCoins, "invalid reserve")
		return
	}

	if !baseCoin.IsPositive() || !quoteCoin.IsPositive() {
		err = errors.Wrap(sdkerrors.ErrInvalidCoins, "non positive reserve provided")
		return
	}

	return
}

func (k Keeper) mintDepositLpTokens(ctx sdk.Context, baseAmount, quoteAmount, poolBaseReserve, poolQuoteReserve *math.Int, lp *types.LiquidityPool) (mintedLp sdk.Coin, err error) {
	lpSupply := k.bankKeeper.GetSupply(ctx, lp.GetLpDenom())
	if !lpSupply.IsPositive() {
		return mintedLp, errors.Wrapf(types.ErrInvalidDenom, "could not find supply for pool %s", lp.GetId())
	}

	baseRatio := math.LegacyNewDecFromInt(*baseAmount).Quo(math.LegacyNewDecFromInt(*poolBaseReserve))
	quoteRatio := math.LegacyNewDecFromInt(*quoteAmount).Quo(math.LegacyNewDecFromInt(*poolQuoteReserve))

	var mintRatio math.LegacyDec
	if baseRatio.LT(quoteRatio) {
		mintRatio = baseRatio
	} else {
		mintRatio = quoteRatio
	}

	tokensToMint := mintRatio.Mul(math.LegacyNewDecFromInt(lpSupply.Amount)).TruncateInt()
	if !tokensToMint.IsPositive() {
		return mintedLp, errors.Wrap(sdkerrors.ErrInvalidCoins, "resulted LP shares is not positive")
	}

	mintedLp = sdk.NewCoin(lp.GetLpDenom(), tokensToMint)
	// Mint the LP tokens
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(mintedLp))
	if err != nil {
		return mintedLp, errors.Wrapf(err, "could not mint liquidity pool tokens %s", lp.GetId())
	}

	return mintedLp, nil
}
