package v2

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/tradebin/exported"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate migrates params from x/params module to x/tradebin own subspace
func Migrate(
	ctx sdk.Context,
	store prefix.Store,
	legacySubspace exported.Subspace,
	cdc codec.BinaryCodec,
) error {
	var currParams types.Params
	legacySubspace.GetParamSet(ctx, &currParams)

	currParams.NativeDenom = types.DefaultNativeDenom
	
	if err := currParams.Validate(); err != nil {
		return err
	}

	bz := cdc.MustMarshal(&currParams)
	store.Set(types.ParamsKey, bz)

	return nil
}
