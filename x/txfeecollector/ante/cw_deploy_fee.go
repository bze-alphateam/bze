package ante

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/bze-alphateam/bze/x/txfeecollector/keeper"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// CwDeployFeeDecorator charges an additional fee for MsgStoreCode transactions.
// The fee amount and destination are configured via the txfeecollector module params.
// If the user pays tx fees in a non-native denom, the native portion of the deploy
// fee is converted to that denom using the spot price.
type CwDeployFeeDecorator struct {
	tradeKeeper       types.TradeKeeper
	bankKeeper        types.BankKeeper
	txCollectorKeeper *keeper.Keeper
}

func NewCwDeployFeeDecorator(
	tk types.TradeKeeper,
	bk types.BankKeeper,
	txk *keeper.Keeper,
) CwDeployFeeDecorator {
	return CwDeployFeeDecorator{
		tradeKeeper:       tk,
		bankKeeper:        bk,
		txCollectorKeeper: txk,
	}
}

func (cfd CwDeployFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	params := cfd.txCollectorKeeper.GetParams(ctx)
	if params.CwDeployFee.IsZero() {
		return next(ctx, tx, simulate)
	}

	storeCodeCount := 0
	for _, msg := range tx.GetMsgs() {
		if _, ok := msg.(*wasmtypes.MsgStoreCode); ok {
			storeCodeCount++
		}
	}

	if storeCodeCount == 0 {
		return next(ctx, tx, simulate)
	}

	// Calculate total fee: cw_deploy_fee * storeCodeCount
	totalFee := sdk.NewCoins()
	for _, coin := range params.CwDeployFee {
		totalFee = totalFee.Add(sdk.NewCoin(coin.Denom, coin.Amount.MulRaw(int64(storeCodeCount))))
	}

	// If the user pays tx fees in a non-native denom, convert the native portion
	// of the deploy fee to that denom so the user can pay entirely in their chosen denom.
	totalFee = cfd.convertNativePortionToFeeDenom(ctx, totalFee)

	// Determine fee payer (first signer)
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}
	feePayer := feeTx.FeePayer()

	// Determine destination module account
	destModule, err := feeDestinationToModule(params.CwDeployFeeDestination)
	if err != nil {
		return ctx, err
	}

	if !simulate {
		err = cfd.bankKeeper.SendCoinsFromAccountToModule(ctx, feePayer, destModule, totalFee)
		if err != nil {
			return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, "failed to charge cw deploy fee: %s", err)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"cw_deploy_fee",
			sdk.NewAttribute("fee_payer", sdk.AccAddress(feePayer).String()),
			sdk.NewAttribute("fee", totalFee.String()),
			sdk.NewAttribute("destination", params.CwDeployFeeDestination),
			sdk.NewAttribute("store_code_count", fmt.Sprintf("%d", storeCodeCount)),
		),
	)

	return next(ctx, tx, simulate)
}

// convertNativePortionToFeeDenom checks the tx fee denom from context. If the user
// pays in a non-native denom, it converts native coins in the fee to that denom using
// the spot price. Non-native coins in the fee are left unchanged.
// If the spot price lookup fails, the original native coin is kept (fallback).
func (cfd CwDeployFeeDecorator) convertNativePortionToFeeDenom(ctx sdk.Context, fee sdk.Coins) sdk.Coins {
	feeDenom := getContextFeeDenom(ctx)
	if feeDenom == "" {
		return fee
	}

	// If the fee denom is native, no conversion needed
	if cfd.tradeKeeper.IsNativeDenom(ctx, feeDenom) {
		return fee
	}

	// Get spot price: how much 1 unit of feeDenom is worth in native coin
	spotPrice, err := cfd.tradeKeeper.GetDenomSpotPriceInNativeCoin(ctx, feeDenom)
	if err != nil || spotPrice.IsZero() {
		// Spot price unavailable — fall back to charging the original native fee
		return fee
	}

	converted := sdk.NewCoins()
	for _, coin := range fee {
		if cfd.tradeKeeper.IsNativeDenom(ctx, coin.Denom) {
			// Convert native amount to fee denom: nativeAmount / spotPrice (ceiling)
			amt := sdkmath.LegacyNewDecFromInt(coin.Amount).Quo(spotPrice.Amount).Ceil().TruncateInt()
			if amt.IsPositive() {
				converted = converted.Add(sdk.NewCoin(feeDenom, amt))
			}
		} else {
			// Non-native coins stay as-is
			converted = converted.Add(coin)
		}
	}

	return converted
}

// feeDestinationToModule maps a CwDeployFeeDestination param value to the
// corresponding module account name used by the txfeecollector module.
func feeDestinationToModule(dest string) (string, error) {
	switch dest {
	case types.FeeDestBurner:
		return types.BurnerFeeCollector, nil
	case types.FeeDestCommunityPool:
		return types.CpFeeCollector, nil
	case types.FeeDestStakers:
		return types.ModuleName, nil
	default:
		return "", errorsmod.Wrapf(sdkerrors.ErrLogic, "unknown cw deploy fee destination: %s", dest)
	}
}
