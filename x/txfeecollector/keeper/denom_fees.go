package keeper

import (
	"context"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// GetDenomFeeRates returns the transfer fee and gas surcharge ratios a denom's DenomConfig sets. Only factory denoms
// can carry them: other denoms, and every denom when no provider is wired, get zeros without a store read.
func (k Keeper) GetDenomFeeRates(ctx sdk.Context, denom string) (transferFee, gasSurcharge math.LegacyDec) {
	if k.denomCpProvider == nil || !strings.HasPrefix(denom, types.FactoryDenomPrefix) {
		return math.LegacyZeroDec(), math.LegacyZeroDec()
	}

	transferFee, gasSurcharge = k.denomCpProvider.GetDenomFeeRates(ctx, denom)
	if transferFee.IsNil() {
		transferFee = math.LegacyZeroDec()
	}
	if gasSurcharge.IsNil() {
		gasSurcharge = math.LegacyZeroDec()
	}

	return transferFee, gasSurcharge
}

// TransferFeeSendRestriction is the x/bank send restriction that charges a factory token's transfer fee. It runs on
// every bank send (account and module sends, every multisend output) and never changes the recipient.
//
// The fee is ceil(amount × transfer fee) for each sent coin whose denom sets one, charged on top of the amount: the
// sender pays amount + fee and the recipient gets the full amount. Rounding up means a send split into dust still
// pays (1 unit at 0.1% pays 1). Module accounts and IBC transfer escrow accounts never pay, so only the initial user
// send of a flow is charged. The fee goes to txfeecollector_cp when the token has a community pool (forwarded there at
// EndBlock) and to txfeecollector_burner when it has none.
//
// It charges nothing while the escrow checker or the denom config provider is unset, or when the context carries
// types.WithSkipTransferFee.
//
// In a multisend the restriction runs once per output, before x/bank debits the input, so the balance check covers
// one output plus its fee at a time. When only the total doesn't fit, x/bank's own debit fails the tx afterwards
// with its generic insufficient funds error; nothing is charged because the tx reverts.
func (k *Keeper) TransferFeeSendRestriction(goCtx context.Context, from, to sdk.AccAddress, amt sdk.Coins) (sdk.AccAddress, error) {
	if types.SkipTransferFee(goCtx) || k.escrowChecker == nil || k.denomCpProvider == nil {
		return to, nil
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	fees := k.transferFees(ctx, amt)
	if fees.IsZero() {
		return to, nil
	}

	if k.isTransferFeeExempt(ctx, from) {
		return to, nil
	}

	// check first, so a caller recovering from the error is left with nothing charged
	spendable := k.bankKeeper.SpendableCoins(ctx, from)
	if !spendable.IsAllGTE(amt.Add(fees...)) {
		return to, errorsmod.Wrapf(
			sdkerrors.ErrInsufficientFunds,
			"spendable balance %s is smaller than %s plus the transfer fee %s",
			spendable, amt, fees,
		)
	}

	// the fee send runs through x/bank too: the flag keeps it from paying a fee of its own
	skipCtx := types.WithSkipTransferFee(ctx)
	for _, fee := range fees {
		destination := k.denomFeeDestination(ctx, fee.Denom)
		if err := k.bankKeeper.SendCoinsFromAccountToModule(skipCtx, from, destination, sdk.NewCoins(fee)); err != nil {
			return to, err
		}

		err := ctx.EventManager().EmitTypedEvent(&types.TransferFeePaidEvent{
			Payer:       from.String(),
			Denom:       fee.Denom,
			Amount:      fee.Amount.String(),
			Destination: destination,
		})
		if err != nil {
			k.Logger().Error("failed to emit transfer fee paid event", "err", err)
		}
	}

	return to, nil
}

// ChargeGasSurcharge moves the gas surcharge out of the tx fee the payer just paid to this module. For each fee coin
// whose denom sets a surcharge s, floor(amount × s / (1 + s)) goes from this module to the denom's fee destination;
// the rest stays here to be converted for the validators as before. The split runs on the fee actually paid.
func (k Keeper) ChargeGasSurcharge(ctx sdk.Context, payer sdk.AccAddress, fees sdk.Coins) error {
	if k.denomCpProvider == nil {
		return nil
	}

	for _, fee := range fees {
		_, surchargeRate := k.GetDenomFeeRates(ctx, fee.Denom)
		if !surchargeRate.IsPositive() || !fee.IsPositive() {
			continue
		}

		// truncated division keeps the floor exact: amount × s is exact at 18 decimals
		surcharge := fee.Amount.ToLegacyDec().Mul(surchargeRate).QuoTruncate(math.LegacyOneDec().Add(surchargeRate)).TruncateInt()
		if !surcharge.IsPositive() {
			continue
		}

		destination := k.denomFeeDestination(ctx, fee.Denom)
		err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, destination, sdk.NewCoins(sdk.NewCoin(fee.Denom, surcharge)))
		if err != nil {
			return err
		}

		err = ctx.EventManager().EmitTypedEvent(&types.GasSurchargePaidEvent{
			Payer:       payer.String(),
			Denom:       fee.Denom,
			Amount:      surcharge.String(),
			Destination: destination,
		})
		if err != nil {
			k.Logger().Error("failed to emit gas surcharge paid event", "err", err)
		}
	}

	return nil
}

// transferFees returns ceil(amount × transfer fee) for each coin whose denom sets a transfer fee. Only factory denoms
// reach a store read.
func (k Keeper) transferFees(ctx sdk.Context, amt sdk.Coins) sdk.Coins {
	fees := sdk.NewCoins()
	for _, c := range amt {
		if !c.IsPositive() {
			continue
		}

		rate, _ := k.GetDenomFeeRates(ctx, c.Denom)
		if !rate.IsPositive() {
			continue
		}

		fees = fees.Add(sdk.NewCoin(c.Denom, c.Amount.ToLegacyDec().Mul(rate).Ceil().TruncateInt()))
	}

	return fees
}

// isTransferFeeExempt reports whether the sender never pays the transfer fee: a module account or an IBC transfer
// escrow account.
func (k Keeper) isTransferFeeExempt(ctx sdk.Context, sender sdk.AccAddress) bool {
	if _, ok := k.accountKeeper.GetAccount(ctx, sender).(sdk.ModuleAccountI); ok {
		return true
	}

	return k.escrowChecker.IsEscrowAddress(ctx, sender)
}

// denomFeeDestination is the module account a denom's transfer fee and gas surcharge go to: the community pool fee
// collector when the denom has a community pool, the burner fee collector when it has none.
func (k Keeper) denomFeeDestination(ctx sdk.Context, denom string) string {
	if _, ok := k.denomCpProvider.GetDenomCommunityPool(ctx, denom); ok {
		return types.CpFeeCollector
	}

	return types.BurnerFeeCollector
}
