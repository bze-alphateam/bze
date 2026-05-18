package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// ----- DepositRecord CRUD -----

// SetDepositRecord writes a DepositRecord row. Used by initial deposit at
// proposal creation, by MsgDeposit top-ups, and by genesis import.
func (k Keeper) SetDepositRecord(ctx context.Context, r types.DepositRecord) error {
	addr, err := sdk.AccAddressFromBech32(r.Depositor)
	if err != nil {
		return err
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.DepositRecordKey(r.DaoId, r.ProposalId, addr), k.cdc.MustMarshal(&r))
	return nil
}

// GetDepositRecord returns (record, true) if (dao, proposal, depositor) has
// a stored deposit, or zero/false otherwise.
func (k Keeper) GetDepositRecord(ctx context.Context, daoID, proposalID uint64, depositor sdk.AccAddress) (types.DepositRecord, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.DepositRecordKey(daoID, proposalID, depositor))
	if bz == nil {
		return types.DepositRecord{}, false
	}
	var r types.DepositRecord
	k.cdc.MustUnmarshal(bz, &r)
	return r, true
}

// DeleteDepositRecord removes a row. Called per-depositor as refund /
// forfeit fan-out completes its bank send.
func (k Keeper) DeleteDepositRecord(ctx context.Context, daoID, proposalID uint64, depositor sdk.AccAddress) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Delete(types.DepositRecordKey(daoID, proposalID, depositor))
}

// IterateDepositRecords walks every DepositRecord for a single proposal.
// Used by refund/forfeit fan-out and by genesis export.
func (k Keeper) IterateDepositRecords(ctx context.Context, daoID, proposalID uint64, cb func(types.DepositRecord) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.DepositRecordsIterationPrefix(daoID, proposalID))
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var r types.DepositRecord
		k.cdc.MustUnmarshal(iter.Value(), &r)
		if cb(r) {
			return
		}
	}
}

// IterateAllDepositRecords walks every DepositRecord in the store (across
// all proposals). Used by genesis export.
func (k Keeper) IterateAllDepositRecords(ctx context.Context, cb func(types.DepositRecord) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.DepositRecordKeyPrefix)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var r types.DepositRecord
		k.cdc.MustUnmarshal(iter.Value(), &r)
		if cb(r) {
			return
		}
	}
}

