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

func (k Keeper) RemoveRaffle(ctx sdk.Context, denom string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RaffleKeyPrefix))
	store.Delete(types.RaffleStoreKey(denom))
}

func (k Keeper) SetRaffleDeleteHook(ctx sdk.Context, raffle types.RaffleDeleteHook) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.GetRaffleDeleteHookPrefix(raffle.EndAt))
	val := k.cdc.MustMarshal(&raffle)
	store.Set(
		types.RaffleStoreKey(raffle.Denom),
		val,
	)
}

func (k Keeper) RemoveRaffleDeleteHook(ctx sdk.Context, raffle types.RaffleDeleteHook) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.GetRaffleDeleteHookPrefix(raffle.EndAt))
	store.Delete(types.RaffleStoreKey(raffle.Denom))
}

func (k Keeper) GetRaffleDeleteHookByEndAtPrefix(ctx sdk.Context, endAt uint64) (list []types.RaffleDeleteHook) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.GetRaffleDeleteHookPrefix(endAt))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.RaffleDeleteHook
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetRaffleWinner(ctx sdk.Context, winner types.RaffleWinner) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.GetRaffleWinnerKeyPrefix(winner.Denom))
	val := k.cdc.MustMarshal(&winner)
	store.Set(
		types.RaffleStoreKey(winner.Index),
		val,
	)
}

func (k Keeper) getAllRaffleWinnersByPrefix(ctx sdk.Context, pref []byte) (list []types.RaffleWinner) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), pref)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.RaffleWinner
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) RemoveRaffleWinner(ctx sdk.Context, winner types.RaffleWinner) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.GetRaffleWinnerKeyPrefix(winner.Denom))
	store.Delete(types.RaffleStoreKey(winner.Index))
}

func (k Keeper) GetRaffleWinners(ctx sdk.Context, denom string) (list []types.RaffleWinner) {

	return k.getAllRaffleWinnersByPrefix(ctx, types.GetRaffleWinnerKeyPrefix(denom))
}
