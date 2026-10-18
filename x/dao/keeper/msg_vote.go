package keeper

import (
	"context"
	"slices"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k msgServer) Vote(goCtx context.Context, msg *types.MsgVote) (*types.MsgVoteResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	proposal, found := k.GetProposal(ctx, msg.ProposalId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrProposalNotFound, "proposal %d", msg.ProposalId)
	}

	// voting closes at voting_end itself (D18)
	if proposal.Status != types.ProposalStatus_VOTING || ctx.BlockTime().Unix() >= proposal.VotingEnd {
		return nil, errorsmod.Wrapf(types.ErrVotingClosed, "proposal %d", proposal.Id)
	}

	voter := sdk.MustAccAddressFromBech32(msg.Voter)
	previous, revote := k.GetBallot(ctx, proposal.Id, voter)
	// a ballot zeroed by a member removal may be cast again with the same
	// option once the voter is a member again (D21)
	if revote && previous.Option == msg.Option && previous.Weight.IsPositive() {
		return nil, errorsmod.Wrapf(types.ErrSameVote, "%s already voted %s on proposal %d", voter, msg.Option, proposal.Id)
	}

	dao, found := k.GetDao(ctx, proposal.DaoId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao %d", proposal.DaoId)
	}

	weight, shares := k.voterWeight(ctx, dao, proposal, voter)
	if !weight.IsPositive() {
		return nil, errorsmod.Wrapf(types.ErrNoVotingPower, "%s has no voting power on proposal %d", voter, proposal.Id)
	}

	// a revote replaces the old ballot at the voter's current weight (D23)
	if revote {
		addToTally(&proposal, previous.Option, previous.Weight.Neg())
		for _, share := range previous.Shares {
			k.removeVoteIndex(ctx, []byte(share.Source), voter, proposal.Id)
		}
	}

	ballot := types.Ballot{
		ProposalId: proposal.Id,
		Voter:      voter.String(),
		Option:     msg.Option,
		Weight:     weight,
		Shares:     shares,
	}
	addToTally(&proposal, ballot.Option, ballot.Weight)
	for _, share := range shares {
		k.setVoteIndex(ctx, []byte(share.Source), voter, proposal.Id)
	}

	k.setBallot(ctx, voter, ballot)
	k.setProposal(ctx, proposal)

	if err := ctx.EventManager().EmitTypedEvent(&types.EventVote{
		ProposalId: proposal.Id,
		Voter:      ballot.Voter,
		Option:     ballot.Option,
		Weight:     ballot.Weight.String(),
	}); err != nil {
		return nil, err
	}

	return &types.MsgVoteResponse{}, nil
}

// voterWeight is the voter's weight on the proposal: 1 for a member of the
// DAO's current list added before the proposal was submitted, or the voter's
// stake in the proposal's pinned sources that joined before it was submitted,
// with one share per source. The DAO itself has no weight on its own
// proposals.
func (k Keeper) voterWeight(ctx sdk.Context, dao types.Dao, proposal types.Proposal, voter sdk.AccAddress) (math.Int, []types.BallotShare) {
	if dao.Type == types.DaoType_DAO_TYPE_STATIC_LIST {
		if k.isEligibleMember(ctx, dao, voter, proposal.SubmitHeight) {
			return math.OneInt(), nil
		}

		return math.ZeroInt(), nil
	}

	if voter.String() == dao.Address {
		return math.ZeroInt(), nil
	}

	weight, sources := k.eligibleWeight(ctx, proposal.PinnedElectorate, voter, proposal.SubmitHeight)
	shares := make([]types.BallotShare, 0, len(sources))
	for _, s := range sources {
		shares = append(shares, types.BallotShare{Source: string(s.source), Amount: s.amount})
	}

	return weight, shares
}

// addToTally adds amount (negative to subtract) to the option's running tally.
func addToTally(proposal *types.Proposal, option types.VoteOption, amount math.Int) {
	switch option {
	case types.VoteOption_YES:
		proposal.Yes = proposal.Yes.Add(amount)
	case types.VoteOption_NO:
		proposal.No = proposal.No.Add(amount)
	case types.VoteOption_ABSTAIN:
		proposal.Abstain = proposal.Abstain.Add(amount)
	}
}

// reduceBallots takes the source's share out of the address's ballots on
// proposals in voting, after the address left the source. Vote-index entries
// of proposals that left voting (or are gone) are only deleted (I6). Missing
// state is skipped: the Exit hooks that call it must not fail.
func (k Keeper) reduceBallots(ctx sdk.Context, source []byte, addr sdk.AccAddress) {
	for _, proposalId := range k.votedProposals(ctx, source, addr) {
		k.removeVoteIndex(ctx, source, addr, proposalId)

		proposal, found := k.GetProposal(ctx, proposalId)
		if !found || proposal.Status != types.ProposalStatus_VOTING {
			continue
		}

		ballot, found := k.GetBallot(ctx, proposalId, addr)
		if !found {
			continue
		}

		i := slices.IndexFunc(ballot.Shares, func(s types.BallotShare) bool { return s.Source == string(source) })
		if i < 0 {
			continue
		}

		amount := ballot.Shares[i].Amount
		ballot.Weight = ballot.Weight.Sub(amount)
		ballot.Shares = slices.Delete(ballot.Shares, i, i+1)
		addToTally(&proposal, ballot.Option, amount.Neg())

		k.setBallot(ctx, addr, ballot)
		k.setProposal(ctx, proposal)

		if err := ctx.EventManager().EmitTypedEvent(&types.EventBallotReduced{
			ProposalId: proposalId,
			Voter:      ballot.Voter,
			Source:     string(source),
			Amount:     amount.String(),
		}); err != nil {
			k.Logger().Error("dao exit hook: failed to emit event", "proposal", proposalId, "error", err)
		}
	}
}
