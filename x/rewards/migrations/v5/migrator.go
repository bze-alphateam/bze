package v5

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate sets the default values of the seven Denom Rewards parameters
// added in consensus version 5. Everything else in the params is left as
// stored on chain.
func Migrate(
	_ sdk.Context,
	store prefix.Store,
	cdc codec.BinaryCodec,
) error {
	var params types.Params
	bz := store.Get(types.ParamsKey)
	if bz != nil {
		cdc.MustUnmarshal(bz, &params)
	}

	params.CreateDenomRewardFee = types.DefaultCreateRewardFee
	params.CreateDenomRewardPrizeFee = types.DefaultCreateRewardFee
	params.AddDenomRewardScheduleFee = types.DefaultAddDenomRewardScheduleFee
	params.MaxPrizeDenomsPerDr = types.DefaultMaxPrizeDenomsPerDr
	params.ExtraGasForDenomExit = types.DefaultExtraGasForDenomExit
	params.DenomRewardLock = types.DefaultDenomRewardLock
	params.DenomRewardMinStake = types.DefaultDenomRewardMinStake

	if err := params.Validate(); err != nil {
		return err
	}

	bz = cdc.MustMarshal(&params)
	store.Set(types.ParamsKey, bz)

	return nil
}
