package keeper

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getMarketStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MarketKeyPrefix))
}

func (k Keeper) getMarketAliasStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MarketAliasKeyPrefix))
}

// SetMarket set a specific market in the store from its index
func (k Keeper) SetMarket(ctx sdk.Context, market types.Market) {
	store := k.getMarketStore(ctx)
	b := k.cdc.MustMarshal(&market)
	store.Set(types.MarketKey(
		market.Asset1,
		market.Asset2,
	), b)

	//store the same market on switched assets as keys in order to make sure the market is unique between two assets
	//we duplicate the same market details in another key. This will help us when searching one asset's markets.
	aStore := k.getMarketAliasStore(ctx)
	aStore.Set(types.MarketKey(
		market.Asset2,
		market.Asset1,
	), b)
}

// GetMarketAlias returns a market from the alias index
func (k Keeper) GetMarketAlias(
	ctx sdk.Context,
	asset1 string,
	asset2 string,

) (val types.Market, found bool) {
	store := k.getMarketAliasStore(ctx)

	b := store.Get(types.MarketKey(
		asset1,
		asset2,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// GetMarket returns a market from its index
func (k Keeper) GetMarket(
	ctx sdk.Context,
	asset1 string,
	asset2 string,

) (val types.Market, found bool) {
	store := k.getMarketStore(ctx)

	b := store.Get(types.MarketKey(
		asset1,
		asset2,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// GetAllMarket returns all market
func (k Keeper) GetAllMarket(ctx sdk.Context) (list []types.Market) {
	store := k.getMarketStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Market
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetAllAssetMarkets returns all markets for an asset
func (k Keeper) GetAllAssetMarkets(ctx sdk.Context, asset string) (list []types.Market) {
	store := k.getMarketStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, types.MarketAssetKey(asset))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Market
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetAllAssetMarketAliases returns all market aliases for an asset
func (k Keeper) GetAllAssetMarketAliases(ctx sdk.Context, asset string) (list []types.Market) {
	store := k.getMarketAliasStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, types.MarketAssetKey(asset))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Market
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
