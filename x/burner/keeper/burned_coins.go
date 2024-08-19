package keeper

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
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

func (k Keeper) GetBurnedCoins(ctx sdk.Context, height string) (val types.BurnedCoins, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BurnedCoinsKeyPrefix))

	b := store.Get(types.BurnedCoinsKey(height))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) SaveBurnedCoins(ctx sdk.Context, coins sdk.Coins) error {
	height := strconv.FormatInt(ctx.BlockHeader().Height, 10)
	b, found := k.GetBurnedCoins(ctx, height)
	if !found {
		b = types.BurnedCoins{
			Burned: coins.String(),
			Height: height,
		}
	} else {
		alreadyBurned, err := sdk.ParseCoinsNormalized(b.Burned)
		if err != nil {
			return err
		}

		alreadyBurned = alreadyBurned.Add(coins...)
		b.Burned = alreadyBurned.String()
	}

	k.SetBurnedCoins(ctx, b)

	return nil
}
