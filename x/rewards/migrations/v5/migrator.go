package v5

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate sets the boost parameters added in consensus version 5 to their
// default values. Params-only: no stored record changed shape in this
// version, so the rest of the store is left untouched.
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

	params.CreateBoostFee = types.DefaultCreateBoostFee
	params.MaxBoostsPerReward = types.DefaultMaxBoostsPerReward

	if err := params.Validate(); err != nil {
		return err
	}

	bz = cdc.MustMarshal(&params)
	store.Set(types.ParamsKey, bz)

	return nil
}
