package keeper

import (
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) getUserDustStore(ctx sdk.Context) prefix.Store {
	return k.getPrefixedStore(ctx, types.KeyPrefix(types.UserDustKeyPrefix))
}

func (k Keeper) GetAllUserDust(ctx sdk.Context) (list []types.UserDust) {
	store := k.getUserDustStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.UserDust
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetUserDust(ctx sdk.Context, address, denom string) (ud types.UserDust, found bool) {
	store := k.getUserDustStore(ctx)

	key := types.UserDustKey(address, denom)
	b := store.Get(key)
	if b == nil {
		return ud, false
	}

	k.cdc.MustUnmarshal(b, &ud)

	return ud, true
}

func (k Keeper) SetUserDust(ctx sdk.Context, ud types.UserDust) {
	store := k.getUserDustStore(ctx)
	b := k.cdc.MustMarshal(&ud)
	key := types.UserDustKey(ud.Owner, ud.Denom)
	store.Set(key, b)
}

func (k Keeper) RemoveUserDust(ctx sdk.Context, ud types.UserDust) {
	store := k.getUserDustStore(ctx)
	key := types.UserDustKey(ud.Owner, ud.Denom)
	store.Delete(key)
}

func (k Keeper) GetUserDustByOwner(ctx sdk.Context, address string) (list []types.UserDust) {
	store := k.getUserDustStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, types.UserDustKeyAddressPrefix(address))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.UserDust
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) StoreProcessedUserDust(ctx sdk.Context, userDust *types.UserDust, userDustDec *math.LegacyDec) {
	if userDust == nil {
		return
	}

	//if the decimal dust was not received take it from the structure
	if userDustDec == nil {
		userDustDec = &userDust.Amount
	}

	if userDustDec.IsPositive() {
		k.SetUserDust(ctx, *userDust)
	} else {
		k.RemoveUserDust(ctx, *userDust)
	}
}

func (k Keeper) CollectUserDust(ctx sdk.Context, address string, coin sdk.Coin, coinDust math.LegacyDec, isReceiver bool) (sdk.Coin, *types.UserDust, math.LegacyDec, error) {
	zeroDec := math.LegacyZeroDec()
	if coinDust.LTE(zeroDec) {
		return coin, nil, zeroDec, nil
	}

	storageDust, ok := k.GetUserDust(ctx, address, coin.Denom)
	if !ok {
		storageDust = types.UserDust{
			Owner:  address,
			Amount: math.LegacyZeroDec(),
			Denom:  coin.Denom,
		}
	}

	storageDustAmount := storageDust.Amount

	oneDec := math.LegacyOneDec()
	if isReceiver {
		//the receiver should also receive the dust
		//if new user total dust is greater than 1uDenom send the Int part of the dust to the user
		storageDustAmount = storageDustAmount.Add(coinDust)
		//check and send dust if it reached at least 1 uDenom
		if storageDustAmount.GTE(oneDec) {
			coin = coin.AddAmount(storageDustAmount.TruncateInt())
			storageDustAmount = storageDustAmount.Sub(storageDustAmount.TruncateDec())
		}
	} else {
		//if the address is a payer we need to subtract the dust from his pending dust or from his coins
		//check if we can obtain the coin dust from his dust balance
		if storageDustAmount.GTE(coinDust) {
			storageDustAmount = storageDustAmount.Sub(coinDust)
		} else {
			//he does not have enough dust, so we should take it from the coin amount
			//add 1 to coin amount for dust payment
			coin = coin.AddAmount(math.OneInt())
			//calculate the remaining dust from the 1 coin added before and add it to his storage dust
			remainingDust := oneDec.Sub(coinDust)
			storageDustAmount = storageDustAmount.Add(remainingDust)
		}
	}
	storageDust.Amount = storageDustAmount

	return coin, &storageDust, storageDustAmount, nil
}
