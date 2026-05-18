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

// ----- Poll DepositRecord CRUD -----
//
// Mirrors deposit.go (proposal deposits) but writes/reads under the
// PollDepositRecord prefix. Same DepositRecord proto shape; the
// `ProposalId` field stores the poll_id (the proto field is generic over
// the entity id — naming is legacy-from-Epic-4).

func (k Keeper) SetPollDepositRecord(ctx context.Context, r types.DepositRecord) error {
	addr, err := sdk.AccAddressFromBech32(r.Depositor)
	if err != nil {
		return err
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.PollDepositRecordKey(r.DaoId, r.ProposalId, addr), k.cdc.MustMarshal(&r))
	return nil
}

func (k Keeper) GetPollDepositRecord(ctx context.Context, daoID, pollID uint64, depositor sdk.AccAddress) (types.DepositRecord, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.PollDepositRecordKey(daoID, pollID, depositor))
	if bz == nil {
		return types.DepositRecord{}, false
	}
	var r types.DepositRecord
	k.cdc.MustUnmarshal(bz, &r)
	return r, true
}

func (k Keeper) DeletePollDepositRecord(ctx context.Context, daoID, pollID uint64, depositor sdk.AccAddress) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Delete(types.PollDepositRecordKey(daoID, pollID, depositor))
}

func (k Keeper) IteratePollDepositRecords(ctx context.Context, daoID, pollID uint64, cb func(types.DepositRecord) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.PollDepositRecordsIterationPrefix(daoID, pollID))
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

func (k Keeper) IterateAllPollDepositRecords(ctx context.Context, cb func(types.DepositRecord) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.PollDepositRecordKeyPrefix)
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

func (k Keeper) PaginatedPollDepositRecords(ctx context.Context, daoID, pollID uint64, pageReq *query.PageRequest) ([]types.DepositRecord, *query.PageResponse, error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),
		types.PollDepositRecordsIterationPrefix(daoID, pollID))

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

// collectInitialPollDeposit moves `amount` from proposer to the DAO's
// shared escrow and persists a poll DepositRecord. Parallel to
// collectInitialDeposit; escrow address is shared with proposals.
func (k Keeper) collectInitialPollDeposit(ctx context.Context, dao types.Dao, pollID uint64, proposer sdk.AccAddress, amount sdk.Coin) error {
	if amount.IsZero() {
		return nil
	}
	escrow := types.DepositEscrowAddress(dao.Id)
	if err := k.bankKeeper.SendCoins(ctx, proposer, escrow, sdk.NewCoins(amount)); err != nil {
		return fmt.Errorf("collect initial poll deposit: %w", err)
	}
	return k.SetPollDepositRecord(ctx, types.DepositRecord{
		DaoId:      dao.Id,
		ProposalId: pollID, // generic id field
		Depositor:  proposer.String(),
		Amount:     amount,
	})
}

// collectPollDeposit is the MsgDepositOnPoll top-up path.
func (k Keeper) collectPollDeposit(ctx context.Context, dao types.Dao, pollID uint64, depositor sdk.AccAddress, amount sdk.Coin) error {
	if amount.IsZero() {
		return nil
	}
	escrow := types.DepositEscrowAddress(dao.Id)
	if err := k.bankKeeper.SendCoins(ctx, depositor, escrow, sdk.NewCoins(amount)); err != nil {
		return fmt.Errorf("collect poll deposit: %w", err)
	}
	existing, ok := k.GetPollDepositRecord(ctx, dao.Id, pollID, depositor)
	if ok {
		newAmt := existing.Amount.Add(amount)
		return k.SetPollDepositRecord(ctx, types.DepositRecord{
			DaoId:      dao.Id,
			ProposalId: pollID,
			Depositor:  depositor.String(),
			Amount:     newAmt,
		})
	}
	return k.SetPollDepositRecord(ctx, types.DepositRecord{
		DaoId:      dao.Id,
		ProposalId: pollID,
		Depositor:  depositor.String(),
		Amount:     amount,
	})
}

// ----- Refund / forfeit dispatch -----

