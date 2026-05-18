package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// UpdateMembers adds and/or removes members of a STATIC DAO. Admin-gated.
//
// Order of operations:
//  1. ValidateBasic (signer + dao_id + per-list rules)
//  2. assertAdmin — only the DAO's current admin may update members
//  3. Require STATIC backend (REWARD_STAKED's membership lives in rewards)
//  4. applyMemberUpdates does removes-then-upserts and maintains the
//     cached total power; returns post-update count
//  5. Post-update count must be in [1, MaxStaticMembers] — STATIC DAOs
//     cannot be left empty, and the cap that SnapshotAll iteration relies
//     on must hold across UPDATE flows too (ValidateBasic only sizes the
//     `add` slice, not the resulting set)
//  6. Emit event
func (k msgServer) UpdateMembers(goCtx context.Context, msg *types.MsgUpdateMembers) (*types.MsgUpdateMembersResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, err := k.assertAdmin(ctx, msg.DaoId, msg.Authority)
	if err != nil {
		return nil, err
	}
	if dao.VotingBackend != types.VotingBackendType_VOTING_BACKEND_STATIC {
		return nil, errorsmod.Wrapf(types.ErrNotStaticBackend,
			"DAO %d is %s; member updates are STATIC-only", dao.Id, dao.VotingBackend)
	}

	postCount, err := k.applyMemberUpdates(ctx, dao.Id, msg.Add, msg.Remove)
	if err != nil {
		return nil, err
	}
	if postCount == 0 {
		return nil, errorsmod.Wrapf(types.ErrEmptyMembership,
			"update would leave DAO %d with zero members", dao.Id)
	}
	// Enforce the upper bound on the resulting member set. ValidateBasic
	// only caps len(msg.Add) at MaxStaticMembers, so an admin could otherwise
	// grow a near-cap DAO past MaxStaticMembers over successive updates,
	// breaking the iteration-cost bound SnapshotAll relies on. The tx is
	// reverted on error (CacheContext rollback), so the partial writes
	// applyMemberUpdates produced before this check do not persist.
	if postCount > uint64(types.MaxStaticMembers) {
		return nil, errorsmod.Wrapf(types.ErrInvalidStaticMembers,
			"post-update member count %d exceeds max %d", postCount, types.MaxStaticMembers)
	}

	total := k.getStaticTotalPower(ctx, dao.Id)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateMembers,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", dao.Id)),
		sdk.NewAttribute(types.AttributeKeyAdmin, msg.Authority),
		sdk.NewAttribute(types.AttributeKeyAddedCount, fmt.Sprintf("%d", len(msg.Add))),
		sdk.NewAttribute(types.AttributeKeyRemovedCount, fmt.Sprintf("%d", len(msg.Remove))),
		sdk.NewAttribute(types.AttributeKeyTotalPower, fmt.Sprintf("%d", total)),
	))

	return &types.MsgUpdateMembersResponse{}, nil
}
