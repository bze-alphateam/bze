package v2

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/burner/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// Migrate creates new empty params for burner module
func Migrate(
	store prefix.Store,
	cdc codec.BinaryCodec,
) error {
	currParams := types.NewParams(4)

	bz := cdc.MustMarshal(&currParams)
	store.Set(types.ParamsKey, bz)

	return nil
}
