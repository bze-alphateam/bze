package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// EndBlock finalizes every proposal whose deadline has passed by the
// current block time. A proposal's deadline is:
//
//   DEPOSIT_PERIOD → deposit_deadline. Failing to reach min_deposit by
//     this point transitions to REJECTED_NO_DEPOSIT and forfeits all
//     collected deposits per deposit_snapshot.forfeit_destination.
//   VOTING → voting_end. Tally is computed against the frozen
//     governance_snapshot and the proposal transitions to PASSED or
//     REJECTED; deposit_snapshot.voting_refund_policy then decides
//     refund vs. forfeit.
//
// Buffer-then-write pattern: collect expired proposals first, then
// mutate state. Mutating mid-iteration on Cosmos's IAVL is technically
// allowed but produces undefined ordering.
//
// Epic 6: the function ALSO drains the parallel ExpiringPollKey queue
// after the proposal pass — polls have a separate queue (different key
// prefix) so the dispatch fan-out stays clean per family.
func (k Keeper) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := uint64(sdkCtx.BlockTime().UnixNano())

	// --- Proposals ---
	var expired []types.Proposal
	k.IterateExpiredProposals(ctx, now, func(p types.Proposal) bool {
		expired = append(expired, p)
		return false
	})

	for _, p := range expired {
		switch p.Status {
		case types.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD:
			if err := k.finalizeDepositPeriodExpiry(ctx, p); err != nil {
				return err
			}
		case types.ProposalStatus_PROPOSAL_STATUS_VOTING:
			if err := k.finalizeVotingExpiry(ctx, p); err != nil {
				return err
			}
		default:
			// Defensive: only DEPOSIT_PERIOD / VOTING proposals belong on
			// the queue. Stale queue rows for terminal proposals are
			// silently dropped (we still issue a dequeue so the bad row
			// doesn't reappear on the next block).
			k.DequeueExpiringProposal(ctx, p)
		}
	}

	// --- Polls (Epic 6) ---
	var expiredPolls []types.Poll
	k.IterateExpiredPolls(ctx, now, func(p types.Poll) bool {
		expiredPolls = append(expiredPolls, p)
		return false
	})
	for _, p := range expiredPolls {
		switch p.Status {
		case types.PollStatus_POLL_STATUS_DEPOSIT_PERIOD:
			if err := k.finalizePollDepositPeriodExpiry(ctx, p); err != nil {
				return err
			}
		case types.PollStatus_POLL_STATUS_VOTING:
			if err := k.finalizePollVotingExpiry(ctx, p); err != nil {
				return err
			}
		default:
			k.DequeueExpiringPoll(ctx, p)
		}
	}
	return nil
}

// finalizePollDepositPeriodExpiry handles a poll whose deposit_deadline
// passed without reaching min_deposit. Transitions to
// REJECTED_NO_DEPOSIT and forfeits all collected deposits.
func (k Keeper) finalizePollDepositPeriodExpiry(ctx context.Context, p types.Poll) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.DequeueExpiringPoll(ctx, p)
	p.Status = types.PollStatus_POLL_STATUS_REJECTED_NO_DEPOSIT
	if err := k.UpdatePoll(ctx, p); err != nil {
		return fmt.Errorf("end-blocker: update poll %d/%d: %w", p.DaoId, p.PollId, err)
	}
	if err := k.handlePollTerminalDeposits(ctx, p); err != nil {
		return fmt.Errorf("end-blocker: disburse poll %d/%d: %w", p.DaoId, p.PollId, err)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypePollDepositExpired,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
		sdk.NewAttribute(types.AttributeKeyPollID, fmt.Sprintf("%d", p.PollId)),
		sdk.NewAttribute(types.AttributeKeyDepositCollected, p.DepositCollected.String()),
		sdk.NewAttribute(types.AttributeKeyMinDeposit, p.DepositSnapshot.MinDeposit.String()),
	))
	return nil
}

