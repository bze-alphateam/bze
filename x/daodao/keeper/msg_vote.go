package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Vote implements MsgVote.
//
// Order of operations:
//  1. ValidateBasic (signer + option in YES/NO/ABSTAIN + ids non-zero).
//  2. Load the proposal; reject if missing or not in VOTING status.
//  3. Read voter's snapshot power; reject if zero.
//  4. If voter already has a Vote on this proposal:
//       - reject when governance_snapshot.allow_revote == false.
//       - otherwise subtract the old option's power from the tally.
//  5. Add the new option's power to the tally; persist Vote.
//  6. Persist the updated Proposal.
//  7. If revote is disabled, run the early-close check; transition + dequeue
//     if the outcome is locked in.
//  8. Emit event.
func (k msgServer) Vote(goCtx context.Context, msg *types.MsgVote) (*types.MsgVoteResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	p, ok := k.GetProposal(ctx, msg.DaoId, msg.ProposalId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrProposalNotFound,
			"dao=%d proposal=%d", msg.DaoId, msg.ProposalId)
	}
	if p.Status != types.ProposalStatus_PROPOSAL_STATUS_VOTING {
		return nil, errorsmod.Wrapf(types.ErrProposalNotVoting,
			"proposal %d/%d is %s", p.DaoId, p.ProposalId, p.Status)
	}

	voterAddr, err := sdk.AccAddressFromBech32(msg.Voter)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}

	// Snapshot read: power at proposal creation, not current. This is what
	// makes the vote stable against post-creation membership / stake moves.
	power := k.SnapshotPower(ctx, p.DaoId, p.SnapshotId, voterAddr)
	if power == 0 {
		return nil, errorsmod.Wrapf(types.ErrNoVotingPower,
			"voter %s has no snapshot power on proposal %d/%d",
			msg.Voter, p.DaoId, p.ProposalId)
	}

	prev, hadPrev := k.GetVote(ctx, p.DaoId, p.ProposalId, voterAddr)
	if hadPrev {
		if !p.GovernanceSnapshot.AllowRevote {
			return nil, errorsmod.Wrapf(types.ErrRevoteNotAllowed,
				"voter %s already voted on proposal %d/%d", msg.Voter, p.DaoId, p.ProposalId)
		}
		// Subtract the previous contribution. We trust prev.Power as the
		// authoritative number — it was the snapshot power at the time of
		// the original vote, which is necessarily the same value we'd read
		// today (snapshot is immutable).
		subtractTally(&p.Tally, prev.Option, prev.Power)
	}

	addTally(&p.Tally, msg.Option, power)

	newVote := types.Vote{
		DaoId:       p.DaoId,
		ProposalId:  p.ProposalId,
		Voter:       msg.Voter,
		Option:      msg.Option,
		Power:       power,
		VotedHeight: ctx.BlockHeight(),
	}
	if err := k.SetVote(ctx, newVote); err != nil {
		return nil, err
	}
	if err := k.UpdateProposal(ctx, p); err != nil {
		return nil, err
	}

	// Early-close only fires when revote is disabled — otherwise any vote
	// could be undone and a "locked in" outcome isn't actually locked in.
	if !p.GovernanceSnapshot.AllowRevote {
		decision := checkEarlyClose(p.GovernanceSnapshot, p.Tally)
		if decision != earlyCloseNone {
			// Dequeue first while status is still VOTING (proposalDeadlineNs
			// uses status to pick the right deadline key).
			k.DequeueExpiringProposal(ctx, p)
			if decision == earlyClosePass {
				p.Status = types.ProposalStatus_PROPOSAL_STATUS_PASSED
			} else {
				p.Status = types.ProposalStatus_PROPOSAL_STATUS_REJECTED
			}
			if err := k.UpdateProposal(ctx, p); err != nil {
				return nil, err
			}
			// Epic 4: route deposits per voting_refund_policy at terminal
			// status — same call the end-blocker uses for the timeout path.
			if err := k.handleTerminalDeposits(ctx, p); err != nil {
				return nil, err
			}

			ctx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventTypeProposalEarlyClosed,
				sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
				sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", p.ProposalId)),
				sdk.NewAttribute(types.AttributeKeyOutcome, p.Status.String()),
			))
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeVote,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
		sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", p.ProposalId)),
		sdk.NewAttribute(types.AttributeKeyVoter, msg.Voter),
		sdk.NewAttribute(types.AttributeKeyVoteOption, msg.Option.String()),
		sdk.NewAttribute(types.AttributeKeyVotePower, fmt.Sprintf("%d", power)),
	))

	return &types.MsgVoteResponse{}, nil
}

// addTally adds `power` to the bucket selected by `opt`.
func addTally(t *types.Tally, opt types.VoteOption, power uint64) {
	switch opt {
	case types.VoteOption_VOTE_OPTION_YES:
		t.YesPower += power
	case types.VoteOption_VOTE_OPTION_NO:
		t.NoPower += power
	case types.VoteOption_VOTE_OPTION_ABSTAIN:
		t.AbstainPower += power
	}
}

// subtractTally subtracts `power` from the bucket selected by `opt`.
// Assumes prior addTally accounted for the same (opt, power) pair.
func subtractTally(t *types.Tally, opt types.VoteOption, power uint64) {
	switch opt {
	case types.VoteOption_VOTE_OPTION_YES:
		t.YesPower -= power
	case types.VoteOption_VOTE_OPTION_NO:
		t.NoPower -= power
	case types.VoteOption_VOTE_OPTION_ABSTAIN:
		t.AbstainPower -= power
	}
}
