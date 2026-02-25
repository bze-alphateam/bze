package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

// GetParams get all parameters as v2types.Params
func (k Keeper) GetParams(ctx context.Context) (params v2types.Params) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return v2types.DefaultParams()
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams set the params
func (k Keeper) SetParams(ctx context.Context, params v2types.Params) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)

	return nil
}

// CreateMarketFee returns the CreateMarketFee param
func (k Keeper) CreateMarketFee(ctx sdk.Context) sdk.Coin {
	return k.GetParams(ctx).CreateMarketFee
}

// MarketMakerFee returns the MarketMakerFee param
func (k Keeper) MarketMakerFee(ctx sdk.Context) sdk.Coin {
	return k.GetParams(ctx).MarketMakerFee
}

// MarketTakerFee returns the MarketTakerFee param
func (k Keeper) MarketTakerFee(ctx sdk.Context) sdk.Coin {
	return k.GetParams(ctx).MarketTakerFee
}

// MakerFeeDestination returns the MakerFeeDestination param
func (k Keeper) MakerFeeDestination(ctx sdk.Context) string {
	return k.GetParams(ctx).MakerFeeDestination
}

// TakerFeeDestination returns the TakerFeeDestination param
func (k Keeper) TakerFeeDestination(ctx sdk.Context) string {
	return k.GetParams(ctx).TakerFeeDestination
}
