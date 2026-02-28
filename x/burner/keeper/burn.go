package keeper

import (
	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EnqueuePeriodicBurn sets the periodic burn queue as pending.
// If a burn is already pending (mid-processing), it does NOT re-enqueue,
// ensuring we don't restart mid-processing.
func (k Keeper) EnqueuePeriodicBurn(ctx sdk.Context) {
	q, found := k.GetPeriodicBurnQueue(ctx)
	if found && q.Pending {
		return
	}

	k.SetPeriodicBurnQueue(ctx, types.PeriodicBurnQueue{
		Pending: true,
	})
}

// ProcessPeriodicBurnQueue processes the periodic burn queue if active, burning coins in batches and emitting events.
func (k Keeper) ProcessPeriodicBurnQueue(ctx sdk.Context) error {
	queue, found := k.GetPeriodicBurnQueue(ctx)
	if !found || !queue.Pending {
		return nil
	}

	logger := k.Logger().With("method", "ProcessPeriodicBurnQueue")

	moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
	allCoins := k.bankKeeper.GetAllBalances(ctx, moduleAcc.GetAddress())
	if allCoins.IsZero() {
		k.RemovePeriodicBurnQueue(ctx)
		return nil
	}

	// Take at most MaxDenomsBurnPerBlock coins to process
	batch := allCoins
	isLastBatch := true
	if len(allCoins) > types.MaxDenomsBurnPerBlock {
		batch = allCoins[:types.MaxDenomsBurnPerBlock]
		isLastBatch = false
	}

	burned, err := k.BurnAnyCoins(ctx, types.ModuleName, batch)
	if err != nil {
		return err
	}

	if !burned.IsZero() {
		if err = k.SaveBurnedCoins(ctx, burned); err != nil {
			return err
		}

		if err = ctx.EventManager().EmitTypedEvent(&types.CoinsBurnedEvent{Burned: burned.String()}); err != nil {
			return err
		}

		logger.With("coins", burned.String()).Info("coins successfully burned in batch")
	}

	if isLastBatch {
		logger.Info("burning queue is empty, removing it")
		k.RemovePeriodicBurnQueue(ctx)
	}

	return nil
}

// EnqueueRaffleCleanup appends the given epoch number to the raffle cleanup queue.
// It is idempotent: if the epoch is already in the queue, it will not be added again.
func (k Keeper) EnqueueRaffleCleanup(ctx sdk.Context, epochNumber uint64) {
	q, found := k.GetRaffleCleanupQueue(ctx)
	if !found {
		q = types.RaffleCleanupQueue{}
	}

	for _, e := range q.PendingEpochs {
		if e == epochNumber {
			return
		}
	}

	q.PendingEpochs = append(q.PendingEpochs, epochNumber)
	k.SetRaffleCleanupQueue(ctx, q)
}

// ProcessRaffleCleanupQueue processes the raffle cleanup queue in bounded batches.
// It processes at most MaxRafflesCleanupPerBlock raffles per block from the first pending epoch.
func (k Keeper) ProcessRaffleCleanupQueue(ctx sdk.Context) error {
	q, found := k.GetRaffleCleanupQueue(ctx)
	if !found || len(q.PendingEpochs) == 0 {
		return nil
	}

	logger := k.Logger().With("method", "ProcessRaffleCleanupQueue")

	epoch := q.PendingEpochs[0]
	batch := k.GetRaffleDeleteHookByEndAtPrefixBatched(ctx, epoch, types.MaxRafflesCleanupPerBlock)

	if len(batch) == 0 {
		// This epoch is fully processed, pop it from the queue
		q.PendingEpochs = q.PendingEpochs[1:]
		if len(q.PendingEpochs) == 0 {
			k.RemoveRaffleCleanupQueue(ctx)
		} else {
			k.SetRaffleCleanupQueue(ctx, q)
		}
		return nil
	}

	for _, item := range batch {
		itemLogger := logger.With("epoch", epoch, "denom", item.Denom)
		k.RemoveRaffleDeleteHook(ctx, item)
		k.RemoveRaffle(ctx, item.Denom)
		winners := k.GetRaffleWinners(ctx, item.Denom)
		for _, w := range winners {
			k.RemoveRaffleWinner(ctx, w)
		}

		raffleAcc := k.accountKeeper.GetModuleAccount(ctx, types.RaffleModuleName)
		if raffleAcc == nil {
			itemLogger.Error("could not find module account")
			continue
		}

		currentPot := k.bankKeeper.GetBalance(ctx, raffleAcc.GetAddress(), item.Denom)
		if !currentPot.IsPositive() {
			itemLogger.Info("no coins to burn for this raffle that we delete")
			continue
		}

		burned, err := k.BurnAnyCoins(ctx, types.RaffleModuleName, sdk.NewCoins(currentPot))
		if err != nil {
			itemLogger.Error("failed to burn raffle remaining coins", "error", err)
			continue
		}

		if burned.IsZero() {
			itemLogger.Info("no raffle coins to burn")
			continue
		}

		err = k.SaveBurnedCoins(ctx, burned)
		if err != nil {
			itemLogger.Error("failed to save burned coins", "error", err)
		}

		itemLogger.Debug("burned raffle coins", "burned_current_pot", burned)

		err = ctx.EventManager().EmitTypedEvent(&types.RaffleFinishedEvent{Denom: item.Denom})
		if err != nil {
			itemLogger.Error("failed to emit raffle finished event", "error", err)
		}
	}

	// If we processed fewer than the limit, this epoch is done
	if len(batch) < types.MaxRafflesCleanupPerBlock {
		q.PendingEpochs = q.PendingEpochs[1:]
		if len(q.PendingEpochs) == 0 {
			k.RemoveRaffleCleanupQueue(ctx)
		} else {
			k.SetRaffleCleanupQueue(ctx, q)
		}
	}

	return nil
}

// BurnAnyCoins attempts to burn, lock, or exchange specified coins from a module account.
// It directly burns native and token factory denominations, locks LP tokens, and exchanges IBC tokens for native ones.
// It returns the coins that were successfully burned.
func (k Keeper) BurnAnyCoins(ctx sdk.Context, fromModule string, coins sdk.Coins) (sdk.Coins, error) {
	logger := k.Logger().With("method", "BurnAnyCoins", "fromModule", fromModule)
	logger.Debug("iterating through provided coins")

	burnable, exchangeable, lockable := k.filterCoinsToBurn(ctx, coins)
	if len(exchangeable) > 0 {
		logger.Debug("there are coins that can be added to liquidity with native denom")

		//use exchangeable coins to add liquidity with native denom
		added, refunded, err := k.tradeKeeper.ModuleAddLiquidityWithNativeDenom(ctx, fromModule, exchangeable)
		if err != nil {
			logger.Error("error on module add liquidity with native", "error", err)
		} else {
			logger.Debug("added liquidity with native denom", "added", added.String(), "refunded", refunded.String())
		}
	}

	//lock coins in black hole address
	if len(lockable) > 0 {
		err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromModule, types.BlackHoleModuleName, lockable)
		if err != nil {
			return nil, err
		}
		logger.Debug("coins locked", "locked", lockable.String())
	}

	if burnable.IsAllPositive() {
		//burn coins eligible to burn
		err := k.bankKeeper.BurnCoins(ctx, fromModule, burnable)
		if err != nil {
			return nil, err
		}
		logger.Debug("coins burned", "burned", burnable.String())
	}

	return burnable, nil
}

