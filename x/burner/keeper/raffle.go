package keeper

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetAllRaffle(ctx sdk.Context) (list []types.Raffle) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RaffleKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Raffle
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetRaffle(ctx sdk.Context, denom string) (val types.Raffle, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RaffleKeyPrefix))

	b := store.Get(types.RaffleStoreKey(denom))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) SetRaffle(ctx sdk.Context, raffle types.Raffle) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RaffleKeyPrefix))
	val := k.cdc.MustMarshal(&raffle)
	store.Set(
		types.RaffleStoreKey(raffle.Denom),
		val,
	)
}

func (k Keeper) SetRaffleDeleteHook(ctx sdk.Context, raffle types.RaffleDeleteHook) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.GetRaffleDeleteHookPrefix(raffle.EndAt))
	val := k.cdc.MustMarshal(&raffle)
	store.Set(
		types.RaffleStoreKey(raffle.Denom),
		val,
	)
}

func (k Keeper) SaveRaffleWinner(ctx sdk.Context, raffle types.RaffleWinner) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.GetRaffleWinnerKeyPrefix(raffle.Denom))
	val := k.cdc.MustMarshal(&raffle)
	store.Set(
		types.RaffleStoreKey(raffle.Index),
		val,
	)
}

func (k Keeper) GetAllRaffleWinners(ctx sdk.Context) (list []types.RaffleWinner) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RaffleWinnerKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.RaffleWinner
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
