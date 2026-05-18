package keeper

import (
	"context"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Deposit implements MsgDeposit.
//
// Order of operations:
//  1. ValidateBasic (signer + ids + positive amount).
//  2. Load the proposal; reject if missing or not in DEPOSIT_PERIOD.
//  3. amount denom must match proposal.deposit_snapshot.min_deposit.denom.
//  4. Move coins into the DAO's escrow and upsert the DepositRecord row.
//  5. Update deposit_collected on the proposal.
//  6. If collected >= min_deposit, transition to VOTING:
//       - dequeue at the deposit_deadline,
//       - set voting_end = blockTime + governance_snapshot.voting_period,
//       - set status = VOTING,
//       - enqueue at the new voting_end.
//  7. Emit event.
func (k msgServer) Deposit(goCtx context.Context, msg *types.MsgDeposit) (*types.MsgDepositResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	p, ok := k.GetProposal(ctx, msg.DaoId, msg.ProposalId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrProposalNotFound,
			"dao=%d proposal=%d", msg.DaoId, msg.ProposalId)
	}
	if p.Status != types.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD {
		return nil, errorsmod.Wrapf(types.ErrProposalNotInDepositPeriod,
			"proposal %d/%d is %s", p.DaoId, p.ProposalId, p.Status)
	}
	if msg.Amount.Denom != p.DepositSnapshot.MinDeposit.Denom {
		return nil, errorsmod.Wrapf(types.ErrInvalidDepositAmount,
			"amount denom %q != min_deposit denom %q",
			msg.Amount.Denom, p.DepositSnapshot.MinDeposit.Denom)
	}

	depositor, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}

	dao, ok := k.GetDao(ctx, p.DaoId)
	if !ok {
		// Defensive — the parent invariant is that a proposal's DAO exists.
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao_id=%d", p.DaoId)
	}

	if err := k.collectDeposit(ctx, dao, p.ProposalId, depositor, msg.Amount); err != nil {
		return nil, err
	}

	// Update the cached deposit_collected total on the proposal record.
	p.DepositCollected = p.DepositCollected.Add(msg.Amount)

	// Transition if we've now met the minimum.
	if p.DepositCollected.Amount.GTE(p.DepositSnapshot.MinDeposit.Amount) {
		// IMPORTANT: dequeue with the OLD status (DEPOSIT_PERIOD) so the
		// helper picks deposit_deadline; THEN flip status; THEN enqueue at
		// voting_end. See the comment on proposalDeadlineNs.
		k.DequeueExpiringProposal(ctx, p)
		p.Status = types.ProposalStatus_PROPOSAL_STATUS_VOTING
		// voting_end is anchored at the TRANSITION block time, not at
		// proposal creation. The proposer doesn't get a longer voting
		// window for procrastinating on deposits.
		p.VotingEnd = ctx.BlockTime().Add(p.GovernanceSnapshot.VotingPeriod)
	}

	if err := k.UpdateProposal(ctx, p); err != nil {
		return nil, err
	}
	if p.Status == types.ProposalStatus_PROPOSAL_STATUS_VOTING {
		// Enqueue at the new deadline (post-status-flip the helper picks
		// voting_end).
		k.EnqueueExpiringProposal(ctx, p)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeDeposit,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
		sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", p.ProposalId)),
		sdk.NewAttribute(types.AttributeKeyDepositor, msg.Depositor),
		sdk.NewAttribute(types.AttributeKeyDepositAmount, msg.Amount.String()),
		sdk.NewAttribute(types.AttributeKeyDepositCollected, p.DepositCollected.String()),
		sdk.NewAttribute(types.AttributeKeyOutcome, p.Status.String()),
		sdk.NewAttribute(types.AttributeKeyVotingEnd, p.VotingEnd.Format(time.RFC3339Nano)),
	))

	return &types.MsgDepositResponse{}, nil
}