// finalizePollVotingExpiry runs computePollOutcome on a VOTING poll and
// applies the resulting status + winning_choice_index, then routes
// deposits per voting_refund_policy.
func (k Keeper) finalizePollVotingExpiry(ctx context.Context, p types.Poll) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	outcome := types.ComputePollOutcome(p)
	k.DequeueExpiringPoll(ctx, p)
	p.Status = outcome.Status
	if outcome.Status == types.PollStatus_POLL_STATUS_CONCLUDED {
		p.WinningChoiceIndex = outcome.WinningChoiceIndex
	}
	if err := k.UpdatePoll(ctx, p); err != nil {
		return fmt.Errorf("end-blocker: update poll %d/%d: %w", p.DaoId, p.PollId, err)
	}
	if err := k.handlePollTerminalDeposits(ctx, p); err != nil {
		return fmt.Errorf("end-blocker: disburse poll %d/%d: %w", p.DaoId, p.PollId, err)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypePollFinalized,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
		sdk.NewAttribute(types.AttributeKeyPollID, fmt.Sprintf("%d", p.PollId)),
		sdk.NewAttribute(types.AttributeKeyOutcome, p.Status.String()),
		sdk.NewAttribute(types.AttributeKeyWinningChoiceIndex, fmt.Sprintf("%d", p.WinningChoiceIndex)),
		sdk.NewAttribute(types.AttributeKeyTotalPower, fmt.Sprintf("%d", p.Tally.TotalPower)),
	))
	return nil
}

// finalizeDepositPeriodExpiry handles a proposal whose deposit_deadline
// has passed without reaching min_deposit. Transitions to
// REJECTED_NO_DEPOSIT and forfeits all collected deposits.
func (k Keeper) finalizeDepositPeriodExpiry(ctx context.Context, p types.Proposal) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Dequeue BEFORE flipping status so proposalDeadlineNs picks the right
	// (deposit_deadline) key.
	k.DequeueExpiringProposal(ctx, p)
	p.Status = types.ProposalStatus_PROPOSAL_STATUS_REJECTED_NO_DEPOSIT
	if err := k.UpdateProposal(ctx, p); err != nil {
		return fmt.Errorf("end-blocker: update proposal %d/%d: %w", p.DaoId, p.ProposalId, err)
	}
	if err := k.handleTerminalDeposits(ctx, p); err != nil {
		return fmt.Errorf("end-blocker: disburse %d/%d: %w", p.DaoId, p.ProposalId, err)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeDepositPeriodExpired,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
		sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", p.ProposalId)),
		sdk.NewAttribute(types.AttributeKeyDepositCollected, p.DepositCollected.String()),
		sdk.NewAttribute(types.AttributeKeyMinDeposit, p.DepositSnapshot.MinDeposit.String()),
	))
	return nil
}

// finalizeVotingExpiry handles a proposal whose voting_end has passed:
// compute outcome, transition to PASSED or REJECTED, route deposits per
// voting_refund_policy.
func (k Keeper) finalizeVotingExpiry(ctx context.Context, p types.Proposal) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	outcome := types.ComputeOutcome(p.GovernanceSnapshot, p.Tally)
	k.DequeueExpiringProposal(ctx, p)
	if outcome == types.OutcomePass {
		p.Status = types.ProposalStatus_PROPOSAL_STATUS_PASSED
	} else {
		p.Status = types.ProposalStatus_PROPOSAL_STATUS_REJECTED
	}
	if err := k.UpdateProposal(ctx, p); err != nil {
		return fmt.Errorf("end-blocker: update proposal %d/%d: %w", p.DaoId, p.ProposalId, err)
	}
	if err := k.handleTerminalDeposits(ctx, p); err != nil {
		return fmt.Errorf("end-blocker: disburse %d/%d: %w", p.DaoId, p.ProposalId, err)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeProposalFinalized,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
		sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", p.ProposalId)),
		sdk.NewAttribute(types.AttributeKeyOutcome, p.Status.String()),
		sdk.NewAttribute(types.AttributeKeyYesPower, fmt.Sprintf("%d", p.Tally.YesPower)),
		sdk.NewAttribute(types.AttributeKeyNoPower, fmt.Sprintf("%d", p.Tally.NoPower)),
		sdk.NewAttribute(types.AttributeKeyAbstainPower, fmt.Sprintf("%d", p.Tally.AbstainPower)),
		sdk.NewAttribute(types.AttributeKeyTotalPower, fmt.Sprintf("%d", p.Tally.TotalPower)),
	))
	return nil
}
