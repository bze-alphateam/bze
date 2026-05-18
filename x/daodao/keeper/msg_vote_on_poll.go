package keeper

import (
	"context"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// VoteOnPoll implements MsgVoteOnPoll.
//
// Order of operations:
//  1. ValidateBasic (signer + ids + non-empty selection).
//  2. Load Poll; reject if not in VOTING.
//  3. Validate selection set against the poll's stored choices and
//     max_selections (NOTA exclusivity / range / dedup).
//  4. Read voter's snapshot power; reject if 0.
//  5. Revote handling:
//       - If existing vote AND allow_revote_snapshot == false: reject.
//       - If existing vote AND allow_revote_snapshot == true: subtract
//         the old contribution from choice_power (total_voted_power
//         unchanged — still one distinct voter).
//       - If no existing vote: total_voted_power += power.
//  6. Add the new selection's power to each chosen choice_power[i].
//  7. Persist PollVote; persist Poll (tally update).
//  8. Emit event. No early-close — polls always run their full voting period.
func (k msgServer) VoteOnPoll(goCtx context.Context, msg *types.MsgVoteOnPoll) (*types.MsgVoteOnPollResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	poll, ok := k.GetPoll(ctx, msg.DaoId, msg.PollId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrPollNotFound,
			"dao=%d poll=%d", msg.DaoId, msg.PollId)
	}
	if poll.Status != types.PollStatus_POLL_STATUS_VOTING {
		return nil, errorsmod.Wrapf(types.ErrPollNotVoting,
			"poll %d/%d is %s", poll.DaoId, poll.PollId, poll.Status)
	}

	if err := types.ValidatePollSelection(msg.ChoiceIndices, len(poll.Choices), poll.MaxSelections, poll.IncludeNota); err != nil {
		return nil, err
	}

	voterAddr, err := sdk.AccAddressFromBech32(msg.Voter)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}

	power := k.SnapshotPower(ctx, poll.DaoId, poll.SnapshotId, voterAddr)
	if power == 0 {
		return nil, errorsmod.Wrapf(types.ErrNoVotingPower,
			"voter %s has no snapshot power on poll %d/%d",
			msg.Voter, poll.DaoId, poll.PollId)
	}

	prev, hadPrev := k.GetPollVote(ctx, poll.DaoId, poll.PollId, voterAddr)
	if hadPrev {
		if !poll.AllowRevoteSnapshot {
			return nil, errorsmod.Wrapf(types.ErrRevoteNotAllowed,
				"voter %s already voted on poll %d/%d", msg.Voter, poll.DaoId, poll.PollId)
		}
		// Subtract the prior contribution. Trust prev.Power as the
		// authoritative number (frozen snapshot).
		for _, idx := range prev.ChoiceIndices {
			poll.Tally.ChoicePower[idx] -= prev.Power
		}
		// total_voted_power unchanged — same distinct voter.
	} else {
		poll.Tally.TotalVotedPower += power
	}

	for _, idx := range msg.ChoiceIndices {
		poll.Tally.ChoicePower[idx] += power
	}

	newVote := types.PollVote{
		DaoId:         poll.DaoId,
		PollId:        poll.PollId,
		Voter:         msg.Voter,
		ChoiceIndices: msg.ChoiceIndices,
		Power:         power,
		VotedHeight:   ctx.BlockHeight(),
	}
	if err := k.SetPollVote(ctx, newVote); err != nil {
		return nil, err
	}
	if err := k.UpdatePoll(ctx, poll); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeVoteOnPoll,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", poll.DaoId)),
		sdk.NewAttribute(types.AttributeKeyPollID, fmt.Sprintf("%d", poll.PollId)),
		sdk.NewAttribute(types.AttributeKeyVoter, msg.Voter),
		sdk.NewAttribute(types.AttributeKeyChoiceIndices, joinUint32(msg.ChoiceIndices)),
		sdk.NewAttribute(types.AttributeKeyVotePower, fmt.Sprintf("%d", power)),
	))

	return &types.MsgVoteOnPollResponse{}, nil
}

// joinUint32 renders a slice of indices as "a,b,c" for event attributes.
func joinUint32(xs []uint32) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = fmt.Sprintf("%d", x)
	}
	return strings.Join(parts, ",")
}
