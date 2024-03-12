package keeper

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.CreateMarketFee(ctx),
		k.MarketMakerFee(ctx),
		k.MarketTakerFee(ctx),
		k.MakerFeeDestination(ctx),
		k.TakerFeeDestination(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// CreateMarketFee returns the CreateMarketFee param
func (k Keeper) CreateMarketFee(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyCreateMarketFee, &res)
	return
}

// MarketMakerFee returns the MarketMakerFee param
func (k Keeper) MarketMakerFee(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyMarketMakerFee, &res)
	return
}

// MarketTakerFee returns the MarketTakerFee param
func (k Keeper) MarketTakerFee(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyMarketTakerFee, &res)
	return
}

// MakerFeeDestination returns the MakerFeeDestination param
func (k Keeper) MakerFeeDestination(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyMakerFeeDestination, &res)
	return
}

// TakerFeeDestination returns the TakerFeeDestination param
func (k Keeper) TakerFeeDestination(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyTakerFeeDestination, &res)
	return
}
