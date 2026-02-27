package keeper

import (
	"strconv"

	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetAllBurnedCoins(ctx sdk.Context) (list []types.BurnedCoins) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BurnedCoinsKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.BurnedCoins
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetBurnedCoins(ctx sdk.Context, burnedCoins types.BurnedCoins) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BurnedCoinsKeyPrefix))
	val := k.cdc.MustMarshal(&burnedCoins)
	store.Set(
		types.BurnedCoinsKey(burnedCoins.Height),
		val,
	)
}

func (k Keeper) GetBurnedCoins(ctx sdk.Context, height string) (val types.BurnedCoins, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.BurnedCoinsKeyPrefix))

	b := store.Get(types.BurnedCoinsKey(height))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) SetPeriodicBurnQueue(ctx sdk.Context, q types.PeriodicBurnQueue) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PeriodicBurnQueueKey))
	b := k.cdc.MustMarshal(&q)
	store.Set([]byte{1}, b)
}

func (k Keeper) GetPeriodicBurnQueue(ctx sdk.Context) (val types.PeriodicBurnQueue, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PeriodicBurnQueueKey))
	b := store.Get([]byte{1})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) RemovePeriodicBurnQueue(ctx sdk.Context) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.PeriodicBurnQueueKey))
	store.Delete([]byte{1})
}

func (k Keeper) SetRaffleCleanupQueue(ctx sdk.Context, q types.RaffleCleanupQueue) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.RaffleCleanupQueueKey))
	b := k.cdc.MustMarshal(&q)
	store.Set([]byte{1}, b)
}

func (k Keeper) GetRaffleCleanupQueue(ctx sdk.Context) (val types.RaffleCleanupQueue, found bool) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.RaffleCleanupQueueKey))
	b := store.Get([]byte{1})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) RemoveRaffleCleanupQueue(ctx sdk.Context) {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.RaffleCleanupQueueKey))
	store.Delete([]byte{1})
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
