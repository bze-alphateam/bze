package v3

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate reads existing params, sets defaults for the new CwDeployFee fields,
// and writes the updated params back to the store.
func Migrate(
	_ sdk.Context,
	store prefix.Store,
	cdc codec.BinaryCodec,
) error {
	var params types.Params

	bz := store.Get(types.ParamsKey)
	if bz != nil {
		cdc.MustUnmarshal(bz, &params)
	} else {
		params = types.DefaultParams()
	}

	// Set defaults for the new fields
	params.CwDeployFeeDestination = types.DefaultCwDeployFeeDestination
	params.CwDeployFee = sdk.NewCoins(sdk.NewInt64Coin("ubze", 5000000000))

	if err := params.Validate(); err != nil {
		return err
	}

	bz = cdc.MustMarshal(&params)
	store.Set(types.ParamsKey, bz)

	return nil
}
