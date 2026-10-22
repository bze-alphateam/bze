package keeper

import (
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

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

// denomFeeDestination is the module account a denom's gas surcharge goes to: the community pool fee
// collector when the denom has a community pool, the burner fee collector when it has none.
func (k Keeper) denomFeeDestination(ctx sdk.Context, denom string) string {
	if _, ok := k.denomCpProvider.GetDenomCommunityPool(ctx, denom); ok {
		return types.CpFeeCollector
	}

	return types.BurnerFeeCollector
}
