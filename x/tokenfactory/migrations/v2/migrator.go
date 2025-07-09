package v2

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/tokenfactory/exported"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	"github.com/bze-alphateam/bze/x/tokenfactory/v1types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate migrates params from x/params module to x/tokenfactory own subspace
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

	coinParam, err := sdk.ParseCoinNormalized(currParams.CreateDenomFee)
	if err != nil {
		return err
	}

	newTypes := types.NewParams(coinParam)
	bz := cdc.MustMarshal(&newTypes)
	store.Set(types.ParamsKey, bz)

	return nil
}
