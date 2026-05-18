package keeper

import (
	"context"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// DepositOnPoll implements MsgDepositOnPoll. Mirror of MsgDeposit's
// orchestrator but on a Poll.
//
// Order of operations:
//  1. ValidateBasic.
//  2. Load Poll; reject if not in DEPOSIT_PERIOD.
//  3. Denom matches poll.deposit_snapshot.min_deposit.denom.
//  4. Move coins into the DAO's escrow; upsert PollDepositRecord row.
//  5. Update deposit_collected.
//  6. If collected >= min_deposit, transition to VOTING (dequeue at
//     deposit_deadline → flip → enqueue at NEW voting_end).
//  7. Emit event.
func (k msgServer) DepositOnPoll(goCtx context.Context, msg *types.MsgDepositOnPoll) (*types.MsgDepositOnPollResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	poll, ok := k.GetPoll(ctx, msg.DaoId, msg.PollId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrPollNotFound,
			"dao=%d poll=%d", msg.DaoId, msg.PollId)
	}
	if poll.Status != types.PollStatus_POLL_STATUS_DEPOSIT_PERIOD {
		return nil, errorsmod.Wrapf(types.ErrPollNotInDepositPeriod,
			"poll %d/%d is %s", poll.DaoId, poll.PollId, poll.Status)
	}
	if msg.Amount.Denom != poll.DepositSnapshot.MinDeposit.Denom {
		return nil, errorsmod.Wrapf(types.ErrInvalidDepositAmount,
			"amount denom %q != min_deposit denom %q",
			msg.Amount.Denom, poll.DepositSnapshot.MinDeposit.Denom)
	}

	depositor, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}

	dao, ok := k.GetDao(ctx, poll.DaoId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao_id=%d", poll.DaoId)
	}

	if err := k.collectPollDeposit(ctx, dao, poll.PollId, depositor, msg.Amount); err != nil {
		return nil, err
	}

	poll.DepositCollected = poll.DepositCollected.Add(msg.Amount)

	if poll.DepositCollected.Amount.GTE(poll.DepositSnapshot.MinDeposit.Amount) {
		// Dequeue with OLD status (DEPOSIT_PERIOD → deposit_deadline key);
		// flip; enqueue at NEW voting_end.
		k.DequeueExpiringPoll(ctx, poll)
		poll.Status = types.PollStatus_POLL_STATUS_VOTING
		poll.VotingEnd = ctx.BlockTime().Add(poll.VotingPeriodSnapshot)
	}

	if err := k.UpdatePoll(ctx, poll); err != nil {
		return nil, err
	}
	if poll.Status == types.PollStatus_POLL_STATUS_VOTING {
		k.EnqueueExpiringPoll(ctx, poll)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeDepositOnPoll,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", poll.DaoId)),
		sdk.NewAttribute(types.AttributeKeyPollID, fmt.Sprintf("%d", poll.PollId)),
		sdk.NewAttribute(types.AttributeKeyDepositor, msg.Depositor),
		sdk.NewAttribute(types.AttributeKeyDepositAmount, msg.Amount.String()),
		sdk.NewAttribute(types.AttributeKeyDepositCollected, poll.DepositCollected.String()),
		sdk.NewAttribute(types.AttributeKeyOutcome, poll.Status.String()),
		sdk.NewAttribute(types.AttributeKeyVotingEnd, poll.VotingEnd.Format(time.RFC3339Nano)),
	))

	return &types.MsgDepositOnPollResponse{}, nil
}
