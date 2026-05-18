package keeper

import (
	"context"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// CreateProposal implements MsgCreateProposal.
//
// Order of operations:
//  1. ValidateBasic (proposer + dao_id + title/description + msgs decode +
//     initial_deposit is a valid Coin).
//  2. Load the DAO; reject if missing.
//  3. Param-bound msgs cardinality check.
//  4. initial_deposit denom validation (must equal DAO's min_deposit denom).
//  5. Resolve voting power and apply Epic 4 submission gating:
//       members (power > 0): any initial_deposit.amount accepted.
//       non-members:         initial_deposit.amount >= min_deposit.amount required.
//  6. Take a snapshot via the DAO's voting backend.
//  7. Move initial_deposit (if > 0) from proposer to the DAO's escrow and
//     persist a DepositRecord row.
//  8. Allocate proposal id; persist Proposal with:
//       deposit_collected = initial_deposit
//       deposit_deadline  = blockTime + deposit_snapshot.deposit_period
//       status            = VOTING if initial_deposit >= min_deposit
//                           else DEPOSIT_PERIOD
//  9. Enqueue on the end-blocker timer at the appropriate deadline.
//  10. Emit event.
func (k msgServer) CreateProposal(goCtx context.Context, msg *types.MsgCreateProposal) (*types.MsgCreateProposalResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, ok := k.GetDao(ctx, msg.DaoId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao_id=%d", msg.DaoId)
	}

	params := k.GetParams(ctx)
	if uint32(len(msg.Msgs)) > params.MaxMsgsPerProposal {
		return nil, errorsmod.Wrapf(types.ErrInvalidProposalContent,
			"msgs has %d entries; chain cap is %d (Params.max_msgs_per_proposal)",
			len(msg.Msgs), params.MaxMsgsPerProposal)
	}

	// Epic 5: every entry of msgs[] must declare the DAO's account address
	// as its sole signer. Run this BEFORE persistence so a bad bundle
	// can never reach VOTING. The same check fires again at
	// MsgExecuteProposal time as defense-in-depth.
	if err := k.validateProposalMsgSigners(msg.DaoId, msg.Msgs); err != nil {
		return nil, err
	}

	proposerAddr, err := sdk.AccAddressFromBech32(msg.Proposer)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}

	// initial_deposit denom must match the DAO's configured min_deposit
	// denom. We allow a zero-amount initial_deposit with the right denom
	// (member submission path) AND with the wrong denom is rejected
	// regardless of amount — there's no semantic for a "different denom"
	// zero deposit, and accepting it would let a non-member submit with
	// effectively no escrow tracking.
	if msg.InitialDeposit.Denom != dao.Deposit.MinDeposit.Denom {
		return nil, errorsmod.Wrapf(types.ErrInvalidDepositAmount,
			"initial_deposit denom %q != DAO min_deposit denom %q",
			msg.InitialDeposit.Denom, dao.Deposit.MinDeposit.Denom)
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
	// Non-members must include >= min_deposit upfront. Members may submit
	// at any amount including zero (the proposal then starts in
	// DEPOSIT_PERIOD waiting for top-ups).
	if !isMember {
		if msg.InitialDeposit.Amount.LT(dao.Deposit.MinDeposit.Amount) {
			return nil, errorsmod.Wrapf(types.ErrNoVotingPower,
				"non-member proposer %s must attach >= min_deposit (%s); got %s",
				msg.Proposer, dao.Deposit.MinDeposit.String(), msg.InitialDeposit.String())
		}
	}

	// Snapshot first — even when starting in DEPOSIT_PERIOD, we snapshot at
	// creation so voting power is locked in. README D14: predictability over
	// late-binding.
	snapshotID, err := k.CreateSnapshot(ctx, dao)
	if err != nil {
		return nil, fmt.Errorf("snapshot at proposal create: %w", err)
	}
	total := k.SnapshotTotal(ctx, dao.Id, snapshotID)

	proposalID := k.ConsumeNextProposalID(ctx, dao.Id)
	now := ctx.BlockTime()
	depositDeadline := now.Add(dao.Deposit.DepositPeriod)
	votingEnd := now.Add(dao.Governance.VotingPeriod)

	// Move the initial deposit into escrow. SendCoins fails if the proposer
	// doesn't have the spendable balance; we surface that as-is.
	if err := k.collectInitialDeposit(ctx, dao, proposalID, proposerAddr, msg.InitialDeposit); err != nil {
		return nil, err
	}

	// Branch on whether the initial deposit alone met the minimum. If it
	// did, skip DEPOSIT_PERIOD entirely; otherwise start at DEPOSIT_PERIOD
	// and wait for top-ups or expiry.
	status := types.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD
	if msg.InitialDeposit.Amount.GTE(dao.Deposit.MinDeposit.Amount) {
		status = types.ProposalStatus_PROPOSAL_STATUS_VOTING
	}

	p := types.Proposal{
		DaoId:              dao.Id,
		ProposalId:         proposalID,
		Proposer:           msg.Proposer,
		Title:              msg.Title,
		Description:        msg.Description,
		Msgs:               msg.Msgs,
		SnapshotId:         snapshotID,
		CreatedHeight:      ctx.BlockHeight(),
		VotingEnd:          votingEnd,
		Status:             status,
		Tally:              types.Tally{TotalPower: total},
		GovernanceSnapshot: dao.Governance,
		DepositCollected:   msg.InitialDeposit,
		DepositDeadline:    depositDeadline,
		DepositSnapshot:    dao.Deposit,
	}
	k.SetProposalNew(ctx, p)
	k.EnqueueExpiringProposal(ctx, p)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreateProposal,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", dao.Id)),
		sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposalID)),
		sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
		sdk.NewAttribute(types.AttributeKeySnapshotID, fmt.Sprintf("%d", snapshotID)),
		sdk.NewAttribute(types.AttributeKeyTotalPower, fmt.Sprintf("%d", total)),
		sdk.NewAttribute(types.AttributeKeyVotingEnd, votingEnd.Format(time.RFC3339Nano)),
		sdk.NewAttribute(types.AttributeKeyDepositCollected, msg.InitialDeposit.String()),
		sdk.NewAttribute(types.AttributeKeyDepositDeadline, depositDeadline.Format(time.RFC3339Nano)),
	))

	return &types.MsgCreateProposalResponse{ProposalId: proposalID}, nil
}
