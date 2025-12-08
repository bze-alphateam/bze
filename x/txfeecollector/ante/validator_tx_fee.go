package ante

import (
	"math"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// checkTxFeeWithValidatorMinGasPrices implements the default fee logic, where the minimum price per
// unit of gas is fixed and set by each validator, can the tx priority is computed from the gas price.
func (dfd DeductFeeDecorator) checkTxFeeWithValidatorMinGasPrices(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil, 0, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()

	// Ensure that the provided fees meet a minimum threshold for the validator,
	// if this is a CheckTx. This is only for local mempool purposes, and thus
	// is only ran on check tx.
	if ctx.IsCheckTx() {
		minGasPrices := dfd.getContextMinGasPrices(ctx)
		if !minGasPrices.IsZero() {
			requiredFees := make(sdk.Coins, len(minGasPrices))

			// Determine the required fees by multiplying each required minimum gas
			// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
			glDec := sdkmath.LegacyNewDec(int64(gas))
			for i, gp := range minGasPrices {
				fee := gp.Amount.Mul(glDec)
				requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
			}

			if !feeCoins.IsAnyGTE(requiredFees) {
				return nil, 0, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s required: %s", feeCoins, requiredFees)
			}
		}
	}

	priority := getTxPriority(feeCoins, int64(gas))
	return feeCoins, priority, nil
}

func (dfd DeductFeeDecorator) getContextMinGasPrices(ctx sdk.Context) sdk.DecCoins {
	params := dfd.txCollectorKeeper.GetParams(ctx)
	txDenom := getContextFeeDenom(ctx)
	if txDenom == "" {
		txDenom = params.ValidatorMinGasFee.Denom
	}

	localGasPrice := ctx.MinGasPrices().AmountOf(txDenom)
	//if the denom is the native one (the one in our params) then we need to make sure it meets the minimum fee
	if txDenom == params.ValidatorMinGasFee.Denom {
		if localGasPrice.LT(params.ValidatorMinGasFee.Amount) {
			localGasPrice = params.ValidatorMinGasFee.Amount
		}

		return sdk.NewDecCoins(sdk.NewDecCoinFromDec(txDenom, localGasPrice))
	}

	//get local native gas price (ubze)
	localNativeGasPrice := ctx.MinGasPrices().AmountOf(params.ValidatorMinGasFee.Denom)
	if localNativeGasPrice.LT(params.ValidatorMinGasFee.Amount) {
		//ensure it meets the minimum fee in the params
		localNativeGasPrice = params.ValidatorMinGasFee.Amount
	}

	txDenomPrice, err := dfd.tradeKeeper.GetDenomSpotPriceInNativeCoin(ctx, txDenom)
	if err != nil || txDenomPrice.IsZero() {
		dfd.txCollectorKeeper.Logger().Error("failed to get denom spot price", "denom", txDenom, "error", err)

		// we failed to get the spot price, so we return the minimum gas price either from local or from params
		return sdk.NewDecCoins(sdk.NewDecCoinFromDec(params.ValidatorMinGasFee.Denom, localNativeGasPrice))
	}

	txDenomGasPrice := localNativeGasPrice.Quo(txDenomPrice.Amount)
	if txDenomGasPrice.LT(localGasPrice) {
		txDenomGasPrice = localGasPrice
	}

	return sdk.NewDecCoins(sdk.NewDecCoinFromDec(txDenom, txDenomGasPrice))
}

func getContextFeeDenom(ctx sdk.Context) string {
	feeDenomVal := ctx.Value(FeeDenomKey)
	if feeDenomVal == nil {
		return ""
	}

	feeDenom, ok := feeDenomVal.(string)
	if !ok {
		return ""
	}

	return feeDenom
}

// getTxPriority returns a naive tx priority based on the amount of the smallest denomination of the gas price
// provided in a transaction.
// NOTE: This implementation should be used with a great consideration as it opens potential attack vectors
// where txs with multiple coins could not be prioritize as expected.
func getTxPriority(fee sdk.Coins, gas int64) int64 {
	var priority int64
	for _, c := range fee {
		p := int64(math.MaxInt64)
		gasPrice := c.Amount.QuoRaw(gas)
		if gasPrice.IsInt64() {
			p = gasPrice.Int64()
		}
		if priority == 0 || p < priority {
			priority = p
		}
	}

	return priority
}
