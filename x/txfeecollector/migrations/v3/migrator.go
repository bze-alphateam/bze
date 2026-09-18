package v3

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate sets the default value of the BlockedIbcInbound parameter added in
// consensus version 3: an empty list, so no inbound IBC transfer is blocked.
// Everything else in the params is left as stored on chain.
//
// The v8.2.0 upgrade handler fills the list in afterwards on the chains that need
// it; on every other chain the parameter stays empty until governance edits it.
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

	params.BlockedIbcInbound = types.DefaultBlockedIbcInbound()

	if err := params.Validate(); err != nil {
		return err
	}

	bz = cdc.MustMarshal(&params)
	store.Set(types.ParamsKey, bz)

	return nil
}
