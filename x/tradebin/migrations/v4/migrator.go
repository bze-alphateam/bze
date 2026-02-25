package v4

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate converts parameters from types.Params (string fees) to v2types.Params (sdk.Coin fees)
// and sets new default values for parameters added in consensus version 4.
func Migrate(
	_ sdk.Context,
	store prefix.Store,
	cdc codec.BinaryCodec,
) error {
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		// No params stored yet — write defaults directly
		newParams := v2types.DefaultParams()
		bz = cdc.MustMarshal(&newParams)
		store.Set(types.ParamsKey, bz)
		return nil
	}

	var oldParams types.Params
	cdc.MustUnmarshal(bz, &oldParams)

	createMarketFee, err := sdk.ParseCoinNormalized(oldParams.CreateMarketFee)
	if err != nil {
		return err
	}

	marketMakerFee, err := sdk.ParseCoinNormalized(oldParams.MarketMakerFee)
	if err != nil {
		return err
	}

	marketTakerFee, err := sdk.ParseCoinNormalized(oldParams.MarketTakerFee)
	if err != nil {
		return err
	}

	newParams := v2types.Params{
		CreateMarketFee:                   createMarketFee,
		MarketMakerFee:                    marketMakerFee,
		MarketTakerFee:                    marketTakerFee,
		MakerFeeDestination:               oldParams.MakerFeeDestination,
		TakerFeeDestination:               oldParams.TakerFeeDestination,
		NativeDenom:                       oldParams.NativeDenom,
		OrderBookExtraGasWindow:           v2types.DefaultOrderBookExtraGasWindow,
		OrderBookQueueExtraGas:            v2types.DefaultOrderBookQueueExtraGas,
		FillOrdersExtraGas:                v2types.DefaultFillOrdersExtraGas,
		OrderBookQueueMessageScanExtraGas: v2types.DefaultOrderBookQueueMessageScanExtraGas,
		MinNativeLiquidityForModuleSwap:   v2types.DefaultMinNativeLiquidityForModuleSwap,
		OrderBookPerBlockMessages:         v2types.DefaultOrderBookPerBlockMessages,
	}

	if err := newParams.Validate(); err != nil {
		return err
	}

	bz = cdc.MustMarshal(&newParams)
	store.Set(types.ParamsKey, bz)

	return nil
}
