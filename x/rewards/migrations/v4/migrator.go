package v4

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate sets the new ExtraGasForExitStake parameter default value
// added in consensus version 4.
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

	params.ExtraGasForExitStake = types.DefaultExtraGasForExitStake

	if err := params.Validate(); err != nil {
		return err
	}

	bz = cdc.MustMarshal(&params)
	store.Set(types.ParamsKey, bz)

	return nil
}
