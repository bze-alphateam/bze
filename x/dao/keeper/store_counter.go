package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k Keeper) getCounter(ctx context.Context, key []byte) uint64 {
	bz := k.getPrefixedStore(ctx, types.KeyPrefix(types.CounterKeyPrefix)).Get(key)
	if bz == nil {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

func (k Keeper) setCounter(ctx context.Context, key []byte, value uint64) {
	k.getPrefixedStore(ctx, types.KeyPrefix(types.CounterKeyPrefix)).Set(key, sdk.Uint64ToBigEndian(value))
}

// GetDaoCounter returns the last dao id handed out (0 before the first DAO).
func (k Keeper) GetDaoCounter(ctx context.Context) uint64 {
	return k.getCounter(ctx, types.DaoCounterKey())
}

// SetDaoCounter sets the last dao id handed out.
func (k Keeper) SetDaoCounter(ctx context.Context, value uint64) {
	k.setCounter(ctx, types.DaoCounterKey(), value)
}

// GetProposalCounter returns the last proposal id handed out (0 before the first proposal).
func (k Keeper) GetProposalCounter(ctx context.Context) uint64 {
	return k.getCounter(ctx, types.ProposalCounterKey())
}

// SetProposalCounter sets the last proposal id handed out.
func (k Keeper) SetProposalCounter(ctx context.Context, value uint64) {
	k.setCounter(ctx, types.ProposalCounterKey(), value)
}