// PaginatedDepositRecords backs the Deposits gRPC query.
func (k Keeper) PaginatedDepositRecords(ctx context.Context, daoID, proposalID uint64, pageReq *query.PageRequest) ([]types.DepositRecord, *query.PageResponse, error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.DepositRecordsIterationPrefix(daoID, proposalID))

	var out []types.DepositRecord
	pageRes, err := query.Paginate(store, pageReq, func(_, value []byte) error {
		var r types.DepositRecord
		if err := k.cdc.Unmarshal(value, &r); err != nil {
			return err
		}
		out = append(out, r)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return out, pageRes, nil
}

// ----- Escrow operations -----

// collectInitialDeposit moves `amount` from `proposer` into the DAO's
// deposit escrow and persists a DepositRecord row. Returns the validated
// record (with normalized depositor bech32) and any bank-send error.
//
// Caller (MsgCreateProposal) has already validated that the denom matches
// the DAO's min_deposit and that amount is sane for the proposer (member
// vs non-member rules).
func (k Keeper) collectInitialDeposit(ctx context.Context, dao types.Dao, proposalID uint64, proposer sdk.AccAddress, amount sdk.Coin) error {
	if amount.IsZero() {
		return nil
	}
	escrow := types.DepositEscrowAddress(dao.Id)
	if err := k.bankKeeper.SendCoins(ctx, proposer, escrow, sdk.NewCoins(amount)); err != nil {
		return fmt.Errorf("collect initial deposit: %w", err)
	}
	return k.SetDepositRecord(ctx, types.DepositRecord{
		DaoId:      dao.Id,
		ProposalId: proposalID,
		Depositor:  proposer.String(),
		Amount:     amount,
	})
}

// collectDeposit is the MsgDeposit top-up path. Sends coins from depositor
// → escrow and upserts the DepositRecord (aggregating into the existing
// row if the same address has deposited before).
func (k Keeper) collectDeposit(ctx context.Context, dao types.Dao, proposalID uint64, depositor sdk.AccAddress, amount sdk.Coin) error {
	if amount.IsZero() {
		return nil
	}
	escrow := types.DepositEscrowAddress(dao.Id)
	if err := k.bankKeeper.SendCoins(ctx, depositor, escrow, sdk.NewCoins(amount)); err != nil {
		return fmt.Errorf("collect deposit: %w", err)
	}
	existing, ok := k.GetDepositRecord(ctx, dao.Id, proposalID, depositor)
	if ok {
		// Denoms must match — collectDeposit's caller already enforced this
		// against the proposal's snapshot, so the .Add is safe.
		newAmt := existing.Amount.Add(amount)
		return k.SetDepositRecord(ctx, types.DepositRecord{
			DaoId:      dao.Id,
			ProposalId: proposalID,
			Depositor:  depositor.String(),
			Amount:     newAmt,
		})
	}
	return k.SetDepositRecord(ctx, types.DepositRecord{
		DaoId:      dao.Id,
		ProposalId: proposalID,
		Depositor:  depositor.String(),
		Amount:     amount,
	})
}

// ----- Refund / forfeit dispatch -----

// handleTerminalDeposits is the single entry point that runs at every
// transition out of DEPOSIT_PERIOD or VOTING. Routes the proposal's
// deposits according to the frozen deposit_snapshot:
//
//   PROPOSAL_STATUS_REJECTED_NO_DEPOSIT → forfeit (the deposit period
//     elapsed under min_deposit; no refund path is available).
//   PROPOSAL_STATUS_PASSED / REJECTED   → apply voting_refund_policy:
//       ALWAYS  → refund every depositor.
//       ON_PASS → refund on PASSED, forfeit on REJECTED.
//       NEVER   → forfeit on both.
//
// On forfeit, ForfeitDestination decides where the coins land (BURNER or
// TREASURY).
//
// After fan-out, every DepositRecord row for the proposal is deleted —
// terminal proposals carry zero records (invariant).
func (k Keeper) handleTerminalDeposits(ctx context.Context, p types.Proposal) error {
	switch p.Status {
	case types.ProposalStatus_PROPOSAL_STATUS_REJECTED_NO_DEPOSIT:
		return k.forfeitAllDeposits(ctx, p)
	case types.ProposalStatus_PROPOSAL_STATUS_PASSED:
		switch p.DepositSnapshot.VotingRefundPolicy {
		case types.RefundPolicy_REFUND_POLICY_ALWAYS,
			types.RefundPolicy_REFUND_POLICY_ON_PASS:
			return k.refundAllDeposits(ctx, p)
		case types.RefundPolicy_REFUND_POLICY_NEVER:
			return k.forfeitAllDeposits(ctx, p)
		}
	case types.ProposalStatus_PROPOSAL_STATUS_REJECTED:
		switch p.DepositSnapshot.VotingRefundPolicy {
		case types.RefundPolicy_REFUND_POLICY_ALWAYS:
			return k.refundAllDeposits(ctx, p)
		case types.RefundPolicy_REFUND_POLICY_ON_PASS,
			types.RefundPolicy_REFUND_POLICY_NEVER:
			return k.forfeitAllDeposits(ctx, p)
		}
	}
	// EXECUTED transitions are Epic-5 territory; we don't re-disburse there.
	return nil
}

// refundAllDeposits returns every depositor's amount via bank.SendCoins,
// then deletes the records. Errors abort and propagate.
func (k Keeper) refundAllDeposits(ctx context.Context, p types.Proposal) error {
	escrow := types.DepositEscrowAddress(p.DaoId)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Collect first to avoid mutating-while-iterating on the same prefix.
	var records []types.DepositRecord
	k.IterateDepositRecords(ctx, p.DaoId, p.ProposalId, func(r types.DepositRecord) bool {
		records = append(records, r)
		return false
	})

	for _, r := range records {
		depositor, err := sdk.AccAddressFromBech32(r.Depositor)
		if err != nil {
			return fmt.Errorf("refund: depositor %q: %w", r.Depositor, err)
		}
		if err := k.bankKeeper.SendCoins(ctx, escrow, depositor, sdk.NewCoins(r.Amount)); err != nil {
			return fmt.Errorf("refund send to %s: %w", r.Depositor, err)
		}
		k.DeleteDepositRecord(ctx, p.DaoId, p.ProposalId, depositor)
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeDepositRefund,
			sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", p.ProposalId)),
			sdk.NewAttribute(types.AttributeKeyDepositor, r.Depositor),
			sdk.NewAttribute(types.AttributeKeyDepositAmount, r.Amount.String()),
		))
	}
	return nil
}