func (k Keeper) filterCoinsToBurn(ctx sdk.Context, toBurn sdk.Coins) (burnable, exchangeable, lockable sdk.Coins) {
	logger := k.Logger().With("method", "filterCoinsToBurn")
	for _, c := range toBurn {
		logger.Debug("checking coin", "coin", c.String())
		if !c.IsPositive() {
			continue
		}

		// native coins can be burned directly
		// factory tokens can be burned directly
		if k.tradeKeeper.IsNativeDenom(ctx, c.Denom) || bzeutils.IsTokenFactoryDenom(c.Denom) {
			logger.Debug("coin is native or factory token, can be burned directly", "coin", c.String())
			burnable = burnable.Add(c)
			continue
		}

		//LP shares cannot be burned, but they can be locked
		if bzeutils.IsLpTokenDenom(c.Denom) {
			logger.Debug("coin is LP token, can be locked", "coin", c.String())
			lockable = lockable.Add(c)
			continue
		}
		//it must be an IBC token

		//not native, not LP share, not token factory -> it should be an IBC denom
		if k.tradeKeeper.CanSwapForNativeDenom(ctx, c) {
			logger.Debug("coin is IBC token, can be exchanged to native coin for burning", "coin", c.String())
			exchangeable = exchangeable.Add(c)
			continue
		}

		logger.Debug("coin is IBC token, cannot be burned or exchanged to native coin. Will lock for now", "coin", c.String())
		lockable = lockable.Add(c)
	}

	return burnable, exchangeable, lockable
}
