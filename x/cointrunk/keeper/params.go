package keeper

import (
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.AnonArticleLimit(ctx),
		k.AnonArticleCost(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}


// AnonArticleLimit returns the AnonArticleLimit param
func (k Keeper) AnonArticleLimit(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyAnonArticleLimit, &res)
	return
}

// AnonArticleCost returns the AnonArticleCost param
func (k Keeper) AnonArticleCost(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyAnonArticleCost, &res)
	return
}
