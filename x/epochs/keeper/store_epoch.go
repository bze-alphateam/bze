package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"fmt"
	"github.com/bze-alphateam/bze/x/epochs/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"time"
)

func (k Keeper) getEpochsStore(ctx sdk.Context) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	return prefix.NewStore(storeAdapter, types.KeyPrefixEpoch)
}

// GetEpochInfo returns epoch info by identifier.
func (k Keeper) GetEpochInfo(ctx sdk.Context, identifier string) (epoch types.EpochInfo) {
	store := k.getEpochsStore(ctx)

	b := store.Get([]byte(identifier))
	if b == nil {
		return
	}

	k.cdc.MustUnmarshal(b, &epoch)

	return epoch
}

// AddEpochInfo adds a new epoch info. Will return an error if the epoch fails validation,
// or re-uses an existing identifier.
// This method also sets the start time if left unset, and sets the epoch start height.
func (k Keeper) AddEpochInfo(ctx sdk.Context, epoch types.EpochInfo) error {
	err := epoch.Validate()
	if err != nil {
		return err
	}

	// Check if identifier already exists
	if (k.GetEpochInfo(ctx, epoch.Identifier) != types.EpochInfo{}) {
		return fmt.Errorf("epoch with identifier %s already exists", epoch.Identifier)
	}

	// Initialize empty and default epoch values
	if epoch.StartTime.Equal(time.Time{}) {
		epoch.StartTime = ctx.BlockTime()
	}

	epoch.CurrentEpochStartHeight = ctx.BlockHeight()
	k.setEpochInfo(ctx, epoch)

	return nil
}

// setEpochInfo set epoch info.
func (k Keeper) setEpochInfo(ctx sdk.Context, epoch types.EpochInfo) {
	store := k.getEpochsStore(ctx)
	b := k.cdc.MustMarshal(&epoch)
	store.Set([]byte(epoch.Identifier), b)
}

// IterateEpochInfo iterate through epochs.
func (k Keeper) IterateEpochInfo(ctx sdk.Context, fn func(index int64, epochInfo types.EpochInfo) (stop bool)) {
	store := k.getEpochsStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	i := int64(0)
	for ; iterator.Valid(); iterator.Next() {
		epoch := types.EpochInfo{}
		k.cdc.MustUnmarshal(iterator.Value(), &epoch)
		stop := fn(i, epoch)

		if stop {
			break
		}
		i++
	}
}

// AllEpochInfos iterate through epochs to return all epochs info.
func (k Keeper) AllEpochInfos(ctx sdk.Context) []types.EpochInfo {
	var epochs []types.EpochInfo
	k.IterateEpochInfo(ctx, func(index int64, epochInfo types.EpochInfo) (stop bool) {
		epochs = append(epochs, epochInfo)

		return false
	})

	return epochs
}

func (k Keeper) GetEpochCountByIdentifier(ctx sdk.Context, identifier string) int64 {
	e := k.GetEpochInfo(ctx, identifier)

	return e.CurrentEpoch
}
