package v2

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/cointrunk/exported"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/bze-alphateam/bze/x/cointrunk/v1types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
)

// MigrateParams migrates params from x/params module to x/cointrunk own subspace
func MigrateParams(
	ctx sdk.Context,
	store prefix.Store,
	legacySubspace exported.Subspace,
	cdc codec.BinaryCodec,
) error {
	var currParams types.Params
	legacySubspace.GetParamSet(ctx, &currParams)

	if err := currParams.Validate(); err != nil {
		return err
	}

	bz := cdc.MustMarshal(&currParams)
	store.Set(types.ParamsKey, bz)

	return nil
}

// MigratePublishers migrates publishers from v1 to v2.
// in v1 publishers had "respect" field int64, and in v2 it's string
func MigratePublishers(
	store prefix.Store,
	cdc codec.BinaryCodec,
) error {
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var old v1types.Publisher
		cdc.MustUnmarshal(iterator.Value(), &old)
		//convert to the new structure
		newPublisher := types.Publisher{
			Name:          old.Name,
			Address:       old.Address,
			Active:        old.Active,
			ArticlesCount: old.ArticlesCount,
			CreatedAt:     old.CreatedAt,
			Respect:       strconv.FormatInt(old.Respect, 10),
		}

		//marshal new structure
		p := cdc.MustMarshal(&newPublisher)
		//set in store
		store.Set(types.PublisherKey(newPublisher.Address), p)
	}

	return nil
}
