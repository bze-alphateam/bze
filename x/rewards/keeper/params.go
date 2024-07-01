package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.CreateStakingRewardFee(ctx),
		k.CreateTradingRewardFee(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// CreateStakingRewardFee returns the CreateStakingRewardFee param
func (k Keeper) CreateStakingRewardFee(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyCreateStakingRewardFee, &res)
	return
}

// CreateTradingRewardFee returns the CreateTradingRewardFee param
func (k Keeper) CreateTradingRewardFee(ctx sdk.Context) (res string) {
	k.paramstore.Get(ctx, types.KeyCreateTradingRewardFee, &res)
	return
}
