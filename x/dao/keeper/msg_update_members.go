package keeper

import (
	"context"
	"slices"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k msgServer) UpdateMembers(goCtx context.Context, msg *types.MsgUpdateMembers) (*types.MsgUpdateMembersResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, found := k.GetDao(ctx, msg.DaoId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao %d", msg.DaoId)
	}

	// only the DAO account: its voters through a proposal, or its admin
	// through MsgAdminExecute, never the admin signing directly
	authority := sdk.MustAccAddressFromBech32(msg.Authority)
	if authority.String() != dao.Address {
		return nil, errorsmod.Wrapf(types.ErrUnauthorizedSigner, "expected dao %d's address %s, got %s", dao.Id, dao.Address, authority)
	}

	if dao.Type != types.DaoType_DAO_TYPE_STATIC_LIST {
		return nil, errorsmod.Wrapf(types.ErrInvalidDaoType, "dao %d is not a static-list dao", dao.Id)
	}

	added := canonicalAddresses(msg.Add)
	removed := canonicalAddresses(msg.Remove)

	for _, addr := range added {
		if addr == dao.Address {
			return nil, errorsmod.Wrapf(types.ErrInvalidMembers, "dao %d cannot be a member of itself", dao.Id)
		}

		if slices.Contains(dao.Members, addr) {
			return nil, errorsmod.Wrapf(types.ErrInvalidMembers, "%s is already a member of dao %d", addr, dao.Id)
		}
	}

	for _, addr := range removed {
		if !slices.Contains(dao.Members, addr) {
			return nil, errorsmod.Wrapf(types.ErrInvalidMembers, "%s is not a member of dao %d", addr, dao.Id)
		}
	}

	members := slices.DeleteFunc(slices.Clone(dao.Members), func(m string) bool { return slices.Contains(removed, m) })
	members = append(members, added...)

	// an empty list leaves no voter, so it also keeps a self-owned DAO live (D11)
	if len(members) == 0 {
		return nil, errorsmod.Wrapf(types.ErrInvalidMembers, "dao %d needs at least one member", dao.Id)
	}

	if maxMembers := k.GetParams(ctx).MaxStaticMembers; uint64(len(members)) > maxMembers {
		return nil, errorsmod.Wrapf(types.ErrInvalidMembers, "a static-list dao can have at most %d members, got %d", maxMembers, len(members))
	}

	dao.Members = members
	k.setDao(ctx, dao)

	// removed members stop counting at once (D21); added ones vote only on
	// proposals submitted after this block, as stake that joins does (D3)
	for _, addr := range removed {
		member := sdk.MustAccAddressFromBech32(addr)
		k.removeMemberRecord(ctx, dao.Id, member)
		if err := k.zeroMemberBallots(ctx, dao.Id, member); err != nil {
			return nil, err
		}
	}

	for _, addr := range added {
		k.setMemberRecord(ctx, dao.Id, sdk.MustAccAddressFromBech32(addr), ctx.BlockHeight())
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.EventDaoMembersUpdated{
		Id:      dao.Id,
		Added:   added,
		Removed: removed,
	}); err != nil {
		return nil, err
	}

	return &types.MsgUpdateMembersResponse{}, nil
}

// zeroMemberBallots takes a removed member's ballots on the DAO's proposals
// still being voted on out of their tallies. The ballot stays, at weight 0, so its record of
// the vote remains. The scan is bounded by max_open_proposals.
func (k Keeper) zeroMemberBallots(ctx sdk.Context, daoId uint64, member sdk.AccAddress) error {
	for _, proposalId := range k.openProposalIds(ctx, daoId) {
		ballot, found := k.GetBallot(ctx, proposalId, member)
		if !found || !ballot.Weight.IsPositive() {
			continue
		}

		// from voting_end on the tally is final, as for exits
		proposal, found := k.GetProposal(ctx, proposalId)
		if !found || proposal.VotingEnded(ctx.BlockTime()) {
			continue
		}

		amount := ballot.Weight
		addToTally(&proposal, ballot.Option, amount.Neg())
		ballot.Weight = ballot.Weight.Sub(amount)

		k.setBallot(ctx, member, ballot)
		k.setProposal(ctx, proposal)

		if err := ctx.EventManager().EmitTypedEvent(&types.EventBallotReduced{
			ProposalId: proposalId,
			Voter:      ballot.Voter,
			Amount:     amount.String(),
		}); err != nil {
			return err
		}
	}

	return nil
}

// openProposalIds returns the ids of the DAO's proposals in voting.
func (k Keeper) openProposalIds(ctx context.Context, daoId uint64) []uint64 {
	store := k.getPrefixedStore(ctx, types.KeyPrefix(types.OpenProposalKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, types.DaoProposalsPrefix(daoId))
	defer iterator.Close()

	var ids []uint64
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		ids = append(ids, sdk.BigEndianToUint64(key[len(key)-8:]))
	}

	return ids
}
