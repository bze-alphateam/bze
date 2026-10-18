package keeper

import (
	"context"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// GetMemberRecord returns the height the member was added to the DAO at, if it
// was added by MsgUpdateMembers.
func (k Keeper) GetMemberRecord(ctx context.Context, daoId uint64, member sdk.AccAddress) (types.MemberRecord, bool) {
	bz := k.getPrefixedStore(ctx, types.KeyPrefix(types.MemberKeyPrefix)).Get(types.MemberKey(daoId, member))
	if bz == nil {
		return types.MemberRecord{}, false
	}

	return types.MemberRecord{
		DaoId:   daoId,
		Address: member.String(),
		Height:  int64(sdk.BigEndianToUint64(bz)), //nolint:gosec // written from a non-negative block height
	}, true
}

func (k Keeper) setMemberRecord(ctx context.Context, daoId uint64, member sdk.AccAddress, height int64) {
	//nolint:gosec // block heights are never negative
	k.getPrefixedStore(ctx, types.KeyPrefix(types.MemberKeyPrefix)).Set(types.MemberKey(daoId, member), sdk.Uint64ToBigEndian(uint64(height)))
}

func (k Keeper) removeMemberRecord(ctx context.Context, daoId uint64, member sdk.AccAddress) {
	k.getPrefixedStore(ctx, types.KeyPrefix(types.MemberKeyPrefix)).Delete(types.MemberKey(daoId, member))
}

// isEligibleMember reports whether the address is a member of the DAO's
// current list that was there before refHeight: a member added in the
// reference block or later cannot vote on a proposal submitted at it, the
// rule stake follows through join records (D3).
func (k Keeper) isEligibleMember(ctx context.Context, dao types.Dao, addr sdk.AccAddress, refHeight int64) bool {
	if !slices.Contains(dao.Members, addr.String()) {
		return false
	}

	record, found := k.GetMemberRecord(ctx, dao.Id, addr)

	return !found || record.Height < refHeight
}

// GetAllMemberRecords returns every member record, by dao id and address.
func (k Keeper) GetAllMemberRecords(ctx context.Context) []types.MemberRecord {
	iterator := k.getPrefixedStore(ctx, types.KeyPrefix(types.MemberKeyPrefix)).Iterator(nil, nil)
	defer iterator.Close()

	var records []types.MemberRecord
	for ; iterator.Valid(); iterator.Next() {
		daoId, addr := types.SplitMemberKey(iterator.Key())
		records = append(records, types.MemberRecord{
			DaoId:   daoId,
			Address: addr.String(),
			Height:  int64(sdk.BigEndianToUint64(iterator.Value())), //nolint:gosec // written from a non-negative block height
		})
	}

	return records
}

// SetAllMemberRecords imports member records (genesis) as they are. They must
// pass GenesisState.Validate.
func (k Keeper) SetAllMemberRecords(ctx context.Context, records []types.MemberRecord) {
	for _, record := range records {
		k.setMemberRecord(ctx, record.DaoId, sdk.MustAccAddressFromBech32(record.Address), record.Height)
	}
}
