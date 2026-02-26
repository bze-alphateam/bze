package keeper

import (
	"github.com/bze-alphateam/bze/bzeutils"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	oneWeekInHours = 7 * 24
)

// EnqueueExpiredTradingRewardRemoval checks if any pending trading reward expirations exist for the given epoch
// and enqueues the epoch number for processing. The module will process the queue in the following blocks.
func (k Keeper) EnqueueExpiredTradingRewardRemoval(ctx sdk.Context, epochNumber int64) {
	expirations := k.GetBatchPendingTradingRewardExpirationByExpireAt(ctx, uint32(epochNumber), 1)
	if len(expirations) == 0 {
		return
	}

	queue, found := k.GetTradingRewardExpirationQueue(ctx)
	if !found {
		queue = types.TradingRewardExpirationQueue{RemovalEpochs: make([]uint32, 0)}
	}

	queue.RemovalEpochs = append(queue.RemovalEpochs, uint32(epochNumber))
	k.SetTradingRewardExpirationQueue(ctx, queue)
}

// ProcessExpiredTradingRewardRemovalQueue processes expired pending trading reward removals in bounded batches.
// It collects entries to process in batches to avoid unbounded iteration within a single block.
func (k Keeper) ProcessExpiredTradingRewardRemovalQueue(ctx sdk.Context) {
	queue, found := k.GetTradingRewardExpirationQueue(ctx)
	if !found || len(queue.RemovalEpochs) == 0 {
		return
	}

	type epochBatch struct {
		epoch   uint32
		entries []types.TradingRewardExpiration
	}

	var batches []epochBatch
	finishedEpochs := make(map[uint32]struct{})
	toProcess := types.MaxTradingRewardRemovalsPerBlock

	// Phase 1: collect entries to process (read-only)
	for _, epoch := range queue.RemovalEpochs {
		if toProcess <= 0 {
			break
		}

		entries := k.GetBatchPendingTradingRewardExpirationByExpireAt(ctx, epoch, toProcess)
		if len(entries) == 0 {
			finishedEpochs[epoch] = struct{}{}
			continue
		}

		if len(entries) < toProcess {
			finishedEpochs[epoch] = struct{}{}
		}

		batches = append(batches, epochBatch{epoch: epoch, entries: entries})
		toProcess -= len(entries)
	}

	// Phase 2: process collected entries
	for _, batch := range batches {
		for _, exp := range batch.entries {
			_ = bzeutils.ApplyFuncIfNoError(ctx, func(c sdk.Context) error {
				err := k.processExpiredPendingTradingReward(c, exp)
				if err != nil {
					delete(finishedEpochs, batch.epoch)
				}

				return err
			})
		}
	}

	var remainingEpochs []uint32
	for _, e := range queue.RemovalEpochs {
		if _, ok := finishedEpochs[e]; ok {
			continue
		}

		remainingEpochs = append(remainingEpochs, e)
	}

	queue.RemovalEpochs = remainingEpochs
	k.SetTradingRewardExpirationQueue(ctx, queue)
}

func (k Keeper) processExpiredPendingTradingReward(ctx sdk.Context, exp types.TradingRewardExpiration) error {
	logger := k.Logger().With("trading_reward_expiration", exp)

	k.RemovePendingTradingRewardExpiration(ctx, exp.ExpireAt, exp.RewardId)
	tr, found := k.GetPendingTradingReward(ctx, exp.RewardId)
	if !found {
		logger.Error("trading reward not found for this trading reward expiration")
		// not returning an error here since the expiration was already removed and retrying won't help
		return nil
	}
	k.RemovePendingTradingReward(ctx, exp.RewardId)

	//burn coins that were captured
	toBurn, err := k.getAmountToCapture(tr.PrizeDenom, tr.PrizeAmount.String(), int64(tr.Slots))
	if err != nil {
		logger.With("trading_reward", tr).Error("could not create amount to burn from trading reward")
		return err
	}

	err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, burnermoduletypes.ModuleName, toBurn)
	if err != nil {
		logger.With("trading_reward", tr, "to_burn", toBurn).Error("could not burn coins for trading reward")
		return err
	}

	logger.With("trading_reward", tr, "to_burn", toBurn).
		Debug("removed expired trading reward and burnt the tokens associated with it")

	err = ctx.EventManager().EmitTypedEvent(
		&types.TradingRewardExpireEvent{
			RewardId: exp.RewardId,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return nil
}

func (k Keeper) distributeTradingRewards(ctx sdk.Context, epochNumber int64) {
	logger := k.Logger().With("epoch_number", epochNumber)
	toPay := k.GetAllActiveTradingRewardExpirationByExpireAt(ctx, uint32(epochNumber))
	for _, exp := range toPay {
		tr, found := k.GetActiveTradingReward(ctx, exp.RewardId)
		if !found {
			logger.With("trading_reward_expiration", exp).Error("trading reward not found for this trading reward expiration")
			continue
		}
		logger = logger.With("trading_reward", tr)

		//delete those already paid in a previous epoch that were extended just so we can display them for a few more days
		if int64(tr.ExpireAt) != epochNumber {
			k.RemoveActiveTradingRewardExpiration(ctx, exp.ExpireAt, exp.RewardId)
			k.RemoveActiveTradingReward(ctx, exp.RewardId)
			k.RemoveTradingRewardLeaderboard(ctx, exp.RewardId)
			k.cleanupTradingRewardCandidates(ctx, exp.RewardId)

			continue
		}

		rewardPerSlot, err := k.getAmountToCapture(tr.PrizeDenom, tr.PrizeAmount.String(), 1)
		if err != nil {
			logger.Error("could not get reward per slot")
			continue
		}

		leaderboard, found := k.GetTradingRewardLeaderboard(ctx, tr.RewardId)
		if !found {
			logger.Error("trading reward leaderboard not found")
			continue
		}

		var eventWinners []string
		for _, winner := range leaderboard.List {
			acc, err := sdk.AccAddressFromBech32(winner.Address)
			if err != nil {
				panic(err)
			}

			err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, rewardPerSlot)
			if err != nil {
				//should never happen
				logger.Error("error sending coins to winner", "winner", winner.Address, "err", err.Error())
			}
			eventWinners = append(eventWinners, winner.Address)
		}

		//the trading reward is finished. remove the market id
		k.RemoveMarketIdRewardId(ctx, tr.MarketId)
		//extend the expiration of this paid trading reward for another period just to display the winners
		exp.ExpireAt += oneWeekInHours
		k.SetActiveTradingRewardExpiration(ctx, exp)

		err = ctx.EventManager().EmitTypedEvent(
			&types.TradingRewardDistributionEvent{
				RewardId:    exp.RewardId,
				PrizeAmount: rewardPerSlot.String(),
				PrizeDenom:  tr.PrizeDenom,
				Winners:     eventWinners,
			},
		)

		if err != nil {
			k.Logger().Error(err.Error())
		}
	}
}

func (k Keeper) cleanupTradingRewardCandidates(ctx sdk.Context, rewardId string) {
	toRemove := k.GetTradingRewardCandidateByRewardId(ctx, rewardId)
	for _, trc := range toRemove {
		k.RemoveTradingRewardCandidate(ctx, trc.RewardId, trc.Address)
	}
}
