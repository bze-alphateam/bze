package keeper

import (
	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// UnlockAllPendingUnlockParticipantsByEpoch processes and queues all pending unlock participants for the specified epoch.
// we save the epoch number in the queue. The module will process the queue in the following blocks and after it
// finishes it deletes the epoch from the queue.
func (k Keeper) UnlockAllPendingUnlockParticipantsByEpoch(ctx sdk.Context, epochNumber int64) {
	var hasPendingUnlock bool
	k.IterateAllEpochPendingUnlockParticipant(ctx, epochNumber, func(ctx sdk.Context, sr types.PendingUnlockParticipant) (stop bool) {
		hasPendingUnlock = true
		return true
	})

	if !hasPendingUnlock {
		return
	}

	queue, found := k.GetUnlockParticipantsQueue(ctx)
	if !found {
		queue = types.UnlockParticipantsQueue{UnlockEpochs: make([]uint64, 0)}
	}

	queue.UnlockEpochs = append(queue.UnlockEpochs, uint64(epochNumber))

	k.SetUnlockParticipantsQueue(ctx, queue)
}

// ProcessUnlockParticipantsQueue processes the unlock participants queue by unlocking eligible participants batch-wise.
// It collects entries to be unlocked in batches to avoid deleting while the iterator is open
// and to avoid opening the iterator in safe write ctx (which uses cached context) as per ApplyFuncIfNoError notes.
func (k Keeper) ProcessUnlockParticipantsQueue(ctx sdk.Context) {
	queue, found := k.GetUnlockParticipantsQueue(ctx)
	if !found || len(queue.UnlockEpochs) == 0 {
		return
	}

	type epochBatch struct {
		epoch   uint64
		entries []types.PendingUnlockParticipant
	}

	var batches []epochBatch
	finishedEpochs := make(map[uint64]struct{})
	toProcess := types.MaxUnlocksPerBlock

	// Phase 1: collect entries to unlock
	for _, epoch := range queue.UnlockEpochs {
		if toProcess <= 0 {
			break
		}

		entries := k.GetBatchEpochPendingUnlockParticipant(ctx, int64(epoch), toProcess)
		if len(entries) == 0 {
			finishedEpochs[epoch] = struct{}{}
			continue
		}

		//if we have less than the number of entries to process, we can stop processing this epoch
		if len(entries) < toProcess {
			finishedEpochs[epoch] = struct{}{}
		}

		batches = append(batches, epochBatch{epoch: epoch, entries: entries})
		toProcess -= len(entries)
	}

	// Phase 2: process collected entries
	err := bzeutils.ApplyFuncIfNoError(ctx, func(c sdk.Context) error {
		for _, batch := range batches {
			var hasErrors bool
			for _, p := range batch.entries {
				logger := k.Logger().With("pending_unlock", p)

				err := k.performUnlock(c, &p)
				if err != nil {
					logger.Error(err.Error())
					hasErrors = true
					continue
				}

				k.RemovePendingUnlockParticipant(c, p)
				logger.Debug("pending unlock participant processed successfully")
			}

			// if we encounter an error, we consider the epoch not fully processed
			if hasErrors {
				delete(finishedEpochs, batch.epoch)
			}
		}

		var remainingEpochs []uint64
		for _, e := range queue.UnlockEpochs {
			if _, ok := finishedEpochs[e]; ok {
				continue
			}

			remainingEpochs = append(remainingEpochs, e)
		}

		queue.UnlockEpochs = remainingEpochs
		k.SetUnlockParticipantsQueue(c, queue)

		return nil
	})

	if err != nil {
		k.Logger().Error("error processing unlock participants queue", "err", err)
	}
}

func (k Keeper) performUnlock(ctx sdk.Context, p *types.PendingUnlockParticipant) error {
	partCoins, err := k.getAmountToCapture(p.Denom, p.Amount.String(), int64(1))
	if err != nil {
		return err
	}

	acc, err := sdk.AccAddressFromBech32(p.Address)
	if err != nil {
		return err
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, partCoins)
	if err != nil {
		return err
	}

	return nil
}
