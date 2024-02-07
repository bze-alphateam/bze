package keeper

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.CreateDenomFee(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// CreateDenomFee returns the CreateDenomFee param
func (k Keeper) CreateDenomFee(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyCreateDenomFee, &res)
	return
}
