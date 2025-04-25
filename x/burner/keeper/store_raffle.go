package keeper

import (
	"encoding/binary"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/burner/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getPrefixedStore(ctx sdk.Context, p []byte) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	return prefix.NewStore(storeAdapter, p)
}

func (k Keeper) getBurnerStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.RaffleKeyPrefix))
}

func (k Keeper) GetAllRaffle(ctx sdk.Context) (list []types.Raffle) {
	store := k.getBurnerStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Raffle
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetRaffle(ctx sdk.Context, denom string) (val types.Raffle, found bool) {
	store := k.getBurnerStore(ctx)

	b := store.Get(types.RaffleStoreKey(denom))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) SetRaffle(ctx sdk.Context, raffle types.Raffle) {
	store := k.getBurnerStore(ctx)
	val := k.cdc.MustMarshal(&raffle)
	store.Set(
		types.RaffleStoreKey(raffle.Denom),
		val,
	)
}

func (k Keeper) RemoveRaffle(ctx sdk.Context, denom string) {
	store := k.getBurnerStore(ctx)
	store.Delete(types.RaffleStoreKey(denom))
}

func (k Keeper) SetRaffleDeleteHook(ctx sdk.Context, raffle types.RaffleDeleteHook) {
	store := k.getPrefixedStore(ctx, types.GetRaffleDeleteHookPrefix(raffle.EndAt))
	val := k.cdc.MustMarshal(&raffle)
	store.Set(
		types.RaffleStoreKey(raffle.Denom),
		val,
	)
}

func (k Keeper) RemoveRaffleDeleteHook(ctx sdk.Context, raffle types.RaffleDeleteHook) {
	store := k.getPrefixedStore(ctx, types.GetRaffleDeleteHookPrefix(raffle.EndAt))
	store.Delete(types.RaffleStoreKey(raffle.Denom))
}

func (k Keeper) GetRaffleDeleteHookByEndAtPrefix(ctx sdk.Context, endAt uint64) (list []types.RaffleDeleteHook) {
	store := k.getPrefixedStore(ctx, types.GetRaffleDeleteHookPrefix(endAt))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.RaffleDeleteHook
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) SetRaffleWinner(ctx sdk.Context, winner types.RaffleWinner) {
	store := k.getPrefixedStore(ctx, types.GetRaffleWinnerKeyPrefix(winner.Denom))
	val := k.cdc.MustMarshal(&winner)
	store.Set(
		types.RaffleStoreKey(winner.Index),
		val,
	)
}

func (k Keeper) getAllRaffleWinnersByPrefix(ctx sdk.Context, pref []byte) (list []types.RaffleWinner) {
	store := k.getPrefixedStore(ctx, pref)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.RaffleWinner
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) RemoveRaffleWinner(ctx sdk.Context, winner types.RaffleWinner) {
	store := k.getPrefixedStore(ctx, types.GetRaffleWinnerKeyPrefix(winner.Denom))
	store.Delete(types.RaffleStoreKey(winner.Index))
}

func (k Keeper) GetRaffleWinners(ctx sdk.Context, denom string) (list []types.RaffleWinner) {
	return k.getAllRaffleWinnersByPrefix(ctx, types.GetRaffleWinnerKeyPrefix(denom))
}

func (k Keeper) getRaffleParticipantsByPrefix(ctx sdk.Context, pref []byte) (list []types.RaffleParticipant) {
	store := k.getPrefixedStore(ctx, pref)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.RaffleParticipant
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetAllRaffleParticipants(ctx sdk.Context) []types.RaffleParticipant {

	return k.getRaffleParticipantsByPrefix(ctx, types.KeyPrefix(types.RaffleParticipantPrefix))
}

func (k Keeper) GetAllPrefixedRaffleParticipants(ctx sdk.Context, pref int64) []types.RaffleParticipant {

	return k.getRaffleParticipantsByPrefix(ctx, types.GetRaffleParticipantPrefixedKey(pref))
}

func (k Keeper) SetRaffleParticipant(ctx sdk.Context, part types.RaffleParticipant) {
	store := k.getPrefixedStore(ctx, types.GetRaffleParticipantPrefixedKey(part.ExecuteAt))
	val := k.cdc.MustMarshal(&part)
	store.Set(
		types.GetRaffleParticipantKey(part.Index),
		val,
	)
	k.incrementParticipantCounter(ctx)
}

func (k Keeper) RemoveRaffleParticipant(ctx sdk.Context, part types.RaffleParticipant) {
	store := k.getPrefixedStore(ctx, types.GetRaffleParticipantPrefixedKey(part.ExecuteAt))
	store.Delete(types.GetRaffleParticipantKey(part.Index))
}

func (k Keeper) getParticipantCounterStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.RaffleParticipantCounterPrefix))
}

func (k Keeper) GetParticipantCounter(ctx sdk.Context) uint64 {
	counterStore := k.getParticipantCounterStore(ctx)
	counter := counterStore.Get(types.GetRaffleParticipantCounterKey())
	if counter == nil {
		return 0
	}

	return binary.BigEndian.Uint64(counter)
}

func (k Keeper) SetParticipantCounter(ctx sdk.Context, counter uint64) {
	counterStore := k.getParticipantCounterStore(ctx)
	record := make([]byte, 8)
	binary.BigEndian.PutUint64(record, counter)

	counterStore.Set(
		types.GetRaffleParticipantCounterKey(),
		record,
	)
}

func (k Keeper) incrementParticipantCounter(ctx sdk.Context) uint64 {
	counter := k.GetParticipantCounter(ctx)
	counter++
	k.SetParticipantCounter(ctx, counter)

	return counter
}