// forfeitAllDeposits routes all records' coins per the proposal's frozen
// ForfeitDestination, then deletes the records.
//
// BURNER:   escrow → bank-from-account-to-module(burner) — burner module
//           processes the burn via its own pipeline.
// TREASURY: escrow → dao.account_address.
//
// We batch into a single send per destination (one per BURNER call, one
// per TREASURY call) rather than per-depositor. That keeps the bank op
// count small. We still walk per-record so events name each depositor.
func (k Keeper) forfeitAllDeposits(ctx context.Context, p types.Proposal) error {
	escrow := types.DepositEscrowAddress(p.DaoId)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	var records []types.DepositRecord
	k.IterateDepositRecords(ctx, p.DaoId, p.ProposalId, func(r types.DepositRecord) bool {
		records = append(records, r)
		return false
	})
	if len(records) == 0 {
		return nil
	}

	total := sdk.NewCoins()
	for _, r := range records {
		total = total.Add(r.Amount)
	}

	switch p.DepositSnapshot.ForfeitDestination {
	case types.ForfeitDestination_FORFEIT_DESTINATION_BURNER:
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, escrow, burnertypes.ModuleName, total); err != nil {
			return fmt.Errorf("forfeit to burner: %w", err)
		}
	case types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY:
		// We don't denormalize dao.account_address onto the proposal —
		// look it up. The DAO record is guaranteed to exist for a proposal
		// in our store (parent invariant).
		dao, ok := k.GetDao(ctx, p.DaoId)
		if !ok {
			return fmt.Errorf("forfeit to treasury: DAO %d not found", p.DaoId)
		}
		treasury, err := sdk.AccAddressFromBech32(dao.AccountAddress)
		if err != nil {
			return fmt.Errorf("forfeit to treasury: invalid dao address: %w", err)
		}
		if err := k.bankKeeper.SendCoins(ctx, escrow, treasury, total); err != nil {
			return fmt.Errorf("forfeit to treasury: %w", err)
		}
	default:
		return fmt.Errorf("forfeit: unknown destination %v", p.DepositSnapshot.ForfeitDestination)
	}

	for _, r := range records {
		depositor, err := sdk.AccAddressFromBech32(r.Depositor)
		if err != nil {
			return fmt.Errorf("forfeit: depositor %q: %w", r.Depositor, err)
		}
		k.DeleteDepositRecord(ctx, p.DaoId, p.ProposalId, depositor)
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeDepositForfeit,
			sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", p.ProposalId)),
			sdk.NewAttribute(types.AttributeKeyDepositor, r.Depositor),
			sdk.NewAttribute(types.AttributeKeyDepositAmount, r.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyForfeitDest, p.DepositSnapshot.ForfeitDestination.String()),
		))
	}
	return nil
}
