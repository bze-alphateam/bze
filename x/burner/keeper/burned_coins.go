package keeper

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetAllBurnedCoins(ctx sdk.Context) (list []types.BurnedCoins) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BurnedCoinsKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.BurnedCoins
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetBurnedCoins(ctx sdk.Context, burnedCoins types.BurnedCoins) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BurnedCoinsKeyPrefix))
	val := k.cdc.MustMarshal(&burnedCoins)
	store.Set(
		types.BurnedCoinsKey(burnedCoins.Height),
		val,
	)
}
