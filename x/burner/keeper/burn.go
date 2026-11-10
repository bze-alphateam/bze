package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
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
// There is no cursor: every block re-reads the burner's balances from the start, so every coin of a batch must leave
// the account. BurnAnyCoins guarantees it for exchangeable coins: one that can't be added as liquidity (an error, a
// refund or a panic) is locked instead of being retried at the front of the account forever.
func (k Keeper) ProcessPeriodicBurnQueue(ctx sdk.Context) error {
	queue, found := k.GetPeriodicBurnQueue(ctx)
	if !found || !queue.Pending {
		return nil
	}

	logger := k.Logger().With("method", "ProcessPeriodicBurnQueue")

	moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
	// Read one balance more than a batch, from the start of the account: burned, locked and pooled coins leave it, so
	// the next block starts over without a cursor, and the extra balance only tells whether another batch follows.
	page, _ := k.bankKeeper.GetBalancesPage(ctx, moduleAcc.GetAddress(), "", types.MaxDenomsBurnPerBlock+1)
	if page.IsZero() {
		k.RemovePeriodicBurnQueue(ctx)
		return nil
	}

	// Take at most MaxDenomsBurnPerBlock coins to process
	batch := page
	isLastBatch := true
	if len(page) > types.MaxDenomsBurnPerBlock {
		batch = page[:types.MaxDenomsBurnPerBlock]
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
		// each raffle is cleaned up in its own cached context: one that fails (e.g. a panic on an oversized pot)
		// doesn't discard the others or keep the epoch in the queue
		err := bzeutils.ApplyFuncIfNoError(ctx, func(cached sdk.Context) error {
			k.cleanupRaffle(cached, itemLogger, item)

			return nil
		})
		if err != nil {
			itemLogger.Error("raffle cleanup failed, removing the raffle and locking its pot", "error", err)
			k.cleanupFailedRaffle(ctx, itemLogger, item)
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

// cleanupRaffle removes a finished raffle (its delete hook, the raffle and its winners) and burns what is left in its
// pot. A pot that can't be burned or exchanged is locked by BurnAnyCoins; a burn that returns an error leaves the pot in
// the raffle module.
func (k Keeper) cleanupRaffle(ctx sdk.Context, itemLogger log.Logger, item types.RaffleDeleteHook) {
	k.removeRaffleData(ctx, item)

	raffleAcc := k.accountKeeper.GetModuleAccount(ctx, types.RaffleModuleName)
	if raffleAcc == nil {
		itemLogger.Error("could not find module account")
		return
	}

	currentPot := k.bankKeeper.GetBalance(ctx, raffleAcc.GetAddress(), item.Denom)
	if !currentPot.IsPositive() {
		itemLogger.Info("no coins to burn for this raffle that we delete")
		return
	}

	burned, err := k.BurnAnyCoins(ctx, types.RaffleModuleName, sdk.NewCoins(currentPot))
	if err != nil {
		itemLogger.Error("failed to burn raffle remaining coins", "error", err)
		return
	}

	if burned.IsZero() {
		itemLogger.Info("no raffle coins to burn")
		return
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

// cleanupFailedRaffle is the fallback when cleanupRaffle failed and its writes were discarded: it removes the raffle's
// delete hook, the raffle and its winners, so the item is not retried forever and the epoch completes, then locks the
// pot in the black hole module (in its own cached context; on failure the pot stays in the raffle module).
func (k Keeper) cleanupFailedRaffle(ctx sdk.Context, itemLogger log.Logger, item types.RaffleDeleteHook) {
	k.removeRaffleData(ctx, item)

	err := bzeutils.ApplyFuncIfNoError(ctx, func(cached sdk.Context) error {
		raffleAcc := k.accountKeeper.GetModuleAccount(cached, types.RaffleModuleName)
		if raffleAcc == nil {
			return fmt.Errorf("could not find module account %s", types.RaffleModuleName)
		}

		pot := k.bankKeeper.GetBalance(cached, raffleAcc.GetAddress(), item.Denom)
		if !pot.IsPositive() {
			return nil
		}

		return k.bankKeeper.SendCoinsFromModuleToModule(cached, types.RaffleModuleName, types.BlackHoleModuleName, sdk.NewCoins(pot))
	})
	if err != nil {
		itemLogger.Error("could not lock the pot of the failed raffle", "error", err)
	}
}

// removeRaffleData removes the raffle's delete hook, the raffle and its winners.
func (k Keeper) removeRaffleData(ctx sdk.Context, item types.RaffleDeleteHook) {
	k.RemoveRaffleDeleteHook(ctx, item)
	k.RemoveRaffle(ctx, item.Denom)
	winners := k.GetRaffleWinners(ctx, item.Denom)
	for _, w := range winners {
		k.RemoveRaffleWinner(ctx, w)
	}
}

// BurnAnyCoins attempts to burn, lock, or exchange specified coins from a module account.
// It directly burns native and token factory denominations, locks LP tokens, and exchanges IBC tokens for native ones.
// An exchangeable coin is added as liquidity on its own (see addLiquidityWithNative); whatever of it is not added is
// locked like a non-exchangeable coin, so it never stays in fromModule to be retried every block.
// It returns the coins that were successfully burned.
func (k Keeper) BurnAnyCoins(ctx sdk.Context, fromModule string, coins sdk.Coins) (sdk.Coins, error) {
	logger := k.Logger().With("method", "BurnAnyCoins", "fromModule", fromModule)
	logger.Debug("iterating through provided coins")

	burnable, exchangeable, lockable := k.filterCoinsToBurn(ctx, coins)
	for _, c := range exchangeable {
		//use exchangeable coins to add liquidity with native denom, one coin at a time; what can't be added is locked
		notAdded := k.addLiquidityWithNative(ctx, fromModule, c)
		if notAdded.IsPositive() {
			lockable = lockable.Add(notAdded)
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
		if k.canExchange(ctx, c) {
			logger.Debug("coin is IBC token, can be exchanged to native coin for burning", "coin", c.String())
			exchangeable = exchangeable.Add(c)
			continue
		}

		logger.Debug("coin is IBC token, cannot be burned or exchanged to native coin. Will lock for now", "coin", c.String())
		lockable = lockable.Add(c)
	}

	return burnable, exchangeable, lockable
}

// canExchange reports whether the coin can be swapped for native denom. A check that panics (e.g. on an oversized
// amount) counts as not exchangeable, so the coin is locked instead of failing the whole batch.
func (k Keeper) canExchange(ctx sdk.Context, c sdk.Coin) bool {
	can := false
	err := bzeutils.ApplyFuncIfNoError(ctx, func(cached sdk.Context) error {
		can = k.tradeKeeper.CanSwapForNativeDenom(cached, c)

		return nil
	})

	return err == nil && can
}

// addLiquidityWithNative adds one exchangeable coin as liquidity with native denom, in its own cached context, and
// returns the part of it that is still in fromModule afterwards:
//   - add-liquidity fails (an error, or a panic such as an arithmetic overflow): its writes are discarded and the
//     whole coin is returned;
//   - add-liquidity refunds the coin (it could not be added): the refunded amount is returned.
//
// Other refunds (LP shares, native leftovers) stay in fromModule as before.
func (k Keeper) addLiquidityWithNative(ctx sdk.Context, fromModule string, c sdk.Coin) sdk.Coin {
	logger := k.Logger().With("method", "addLiquidityWithNative", "fromModule", fromModule, "coin", c.String())

	notAdded := sdk.NewCoin(c.Denom, math.ZeroInt())
	err := bzeutils.ApplyFuncIfNoError(ctx, func(cached sdk.Context) error {
		added, refunded, err := k.tradeKeeper.ModuleAddLiquidityWithNativeDenom(cached, fromModule, sdk.NewCoins(c))
		if err != nil {
			return err
		}
		logger.Debug("added liquidity with native denom", "added", added.String(), "refunded", refunded.String())
		notAdded = sdk.NewCoin(c.Denom, refunded.AmountOf(c.Denom))

		return nil
	})
	if err != nil {
		logger.Error("error on module add liquidity with native, locking the coin", "error", err)

		return c
	}

	return notAdded
}