// handlePollTerminalDeposits routes poll deposits at terminal status per
// the poll's frozen deposit_snapshot. Parallel to handleTerminalDeposits.
//
//   REJECTED_NO_DEPOSIT → forfeit (deposit period elapsed underfunded).
//   CONCLUDED / REJECTED → apply voting_refund_policy:
//       ALWAYS  → refund.
//       ON_PASS → refund on CONCLUDED, forfeit on REJECTED.
//       NEVER   → forfeit on both.
func (k Keeper) handlePollTerminalDeposits(ctx context.Context, p types.Poll) error {
	switch p.Status {
	case types.PollStatus_POLL_STATUS_REJECTED_NO_DEPOSIT:
		return k.forfeitAllPollDeposits(ctx, p)
	case types.PollStatus_POLL_STATUS_CONCLUDED:
		switch p.DepositSnapshot.VotingRefundPolicy {
		case types.RefundPolicy_REFUND_POLICY_ALWAYS,
			types.RefundPolicy_REFUND_POLICY_ON_PASS:
			return k.refundAllPollDeposits(ctx, p)
		case types.RefundPolicy_REFUND_POLICY_NEVER:
			return k.forfeitAllPollDeposits(ctx, p)
		}
	case types.PollStatus_POLL_STATUS_REJECTED:
		switch p.DepositSnapshot.VotingRefundPolicy {
		case types.RefundPolicy_REFUND_POLICY_ALWAYS:
			return k.refundAllPollDeposits(ctx, p)
		case types.RefundPolicy_REFUND_POLICY_ON_PASS,
			types.RefundPolicy_REFUND_POLICY_NEVER:
			return k.forfeitAllPollDeposits(ctx, p)
		}
	}
	return nil
}

func (k Keeper) refundAllPollDeposits(ctx context.Context, p types.Poll) error {
	escrow := types.DepositEscrowAddress(p.DaoId)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	var records []types.DepositRecord
	k.IteratePollDepositRecords(ctx, p.DaoId, p.PollId, func(r types.DepositRecord) bool {
		records = append(records, r)
		return false
	})

	for _, r := range records {
		depositor, err := sdk.AccAddressFromBech32(r.Depositor)
		if err != nil {
			return fmt.Errorf("poll refund: depositor %q: %w", r.Depositor, err)
		}
		if err := k.bankKeeper.SendCoins(ctx, escrow, depositor, sdk.NewCoins(r.Amount)); err != nil {
			return fmt.Errorf("poll refund send to %s: %w", r.Depositor, err)
		}
		k.DeletePollDepositRecord(ctx, p.DaoId, p.PollId, depositor)
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeDepositRefund,
			sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
			sdk.NewAttribute(types.AttributeKeyPollID, fmt.Sprintf("%d", p.PollId)),
			sdk.NewAttribute(types.AttributeKeyDepositor, r.Depositor),
			sdk.NewAttribute(types.AttributeKeyDepositAmount, r.Amount.String()),
		))
	}
	return nil
}

func (k Keeper) forfeitAllPollDeposits(ctx context.Context, p types.Poll) error {
	escrow := types.DepositEscrowAddress(p.DaoId)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	var records []types.DepositRecord
	k.IteratePollDepositRecords(ctx, p.DaoId, p.PollId, func(r types.DepositRecord) bool {
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
			return fmt.Errorf("poll forfeit to burner: %w", err)
		}
	case types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY:
		dao, ok := k.GetDao(ctx, p.DaoId)
		if !ok {
			return fmt.Errorf("poll forfeit to treasury: DAO %d not found", p.DaoId)
		}
		treasury, err := sdk.AccAddressFromBech32(dao.AccountAddress)
		if err != nil {
			return fmt.Errorf("poll forfeit to treasury: invalid dao address: %w", err)
		}
		if err := k.bankKeeper.SendCoins(ctx, escrow, treasury, total); err != nil {
			return fmt.Errorf("poll forfeit to treasury: %w", err)
		}
	default:
		return fmt.Errorf("poll forfeit: unknown destination %v", p.DepositSnapshot.ForfeitDestination)
	}

	for _, r := range records {
		depositor, err := sdk.AccAddressFromBech32(r.Depositor)
		if err != nil {
			return fmt.Errorf("poll forfeit: depositor %q: %w", r.Depositor, err)
		}
		k.DeletePollDepositRecord(ctx, p.DaoId, p.PollId, depositor)
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeDepositForfeit,
			sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", p.DaoId)),
			sdk.NewAttribute(types.AttributeKeyPollID, fmt.Sprintf("%d", p.PollId)),
			sdk.NewAttribute(types.AttributeKeyDepositor, r.Depositor),
			sdk.NewAttribute(types.AttributeKeyDepositAmount, r.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyForfeitDest, p.DepositSnapshot.ForfeitDestination.String()),
		))
	}
	return nil
}
