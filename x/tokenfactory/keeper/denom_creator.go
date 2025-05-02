package keeper

import (
	"cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) addDenomFromCreator(ctx sdk.Context, creator, denom string) {
	s := k.GetCreatorPrefixStore(ctx, creator)
	s.Set([]byte(denom), []byte(denom))
}

func (k Keeper) GetAllDenomsIterator(ctx sdk.Context) store.Iterator {
	return k.GetCreatorsPrefixStore(ctx).Iterator(nil, nil)
}
