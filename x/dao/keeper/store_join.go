package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// GetJoinRecord returns the height the address joined the source at, if the
// source was pinned when it joined.
func (k Keeper) GetJoinRecord(ctx context.Context, source []byte, addr sdk.AccAddress) (types.JoinRecord, bool) {
	bz := k.getPrefixedStore(ctx, types.KeyPrefix(types.JoinKeyPrefix)).Get(types.JoinKey(source, addr))
	if bz == nil {
		return types.JoinRecord{}, false
	}

	return types.JoinRecord{
		Source:  string(source),
		Address: addr.String(),
		Height:  int64(sdk.BigEndianToUint64(bz)), //nolint:gosec // written from a non-negative block height
	}, true
}

func (k Keeper) setJoinRecord(ctx context.Context, source []byte, addr sdk.AccAddress, height int64) {
	//nolint:gosec // block heights are never negative
	k.getPrefixedStore(ctx, types.KeyPrefix(types.JoinKeyPrefix)).Set(types.JoinKey(source, addr), sdk.Uint64ToBigEndian(uint64(height)))
}

func (k Keeper) removeJoinRecord(ctx context.Context, source []byte, addr sdk.AccAddress) {
	k.getPrefixedStore(ctx, types.KeyPrefix(types.JoinKeyPrefix)).Delete(types.JoinKey(source, addr))
}

// GetAllJoinRecords returns every join record, by source and address.
func (k Keeper) GetAllJoinRecords(ctx context.Context) []types.JoinRecord {
	iterator := k.getPrefixedStore(ctx, types.KeyPrefix(types.JoinKeyPrefix)).Iterator(nil, nil)
	defer iterator.Close()

	var records []types.JoinRecord
	for ; iterator.Valid(); iterator.Next() {
		source, addr, _ := types.SplitSourceAddressKey(iterator.Key())
		records = append(records, types.JoinRecord{
			Source:  string(source),
			Address: addr.String(),
			Height:  int64(sdk.BigEndianToUint64(iterator.Value())), //nolint:gosec // written from a non-negative block height
		})
	}

	return records
}

// SetAllJoinRecords imports join records (genesis) as they are. They must pass
// GenesisState.Validate.
func (k Keeper) SetAllJoinRecords(ctx context.Context, records []types.JoinRecord) {
	for _, record := range records {
		k.setJoinRecord(ctx, []byte(record.Source), sdk.MustAccAddressFromBech32(record.Address), record.Height)
	}
}
