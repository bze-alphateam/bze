package keeper

import (
	"context"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// CreatePoll implements MsgCreatePoll.
//
// Order of operations:
//  1. ValidateBasic (proposer, dao_id, title/description, choices,
//     max_selections, quorum_bps, initial_deposit Coin shape).
//  2. Load DAO.
//  3. Append NOTA to the choice list if include_nota = true. (The
//     stored Poll.choices is the post-append form.)
//  4. Validate initial_deposit denom matches DAO's min_deposit denom.
//  5. Resolve voting power; non-member must attach >= min_deposit.
//  6. Take a snapshot.
//  7. Move initial_deposit to the shared escrow; persist a poll
//     DepositRecord row.
//  8. Persist the Poll with:
//       status = VOTING if deposit met else DEPOSIT_PERIOD
//       deposit_deadline = blockTime + deposit.deposit_period
//       voting_end       = blockTime + governance.voting_period
//       tally.choice_power = make(len(choices))
//       tally.total_power  = snapshot total
//  9. Enqueue on the end-blocker queue.
// 10. Emit event.
func (k msgServer) CreatePoll(goCtx context.Context, msg *types.MsgCreatePoll) (*types.MsgCreatePollResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, ok := k.GetDao(ctx, msg.DaoId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao_id=%d", msg.DaoId)
	}

	// Construct the stored choices slice: user labels + optional NOTA tail.
	choices := append([]string{}, msg.Choices...)
	if msg.IncludeNota {
		choices = append(choices, types.NotaLabel)
	}

	if msg.InitialDeposit.Denom != dao.Deposit.MinDeposit.Denom {
		return nil, errorsmod.Wrapf(types.ErrInvalidDepositAmount,
			"initial_deposit denom %q != DAO min_deposit denom %q",
			msg.InitialDeposit.Denom, dao.Deposit.MinDeposit.Denom)
	}

	proposerAddr, err := sdk.AccAddressFromBech32(msg.Proposer)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}

	backend, err := k.backendFor(dao)
	if err != nil {
		return nil, err
	}
	power, err := backend.Power(ctx, dao, proposerAddr)
	if err != nil {
		return nil, err
	}
	isMember := power > 0
	if !isMember {
		if msg.InitialDeposit.Amount.LT(dao.Deposit.MinDeposit.Amount) {
			return nil, errorsmod.Wrapf(types.ErrNoVotingPower,
				"non-member proposer %s must attach >= min_deposit (%s); got %s",
				msg.Proposer, dao.Deposit.MinDeposit.String(), msg.InitialDeposit.String())
		}
	}

	// Snapshot — voting power locked in even when starting in DEPOSIT_PERIOD.
	snapshotID, err := k.CreateSnapshot(ctx, dao)
	if err != nil {
		return nil, fmt.Errorf("snapshot at poll create: %w", err)
	}
	total := k.SnapshotTotal(ctx, dao.Id, snapshotID)

	pollID := k.ConsumeNextPollID(ctx, dao.Id)
	now := ctx.BlockTime()
	depositDeadline := now.Add(dao.Deposit.DepositPeriod)
	votingEnd := now.Add(dao.Governance.VotingPeriod)

	if err := k.collectInitialPollDeposit(ctx, dao, pollID, proposerAddr, msg.InitialDeposit); err != nil {
		return nil, err
	}

	status := types.PollStatus_POLL_STATUS_DEPOSIT_PERIOD
	if msg.InitialDeposit.Amount.GTE(dao.Deposit.MinDeposit.Amount) {
		status = types.PollStatus_POLL_STATUS_VOTING
	}

	p := types.Poll{
		DaoId:                dao.Id,
		PollId:               pollID,
		Proposer:             msg.Proposer,
		Title:                msg.Title,
		Description:          msg.Description,
		Choices:              choices,
		MaxSelections:        msg.MaxSelections,
		QuorumBps:            msg.QuorumBps,
		IncludeNota:          msg.IncludeNota,
		SnapshotId:           snapshotID,
		CreatedHeight:        ctx.BlockHeight(),
		DepositDeadline:      depositDeadline,
		VotingEnd:            votingEnd,
		Status:               status,
		VotingPeriodSnapshot: dao.Governance.VotingPeriod,
		AllowRevoteSnapshot:  dao.Governance.AllowRevote,
		DepositSnapshot:      dao.Deposit,
		Tally: types.PollTally{
			ChoicePower:     make([]uint64, len(choices)),
			TotalVotedPower: 0,
			TotalPower:      total,
		},
		DepositCollected: msg.InitialDeposit,
	}
	k.SetPollNew(ctx, p)
	k.EnqueueExpiringPoll(ctx, p)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreatePoll,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", dao.Id)),
		sdk.NewAttribute(types.AttributeKeyPollID, fmt.Sprintf("%d", pollID)),
		sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
		sdk.NewAttribute(types.AttributeKeySnapshotID, fmt.Sprintf("%d", snapshotID)),
		sdk.NewAttribute(types.AttributeKeyTotalPower, fmt.Sprintf("%d", total)),
		sdk.NewAttribute(types.AttributeKeyVotingEnd, votingEnd.Format(time.RFC3339Nano)),
		sdk.NewAttribute(types.AttributeKeyDepositCollected, msg.InitialDeposit.String()),
		sdk.NewAttribute(types.AttributeKeyDepositDeadline, depositDeadline.Format(time.RFC3339Nano)),
	))
	return &types.MsgCreatePollResponse{PollId: pollID}, nil
}
