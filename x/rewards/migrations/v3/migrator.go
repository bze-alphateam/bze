package v3

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/rewards/exported"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/bze-alphateam/bze/x/rewards/v1types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate migrates params from x/params module to x/rewards own subspace
func Migrate(
	ctx sdk.Context,
	store prefix.Store,
	legacySubspace exported.Subspace,
	cdc codec.BinaryCodec,
) error {
	var currParams v1types.Params
	legacySubspace.GetParamSet(ctx, &currParams)

	if err := currParams.Validate(); err != nil {
		return err
	}

	srf, err := sdk.ParseCoinNormalized(currParams.CreateStakingRewardFee)
	if err != nil {
		return err
	}

	trf, err := sdk.ParseCoinNormalized(currParams.CreateTradingRewardFee)
	if err != nil {
		return err
	}

	newParams := types.NewParams(srf, trf, types.DefaultExtraGasForExitStake)
	bz := cdc.MustMarshal(&newParams)
	store.Set(types.ParamsKey, bz)

	return nil
}
