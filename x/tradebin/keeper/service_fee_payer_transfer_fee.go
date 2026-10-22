package keeper

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	txfeecollectormoduletypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// preferredDenomTransferFee returns the transfer fee a user pays on top of a protocol fee paid in a factory token:
// ceil(amount × transfer fee) of the token, empty when the token sets none. Only factory denoms reach a store read.
func (k Keeper) preferredDenomTransferFee(ctx sdk.Context, paid sdk.Coin) sdk.Coins {
	if !strings.HasPrefix(paid.Denom, txfeecollectormoduletypes.FactoryDenomPrefix) || !paid.IsPositive() {
		return sdk.NewCoins()
	}

	if k.denomFeeRates == nil {
		k.Logger().Warn("denom fee rates provider is not set, the factory token transfer fee is not charged", "denom", paid.Denom)

		return sdk.NewCoins()
	}

	rate, _ := k.denomFeeRates.GetDenomFeeRates(ctx, paid.Denom)
	if rate.IsNil() || !rate.IsPositive() {
		return sdk.NewCoins()
	}

	return sdk.NewCoins(sdk.NewCoin(paid.Denom, paid.Amount.ToLegacyDec().Mul(rate).Ceil().TruncateInt()))
}

// payDenomTransferFee sends the transfer fee from the payer to txfeecollector_cp when the token has a community pool
// (forwarded there at EndBlock) and to txfeecollector_burner when it has none.
func (k Keeper) payDenomTransferFee(ctx sdk.Context, payer sdk.AccAddress, transferFee sdk.Coins) error {
	for _, c := range transferFee {
		destination := txfeecollectormoduletypes.BurnerFeeCollector
		if _, ok := k.denomFeeRates.GetDenomCommunityPool(ctx, c.Denom); ok {
			destination = txfeecollectormoduletypes.CpFeeCollector
		}

		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, payer, destination, sdk.NewCoins(c)); err != nil {
			return err
		}
	}

	return nil
}
