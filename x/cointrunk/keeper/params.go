package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams set the params
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)

	return nil
}

// AnonArticleLimit returns the AnonArticleLimit param
func (k Keeper) AnonArticleLimit(ctx sdk.Context) (res uint64) {
	p := k.GetParams(ctx)

	return p.AnonArticleLimit
}

// AnonArticleCost returns the AnonArticleCost param
func (k Keeper) AnonArticleCost(ctx sdk.Context) (res sdk.Coin) {
	p := k.GetParams(ctx)

	return p.AnonArticleCost
}

// PublisherRespectParams returns the PublisherRespectParams param
func (k Keeper) PublisherRespectParams(ctx sdk.Context) (res types.PublisherRespectParams) {
	p := k.GetParams(ctx)

	return p.PublisherRespectParams
}
