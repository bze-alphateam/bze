package v2

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate sets default parameters for the txfeecollector module
// added in consensus version 2.
func Migrate(
	_ sdk.Context,
	store prefix.Store,
	cdc codec.BinaryCodec,
) error {
	defaultParams := types.DefaultParams()

	if err := defaultParams.Validate(); err != nil {
		return err
	}

	bz := cdc.MustMarshal(&defaultParams)
	store.Set(types.ParamsKey, bz)

	return nil
}
